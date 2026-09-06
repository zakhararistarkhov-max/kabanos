// Package scheduler defines the planning/reminders abstraction. The product
// lets users place diets, workouts and activities on specific dates and get
// reminders. We start with a built-in Planner backed by Postgres + the outbox,
// but everything is expressed through the Planner interface so a Google
// Calendar-backed implementation can be added later without touching callers.
//
// This file intentionally contains only the seam (types + interface). The
// concrete Postgres planner and the reminder-dispatch worker land alongside the
// nutrition and workout domains, which produce the plannable items.
package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PlanItemKind enumerates what can be scheduled on a day.
type PlanItemKind string

const (
	KindDiet     PlanItemKind = "diet"
	KindWorkout  PlanItemKind = "workout"
	KindActivity PlanItemKind = "activity"
	KindPill     PlanItemKind = "pill"
)

// PlanItem is a single scheduled entry on a user's calendar.
type PlanItem struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Kind      PlanItemKind
	RefID     *uuid.UUID // optional link to a workout/dish/etc.
	Title     string
	Date      time.Time // the local calendar day
	RemindAt  *time.Time
	Notes     string
	CreatedAt time.Time
}

// Planner is the calendar seam. The built-in implementation stores items in
// Postgres; a future GoogleCalendarPlanner would satisfy the same contract.
type Planner interface {
	Add(ctx context.Context, item PlanItem) (PlanItem, error)
	ListForDay(ctx context.Context, userID uuid.UUID, day time.Time) ([]PlanItem, error)
	Remove(ctx context.Context, userID, id uuid.UUID) error
	// DueReminders returns items whose RemindAt has passed and that have not yet
	// been notified, so the worker can enqueue digests/notifications.
	DueReminders(ctx context.Context, now time.Time, limit int) ([]PlanItem, error)
}
