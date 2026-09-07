// Package reminders implements user-configurable recurring reminders delivered
// via Web Push. The scheduling logic (DueAt) is pure and timezone-aware; the
// worker evaluates it on a tick and sends notifications for due reminders.
package reminders

import (
	"time"

	"github.com/google/uuid"
)

// fireTolerance bounds how late a fixed-time slot may fire (e.g. if the worker
// was briefly down) before it is considered missed — avoids back-filling old
// slots when a reminder is created mid-day.
const fireTolerance = 30 * time.Minute

type Reminder struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Title           string
	Body            string
	URL             string
	Mode            string // "interval" | "times"
	IntervalMinutes *int
	WindowStart     string // "HH:MM"
	WindowEnd       string // "HH:MM"
	Times           []string
	Days            []int // 0..6 (0=Sunday); empty = every day
	Condition       string // "" | "water_below_goal" | "meds_due"
	Timezone        string
	Enabled         bool
	LastFiredAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Input carries the mutable fields for create/update.
type Input struct {
	Title           string
	Body            string
	URL             string
	Mode            string
	IntervalMinutes *int
	WindowStart     string
	WindowEnd       string
	Times           []string
	Days            []int
	Condition       string
	Timezone        string
	Enabled         bool
}

// DueAt reports whether the reminder should fire at instant `now`, considering
// its timezone, weekday filter, schedule and last fire time.
func (r Reminder) DueAt(now time.Time) bool {
	loc, err := time.LoadLocation(r.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)

	if len(r.Days) > 0 && !containsInt(r.Days, int(local.Weekday())) {
		return false
	}

	switch r.Mode {
	case "times":
		for _, t := range r.Times {
			slot, ok := atLocalTime(local, t, loc)
			if !ok || slot.After(now) {
				continue
			}
			if now.Sub(slot) <= fireTolerance && (r.LastFiredAt == nil || r.LastFiredAt.Before(slot)) {
				return true
			}
		}
		return false
	case "interval":
		if r.IntervalMinutes == nil || !withinWindow(local, r.WindowStart, r.WindowEnd) {
			return false
		}
		if r.LastFiredAt == nil {
			return true
		}
		return now.Sub(*r.LastFiredAt) >= time.Duration(*r.IntervalMinutes)*time.Minute
	}
	return false
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// atLocalTime returns the instant for "HH:MM" on the given local day.
func atLocalTime(local time.Time, hhmm string, loc *time.Location) (time.Time, bool) {
	h, m, ok := parseHM(hhmm)
	if !ok {
		return time.Time{}, false
	}
	return time.Date(local.Year(), local.Month(), local.Day(), h, m, 0, 0, loc), true
}

// withinWindow reports whether the local clock time is within [start, end].
func withinWindow(local time.Time, start, end string) bool {
	sh, sm, ok1 := parseHM(start)
	eh, em, ok2 := parseHM(end)
	if !ok1 || !ok2 {
		return true // misconfigured window → don't block
	}
	cur := local.Hour()*60 + local.Minute()
	from := sh*60 + sm
	to := eh*60 + em
	if to < from {
		return true // overnight window not supported; allow
	}
	return cur >= from && cur <= to
}

func parseHM(s string) (int, int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, false
	}
	return t.Hour(), t.Minute(), true
}
