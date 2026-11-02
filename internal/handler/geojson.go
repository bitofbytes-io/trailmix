package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/bitofbytes-io/trailmix/internal/filter"
	"github.com/bitofbytes-io/trailmix/internal/format"
	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/bitofbytes-io/trailmix/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var errBadSelected = errors.New("Selected trail is not a valid id.")

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Geometry   model.LineString `json:"geometry"`
	Properties properties       `json:"properties"`
}

type properties struct {
	Name          string  `json:"name"`
	Href          string  `json:"href"`
	Park          string  `json:"park"`
	County        string  `json:"county"`
	DistanceMiles float64 `json:"distance_miles"`
	Distance      string  `json:"distance"`
	Difficulty    string  `json:"difficulty"`
	Dog           string  `json:"dog"`
	Logged        string  `json:"logged"`
	Selected      string  `json:"selected"`
	TrailheadLng  float64 `json:"trailhead_lng"`
	TrailheadLat  float64 `json:"trailhead_lat"`
}

// TrailsGeoJSON serves the trails currently shown by Find trails.
// The query string uses the same filters. selected=<uuid> emphasizes one line.
func (h *Handler) TrailsGeoJSON(w http.ResponseWriter, r *http.Request) {
	trails, err := h.trails.ListTrails(r.Context())
	if err != nil {
		h.failJSON(w, r, err)
		return
	}
	query, err := filter.Parse(r.URL.Query())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	selected, err := selectedTrailID(r.URL.Query())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	logged, err := h.loggedTrailIDs(r.Context())
	if err != nil {
		h.failJSON(w, r, err)
		return
	}
	writeGeoJSON(w, http.StatusOK, trailsFeatureCollection(filter.Apply(trails, query), logged, selected))
}

// TrailGeoJSON serves one trail. The detail map uses it.
func (h *Handler) TrailGeoJSON(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "That link is not valid.")
		return
	}
	trail, err := h.trails.GetTrail(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "That trail is not in the catalog.")
		return
	}
	if err != nil {
		h.failJSON(w, r, err)
		return
	}
	logged, err := h.loggedTrailIDs(r.Context())
	if err != nil {
		h.failJSON(w, r, err)
		return
	}
	writeGeoJSON(w, http.StatusOK, trailsFeatureCollection([]model.Trail{trail}, logged, trail.ID))
}

func (h *Handler) loggedTrailIDs(ctx context.Context) (map[uuid.UUID]struct{}, error) {
	entries, err := h.logs.ListEntries(ctx)
	if err != nil {
		return nil, err
	}
	ids := make(map[uuid.UUID]struct{}, len(entries))
	for _, entry := range entries {
		ids[entry.Trail.ID] = struct{}{}
	}
	return ids, nil
}

func (h *Handler) failJSON(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "error", err, "path", r.URL.Path)
	writeJSONError(w, http.StatusInternalServerError, "Something went wrong.")
}

func trailsFeatureCollection(trails []model.Trail, logged map[uuid.UUID]struct{}, selected uuid.UUID) featureCollection {
	features := make([]feature, len(trails))
	for i, trail := range trails {
		_, isLogged := logged[trail.ID]
		features[i] = feature{
			Type:     "Feature",
			ID:       trail.ID.String(),
			Geometry: trail.Geometry,
			Properties: properties{
				Name:          trail.Name,
				Href:          "/trails/" + trail.ID.String(),
				Park:          trail.Park.Name,
				County:        trail.Park.County,
				DistanceMiles: trail.DistanceMiles,
				Distance:      format.Miles(trail.DistanceMiles),
				Difficulty:    trail.Difficulty.Label(),
				Dog:           dogLabel(trail.DogRule),
				Logged:        yesNo(isLogged),
				Selected:      yesNo(selected != uuid.Nil && trail.ID == selected),
				TrailheadLng:  trail.Trailhead.Lon,
				TrailheadLat:  trail.Trailhead.Lat,
			},
		}
	}
	return featureCollection{Type: "FeatureCollection", Features: features}
}

func collectionURL(values url.Values, selected uuid.UUID) string {
	query := url.Values{}
	if values != nil {
		for _, key := range []string{"q", "max_distance", "difficulty", "max_grade", "park", "dog"} {
			if value := strings.TrimSpace(values.Get(key)); value != "" {
				query.Set(key, value)
			}
		}
	}
	if selected != uuid.Nil {
		query.Set("selected", selected.String())
	}
	if encoded := query.Encode(); encoded != "" {
		return "/api/trails.geojson?" + encoded
	}
	return "/api/trails.geojson"
}

func trailGeoJSONURL(id uuid.UUID) string {
	return "/api/trails/" + id.String() + "/geojson"
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func selectedTrailID(values url.Values) (uuid.UUID, error) {
	raw := strings.TrimSpace(values.Get("selected"))
	if raw == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errBadSelected
	}
	return id, nil
}

func writeGeoJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, status, body)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Debug("write json", "error", err)
	}
}
