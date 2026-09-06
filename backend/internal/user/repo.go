package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// Repo is the data-access layer for users.
type Repo struct {
	db *postgres.DB
}

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

// selectColumns lists user columns in a fixed order. NUMERIC is cast to float8
// so it scans cleanly into *float64.
const selectColumns = `
	id, email, password_hash, display_name,
	height_cm::float8, sex, birth_date,
	telegram_username, telegram_chat_id,
	email_verified_at, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.HeightCm, &u.Sex, &u.BirthDate,
		&u.TelegramUsername, &u.TelegramChatID,
		&u.EmailVerifiedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateTx inserts a new user using the caller's Querier (pool or transaction).
// The email must already be normalized. Returns postgres.ErrConflict if the
// email is taken.
func (r *Repo) CreateTx(ctx context.Context, q postgres.Querier, email, passwordHash, displayName string) (*User, error) {
	const sql = `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING ` + selectColumns
	u, err := scanUser(q.QueryRow(ctx, sql, email, passwordHash, displayName))
	if err != nil {
		if postgres.IsUniqueViolation(err) {
			return nil, postgres.ErrConflict
		}
		return nil, err
	}
	return u, nil
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `SELECT ` + selectColumns + ` FROM users WHERE id = $1`
	return scanUser(r.db.Pool.QueryRow(ctx, q, id))
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT ` + selectColumns + ` FROM users WHERE lower(email) = lower($1)`
	return scanUser(r.db.Pool.QueryRow(ctx, q, email))
}

func (r *Repo) SetEmailVerified(ctx context.Context, q postgres.Querier, id uuid.UUID, at time.Time) error {
	const sql = `UPDATE users SET email_verified_at = $2, updated_at = now() WHERE id = $1`
	ct, err := q.Exec(ctx, sql, id, at)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) UpdatePassword(ctx context.Context, q postgres.Querier, id uuid.UUID, passwordHash string) error {
	const sql = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`
	ct, err := q.Exec(ctx, sql, id, passwordHash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// UpdateProfile applies a partial update using COALESCE so nil fields are left
// untouched in a single round-trip.
func (r *Repo) UpdateProfile(ctx context.Context, id uuid.UUID, p ProfileUpdate) (*User, error) {
	const q = `
		UPDATE users SET
			display_name      = COALESCE($2, display_name),
			height_cm         = COALESCE($3, height_cm),
			sex               = COALESCE($4, sex),
			birth_date        = COALESCE($5, birth_date),
			telegram_username = COALESCE($6, telegram_username),
			updated_at        = now()
		WHERE id = $1
		RETURNING ` + selectColumns
	return scanUser(r.db.Pool.QueryRow(ctx, q, id, p.DisplayName, p.HeightCm, p.Sex, p.BirthDate, p.TelegramUsername))
}
