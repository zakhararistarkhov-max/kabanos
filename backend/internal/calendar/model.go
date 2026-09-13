// Package calendar implements two-way synchronisation between the GTD calendar
// bucket and an external CalDAV calendar (Yandex.Calendar). Credentials (an
// app-specific password) are stored encrypted; GTD calendar items are pushed as
// VEVENTs and remote events are pulled back into the same bucket.
package calendar

import (
	"time"

	"github.com/google/uuid"
)

// Connection is a user's link to their CalDAV calendar.
type Connection struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Provider     string
	Login        string
	PasswordEnc  []byte
	CalendarURL  string
	CalendarName string
	Enabled      bool
	LastSyncAt   *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RemoteCalendar is a calendar collection offered by the server (for the picker).
type RemoteCalendar struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// pushItem is a GTD calendar item that needs to be created/updated remotely.
type pushItem struct {
	ID           uuid.UUID
	Title        string
	Notes        string
	ScheduledAt  time.Time
	EndAt        *time.Time
	AllDay       bool
	ExternalUID  *string
	ExternalHref *string
}

// remoteEvent is a VEVENT pulled from the server, mapped to our fields.
type remoteEvent struct {
	UID     string
	Href    string
	ETag    string
	Summary string
	Notes   string
	Start   time.Time
	End     *time.Time
	AllDay  bool
}

// SyncResult summarises one sync run.
type SyncResult struct {
	Pushed  int `json:"pushed"`
	Pulled  int `json:"pulled"`
	Deleted int `json:"deleted"`
}

// tombstone is a locally-deleted synced item to remove on the server.
type tombstone struct {
	ID   uuid.UUID
	Href string
	ETag string
}
