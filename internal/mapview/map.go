// Package mapview draws trails on a MapLibre GL map. The style URL and the
// attribution string are supplied by the caller (see internal/config) so a
// different MapLibre style does not require a handler change.
package mapview

import "github.com/a-h/templ"

// Renderer draws the trail map for one GeoJSON FeatureCollection URL.
type Renderer interface {
	Render(geoJSONURL string) templ.Component
}

// MapLibre is the MapLibre GL renderer.
type MapLibre struct {
	StyleURL    string
	Attribution string
}

type mapProps struct {
	GeoJSONURL  string
	StyleURL    string
	Attribution string
}

// Render returns the map for geoJSONURL. The browser loads that URL and
// draws the lines on top of StyleURL.
func (m MapLibre) Render(geoJSONURL string) templ.Component {
	return Map(mapProps{
		GeoJSONURL:  geoJSONURL,
		StyleURL:    m.StyleURL,
		Attribution: m.Attribution,
	})
}
