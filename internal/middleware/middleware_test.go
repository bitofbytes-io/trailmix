package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	SecurityHeaders(true)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("expected HSTS when secure cookies are on")
	}

	rec = httptest.NewRecorder()
	SecurityHeaders(false)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("local HTTP should not send HSTS")
	}
}

func TestSameOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := SameOrigin(next)

	ok := httptest.NewRequest(http.MethodPost, "http://example.test/log", nil)
	ok.Host = "example.test"
	ok.Header.Set("Origin", "http://example.test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, ok)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("matching origin status = %d", rec.Code)
	}

	blocked := httptest.NewRequest(http.MethodPost, "http://example.test/log", nil)
	blocked.Host = "example.test"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, blocked)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing origin status = %d", rec.Code)
	}
}
