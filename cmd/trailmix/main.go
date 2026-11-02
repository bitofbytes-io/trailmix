package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bitofbytes-io/trailmix/internal/assets"
	"github.com/bitofbytes-io/trailmix/internal/config"
	"github.com/bitofbytes-io/trailmix/internal/repository"
	"github.com/bitofbytes-io/trailmix/internal/server"
	"github.com/bitofbytes-io/trailmix/internal/ui/layout"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	version  = "dev"
	revision = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
	slog.Info("starting trailmix", "port", cfg.Port, "version", version, "revision", revision)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	slog.Info("connected to database")

	assetsVersion, err := assets.Version(
		filepath.Join("static", "styles.css"),
		filepath.Join("static", "map.js"),
		filepath.Join("static", "vendor", "maplibre", "maplibre-gl.css"),
		filepath.Join("static", "vendor", "maplibre", "maplibre-gl.mjs"),
		filepath.Join("static", "vendor", "maplibre", "maplibre-gl-worker.mjs"),
	)
	if err != nil {
		slog.Warn("asset version unavailable", "error", err)
	} else {
		layout.SetAssetsVersion(assetsVersion)
	}

	srv := server.New(
		cfg,
		pool,
		repository.NewTrailRepository(pool),
		repository.NewLogRepository(pool),
		nil,
	)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serve(sigCtx, httpServer)
}

// serve listens until ctx is cancelled, then shuts down. A port that is
// already taken returns immediately.
func serve(ctx context.Context, httpServer *http.Server) error {
	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", httpServer.Addr)
		serverErr <- httpServer.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
	}
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	slog.Info("server stopped")
	return nil
}
