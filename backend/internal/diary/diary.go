// Package diary implements a personal day-by-day journal: free-text entries
// with photo/video attachments, browsable by date. Media is uploaded straight
// to object storage via presigned URLs; entries store only object references.
package diary

import (
	"time"

	"github.com/google/uuid"
)

// Attachment references one uploaded media object.
type Attachment struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"` // image | video
	ContentType string `json:"contentType"`
}

// Entry is a single diary note on a given day.
type Entry struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Date        string // YYYY-MM-DD
	Body        string
	Attachments []Attachment
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DateCount is used to show which days have entries when browsing.
type DateCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}
