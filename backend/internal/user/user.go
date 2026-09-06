// Package user owns the user aggregate and its persistence. Authentication
// logic lives in the auth package; this package is purely about the user record
// and profile.
package user

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain model. Password handling is the auth package's concern;
// PasswordHash is stored here only because it lives on the same row.
type User struct {
	ID               uuid.UUID
	Email            string
	PasswordHash     string
	DisplayName      string
	HeightCm         *float64
	Sex              *string
	BirthDate        *time.Time
	TelegramUsername *string
	TelegramChatID   *int64
	EmailVerifiedAt  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsEmailVerified reports whether the user has confirmed their email address.
func (u *User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }

// ProfileUpdate carries the mutable profile fields. Nil pointers mean "leave
// unchanged", so a PATCH-style update can touch a single field.
type ProfileUpdate struct {
	DisplayName      *string
	HeightCm         *float64
	Sex              *string
	BirthDate        *time.Time
	TelegramUsername *string
}
