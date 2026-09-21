// Package habits implements a list of habits to build ('good') or break ('bad'),
// each with a mini-diary of status notes and optional push reminders (delivered
// through the shared reminders worker).
package habits

import (
	"time"

	"github.com/google/uuid"
)

// Habit is one tracked habit.
type Habit struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	Kind        string // good | bad
	Description string
	SortOrder   int
	Archived    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Derived (not stored):
	ReminderCount int
	LogCount      int
	LastStatus    string
	LastLogAt     *time.Time
	// Health (derived from recent check-ins):
	Color     string    // green | yellow | red
	Fails     int       // failed/missed days in the recent window
	Successes int       // kept days in the recent window
	Streak    int       // current consecutive-day success streak
	Recent    []Checkin // the recent window (oldest→newest) for the preview strip
}

// Checkin is one day's result for a habit. Day is YYYY-MM-DD; Status is
// "success" | "fail" | "none" (none = the day is within the window but unmarked).
type Checkin struct {
	Day     string `json:"day"`
	Status  string `json:"status"`
	Success bool   `json:"-"`
}

// Log is one mini-diary entry for a habit.
type Log struct {
	ID        uuid.UUID
	HabitID   uuid.UUID
	Note      string
	Status    string // '' | positive | negative
	CreatedAt time.Time
}

// Input carries the mutable fields for create/update of a habit.
type Input struct {
	Name        string
	Kind        string
	Description string
	Archived    bool
}
