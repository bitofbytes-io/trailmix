package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/bitofbytes-io/trailmix/internal/repository"
)

type entryDraft struct {
	Present     bool
	VisitedOn   string
	Notes       string
	Duration    string
	RemovePhoto bool
}

type formError struct {
	status  int
	message string
	draft   entryDraft
}

func (e formError) Error() string {
	return e.message
}

func (h *Handler) parseEntryForm(w http.ResponseWriter, r *http.Request, updating bool) (repository.EntryInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return repository.EntryInput{}, formError{status: http.StatusRequestEntityTooLarge, message: "That upload is too large."}
		}
		return repository.EntryInput{}, formError{status: http.StatusBadRequest, message: "The form could not be read."}
	}

	draft := entryDraft{
		Present:     true,
		VisitedOn:   strings.TrimSpace(r.FormValue("visited_on")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
		Duration:    strings.TrimSpace(r.FormValue("duration_minutes")),
		RemovePhoto: r.FormValue("remove_photo") == "1",
	}
	reject := func(status int, message string) error {
		return formError{status: status, message: message, draft: draft}
	}

	visited, err := model.ParseDate(draft.VisitedOn)
	if err != nil {
		return repository.EntryInput{}, reject(http.StatusBadRequest, "Choose a visit date.")
	}
	if visited.After(ncToday(h.now())) {
		return repository.EntryInput{}, reject(http.StatusBadRequest, "Visit date can't be in the future.")
	}

	if utf8.RuneCountInString(draft.Notes) > maxNotesRunes {
		return repository.EntryInput{}, reject(http.StatusBadRequest, "Notes need to be 4,000 characters or fewer.")
	}

	var duration *int
	if draft.Duration != "" {
		minutes, err := strconv.Atoi(draft.Duration)
		if err != nil || minutes < 0 || minutes > 1440 {
			return repository.EntryInput{}, reject(http.StatusBadRequest, "Duration must be a whole number of minutes, up to 24 hours.")
		}
		duration = &minutes
	}

	photo, err := readPhoto(r)
	if err != nil {
		var formErr formError
		if errors.As(err, &formErr) {
			formErr.draft = draft
			return repository.EntryInput{}, formErr
		}
		return repository.EntryInput{}, err
	}
	if photo == nil && updating && r.FormValue("remove_photo") == "1" {
		photo = &repository.Photo{Clear: true}
	}

	return repository.EntryInput{
		VisitedOn:       visited,
		Notes:           draft.Notes,
		DurationMinutes: duration,
		Photo:           photo,
	}, nil
}

func readPhoto(r *http.Request) (*repository.Photo, error) {
	file, header, err := r.FormFile("photo")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, formError{status: http.StatusBadRequest, message: "The photo could not be read."}
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Debug("close upload", "error", err)
		}
	}()
	if header.Size == 0 {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
	if err != nil {
		return nil, formError{status: http.StatusBadRequest, message: "The photo could not be read."}
	}
	if len(data) > maxPhotoBytes {
		return nil, formError{status: http.StatusBadRequest, message: "Photos need to be 2 MB or smaller."}
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
	default:
		return nil, formError{status: http.StatusBadRequest, message: "Use a JPEG, PNG, WEBP, or GIF."}
	}
	return &repository.Photo{ContentType: contentType, Data: data}, nil
}

func ncToday(now time.Time) model.Date {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.FixedZone("EST", -5*60*60)
	}
	local := now.In(loc)
	return model.Date{Year: local.Year(), Month: local.Month(), Day: local.Day()}
}
