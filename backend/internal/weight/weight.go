// Package weight implements body-weight tracking: a target weight, a per-day
// time series of measurements, and derived BMI.
package weight

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type Goal struct {
	UserID    uuid.UUID
	TargetKg  float64
	UpdatedAt time.Time
}

type Entry struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"-"`
	WeightKg   float64   `json:"weightKg"`
	Note       string    `json:"note"`
	MeasuredOn string    `json:"measuredOn"` // YYYY-MM-DD
	MeasuredAt time.Time `json:"measuredAt"`
}

// Summary is the dashboard payload: current state plus the full series.
type Summary struct {
	LatestKg    *float64 `json:"latestKg"`
	TargetKg    *float64 `json:"targetKg"`
	HeightCm    *float64 `json:"heightCm"`
	BMI         *float64 `json:"bmi"`
	BMICategory string   `json:"bmiCategory"`
	Series      []Entry  `json:"series"`
}

// BMI computes body-mass index rounded to one decimal, or nil when height is
// unknown or invalid.
func BMI(weightKg float64, heightCm *float64) *float64 {
	if heightCm == nil || *heightCm <= 0 {
		return nil
	}
	h := *heightCm / 100.0
	bmi := math.Round(weightKg/(h*h)*10) / 10
	return &bmi
}

// BMICategory maps a BMI value to the WHO classification.
func BMICategory(bmi *float64) string {
	if bmi == nil {
		return ""
	}
	switch {
	case *bmi < 18.5:
		return "underweight"
	case *bmi < 25:
		return "normal"
	case *bmi < 30:
		return "overweight"
	default:
		return "obese"
	}
}
