package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/mapview"
)

func TestParseEntryFormRejectsFutureDate(t *testing.T) {
	h := New(nil, nil, mapview.Placeholder{})
	h.now = func() time.Time { return time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC) }

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("visited_on", "2026-11-03"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/trails/x/log", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := h.parseEntryForm(httptest.NewRecorder(), req, false)
	if err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseEntryFormAcceptsToday(t *testing.T) {
	h := New(nil, nil, mapview.Placeholder{})
	h.now = func() time.Time { return time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC) }

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("visited_on", "2026-11-02"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("notes", "Shady, muddy near the lake."); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("duration_minutes", "45"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/trails/x/log", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	input, err := h.parseEntryForm(httptest.NewRecorder(), req, false)
	if err != nil {
		t.Fatal(err)
	}
	if input.Notes != "Shady, muddy near the lake." || input.DurationMinutes == nil || *input.DurationMinutes != 45 {
		t.Fatalf("input = %+v", input)
	}
	if input.VisitedOn.String() != "2026-11-02" {
		t.Fatalf("visited = %s", input.VisitedOn)
	}
}
