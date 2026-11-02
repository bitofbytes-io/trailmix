package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const readyTimeout = 2 * time.Second

// HealthHandler serves liveness and readiness.
type HealthHandler struct {
	db Pinger
}

// NewHealthHandler returns the health endpoints.
func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db}
}

type readyResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// Health reports that the process is up.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		slog.Debug("write health", "error", err)
	}
}

// Ready pings the database. The response never includes the error text.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
	defer cancel()

	resp := readyResponse{Status: "ok", DB: "ok"}
	code := http.StatusOK
	if err := h.db.Ping(ctx); err != nil {
		slog.Warn("readiness check failed", "dependency", "db", "error", err)
		resp = readyResponse{Status: "degraded", DB: "error"}
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Debug("write ready", "error", err)
	}
}
