// Package model holds the trail, park, dog-rule, and log types shared by
// storage and the HTTP handlers.
package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DogRuleCode is the stable identifier stored in dog_rules.code.
type DogRuleCode string

const (
	DogAllowedLeashed  DogRuleCode = "allowed_leashed"
	DogAllowedOffleash DogRuleCode = "allowed_offleash"
	DogProhibited      DogRuleCode = "prohibited"
	DogUnknown         DogRuleCode = "unknown"
)

// Valid reports whether code is one of the four dog rules.
func (c DogRuleCode) Valid() bool {
	switch c {
	case DogAllowedLeashed, DogAllowedOffleash, DogProhibited, DogUnknown:
		return true
	default:
		return false
	}
}

// Label is the short phrase shown in the UI.
func (c DogRuleCode) Label() string {
	switch c {
	case DogAllowedLeashed:
		return "Leashed"
	case DogAllowedOffleash:
		return "Off leash"
	case DogProhibited:
		return "No dogs"
	case DogUnknown:
		return "Unknown"
	default:
		return string(c)
	}
}

// Difficulty is the coarse grade bucket stored on a trail.
type Difficulty string

const (
	DifficultyEasy     Difficulty = "easy"
	DifficultyModerate Difficulty = "moderate"
	DifficultyHard     Difficulty = "hard"
)

// Valid reports whether d is a stored difficulty value.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyModerate, DifficultyHard:
		return true
	default:
		return false
	}
}

// Label is the title-case difficulty name.
func (d Difficulty) Label() string {
	switch d {
	case DifficultyEasy:
		return "Easy"
	case DifficultyModerate:
		return "Moderate"
	case DifficultyHard:
		return "Hard"
	default:
		return string(d)
	}
}

// Park is a place that contains one or more trails.
type Park struct {
	ID     uuid.UUID
	Name   string
	County string
}

// DogRule is a row from dog_rules.
type DogRule struct {
	Code  DogRuleCode
	Label string
}

// Point is a WGS84 location. Lon is longitude, Lat is latitude.
type Point struct {
	Lon float64
	Lat float64
}

// LineString is a GeoJSON LineString. Coordinates are [longitude, latitude].
type LineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

// Points returns longitude/latitude pairs.
func (g LineString) Points() ([][2]float64, error) {
	if g.Type != "LineString" {
		return nil, fmt.Errorf("geometry type %q", g.Type)
	}
	points := make([][2]float64, 0, len(g.Coordinates))
	for _, coordinate := range g.Coordinates {
		if len(coordinate) < 2 {
			return nil, fmt.Errorf("coordinate length %d", len(coordinate))
		}
		points = append(points, [2]float64{coordinate[0], coordinate[1]})
	}
	if len(points) < 2 {
		return nil, fmt.Errorf("linestring needs two points")
	}
	return points, nil
}

// Trail is one hike. Geometry is the centerline. Trailhead is its first point.
// DistanceMiles is the geodesic length of Geometry.
type Trail struct {
	ID                uuid.UUID
	Name              string
	Park              Park
	DogRule           DogRule
	DistanceMiles     float64
	ElevationGainFeet int
	GradePercent      float64
	Difficulty        Difficulty
	Geometry          LineString
	Trailhead         Point
	Source            string
	Confidence        float64
	Sample            bool
}

// LogEntry is one visit the user recorded.
type LogEntry struct {
	ID              uuid.UUID
	Trail           Trail
	VisitedOn       Date
	Notes           string
	DurationMinutes *int
	HasPhoto        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
