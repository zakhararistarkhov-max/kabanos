// Package decisions implements a decision journal: capture a decision with its
// context and predicted outcome, revisit it on a review date, then record the
// actual result, rate how right it was, and note lessons for the future.
package decisions

import (
	"time"

	"github.com/google/uuid"
)

// Decision is one journal entry.
type Decision struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Title      string
	Context    string
	Decision   string
	Expected   string
	Confidence *int   // 1..5, nil = unset
	DecidedOn  string // YYYY-MM-DD
	ReviewAt   *string
	Status     string // open | reviewed
	Result     string
	Rating     *int // 1..5, nil = unset
	Lessons    string
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Derived (not stored): review date has arrived and it's still open.
	Due bool
}

// Input carries the mutable capture fields (create/update).
type Input struct {
	Title      string
	Context    string
	Decision   string
	Expected   string
	Confidence *int
	DecidedOn  string
	ReviewAt   *string
}

// ReviewInput carries the outcome fields recorded at review time.
type ReviewInput struct {
	Result  string
	Rating  *int
	Lessons string
}
