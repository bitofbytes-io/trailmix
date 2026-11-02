package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestHealthAndReady(t *testing.T) {
	ok := NewHealthHandler(pingFunc(func(context.Context) error { return nil }))
	rec := httptest.NewRecorder()
	ok.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("health = %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	ok.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusOK || rec.Body.String() == "" {
		t.Fatalf("ready = %d %q", rec.Code, rec.Body.String())
	}

	bad := NewHealthHandler(pingFunc(func(context.Context) error { return errors.New("db down") }))
	rec = httptest.NewRecorder()
	bad.Ready(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("degraded ready = %d", rec.Code)
	}
	if rec.Body.String() == "" || strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("ready leaked the database error: %s", rec.Body.String())
	}
}
