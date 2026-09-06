// Package nutrition implements calories & diet: a shared dish catalog with
// ratings/comments/favorites, per-user macro goals, the daily diet log, and
// calorie-burning activities (manual or MET-based).
package nutrition

import (
	"time"

	"github.com/google/uuid"
)

// Macros is the reusable calories + protein/fat/carbs tuple.
type Macros struct {
	Kcal    float64 `json:"kcal"`
	Protein float64 `json:"protein"`
	Fat     float64 `json:"fat"`
	Carbs   float64 `json:"carbs"`
}

// Dish is a catalog item. Macro fields are per 100 g.
type Dish struct {
	ID             uuid.UUID
	CreatedBy      uuid.UUID
	Name           string
	Description    string
	Recipe         string
	ImageKey       *string
	KcalPer100     float64
	ProteinPer100  float64
	FatPer100      float64
	CarbsPer100    float64
	ServingGrams   *float64
	RatingCount    int
	RatingSum      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// Populated for list/detail responses (viewer-specific).
	AuthorName string
	IsFavorite bool
	MyRating   *int
}

// AvgRating returns the mean rating, or 0 when there are none.
func (d *Dish) AvgRating() float64 {
	if d.RatingCount == 0 {
		return 0
	}
	return float64(d.RatingSum) / float64(d.RatingCount)
}

type Comment struct {
	ID         uuid.UUID
	DishID     uuid.UUID
	UserID     uuid.UUID
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

type Goal struct {
	Kcal      int
	Protein   float64
	Fat       float64
	Carbs     float64
	UpdatedAt time.Time
}

// DietEntry is one logged food item with a macro snapshot (authoritative even
// if the source dish later changes).
type DietEntry struct {
	ID         uuid.UUID `json:"id"`
	DishID     *uuid.UUID `json:"dishId"`
	Name       string    `json:"name"`
	Grams      *float64  `json:"grams"`
	Meal       *string   `json:"meal"`
	Source     string    `json:"source"`
	Macros     Macros    `json:"macros"`
	ConsumedAt time.Time `json:"consumedAt"`
}

type Activity struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Kcal        float64   `json:"kcal"`
	DurationMin *int      `json:"durationMin"`
	MET         *float64  `json:"met"`
	Source      string    `json:"source"`
	PerformedAt time.Time `json:"performedAt"`
}

// DaySummary is the payload for the daily diet screen.
type DaySummary struct {
	Date        string      `json:"date"`
	Goal        Goal        `json:"goal"`
	Consumed    Macros      `json:"consumed"`
	BurnedKcal  float64     `json:"burnedKcal"`
	NetKcal     float64     `json:"netKcal"`     // consumed − burned
	RemainingKcal float64   `json:"remainingKcal"` // goal − net (negative = surplus)
	Entries     []DietEntry `json:"entries"`
	Activities  []Activity  `json:"activities"`
}

// DayTotals is one point in the history chart.
type DayTotals struct {
	Date       string  `json:"date"`
	Kcal       float64 `json:"kcal"`
	Protein    float64 `json:"protein"`
	Fat        float64 `json:"fat"`
	Carbs      float64 `json:"carbs"`
	BurnedKcal float64 `json:"burnedKcal"`
}

func validMeal(m string) bool {
	switch m {
	case "breakfast", "lunch", "dinner", "snack":
		return true
	}
	return false
}
