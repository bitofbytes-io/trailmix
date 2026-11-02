package server

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if !strings.Contains(home.Body.String(), "/api/trails.geojson") || !strings.Contains(home.Body.String(), "OpenStreetMap") {
		t.Fatalf("home map = %s", home.Body.String())
	}

	filtered := get(t, router, "/trails?dog=prohibited")
	body := filtered.Body.String()
	if filtered.Code != http.StatusOK || !strings.Contains(body, "Hemlock Bluffs Loop") || strings.Contains(body, "Bond Park Lake Trail") {
		t.Fatalf("filtered = %d %s", filtered.Code, body)
	}
	if !strings.Contains(body, "dog=prohibited") || !strings.Contains(body, config.DefaultMapStyleURL) {
		t.Fatalf("filtered map did not keep the filter or the default style\n%s", body)
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

func TestRejectedFormsKeepTheDraft(t *testing.T) {
	trail := fakeTrail("Hemlock Bluffs Loop", model.DogProhibited, 1.3)
	store := &fakeTrails{trails: []model.Trail{trail}}
	logs := &fakeLogs{}
	router := New(&config.Config{}, pingOK{}, store, logs, nil).Router()

	created := postForm(t, router, "/trails/"+trail.ID.String()+"/log", map[string]string{
		"visited_on":       "2099-01-01",
		"notes":            "muddy creek",
		"duration_minutes": "40",
	})
	if created.Code != http.StatusBadRequest {
		t.Fatalf("create status = %d", created.Code)
	}
	body := created.Body.String()
	for _, want := range []string{"muddy creek", "2099-01-01", "future", `value="40"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("create response missing %q\n%s", want, body)
		}
	}
	if len(logs.entries) != 0 {
		t.Fatal("rejected create was stored")
	}

	saved := model.LogEntry{
		ID:        uuid.New(),
		Trail:     trail,
		VisitedOn: model.Date{Year: 2026, Month: time.October, Day: 1},
		Notes:     "old notes",
		HasPhoto:  true,
	}
	logs.entries = []model.LogEntry{saved}
	edited := postForm(t, router, "/log/"+saved.ID.String(), map[string]string{
		"visited_on":       "2026-10-20",
		"notes":            "fresh mud",
		"duration_minutes": "nope",
		"remove_photo":     "1",
	})
	if edited.Code != http.StatusBadRequest {
		t.Fatalf("edit status = %d", edited.Code)
	}
	body = edited.Body.String()
	for _, want := range []string{"fresh mud", "2026-10-20", `value="nope"`, "whole number", "checked"} {
		if !strings.Contains(body, want) {
			t.Fatalf("edit response missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "old notes") {
		t.Fatal("edit response kept the saved notes instead of the draft")
	}
	if logs.entries[0].Notes != "old notes" {
		t.Fatal("rejected edit changed the stored notes")
	}
}

func postForm(t *testing.T, handler http.Handler, path string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.test"+path, &body)
	req.Host = "example.test"
	req.Header.Set("Origin", "http://example.test")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
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
	coordinates := [][]float64{{-78.8, 35.7}, {-78.7, 35.8}}
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
			Coordinates: coordinates,
		},
		Trailhead: model.Point{Lon: coordinates[0][0], Lat: coordinates[0][1]},
	}
}

func TestTrailsGeoJSONFiltersLoggedAndDetail(t *testing.T) {
	bond := fakeTrail("Bond Park Lake Trail", model.DogAllowedLeashed, 2.1)
	hemlock := fakeTrail("Hemlock Bluffs Loop", model.DogProhibited, 1.3)
	logs := &fakeLogs{entries: []model.LogEntry{{ID: uuid.New(), Trail: bond}}}
	router := New(&config.Config{}, pingOK{}, &fakeTrails{trails: []model.Trail{bond, hemlock}}, logs, nil).Router()

	all := decodeFeatures(t, get(t, router, "/api/trails.geojson"))
	if all.Type != "FeatureCollection" || len(all.Features) != 2 {
		t.Fatalf("all = %+v", all)
	}
	byName := map[string]geoFeature{}
	for _, feature := range all.Features {
		byName[feature.Properties.Name] = feature
		if feature.Geometry.Type != "LineString" || len(feature.Geometry.Coordinates) < 2 {
			t.Fatalf("geometry = %+v", feature.Geometry)
		}
		if feature.ID != feature.Properties.Href[len("/trails/"):] {
			t.Fatalf("id %s href %s", feature.ID, feature.Properties.Href)
		}
		if feature.Properties.TrailheadLng != feature.Geometry.Coordinates[0][0] || feature.Properties.TrailheadLat != feature.Geometry.Coordinates[0][1] {
			t.Fatalf("trailhead = %v,%v", feature.Properties.TrailheadLng, feature.Properties.TrailheadLat)
		}
	}
	if byName[bond.Name].Properties.Logged != "yes" || byName[hemlock.Name].Properties.Logged != "no" {
		t.Fatalf("logged flags = %+v %+v", byName[bond.Name].Properties, byName[hemlock.Name].Properties)
	}
	if byName[bond.Name].Properties.Distance != "2.1 mi" {
		t.Fatalf("distance = %q", byName[bond.Name].Properties.Distance)
	}

	filtered := decodeFeatures(t, get(t, router, "/api/trails.geojson?dog=prohibited&max_distance=2"))
	if len(filtered.Features) != 1 || filtered.Features[0].Properties.Name != hemlock.Name {
		t.Fatalf("filtered = %+v", filtered.Features)
	}

	selected := decodeFeatures(t, get(t, router, "/api/trails.geojson?selected="+bond.ID.String()))
	for _, feature := range selected.Features {
		want := "no"
		if feature.ID == bond.ID.String() {
			want = "yes"
		}
		if feature.Properties.Selected != want {
			t.Fatalf("selected %s = %v", feature.Properties.Name, feature.Properties.Selected)
		}
	}

	empty := decodeFeatures(t, get(t, router, "/api/trails.geojson?q=zzzz"))
	if len(empty.Features) != 0 {
		t.Fatalf("empty search = %+v", empty.Features)
	}

	one := decodeFeatures(t, get(t, router, "/api/trails/"+hemlock.ID.String()+"/geojson"))
	if len(one.Features) != 1 || one.Features[0].Properties.Selected != "yes" || one.Features[0].Properties.Logged != "no" {
		t.Fatalf("detail = %+v", one.Features)
	}
	if one.Features[0].Properties.Href != "/trails/"+hemlock.ID.String() {
		t.Fatalf("href = %s", one.Features[0].Properties.Href)
	}

	missing := get(t, router, "/api/trails/"+uuid.NewString()+"/geojson")
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("missing = %d %s", missing.Code, missing.Header().Get("Content-Type"))
	}
	badID := get(t, router, "/api/trails/not-a-uuid/geojson")
	if badID.Code != http.StatusBadRequest {
		t.Fatalf("bad id = %d", badID.Code)
	}
	badFilter := get(t, router, "/api/trails.geojson?max_distance=nope")
	if badFilter.Code != http.StatusBadRequest || !strings.Contains(badFilter.Body.String(), "Distance") {
		t.Fatalf("bad filter = %d %s", badFilter.Code, badFilter.Body.String())
	}

	entryID := logs.entries[0].ID
	home := get(t, router, "/?entry="+entryID.String())
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "selected="+bond.ID.String()) {
		t.Fatalf("home selected map missing\n%s", home.Body.String())
	}
	detail := get(t, router, "/trails/"+bond.ID.String())
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "/api/trails/"+bond.ID.String()+"/geojson") {
		t.Fatalf("detail map url missing\n%s", detail.Body.String())
	}
}

type geoFeature struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Geometry   model.LineString `json:"geometry"`
	Properties struct {
		Name         string  `json:"name"`
		Href         string  `json:"href"`
		Distance     string  `json:"distance"`
		Logged       string  `json:"logged"`
		Selected     string  `json:"selected"`
		TrailheadLng float64 `json:"trailhead_lng"`
		TrailheadLat float64 `json:"trailhead_lat"`
	} `json:"properties"`
}

type geoCollection struct {
	Type     string       `json:"type"`
	Features []geoFeature `json:"features"`
}

func decodeFeatures(t *testing.T, rec *httptest.ResponseRecorder) geoCollection {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/geo+json") {
		t.Fatalf("content type = %s", rec.Header().Get("Content-Type"))
	}
	var collection geoCollection
	if err := json.Unmarshal(rec.Body.Bytes(), &collection); err != nil {
		t.Fatal(err)
	}
	return collection
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
