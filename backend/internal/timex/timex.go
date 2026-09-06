// Package timex centralises timezone-aware "calendar day" math. Health data is
// naturally bucketed by the user's local day, so every domain that summarises
// per day funnels through here to stay consistent.
package timex

import (
	"errors"
	"time"
)

const dateLayout = "2006-01-02"

// Location resolves an IANA timezone name, falling back to UTC for empty or
// invalid input so a bad client header can never break a request.
func Location(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// Today returns today's date (YYYY-MM-DD) in the given location.
func Today(loc *time.Location) string {
	return time.Now().In(loc).Format(dateLayout)
}

// ParseDate parses a YYYY-MM-DD string as midnight in loc.
func ParseDate(s string, loc *time.Location) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty date")
	}
	return time.ParseInLocation(dateLayout, s, loc)
}

// DayRange returns the [start, end) instants bounding the given local calendar
// day, as UTC-comparable time.Time values suitable for range queries.
func DayRange(dateStr string, loc *time.Location) (from, to time.Time, err error) {
	start, err := ParseDate(dateStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 0, 1), nil
}

// RangeBounds returns the [start, end) instants covering the inclusive local
// date range [fromDate, toDate].
func RangeBounds(fromDate, toDate string, loc *time.Location) (from, to time.Time, err error) {
	start, err := ParseDate(fromDate, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := ParseDate(toDate, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// end is exclusive upper bound → day after toDate.
	return start, end.AddDate(0, 0, 1), nil
}
