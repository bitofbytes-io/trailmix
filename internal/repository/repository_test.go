package repository

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/dbtest"
	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
)

func TestSeedCoversSampleTrailsAndDogRules(t *testing.T) {
	ctx := context.Background()
	trailsRepo := NewTrailRepository(dbtest.Pool(t))
	trails, err := trailsRepo.ListTrails(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(trails) != 12 {
		t.Fatalf("seeded %d trails, want 12", len(trails))
	}
	seen := map[model.DogRuleCode]int{}
	for _, trail := range trails {
		if !trail.Sample {
			t.Fatalf("%s is not marked sample", trail.Name)
		}
		if trail.Source != "sample-fixture" {
			t.Fatalf("%s source = %q", trail.Name, trail.Source)
		}
		if _, err := trail.Geometry.Points(); err != nil {
			t.Fatalf("%s geometry: %v", trail.Name, err)
		}
		seen[trail.DogRule.Code]++
	}
	for _, rule := range []model.DogRuleCode{
		model.DogAllowedLeashed,
		model.DogAllowedOffleash,
		model.DogProhibited,
		model.DogUnknown,
	} {
		if seen[rule] == 0 {
			t.Fatalf("seed has no %s trail", rule)
		}
	}

	rules, err := trailsRepo.ListDogRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 4 {
		t.Fatalf("dog rules = %d, want 4", len(rules))
	}
	parks, err := trailsRepo.ListParks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(parks) < 10 {
		t.Fatalf("parks = %d, want at least 10", len(parks))
	}

	random, err := trailsRepo.RandomTrail(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !random.Sample {
		t.Fatal("random trail was not a sample")
	}
}

func TestLogEntryPhotoRoundTrip(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	trailsRepo := NewTrailRepository(pool)
	logs := NewLogRepository(pool)
	trails, err := trailsRepo.ListTrails(ctx)
	if err != nil {
		t.Fatal(err)
	}
	visited := model.Date{Year: 2026, Month: time.October, Day: 20}
	minutes := 45
	id, err := logs.CreateEntry(ctx, EntryInput{
		TrailID:         trails[0].ID,
		VisitedOn:       visited,
		Notes:           "Shady, muddy near the lake.",
		DurationMinutes: &minutes,
		Photo:           &Photo{ContentType: "image/png", Data: tinyPNG(t)},
	})
	if err != nil {
		t.Fatal(err)
	}

	entry, err := logs.GetEntry(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Notes != "Shady, muddy near the lake." || !entry.HasPhoto || entry.Trail.ID != trails[0].ID {
		t.Fatalf("entry = %+v", entry)
	}
	contentType, data, err := logs.EntryPhoto(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "image/png" || len(data) == 0 {
		t.Fatalf("photo type %q len %d", contentType, len(data))
	}

	if err := logs.UpdateEntry(ctx, id, EntryInput{
		VisitedOn: visited,
		Notes:     "Drier on the way back.",
		Photo:     &Photo{Clear: true},
	}); err != nil {
		t.Fatal(err)
	}
	entry, err = logs.GetEntry(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if entry.HasPhoto || entry.Notes != "Drier on the way back." {
		t.Fatalf("after clear = %+v", entry)
	}
	if _, _, err := logs.EntryPhoto(ctx, id); err != ErrNotFound {
		t.Fatalf("cleared photo error = %v", err)
	}

	entries, err := logs.ListEntries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != id {
		t.Fatalf("list = %+v", entries)
	}
	if err := logs.DeleteEntry(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := logs.GetEntry(ctx, id); err != ErrNotFound {
		t.Fatalf("deleted get = %v", err)
	}
}

func TestMissingTrailAndEntry(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.Pool(t)
	if _, err := NewTrailRepository(pool).GetTrail(ctx, uuid.New()); err != ErrNotFound {
		t.Fatalf("missing trail = %v", err)
	}
	if _, err := NewLogRepository(pool).CreateEntry(ctx, EntryInput{
		TrailID:   uuid.New(),
		VisitedOn: model.Date{Year: 2026, Month: time.January, Day: 2},
	}); err != ErrNotFound {
		t.Fatalf("missing trail create = %v", err)
	}
}

func TestVisitDateSurvivesNonUTCSession(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.PoolInTimeZone(t, "America/Los_Angeles")
	var zone string
	if err := pool.QueryRow(ctx, "SHOW timezone").Scan(&zone); err != nil {
		t.Fatal(err)
	}
	if zone != "America/Los_Angeles" {
		t.Fatalf("session timezone = %q", zone)
	}

	trails, err := NewTrailRepository(pool).ListTrails(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logs := NewLogRepository(pool)
	day := model.Date{Year: 2026, Month: time.November, Day: 2}
	id, err := logs.CreateEntry(ctx, EntryInput{TrailID: trails[0].ID, VisitedOn: day, Notes: "west of UTC"})
	if err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := pool.QueryRow(ctx, "SELECT visited_on::text FROM log_entries WHERE id = $1", id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "2026-11-02" {
		t.Fatalf("stored date = %s, want 2026-11-02", stored)
	}
	entry, err := logs.GetEntry(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if entry.VisitedOn != day {
		t.Fatalf("read date = %s", entry.VisitedOn)
	}

	updated := model.Date{Year: 2026, Month: time.January, Day: 31}
	if err := logs.UpdateEntry(ctx, id, EntryInput{VisitedOn: updated, Notes: "still January"}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT visited_on::text FROM log_entries WHERE id = $1", id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "2026-01-31" {
		t.Fatalf("updated date = %s", stored)
	}
	listed, err := logs.ListEntries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].VisitedOn != updated {
		t.Fatalf("listed date = %+v", listed)
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
