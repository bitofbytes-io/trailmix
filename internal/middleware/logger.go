package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// Logger records one access line per request. Server errors are logged at
// error, client errors at warn, and the rest at debug so a quiet info log
// is not a request trace.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"bytes", wrapped.BytesWritten(),
				"duration", time.Since(start).String(),
				"request_id", chimw.GetReqID(r.Context()),
			}
			switch {
			case status >= http.StatusInternalServerError:
				slog.Error("http request", attrs...)
			case status >= http.StatusBadRequest:
				slog.Warn("http request", attrs...)
			default:
				slog.Debug("http request", attrs...)
			}
		}()
		next.ServeHTTP(wrapped, r)
	})
}
