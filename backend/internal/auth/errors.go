package auth

import "errors"

// Domain-level sentinel errors. The HTTP handler maps these to stable API
// error codes; nothing outside this package should depend on their messages.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrEmailNotVerified   = errors.New("email address not verified")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
)
