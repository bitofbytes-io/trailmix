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

	"github.com/bitofbytes-io/trailmix/internal/repository"
)

type formError struct {
	status  int
	message string
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

	visited, err := time.Parse("2006-01-02", strings.TrimSpace(r.FormValue("visited_on")))
	if err != nil {
		return repository.EntryInput{}, formError{status: http.StatusBadRequest, message: "Choose a visit date."}
	}
	if visited.After(ncToday(h.now())) {
		return repository.EntryInput{}, formError{status: http.StatusBadRequest, message: "Visit date can't be in the future."}
	}

	notes := strings.TrimSpace(r.FormValue("notes"))
	if utf8.RuneCountInString(notes) > maxNotesRunes {
		return repository.EntryInput{}, formError{status: http.StatusBadRequest, message: "Notes need to be 4,000 characters or fewer."}
	}

	var duration *int
	if raw := strings.TrimSpace(r.FormValue("duration_minutes")); raw != "" {
		minutes, err := strconv.Atoi(raw)
		if err != nil || minutes < 0 || minutes > 1440 {
			return repository.EntryInput{}, formError{status: http.StatusBadRequest, message: "Duration must be a whole number of minutes, up to 24 hours."}
		}
		duration = &minutes
	}

	photo, err := readPhoto(r)
	if err != nil {
		return repository.EntryInput{}, err
	}
	if photo == nil && updating && r.FormValue("remove_photo") == "1" {
		photo = &repository.Photo{Clear: true}
	}

	return repository.EntryInput{
		VisitedOn:       visited,
		Notes:           notes,
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

func ncToday(now time.Time) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.FixedZone("EST", -5*60*60)
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
