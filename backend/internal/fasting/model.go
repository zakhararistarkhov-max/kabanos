// Package fasting implements intermittent-fasting tracking: a per-user protocol
// (fasting/eating window lengths) and a log of fasts. The live countdown and
// clock ring are derived from the current phase — an ongoing fast, or the eating
// window that follows the last one.
package fasting

import (
	"time"

	"github.com/google/uuid"
)

// Settings is the user's protocol (e.g. 16:8 → FastingHours 16, EatingHours 8).
type Settings struct {
	FastingHours float64
	EatingHours  float64
}

// Schedule is the optional daily plan: the eating window opens at
// EatStartHour:EatStartMinute (in Timezone); the fast then begins when the
// window closes (eat start + eating hours), which is when we auto-start the fast
// and send push notifications at the start and every elapsed hour.
type Schedule struct {
	Enabled      bool
	EatStartHour int
	EatStartMin  int
	Timezone     string
	AutoStart    bool
	NotifyStart  bool
	NotifyHourly bool
}

// ScheduleRow is a worker-facing snapshot: a user's schedule together with their
// protocol and the per-user notification bookkeeping the worker advances.
type ScheduleRow struct {
	UserID         uuid.UUID
	FastingHours   float64
	EatingHours    float64
	Schedule       Schedule
	AutoStartedOn  *time.Time // last local date auto-started (nil = never)
	NotifyAnchor   *time.Time // active fast start last notified about
	NotifyLastHour int
}

// Session is one fast; EndedAt nil means it is ongoing.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	StartedAt time.Time
	EndedAt   *time.Time
	GoalHours float64
	CreatedAt time.Time
}

// Stats summarises completed fasts.
type Stats struct {
	TotalFasts   int
	LongestHours float64
	AvgHours     float64
}

// State is the computed live view of where the user is in their cycle.
type State struct {
	Phase        string // fasting | eating | idle
	FastingHours float64
	EatingHours  float64
	ActiveID     *uuid.UUID
	PhaseStartAt *time.Time
	PhaseEndAt   *time.Time
	GoalHours    float64 // target length of the current phase (hours)
	Overrun      bool    // current phase has passed its target
	ServerNow    time.Time
	Stats        Stats
	Schedule     Schedule
}
