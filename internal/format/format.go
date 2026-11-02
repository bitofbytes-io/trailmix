// Package format turns trail numbers into the short labels used in the UI.
package format

import (
	"math"
	"strconv"
	"time"
)

// Miles formats a distance as miles with one decimal place.
func Miles(miles float64) string {
	return strconv.FormatFloat(miles, 'f', 1, 64) + " mi"
}

// Grade formats an average grade percent with one decimal place.
func Grade(percent float64) string {
	return strconv.FormatFloat(percent, 'f', 1, 64) + "%"
}

// Feet formats an elevation gain.
func Feet(feet int) string {
	return strconv.Itoa(feet) + " ft"
}

// Confidence formats a 0–1 score as a percent.
func Confidence(score float64) string {
	return strconv.Itoa(int(math.Round(score*100))) + "%"
}

// Visited formats a calendar date. The date is read in UTC so a DATE column
// does not shift a day when the server is in another zone.
func Visited(day time.Time) string {
	return day.UTC().Format("Jan 2, 2006")
}

// ISODate is the value for an HTML date input.
func ISODate(day time.Time) string {
	return day.UTC().Format("2006-01-02")
}

// Duration formats an optional hike length in minutes.
func Duration(minutes *int) string {
	if minutes == nil {
		return ""
	}
	return strconv.Itoa(*minutes) + " min"
}
