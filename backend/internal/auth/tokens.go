package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// generateSecret returns a URL-safe random secret. It is handed to the client
// (in a URL or cookie) while only its hash is persisted, so a database leak
// never exposes a usable token.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashSecret is a fast one-way hash suitable for high-entropy random tokens
// (unlike passwords, these do not need a slow KDF).
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// --- refresh tokens (with rotation) ---

type refreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type refreshRepo struct{ db *postgres.DB }

func (r refreshRepo) create(ctx context.Context, q postgres.Querier, userID uuid.UUID, hash string, expiresAt time.Time, ua, ip string) (uuid.UUID, error) {
	const sql = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id uuid.UUID
	err := q.QueryRow(ctx, sql, userID, hash, expiresAt, ua, ip).Scan(&id)
	return id, err
}

func (r refreshRepo) getByHash(ctx context.Context, q postgres.Querier, hash string) (*refreshToken, error) {
	// FOR UPDATE locks the row so concurrent refreshes of the same token cannot
	// both succeed — the second waits, then observes it already revoked. This
	// must run inside a transaction (rotation always does).
	const sql = `SELECT id, user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE`
	var t refreshToken
	err := q.QueryRow(ctx, sql, hash).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &t.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r refreshRepo) revoke(ctx context.Context, q postgres.Querier, id, replacedBy uuid.UUID) error {
	const sql = `UPDATE refresh_tokens SET revoked_at = now(), replaced_by = $2 WHERE id = $1`
	var replaced any
	if replacedBy != uuid.Nil {
		replaced = replacedBy
	}
	_, err := q.Exec(ctx, sql, id, replaced)
	return err
}

// revokeByHash revokes a single token identified by its hash (used on logout).
func (r refreshRepo) revokeByHash(ctx context.Context, q postgres.Querier, hash string) error {
	const sql = `UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := q.Exec(ctx, sql, hash)
	return err
}

// revokeAllForUser revokes every non-revoked token for a user. Used when
// refresh-token reuse is detected and after a password reset.
func (r refreshRepo) revokeAllForUser(ctx context.Context, q postgres.Querier, userID uuid.UUID) error {
	const sql = `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := q.Exec(ctx, sql, userID)
	return err
}

// --- single-use tokens (email verification, password reset) ---

type singleUseRepo struct {
	db    *postgres.DB
	table string
}

func (r singleUseRepo) create(ctx context.Context, q postgres.Querier, userID uuid.UUID, hash string, expiresAt time.Time) error {
	sql := `INSERT INTO ` + r.table + ` (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err := q.Exec(ctx, sql, userID, hash, expiresAt)
	return err
}

// consume atomically marks a token used and returns its user id. It returns
// ErrInvalidToken if the token is unknown, expired, or already consumed, which
// makes the operation safe against replay.
func (r singleUseRepo) consume(ctx context.Context, q postgres.Querier, hash string) (uuid.UUID, error) {
	sql := `
		UPDATE ` + r.table + `
		SET consumed_at = now()
		WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		RETURNING user_id`
	var userID uuid.UUID
	err := q.QueryRow(ctx, sql, hash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrInvalidToken
		}
		return uuid.Nil, err
	}
	return userID, nil
}

// invalidateAll consumes any outstanding tokens for a user, so issuing a fresh
// one implicitly cancels older links.
func (r singleUseRepo) invalidateAll(ctx context.Context, q postgres.Querier, userID uuid.UUID) error {
	sql := `UPDATE ` + r.table + ` SET consumed_at = now() WHERE user_id = $1 AND consumed_at IS NULL`
	_, err := q.Exec(ctx, sql, userID)
	return err
}
