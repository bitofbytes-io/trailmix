// Package handler serves the trail catalog and the personal log.
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/a-h/templ"
	"github.com/bitofbytes-io/trailmix/internal/filter"
	"github.com/bitofbytes-io/trailmix/internal/mapview"
	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/bitofbytes-io/trailmix/internal/repository"
	"github.com/bitofbytes-io/trailmix/internal/ui/pages"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	maxPhotoBytes = 2 << 20
	maxBodyBytes  = 3 << 20
	maxNotesRunes = 4000
)

// Pinger is the database check used by /ready.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler renders Trail Mix pages.
type Handler struct {
	trails repository.TrailStore
	logs   repository.LogStore
	maps   mapview.Renderer
	now    func() time.Time
}

// New returns a handler. maps may be the placeholder renderer.
func New(trails repository.TrailStore, logs repository.LogStore, maps mapview.Renderer) *Handler {
	return &Handler{trails: trails, logs: logs, maps: maps, now: time.Now}
}

// Home is the log: a random-trail button, the map, the list, and one entry.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	entries, err := h.logs.ListEntries(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	var selected *model.LogEntry
	if raw := r.URL.Query().Get("entry"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			h.render(w, r, http.StatusBadRequest, pages.NoticePage("Bad link", "That entry link is not valid."))
			return
		}
		entry, err := h.logs.GetEntry(r.Context(), id)
		if errors.Is(err, repository.ErrNotFound) {
			h.notFound(w, r, "That log entry is not here.")
			return
		}
		if err != nil {
			h.fail(w, r, err)
			return
		}
		selected = &entry
	} else if len(entries) > 0 {
		selected = &entries[0]
	}

	selectedTrail := uuid.Nil
	if selected != nil {
		selectedTrail = selected.Trail.ID
	}

	view := pages.HomeView{Map: h.maps.Render(collectionURL(nil, selectedTrail))}
	selectedID := uuid.Nil
	if selected != nil {
		selectedID = selected.ID
		view.HasSelected = true
		view.Selected = presentLog(*selected, selectedID)
	}
	view.Entries = make([]pages.LogItem, len(entries))
	for i, entry := range entries {
		view.Entries[i] = presentLog(entry, selectedID)
	}
	h.render(w, r, http.StatusOK, pages.HomePage(view))
}

// Search lists trails that match the filter form.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	catalog, parks, rules, err := h.catalog(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	query, parseErr := filter.Parse(r.URL.Query())
	shown := catalog
	summary := filter.Describe(query)
	errText := ""
	if parseErr != nil {
		errText = parseErr.Error()
		summary = "Showing every sample trail."
	} else {
		shown = filter.Apply(catalog, query)
	}
	values := r.URL.Query()
	mapValues := url.Values(nil)
	if parseErr == nil {
		mapValues = values
	}
	view := pages.SearchView{
		QueryText:   values.Get("q"),
		MaxDistance: values.Get("max_distance"),
		Difficulty:  values.Get("difficulty"),
		MaxGrade:    values.Get("max_grade"),
		ParkID:      values.Get("park"),
		Dog:         values.Get("dog"),
		Parks:       presentParks(parks),
		DogRules:    presentRules(rules),
		Trails:      presentTrails(shown),
		Summary:     summary,
		Error:       errText,
		Map:         h.maps.Render(collectionURL(mapValues, uuid.Nil)),
	}
	status := http.StatusOK
	if parseErr != nil {
		status = http.StatusBadRequest
	}
	h.render(w, r, status, pages.SearchPage(view))
}

