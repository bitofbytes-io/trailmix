package model

import (
	"fmt"
	"time"
)

// Date is a calendar day. It has no clock time and no time zone, which is
// what a Postgres DATE stores.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// ParseDate reads a YYYY-MM-DD value.
func ParseDate(value string) (Date, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return Date{}, fmt.Errorf("parse date: %w", err)
	}
	return Date{Year: parsed.Year(), Month: parsed.Month(), Day: parsed.Day()}, nil
}

// IsZero reports whether the date was never set.
func (d Date) IsZero() bool {
	return d == Date{}
}

// String returns the ISO calendar date.
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// After reports whether d is a later calendar day than other.
func (d Date) After(other Date) bool {
	if d.Year != other.Year {
		return d.Year > other.Year
	}
	if d.Month != other.Month {
		return d.Month > other.Month
	}
	return d.Day > other.Day
}
