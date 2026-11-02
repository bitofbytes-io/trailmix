package mapview

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRenderPassesStyleAttributionAndGeoJSONURL(t *testing.T) {
	var buf bytes.Buffer
	err := MapLibre{
		StyleURL:    "https://tiles.example/topo.json",
		Attribution: "Example Topo Survey",
	}.Render("/api/trails.geojson?dog=prohibited").Render(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	for _, want := range []string{
		`data-style-url="https://tiles.example/topo.json"`,
		`data-geojson="/api/trails.geojson?dog=prohibited"`,
		"Example Topo Survey",
		"/static/vendor/maplibre/maplibre-gl.css",
		"/static/map.js",
		"Trail map",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("map HTML missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "openfreemap.org") {
		t.Fatal("renderer hardcoded the default style instead of using the configured URL")
	}
}
