package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/httpx"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
	userEmailKey
)

// Authenticator returns middleware that validates the Bearer access token and
// injects the authenticated user id into the request context. Requests without
// a valid token are rejected with 401 before reaching the handler.
func (s *Service) Authenticator() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				httpx.Error(w, r, httpx.ErrUnauthorized("missing bearer token"))
				return
			}
			userID, claims, err := s.tokens.ParseAccessToken(raw)
			if err != nil {
				httpx.Error(w, r, httpx.ErrUnauthorized("invalid or expired token"))
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			ctx = context.WithValue(ctx, userEmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// UserID returns the authenticated user's id. It panics if called outside an
// authenticated route (a programming error), so handlers can use it directly.
func UserID(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		panic("auth.UserID called on an unauthenticated request")
	}
	return id
}
