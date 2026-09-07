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
	// Populated on the detail endpoint: the dishes this one is composed of.
	Ingredients []Ingredient
}

// AvgRating returns the mean rating, or 0 when there are none.
func (d *Dish) AvgRating() float64 {
	if d.RatingCount == 0 {
		return 0
	}
	return float64(d.RatingSum) / float64(d.RatingCount)
}

// Ingredient is one component of a composed dish: a reference to another dish
// plus how many grams of it go in. Per100 is a snapshot of the ingredient
// dish's per-100g macros.
type Ingredient struct {
	DishID uuid.UUID
	Name   string
	Grams  float64
	Per100 Macros
}

// Contribution returns the macros this ingredient contributes (Per100 scaled by
// its grams).
func (i Ingredient) Contribution() Macros {
	f := i.Grams / 100.0
	return Macros{Kcal: i.Per100.Kcal * f, Protein: i.Per100.Protein * f, Fat: i.Per100.Fat * f, Carbs: i.Per100.Carbs * f}
}

// IngredientInput is a component supplied when creating/updating a dish.
type IngredientInput struct {
	DishID uuid.UUID
	Grams  float64
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
	Kcal      int       `json:"kcal"`
	Protein   float64   `json:"protein"`
	Fat       float64   `json:"fat"`
	Carbs     float64   `json:"carbs"`
	UpdatedAt time.Time `json:"updatedAt"`
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
