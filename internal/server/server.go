// Package server wires the HTTP router.
package server

import (
	"mime"
	"net/http"

	"github.com/bitofbytes-io/trailmix/internal/config"
	"github.com/bitofbytes-io/trailmix/internal/handler"
	"github.com/bitofbytes-io/trailmix/internal/mapview"
	"github.com/bitofbytes-io/trailmix/internal/middleware"
	"github.com/bitofbytes-io/trailmix/internal/repository"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func init() {
	if err := mime.AddExtensionType(".webmanifest", "application/manifest+json"); err != nil {
		panic(err)
	}
	// MapLibre is an ES module. Browsers refuse a module worker unless the
	// script is served as JavaScript.
	if err := mime.AddExtensionType(".mjs", "text/javascript"); err != nil {
		panic(err)
	}
}

// Server is the Trail Mix HTTP application.
type Server struct {
	cfg    *config.Config
	db     handler.Pinger
	trails repository.TrailStore
	logs   repository.LogStore
	maps   mapview.Renderer
}

// New builds a server. A nil map renderer uses MapLibre with the configured
// style, or the OpenFreeMap default when the style was left empty.
func New(cfg *config.Config, db handler.Pinger, trails repository.TrailStore, logs repository.LogStore, maps mapview.Renderer) *Server {
	if maps == nil {
		style := cfg.MapStyleURL
		if style == "" {
			style = config.DefaultMapStyleURL
		}
		attribution := cfg.MapAttribution
		if attribution == "" {
			attribution = config.DefaultMapAttribution
		}
		maps = mapview.MapLibre{StyleURL: style, Attribution: attribution}
	}
	return &Server{cfg: cfg, db: db, trails: trails, logs: logs, maps: maps}
}

// Router returns the chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders(s.cfg.SecureCookies))
	r.Use(chimw.RequestID)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SameOrigin)

	const staticCacheControl = "public, max-age=86400"
	fileServer := http.FileServer(http.Dir("static"))
	r.Handle("/static/*", withCacheControl(staticCacheControl, http.StripPrefix("/static/", fileServer)))
	r.Get("/favicon.svg", serveStaticFile("static/favicon.svg"))
	r.Get("/site.webmanifest", serveStaticFile("static/site.webmanifest"))

	health := handler.NewHealthHandler(s.db)
	r.Get("/health", health.Health)
	r.Get("/ready", health.Ready)

	pages := handler.New(s.trails, s.logs, s.maps)
	r.Get("/api/trails.geojson", pages.TrailsGeoJSON)
	r.Get("/api/trails/{id}/geojson", pages.TrailGeoJSON)
	r.Get("/", pages.Home)
	r.Get("/trails", pages.Search)
	r.Get("/trails/random", pages.Random)
	r.Get("/trails/{id}", pages.Trail)
	r.Post("/trails/{id}/log", pages.CreateLog)
	r.Get("/log/{id}/photo", pages.Photo)
	r.Get("/log/{id}", pages.Entry)
	r.Post("/log/{id}", pages.UpdateLog)
	r.Post("/log/{id}/delete", pages.DeleteLog)
	return r
}

func serveStaticFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func withCacheControl(cacheControl string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(w, r)
	})
}
