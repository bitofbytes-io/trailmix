package handler

import (
	"strconv"

	"github.com/bitofbytes-io/trailmix/internal/format"
	"github.com/bitofbytes-io/trailmix/internal/model"
	"github.com/bitofbytes-io/trailmix/internal/ui/pages"
	"github.com/google/uuid"
)

func presentLog(entry model.LogEntry, selected uuid.UUID) pages.LogItem {
	item := pages.LogItem{
		ID:            entry.ID.String(),
		Href:          "/?entry=" + entry.ID.String(),
		EditHref:      "/log/" + entry.ID.String(),
		TrailName:     entry.Trail.Name,
		TrailHref:     "/trails/" + entry.Trail.ID.String(),
		Distance:      format.Miles(entry.Trail.DistanceMiles),
		Dog:           dogLabel(entry.Trail.DogRule),
		Visited:       format.Visited(entry.VisitedOn),
		VisitedISO:    format.ISODate(entry.VisitedOn),
		Duration:      format.Duration(entry.DurationMinutes),
		DurationInput: durationInput(entry.DurationMinutes),
		Notes:         entry.Notes,
		HasPhoto:      entry.HasPhoto,
		Selected:      entry.ID == selected,
	}
	if entry.HasPhoto {
		item.PhotoURL = "/log/" + entry.ID.String() + "/photo?v=" + strconv.FormatInt(entry.UpdatedAt.Unix(), 10)
	}
	return item
}

func presentTrail(trail model.Trail) pages.TrailItem {
	return pages.TrailItem{
		ID:         trail.ID.String(),
		Name:       trail.Name,
		Park:       trail.Park.Name,
		County:     trail.Park.County + " County",
		Distance:   format.Miles(trail.DistanceMiles),
		Grade:      format.Grade(trail.GradePercent),
		Elevation:  format.Feet(trail.ElevationGainFeet),
		Difficulty: trail.Difficulty.Label(),
		Dog:        dogLabel(trail.DogRule),
		Href:       "/trails/" + trail.ID.String(),
	}
}

func presentTrails(trails []model.Trail) []pages.TrailItem {
	items := make([]pages.TrailItem, len(trails))
	for i, trail := range trails {
		items[i] = presentTrail(trail)
	}
	return items
}

func presentParks(parks []model.Park) []pages.ParkOption {
	options := make([]pages.ParkOption, len(parks))
	for i, park := range parks {
		options[i] = pages.ParkOption{ID: park.ID.String(), Name: park.Name}
	}
	return options
}

func presentRules(rules []model.DogRule) []pages.DogOption {
	options := make([]pages.DogOption, len(rules))
	for i, rule := range rules {
		options[i] = pages.DogOption{Code: string(rule.Code), Label: dogLabel(rule)}
	}
	return options
}

func dogLabel(rule model.DogRule) string {
	if rule.Label != "" {
		return rule.Label
	}
	return rule.Code.Label()
}

func confidence(score float64) string {
	return format.Confidence(score)
}

func durationInput(minutes *int) string {
	if minutes == nil {
		return ""
	}
	return strconv.Itoa(*minutes)
}
