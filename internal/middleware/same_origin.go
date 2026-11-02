package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// SameOrigin rejects unsafe methods unless Origin or Referer matches this
// request's origin. Safe methods are always allowed.
func SameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if originMatches(r) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func originMatches(r *http.Request) bool {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	requestOrigin := normalize(scheme + "://" + r.Host)
	if requestOrigin == "" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		return normalize(origin) == requestOrigin
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return normalize(referer) == requestOrigin
	}
	return false
}

func normalize(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}
