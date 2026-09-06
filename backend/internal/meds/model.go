// Package meds implements medication & vitamin courses: a per-user list with a
// daily intake schedule (dose per intake × intakes per day) and course window
// (start date + duration), plus intake logging that drives the daily progress
// bar and weekly adherence stats.
package meds

import (
	"time"

	"github.com/google/uuid"
)

// Medication is one course a user is taking.
type Medication struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Name         string
	Unit         string
	Dose         float64
	TimesPerDay  int
	StartDate    time.Time
	DurationDays *int
	Notes        string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Progress is a Medication enriched with the counts needed by the UI for a
// given day: how many intakes were logged today and over the trailing week.
type Progress struct {
	Medication
	TakenToday int
	WeekTaken  int
}

// Input carries the mutable fields for create/update.
type Input struct {
	Name         string
	Unit         string
	Dose         float64
	TimesPerDay  int
	StartDate    time.Time
	DurationDays *int
	Notes        string
	Active       bool
}