// Random sends the browser to one catalog trail.
func (h *Handler) Random(w http.ResponseWriter, r *http.Request) {
	trail, err := h.trails.RandomTrail(r.Context())
	if errors.Is(err, repository.ErrNotFound) {
		h.notFound(w, r, "The catalog is empty.")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/trails/"+trail.ID.String(), http.StatusFound)
}

// Trail shows one trail and the form that logs a visit.
func (h *Handler) Trail(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	h.renderTrail(w, r, id, "", entryDraft{})
}

// CreateLog stores a visit for a trail.
func (h *Handler) CreateLog(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	input, err := h.parseEntryForm(w, r, false)
	if err != nil {
		h.rejectEntry(w, r, id, uuid.Nil, err)
		return
	}
	input.TrailID = id
	entryID, err := h.logs.CreateEntry(r.Context(), input)
	if errors.Is(err, repository.ErrNotFound) {
		h.notFound(w, r, "That trail is not in the catalog.")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/log/"+entryID.String(), http.StatusSeeOther)
}

// Entry is the photo-and-notes page for one visit.
func (h *Handler) Entry(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	h.renderEntry(w, r, id, "", entryDraft{})
}

// UpdateLog saves notes, the date, and an optional photo.
func (h *Handler) UpdateLog(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	input, err := h.parseEntryForm(w, r, true)
	if err != nil {
		h.rejectEntry(w, r, uuid.Nil, id, err)
		return
	}
	if err := h.logs.UpdateEntry(r.Context(), id, input); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.notFound(w, r, "That log entry is not here.")
			return
		}
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/log/"+id.String(), http.StatusSeeOther)
}

// DeleteLog removes a visit and returns to the log.
func (h *Handler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if err := h.logs.DeleteEntry(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.notFound(w, r, "That log entry is not here.")
			return
		}
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Photo serves the image stored on a log entry.
func (h *Handler) Photo(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	contentType, data, err := h.logs.EntryPhoto(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-cache")
	if _, err := w.Write(data); err != nil {
		slog.Debug("write photo", "error", err)
	}
}

func (h *Handler) catalog(ctx context.Context) ([]model.Trail, []model.Park, []model.DogRule, error) {
	trails, err := h.trails.ListTrails(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	parks, err := h.trails.ListParks(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	rules, err := h.trails.ListDogRules(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return trails, parks, rules, nil
}

func (h *Handler) renderTrail(w http.ResponseWriter, r *http.Request, id uuid.UUID, message string, draft entryDraft) {
	trail, err := h.trails.GetTrail(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.notFound(w, r, "That trail is not in the catalog.")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	visited := ncToday(h.now()).String()
	notes := ""
	duration := ""
	if draft.Present {
		visited = draft.VisitedOn
		notes = draft.Notes
		duration = draft.Duration
	}
	view := pages.TrailView{
		TrailItem:     presentTrail(trail),
		Source:        trail.Source,
		Confidence:    confidence(trail.Confidence),
		LowConfidence: trail.Confidence < 0.5,
		Sample:        trail.Sample,
		Visited:       visited,
		Notes:         notes,
		Duration:      duration,
		Map:           h.maps.Render(trailGeoJSONURL(trail.ID)),
		Error:         message,
	}
	status := http.StatusOK
	if message != "" {
		status = http.StatusBadRequest
	}
	h.render(w, r, status, pages.TrailPage(view))
}

func (h *Handler) renderEntry(w http.ResponseWriter, r *http.Request, id uuid.UUID, message string, draft entryDraft) {
	entry, err := h.logs.GetEntry(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.notFound(w, r, "That log entry is not here.")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	view := pages.EntryView{LogItem: presentLog(entry, entry.ID), Error: message}
	if draft.Present {
		view.VisitedISO = draft.VisitedOn
		view.Notes = draft.Notes
		view.DurationInput = draft.Duration
		view.RemovePhoto = draft.RemovePhoto
	}
	status := http.StatusOK
	if message != "" {
		status = http.StatusBadRequest
	}
	h.render(w, r, status, pages.EntryPage(view))
}

func (h *Handler) rejectEntry(w http.ResponseWriter, r *http.Request, trailID, entryID uuid.UUID, err error) {
	var formErr formError
	if !errors.As(err, &formErr) {
		h.fail(w, r, err)
		return
	}
	if formErr.status == http.StatusRequestEntityTooLarge {
		h.render(w, r, formErr.status, pages.NoticePage("Upload too large", formErr.Error()))
		return
	}
	if entryID != uuid.Nil {
		h.renderEntry(w, r, entryID, formErr.Error(), formErr.draft)
		return
	}
	h.renderTrail(w, r, trailID, formErr.Error(), formErr.draft)
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.render(w, r, http.StatusBadRequest, pages.NoticePage("Bad link", "That link is not valid."))
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request, message string) {
	h.render(w, r, http.StatusNotFound, pages.NoticePage("Not found", message))
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "error", err, "path", r.URL.Path)
	h.render(w, r, http.StatusInternalServerError, pages.NoticePage("Something went wrong", "Try again in a moment."))
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, status int, view templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := view.Render(r.Context(), w); err != nil {
		slog.Error("render", "error", err, "path", r.URL.Path)
	}
}
