// Package water implements the hydration tracker: a daily goal and an
// append-only log of intakes, summarised per day in the user's timezone.
package water

import (
	"time"

	"github.com/google/uuid"
)

// DefaultGoalML is used when a user has not set a personal goal yet.
const DefaultGoalML = 2000

// Preset pour sizes shown in the UI. "custom" allows an arbitrary amount.
var PresetAmounts = map[string]int{
	"glass":        250,  // 0.25 L
	"bottle_small": 500,  // 0.5 L
	"bottle_large": 1000, // 1.0 L
}

func ValidSource(s string) bool {
	switch s {
	case "glass", "bottle_small", "bottle_large", "custom":
		return true
	}
	return false
}

type Goal struct {
	UserID    uuid.UUID
	DailyML   int
	UpdatedAt time.Time
}

type Intake struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"-"`
	AmountML   int       `json:"amountMl"`
	Source     string    `json:"source"`
	ConsumedAt time.Time `json:"consumedAt"`
}

// DaySummary is the payload the UI needs to render the "fill the container"
// screen for a single day.
type DaySummary struct {
	Date        string   `json:"date"` // YYYY-MM-DD in the requested timezone
	GoalML      int      `json:"goalMl"`
	ConsumedML  int      `json:"consumedMl"`
	RemainingML int      `json:"remainingMl"`
	Percent     float64  `json:"percent"`
	Intakes     []Intake `json:"intakes"`
}

// DayTotal is one point in the history chart.
type DayTotal struct {
	Date    string `json:"date"`
	TotalML int    `json:"totalMl"`
}
