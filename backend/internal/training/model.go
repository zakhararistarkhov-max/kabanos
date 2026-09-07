// Package training implements the workouts domain: a shared catalog of
// exercises and workouts created by any user, each carrying the same social
// signals as dishes (1–5 ratings, comments, favorites) and a "mine / all /
// favorites" split. A workout is an ordered list of exercises with a per-item
// prescription (sets, reps, duration, rest, weight).
package training

import (
	"time"

	"github.com/google/uuid"
)

// Exercise is a catalog movement. Equipment/Muscles are open tag lists.
type Exercise struct {
	ID          uuid.UUID
	CreatedBy   uuid.UUID
	Name        string
	Description string
	Category    string // strength | cardio | mobility
	Difficulty  string // easy | medium | hard
	JointImpact string // low | medium | high
	Equipment   []string
	Muscles     []string
	ImageKey    *string
	VideoURL    string
	IsPublic    bool
	RatingCount int
	RatingSum   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Viewer-specific, populated for list/detail responses.
	AuthorName string
	IsFavorite bool
	MyRating   *int
}

func (e *Exercise) AvgRating() float64 {
	if e.RatingCount == 0 {
		return 0
	}
	return float64(e.RatingSum) / float64(e.RatingCount)
}

// Workout is an ordered program of exercises.
type Workout struct {
	ID          uuid.UUID
	CreatedBy   uuid.UUID
	Name        string
	Description string
	Difficulty  string
	ImageKey    *string
	IsPublic    bool
	RatingCount int
	RatingSum   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Viewer-specific.
	AuthorName string
	IsFavorite bool
	MyRating   *int
	// Populated on the detail endpoint only.
	Items []WorkoutItem
}

func (w *Workout) AvgRating() float64 {
	if w.RatingCount == 0 {
		return 0
	}
	return float64(w.RatingSum) / float64(w.RatingCount)
}

// WorkoutItem is one exercise slot within a workout, in execution order.
type WorkoutItem struct {
	ID          uuid.UUID
	ExerciseID  uuid.UUID
	Position    int
	Sets        *int
	Reps        *int
	DurationSec *int
	RestSec     *int
	WeightKg    *float64
	Note        string
	// Denormalized snapshot of the linked exercise for rendering the list with
	// links, without a second round-trip.
	Exercise ExerciseRef
}

// ExerciseRef is the lightweight exercise summary embedded in a workout item.
type ExerciseRef struct {
	ID          uuid.UUID
	Name        string
	Category    string
	Difficulty  string
	JointImpact string
	Equipment   []string
	ImageKey    *string
	RatingCount int
	RatingSum   int
}

// Comment is a user comment on an exercise or workout.
type Comment struct {
	ID         uuid.UUID
	EntityID   uuid.UUID
	UserID     uuid.UUID
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

// ItemInput is the per-exercise prescription supplied when saving a workout.
type ItemInput struct {
	ExerciseID  uuid.UUID
	Sets        *int
	Reps        *int
	DurationSec *int
	RestSec     *int
	WeightKg    *float64
	Note        string
}

// ListParams controls a catalog listing for either resource.
type ListParams struct {
	ViewerID   uuid.UUID
	Scope      string // all | mine | favorites
	Query      string
	Sort       string // new | rating | name
	Category   string // exercises only
	Difficulty string
	Equipment  string // exercises only: single equipment tag filter
	Limit      int
	Offset     int
}
