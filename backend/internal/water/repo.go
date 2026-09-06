package water

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
	const q = `SELECT user_id, daily_ml, updated_at FROM water_goals WHERE user_id = $1`
	var g Goal
	err := r.db.Read().QueryRow(ctx, q, userID).Scan(&g.UserID, &g.DailyML, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *Repo) UpsertGoal(ctx context.Context, userID uuid.UUID, dailyML int) (*Goal, error) {
	const q = `
		INSERT INTO water_goals (user_id, daily_ml) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET daily_ml = EXCLUDED.daily_ml, updated_at = now()
		RETURNING user_id, daily_ml, updated_at`
	var g Goal
	if err := r.db.Pool.QueryRow(ctx, q, userID, dailyML).Scan(&g.UserID, &g.DailyML, &g.UpdatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) AddIntake(ctx context.Context, userID uuid.UUID, amountML int, source string, consumedAt time.Time) (*Intake, error) {
	const q = `
		INSERT INTO water_intakes (user_id, amount_ml, source, consumed_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, amount_ml, source, consumed_at`
	var in Intake
	if err := r.db.Pool.QueryRow(ctx, q, userID, amountML, source, consumedAt).
		Scan(&in.ID, &in.UserID, &in.AmountML, &in.Source, &in.ConsumedAt); err != nil {
		return nil, err
	}
	return &in, nil
}

// DeleteIntake removes an intake, scoped to its owner so users cannot delete
// each other's rows.
func (r *Repo) DeleteIntake(ctx context.Context, userID, id uuid.UUID) error {
	const q = `DELETE FROM water_intakes WHERE id = $1 AND user_id = $2`
	ct, err := r.db.Pool.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ListIntakesBetween returns intakes in [from, to) ordered chronologically.
func (r *Repo) ListIntakesBetween(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Intake, error) {
	const q = `
		SELECT id, user_id, amount_ml, source, consumed_at
		FROM water_intakes
		WHERE user_id = $1 AND consumed_at >= $2 AND consumed_at < $3
		ORDER BY consumed_at`
	rows, err := r.db.Read().Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	intakes := []Intake{}
	for rows.Next() {
		var in Intake
		if err := rows.Scan(&in.ID, &in.UserID, &in.AmountML, &in.Source, &in.ConsumedAt); err != nil {
			return nil, err
		}
		intakes = append(intakes, in)
	}
	return intakes, rows.Err()
}

// SumBetween returns the total millilitres consumed in [from, to).
func (r *Repo) SumBetween(ctx context.Context, userID uuid.UUID, from, to time.Time) (int, error) {
	const q = `
		SELECT COALESCE(SUM(amount_ml), 0)
		FROM water_intakes
		WHERE user_id = $1 AND consumed_at >= $2 AND consumed_at < $3`
	var total int
	err := r.db.Read().QueryRow(ctx, q, userID, from, to).Scan(&total)
	return total, err
}

// DailyTotals aggregates intake per calendar day (in the given timezone) across
// [from, to). The date_trunc runs in the requested zone so day boundaries match
// the user's expectation.
func (r *Repo) DailyTotals(ctx context.Context, userID uuid.UUID, from, to time.Time, tz string) ([]DayTotal, error) {
	const q = `
		SELECT to_char(date_trunc('day', consumed_at AT TIME ZONE $4), 'YYYY-MM-DD') AS day,
		       SUM(amount_ml)::int AS total
		FROM water_intakes
		WHERE user_id = $1 AND consumed_at >= $2 AND consumed_at < $3
		GROUP BY day
		ORDER BY day`
	rows, err := r.db.Read().Query(ctx, q, userID, from, to, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totals := []DayTotal{}
	for rows.Next() {
		var d DayTotal
		if err := rows.Scan(&d.Date, &d.TotalML); err != nil {
			return nil, err
		}
		totals = append(totals, d)
	}
	return totals, rows.Err()
}
