// Package gtd implements a "Getting Things Done" workspace: a capture inbox
// whose items are clarified into buckets (next action, waiting, calendar,
// someday, reference) and optionally grouped under multi-step projects. It backs
// the five-step workflow — Capture, Clarify, Organize, Reflect (weekly review)
// and Engage (do) — exposed by the HTTP handler.
package gtd

import (
	"time"

	"github.com/google/uuid"
)

// Project is a desired outcome that takes more than one action.
type Project struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Outcome     string
	Notes       string
	Status      string // active | someday | done | dropped
	Priority    int    // 1..5 (5 = highest); orders the graph panel within a colour band
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	// Derived (not stored): counts for the UI.
	OpenActions int
	NextActions int
}

// Item is one captured thought or a concrete action.
type Item struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ProjectID   *uuid.UUID
	Title       string
	Notes       string
	Bucket      string // inbox | next | waiting | calendar | someday | reference
	Context     string
	WaitingFor  string
	ScheduledAt *time.Time
	EndAt       *time.Time
	AllDay      bool
	DueOn       *string // YYYY-MM-DD
	Energy      string  // '' | low | medium | high
	TimeMinutes *int
	Priority    int
	Done        bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Derived (not stored): the parent project's title, for list display.
	ProjectTitle string
}

// ProjectInput carries the mutable fields for create/update of a project.
type ProjectInput struct {
	Title    string
	Outcome  string
	Notes    string
	Status   string
	Priority int // 1..5
}

// clampPriority normalises a priority into 1..5 (invalid/unset → 3, medium).
func clampPriority(p int) int {
	if p < 1 || p > 5 {
		return 3
	}
	return p
}

// clampItemPriority normalises a task priority into 0..5 (0 = unset).
func clampItemPriority(p int) int {
	if p < 0 {
		return 0
	}
	if p > 5 {
		return 5
	}
	return p
}

// ItemInput carries the mutable fields for create/update of an item.
type ItemInput struct {
	ProjectID   *uuid.UUID
	Title       string
	Notes       string
	Bucket      string
	Context     string
	WaitingFor  string
	ScheduledAt *time.Time
	EndAt       *time.Time
	AllDay      bool
	DueOn       *string
	Energy      string
	TimeMinutes *int
	Priority    int
}

// ItemFilter narrows a list query. Zero values mean "no constraint".
type ItemFilter struct {
	Bucket    string // exact bucket, or "" for any
	Context   string
	ProjectID *uuid.UUID
	Done      *bool // nil = both; false = only open; true = only done
	MaxTime   *int  // engage: items estimated to take <= MaxTime minutes
	Energy    string
}

// Review aggregates the state the weekly review surfaces.
type Review struct {
	InboxCount        int
	NextCount         int
	WaitingCount      int
	SomedayCount      int
	CalendarUpcoming  int
	ByContext         []ContextCount
	StalledProjects   []Project // active projects with no next action
	OverdueCalendar   []Item    // calendar items already in the past, not done
	CompletedThisWeek int
}

// ContextCount is the number of open next actions in one context.
type ContextCount struct {
	Context string
	Count   int
}
