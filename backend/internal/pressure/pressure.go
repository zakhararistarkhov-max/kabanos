// Package pressure implements a blood-pressure diary: a time series of
// systolic/diastolic/pulse measurements with derived averages and an ACC/AHA
// category for the latest reading.
package pressure

import (
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"-"`
	Systolic   int       `json:"systolic"`
	Diastolic  int       `json:"diastolic"`
	Pulse      *int      `json:"pulse"`
	Note       string    `json:"note"`
	MeasuredAt time.Time `json:"measuredAt"`
}

// Averages holds the mean of each metric over the returned series (nil when
// there is nothing to average).
type Averages struct {
	Systolic  *float64 `json:"systolic"`
	Diastolic *float64 `json:"diastolic"`
	Pulse     *float64 `json:"pulse"`
}

// Summary is the diary payload: the recent series, per-metric averages, and the
// latest reading with its category.
type Summary struct {
	Latest   *Entry   `json:"latest"`
	Category string   `json:"category"`
	Averages Averages `json:"averages"`
	Series   []Entry  `json:"series"`
	Count    int      `json:"count"`
}

// Category classifies a reading per the ACC/AHA 2017 scale. Keys are stable and
// mapped to labels/colors on the frontend.
func Category(systolic, diastolic int) string {
	switch {
	case systolic <= 0 || diastolic <= 0:
		return ""
	case systolic >= 180 || diastolic >= 120:
		return "crisis"
	case systolic >= 140 || diastolic >= 90:
		return "high2"
	case systolic >= 130 || diastolic >= 80:
		return "high1"
	case systolic >= 120:
		return "elevated"
	default:
		return "normal"
	}
}
