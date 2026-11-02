package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_URL_FILE", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadReadsDatabaseURLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	if err := os.WriteFile(path, []byte("postgres://trailmix@db/trailmix\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_URL_FILE", path)
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SECURE_COOKIES", "false")
	t.Setenv("MAP_STYLE_URL", "")
	t.Setenv("MAP_ATTRIBUTION", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://trailmix@db/trailmix" {
		t.Fatalf("database URL = %q", cfg.DatabaseURL)
	}
	if cfg.Port != "4600" {
		t.Fatalf("port = %q", cfg.Port)
	}
	if cfg.SecureCookies {
		t.Fatal("SECURE_COOKIES=false should disable secure cookies")
	}
	if cfg.MapStyleURL != DefaultMapStyleURL || cfg.MapAttribution != DefaultMapAttribution {
		t.Fatalf("map config = %q %q", cfg.MapStyleURL, cfg.MapAttribution)
	}
	if !strings.Contains(cfg.MapAttribution, "OpenStreetMap") || !strings.Contains(cfg.MapStyleURL, "openfreemap.org") {
		t.Fatalf("default map provider = %q %q", cfg.MapStyleURL, cfg.MapAttribution)
	}
}

func TestLoadOverridesMapProvider(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://trailmix@db/trailmix")
	t.Setenv("MAP_STYLE_URL", "https://tiles.example/topo.json")
	t.Setenv("MAP_ATTRIBUTION", "Example Topo")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MapStyleURL != "https://tiles.example/topo.json" || cfg.MapAttribution != "Example Topo" {
		t.Fatalf("map config = %q %q", cfg.MapStyleURL, cfg.MapAttribution)
	}
}

func TestLoadRejectsMapStyleThatIsNotAnHTTPURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://trailmix@db/trailmix")
	t.Setenv("MAP_STYLE_URL", "ftp://tiles.example/style.json")
	if _, err := Load(); err == nil {
		t.Fatal("expected style URL error")
	}
}

func TestLoadRejectsBlankAttribution(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://trailmix@db/trailmix")
	t.Setenv("MAP_ATTRIBUTION", "  ")
	if _, err := Load(); err == nil {
		t.Fatal("expected attribution error")
	}
}

func TestLoadRejectsUnknownLogLevel(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://trailmix@db/trailmix")
	t.Setenv("LOG_LEVEL", "loud")
	if _, err := Load(); err == nil {
		t.Fatal("expected log level error")
	}
}
