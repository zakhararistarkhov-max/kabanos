// Package push implements Web Push (browser/PWA) notifications: it stores each
// device's push subscription and delivers encrypted payloads via the Web Push
// protocol using VAPID. Requires HTTPS on the client side (service workers and
// the Push API only work in a secure context).
package push

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Endpoint  string
	P256dh    string
	Auth      string
	CreatedAt time.Time
}

// Notification is the JSON payload delivered to the service worker.
type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}
