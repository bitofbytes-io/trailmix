package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bitofbytes-io/trailmix/internal/config"
	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/bitofbytes-io/trailmix/internal/repository"
	"github.com/google/uuid"
)

func TestRouterHomeSearchAndRandom(t *testing.T) {
	bond := fakeTrail("Bond Park Lake Trail", model.DogAllowedLeashed, 2.1)
	bond.Park.Name = "Fred G. Bond Metro Park"
	hemlock := fakeTrail("Hemlock Bluffs Loop", model.DogProhibited, 1.3)
	hemlock.Park.Name = "Hemlock Bluffs Nature Preserve"
	store := &fakeTrails{trails: []model.Trail{bond, hemlock}}
	logs := &fakeLogs{}
	router := New(&config.Config{}, pingOK{}, store, logs, nil).Router()

	home := get(t, router, "/")
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "Random trail near me") || !strings.Contains(home.Body.String(), "No hikes logged yet") {
		t.Fatalf("home = %d %s", home.Code, home.Body.String())
	}

	filtered := get(t, router, "/trails?dog=prohibited")
	body := filtered.Body.String()
	if filtered.Code != http.StatusOK || !strings.Contains(body, "Hemlock Bluffs Loop") || strings.Contains(body, "Bond Park Lake Trail") {
		t.Fatalf("filtered = %d %s", filtered.Code, body)
	}

	random := get(t, router, "/trails/random")
	if random.Code != http.StatusFound || !strings.HasPrefix(random.Header().Get("Location"), "/trails/") {
		t.Fatalf("random = %d %s", random.Code, random.Header().Get("Location"))
	}
	if strings.Contains(random.Header().Get("Location"), "random") {
		t.Fatal("random route was handled as a trail id")
	}

	blocked := httptest.NewRequest(http.MethodPost, "/trails/"+bond.ID.String()+"/log", nil)
	blocked.Host = "example.test"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, blocked)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin post = %d", rec.Code)
	}

	health := get(t, router, "/health")
	if health.Code != http.StatusOK || health.Body.String() != "ok" {
		t.Fatalf("health = %d %q", health.Code, health.Body.String())
	}
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type pingOK struct{}

func (pingOK) Ping(context.Context) error { return nil }

func fakeTrail(name string, dog model.DogRuleCode, miles float64) model.Trail {
	return model.Trail{
		ID:            uuid.New(),
		Name:          name,
		DistanceMiles: miles,
		GradePercent:  1,
		Difficulty:    model.DifficultyEasy,
		Sample:        true,
		Source:        "sample-fixture",
		Confidence:    0.7,
		Park:          model.Park{ID: uuid.New(), Name: name + " Park", County: "Wake"},
		DogRule:       model.DogRule{Code: dog, Label: dog.Label()},
		Geometry: model.LineString{
			Type:        "LineString",
			Coordinates: [][]float64{{-78.8, 35.7}, {-78.7, 35.8}},
		},
	}
}

type fakeTrails struct {
	trails []model.Trail
}

func (f *fakeTrails) ListTrails(context.Context) ([]model.Trail, error) {
	return f.trails, nil
}

func (f *fakeTrails) ListParks(context.Context) ([]model.Park, error) {
	parks := make([]model.Park, len(f.trails))
	for i, trail := range f.trails {
		parks[i] = trail.Park
	}
	return parks, nil
}

func (f *fakeTrails) ListDogRules(context.Context) ([]model.DogRule, error) {
	return []model.DogRule{
		{Code: model.DogAllowedLeashed, Label: model.DogAllowedLeashed.Label()},
		{Code: model.DogAllowedOffleash, Label: model.DogAllowedOffleash.Label()},
		{Code: model.DogProhibited, Label: model.DogProhibited.Label()},
		{Code: model.DogUnknown, Label: model.DogUnknown.Label()},
	}, nil
}

func (f *fakeTrails) GetTrail(_ context.Context, id uuid.UUID) (model.Trail, error) {
	for _, trail := range f.trails {
		if trail.ID == id {
			return trail, nil
		}
	}
	return model.Trail{}, repository.ErrNotFound
}

func (f *fakeTrails) RandomTrail(context.Context) (model.Trail, error) {
	if len(f.trails) == 0 {
		return model.Trail{}, repository.ErrNotFound
	}
	return f.trails[0], nil
}

type fakeLogs struct {
	entries []model.LogEntry
}

func (f *fakeLogs) ListEntries(context.Context) ([]model.LogEntry, error) {
	return f.entries, nil
}

func (f *fakeLogs) GetEntry(_ context.Context, id uuid.UUID) (model.LogEntry, error) {
	for _, entry := range f.entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return model.LogEntry{}, repository.ErrNotFound
}

func (f *fakeLogs) CreateEntry(_ context.Context, in repository.EntryInput) (uuid.UUID, error) {
	id := uuid.New()
	f.entries = append(f.entries, model.LogEntry{ID: id, Trail: model.Trail{ID: in.TrailID}, Notes: in.Notes, VisitedOn: in.VisitedOn})
	return id, nil
}

func (f *fakeLogs) UpdateEntry(context.Context, uuid.UUID, repository.EntryInput) error {
	return nil
}

func (f *fakeLogs) DeleteEntry(context.Context, uuid.UUID) error { return nil }

func (f *fakeLogs) EntryPhoto(context.Context, uuid.UUID) (string, []byte, error) {
	return "", nil, repository.ErrNotFound
}
