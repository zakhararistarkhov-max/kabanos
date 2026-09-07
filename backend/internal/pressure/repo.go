package pressure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

const cols = `id, user_id, systolic, diastolic, pulse, note, measured_at`

func scan(row pgx.Row) (*Entry, error) {
	var e Entry
	err := row.Scan(&e.ID, &e.UserID, &e.Systolic, &e.Diastolic, &e.Pulse, &e.Note, &e.MeasuredAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *Repo) Insert(ctx context.Context, userID uuid.UUID, systolic, diastolic int, pulse *int, note string, measuredAt time.Time) (*Entry, error) {
	const q = `
		INSERT INTO pressure_entries (user_id, systolic, diastolic, pulse, note, measured_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, userID, systolic, diastolic, pulse, note, measuredAt))
}

func (r *Repo) DeleteEntry(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM pressure_entries WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) Latest(ctx context.Context, userID uuid.UUID) (*Entry, error) {
	const q = `SELECT ` + cols + ` FROM pressure_entries WHERE user_id=$1 ORDER BY measured_at DESC LIMIT 1`
	return scan(r.db.Read().QueryRow(ctx, q, userID))
}

// List returns measurements oldest-first. limit<=0 returns all; otherwise the
// most recent `limit` measurements (still presented chronologically).
func (r *Repo) List(ctx context.Context, userID uuid.UUID, limit int) ([]Entry, error) {
	q := `SELECT ` + cols + ` FROM pressure_entries WHERE user_id=$1 ORDER BY measured_at`
	args := []any{userID}
	if limit > 0 {
		q = `SELECT id, user_id, systolic, diastolic, pulse, note, measured_at FROM (
				SELECT ` + cols + ` FROM pressure_entries WHERE user_id=$1
				ORDER BY measured_at DESC LIMIT $2
			) t ORDER BY measured_at`
		args = append(args, limit)
	}
	rows, err := r.db.Read().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Systolic, &e.Diastolic, &e.Pulse, &e.Note, &e.MeasuredAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
