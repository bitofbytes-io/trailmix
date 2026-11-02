// Package filter narrows the trail catalog. It is pure so the rules can be
// tested without Postgres. The catalog is small enough to filter in memory.
package filter

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/google/uuid"
)

// Query is a search. Empty fields do not constrain results.
type Query struct {
	Text             string
	MaxDistanceMiles *float64
	Difficulty       model.Difficulty
	MaxGradePercent  *float64
	ParkID           *uuid.UUID
	DogRule          model.DogRuleCode
}

// InvalidQueryError is a filter the UI can show back to the user.
type InvalidQueryError struct {
	Message string
}

func (e InvalidQueryError) Error() string {
	return e.Message
}

// Parse reads the search form. Unknown keys are ignored.
func Parse(values url.Values) (Query, error) {
	var query Query
	query.Text = strings.TrimSpace(values.Get("q"))

	if raw := strings.TrimSpace(values.Get("max_distance")); raw != "" {
		miles, err := strconv.ParseFloat(raw, 64)
		if err != nil || miles < 0 {
			return Query{}, InvalidQueryError{Message: "Distance must be a number of miles, zero or more."}
		}
		query.MaxDistanceMiles = &miles
	}

	if raw := strings.TrimSpace(values.Get("difficulty")); raw != "" {
		difficulty := model.Difficulty(strings.ToLower(raw))
		if !difficulty.Valid() {
			return Query{}, InvalidQueryError{Message: "Pick a difficulty of easy, moderate, or hard."}
		}
		query.Difficulty = difficulty
	}

	if raw := strings.TrimSpace(values.Get("max_grade")); raw != "" {
		grade, err := strconv.ParseFloat(raw, 64)
		if err != nil || grade < 0 {
			return Query{}, InvalidQueryError{Message: "Grade must be a percent, zero or more."}
		}
		query.MaxGradePercent = &grade
	}

	if raw := strings.TrimSpace(values.Get("park")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return Query{}, InvalidQueryError{Message: "Pick a park from the list."}
		}
		query.ParkID = &id
	}

	if raw := strings.TrimSpace(values.Get("dog")); raw != "" {
		rule := model.DogRuleCode(raw)
		if !rule.Valid() {
			return Query{}, InvalidQueryError{Message: "Pick a dog rule from the list."}
		}
		query.DogRule = rule
	}

	return query, nil
}

// Apply returns the trails that match query, shortest first.
// The input slice is left in its original order.
func Apply(trails []model.Trail, query Query) []model.Trail {
	text := strings.ToLower(strings.TrimSpace(query.Text))
	matched := make([]model.Trail, 0, len(trails))
	for _, trail := range trails {
		if !matches(trail, query, text) {
			continue
		}
		matched = append(matched, trail)
	}
	slices.SortStableFunc(matched, func(a, b model.Trail) int {
		switch {
		case a.DistanceMiles < b.DistanceMiles:
			return -1
		case a.DistanceMiles > b.DistanceMiles:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	return matched
}

func matches(trail model.Trail, query Query, text string) bool {
	if query.MaxDistanceMiles != nil && trail.DistanceMiles > *query.MaxDistanceMiles {
		return false
	}
	if query.Difficulty != "" && trail.Difficulty != query.Difficulty {
		return false
	}
	if query.MaxGradePercent != nil && trail.GradePercent > *query.MaxGradePercent {
		return false
	}
	if query.ParkID != nil && trail.Park.ID != *query.ParkID {
		return false
	}
	if query.DogRule != "" && trail.DogRule.Code != query.DogRule {
		return false
	}
	if text != "" && !strings.Contains(strings.ToLower(trail.Name), text) {
		return false
	}
	return true
}

// Describe returns a short sentence for the active constraints.
func Describe(query Query) string {
	var parts []string
	if query.Text != "" {
		parts = append(parts, fmt.Sprintf("named %q", query.Text))
	}
	if query.MaxDistanceMiles != nil {
		parts = append(parts, fmt.Sprintf("up to %g mi", *query.MaxDistanceMiles))
	}
	if query.Difficulty != "" {
		parts = append(parts, string(query.Difficulty))
	}
	if query.MaxGradePercent != nil {
		parts = append(parts, fmt.Sprintf("grade up to %g%%", *query.MaxGradePercent))
	}
	if query.DogRule != "" {
		parts = append(parts, dogSummary(query.DogRule))
	}
	if query.ParkID != nil {
		parts = append(parts, "one park")
	}
	if len(parts) == 0 {
		return "Showing every sample trail."
	}
	return "Filtered to " + strings.Join(parts, ", ") + "."
}

func dogSummary(rule model.DogRuleCode) string {
	switch rule {
	case model.DogAllowedLeashed:
		return "leashed dogs"
	case model.DogAllowedOffleash:
		return "off-leash dogs"
	case model.DogProhibited:
		return "no dogs"
	case model.DogUnknown:
		return "unknown dog rules"
	default:
		return "dog rule " + string(rule)
	}
}
