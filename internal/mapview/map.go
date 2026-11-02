// Package mapview draws trails. Placeholder is the only renderer. A tile
// provider can implement Renderer later without changing the handlers.
package mapview

import (
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

const (
	width  = 800
	height = 420
	pad    = 36
	// cos(35.7°) so a degree of longitude is not drawn as wide as a degree of
	// latitude around the Triangle.
	lngScale = 0.8121
)

// Line is one trail stroke.
type Line struct {
	ID       string
	Name     string
	Points   [][2]float64 // longitude, latitude
	Selected bool
}

// ProjectedLine is a line in SVG coordinates.
type ProjectedLine struct {
	Line
	PointsAttr string
}

// Renderer is the map seam.
type Renderer interface {
	Render(lines []Line) templ.Component
}

// Placeholder draws sample lines on a blank panel. It does not load tiles.
type Placeholder struct{}

// Render returns the placeholder map.
func (Placeholder) Render(lines []Line) templ.Component {
	return Map(Project(lines))
}

// Project maps longitude/latitude into the placeholder viewBox.
func Project(lines []Line) []ProjectedLine {
	projected := make([]ProjectedLine, len(lines))
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	seen := false
	for _, line := range lines {
		for _, point := range line.Points {
			seen = true
			x, y := projectPoint(point)
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		}
	}
	spanX := maxX - minX
	spanY := maxY - minY
	if !seen || spanX < 1e-9 {
		spanX = 1
	}
	if !seen || spanY < 1e-9 {
		spanY = 1
	}
	innerW := float64(width - 2*pad)
	innerH := float64(height - 2*pad)
	scale := innerW / spanX
	if innerH/spanY < scale {
		scale = innerH / spanY
	}
	offsetX := float64(pad) + (innerW-spanX*scale)/2
	offsetY := float64(pad) + (innerH-spanY*scale)/2

	for i, line := range lines {
		coords := make([]string, len(line.Points))
		for j, point := range line.Points {
			x, y := projectPoint(point)
			sx := offsetX + (x-minX)*scale
			sy := offsetY + (maxY-y)*scale
			if !seen {
				sx = float64(width) / 2
				sy = float64(height) / 2
			}
			coords[j] = format(sx) + "," + format(sy)
		}
		projected[i] = ProjectedLine{Line: line, PointsAttr: strings.Join(coords, " ")}
	}
	return projected
}

func projectPoint(point [2]float64) (float64, float64) {
	return point[0] * lngScale, point[1]
}

func format(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}
