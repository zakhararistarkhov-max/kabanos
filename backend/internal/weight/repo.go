package weight

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

func (r *Repo) GetGoal(ctx context.Context, userID uuid.UUID) (*Goal, error) {
	const q = `SELECT user_id, target_kg::float8, updated_at FROM weight_goals WHERE user_id = $1`
	var g Goal
	err := r.db.Read().QueryRow(ctx, q, userID).Scan(&g.UserID, &g.TargetKg, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *Repo) UpsertGoal(ctx context.Context, userID uuid.UUID, targetKg float64) (*Goal, error) {
	const q = `
		INSERT INTO weight_goals (user_id, target_kg) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET target_kg = EXCLUDED.target_kg, updated_at = now()
		RETURNING user_id, target_kg::float8, updated_at`
	var g Goal
	if err := r.db.Pool.QueryRow(ctx, q, userID, targetKg).Scan(&g.UserID, &g.TargetKg, &g.UpdatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

// UpsertEntry records (or replaces) the measurement for a given local day.
func (r *Repo) UpsertEntry(ctx context.Context, userID uuid.UUID, weightKg float64, note, measuredOn string, measuredAt time.Time) (*Entry, error) {
	const q = `
		INSERT INTO weight_entries (user_id, weight_kg, note, measured_on, measured_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, measured_on)
		DO UPDATE SET weight_kg = EXCLUDED.weight_kg, note = EXCLUDED.note, measured_at = EXCLUDED.measured_at
		RETURNING id, user_id, weight_kg::float8, note, to_char(measured_on, 'YYYY-MM-DD'), measured_at`
	var e Entry
	if err := r.db.Pool.QueryRow(ctx, q, userID, weightKg, note, measuredOn, measuredAt).
		Scan(&e.ID, &e.UserID, &e.WeightKg, &e.Note, &e.MeasuredOn, &e.MeasuredAt); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) DeleteEntry(ctx context.Context, userID, id uuid.UUID) error {
	const q = `DELETE FROM weight_entries WHERE id = $1 AND user_id = $2`
	ct, err := r.db.Pool.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// Latest returns the most recent measurement, or ErrNotFound if there are none.
func (r *Repo) Latest(ctx context.Context, userID uuid.UUID) (*Entry, error) {
	const q = `
		SELECT id, user_id, weight_kg::float8, note, to_char(measured_on, 'YYYY-MM-DD'), measured_at
		FROM weight_entries WHERE user_id = $1
		ORDER BY measured_on DESC LIMIT 1`
	var e Entry
	err := r.db.Read().QueryRow(ctx, q, userID).Scan(&e.ID, &e.UserID, &e.WeightKg, &e.Note, &e.MeasuredOn, &e.MeasuredAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// List returns measurements ordered chronologically. When limit <= 0 all
// entries are returned.
func (r *Repo) List(ctx context.Context, userID uuid.UUID, limit int) ([]Entry, error) {
	q := `
		SELECT id, user_id, weight_kg::float8, note, to_char(measured_on, 'YYYY-MM-DD'), measured_at
		FROM weight_entries WHERE user_id = $1
		ORDER BY measured_on`
	args := []any{userID}
	if limit > 0 {
		// Take the most recent `limit` days, then present oldest-first.
		q = `
			SELECT id, user_id, weight_kg, note, measured_on, measured_at FROM (
				SELECT id, user_id, weight_kg::float8, note,
				       to_char(measured_on, 'YYYY-MM-DD') AS measured_on, measured_at, measured_on AS mo
				FROM weight_entries WHERE user_id = $1
				ORDER BY measured_on DESC LIMIT $2
			) t ORDER BY mo`
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
		if err := rows.Scan(&e.ID, &e.UserID, &e.WeightKg, &e.Note, &e.MeasuredOn, &e.MeasuredAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
