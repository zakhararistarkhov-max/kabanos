package reminders

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

const cols = `id, user_id, title, body, url, mode, interval_minutes, window_start, window_end,
	times, days, condition, timezone, enabled, last_fired_at, created_at, updated_at`

func scan(row pgx.Row) (*Reminder, error) {
	var r Reminder
	err := row.Scan(&r.ID, &r.UserID, &r.Title, &r.Body, &r.URL, &r.Mode, &r.IntervalMinutes,
		&r.WindowStart, &r.WindowEnd, &r.Times, &r.Days, &r.Condition, &r.Timezone, &r.Enabled,
		&r.LastFiredAt, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, in Input) (*Reminder, error) {
	const q = `
		INSERT INTO reminders (user_id, title, body, url, mode, interval_minutes, window_start, window_end,
			times, days, condition, timezone, enabled, last_fired_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13, now())
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, userID, in.Title, in.Body, in.URL, in.Mode, in.IntervalMinutes,
		in.WindowStart, in.WindowEnd, in.Times, in.Days, in.Condition, in.Timezone, in.Enabled))
}

func (r *Repo) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Reminder, error) {
	const q = `
		UPDATE reminders SET title=$3, body=$4, url=$5, mode=$6, interval_minutes=$7, window_start=$8,
			window_end=$9, times=$10, days=$11, condition=$12, timezone=$13, enabled=$14, updated_at=now()
		WHERE id=$1 AND user_id=$2
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, id, userID, in.Title, in.Body, in.URL, in.Mode, in.IntervalMinutes,
		in.WindowStart, in.WindowEnd, in.Times, in.Days, in.Condition, in.Timezone, in.Enabled))
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM reminders WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) ListByUser(ctx context.Context, userID uuid.UUID) ([]Reminder, error) {
	return r.query(ctx, `SELECT `+cols+` FROM reminders WHERE user_id=$1 ORDER BY created_at`, userID)
}

// ListEnabled returns all enabled reminders across users (for the worker).
func (r *Repo) ListEnabled(ctx context.Context) ([]Reminder, error) {
	return r.query(ctx, `SELECT `+cols+` FROM reminders WHERE enabled`)
}

func (r *Repo) query(ctx context.Context, q string, args ...any) ([]Reminder, error) {
	rows, err := r.db.Read().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reminder{}
	for rows.Next() {
		rem, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rem)
	}
	return out, rows.Err()
}

// ClaimFire atomically marks the reminder as fired only if its last_fired_at is
// still `prev`. Returns true if this caller won the claim — the guard makes
// multi-worker delivery safe (only one replica fires each occurrence).
func (r *Repo) ClaimFire(ctx context.Context, id uuid.UUID, prev *time.Time, now time.Time) (bool, error) {
	const q = `UPDATE reminders SET last_fired_at=$3, updated_at=now()
		WHERE id=$1 AND enabled AND last_fired_at IS NOT DISTINCT FROM $2`
	ct, err := r.db.Pool.Exec(ctx, q, id, prev, now)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// --- condition checks ---

// WaterBelowGoal reports whether the user's water intake in [from,to) is below
// their daily goal (default 2000 ml when unset).
func (r *Repo) WaterBelowGoal(ctx context.Context, userID uuid.UUID, from, to time.Time) (bool, error) {
	var goal int
	if err := r.db.Read().QueryRow(ctx, `SELECT COALESCE((SELECT daily_ml FROM water_goals WHERE user_id=$1), 2000)`, userID).Scan(&goal); err != nil {
		return false, err
	}
	var consumed int
	if err := r.db.Read().QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_ml),0) FROM water_intakes WHERE user_id=$1 AND consumed_at >= $2 AND consumed_at < $3`,
		userID, from, to).Scan(&consumed); err != nil {
		return false, err
	}
	return consumed < goal, nil
}

// MedsDue reports whether the user has any active medication whose scheduled
// intakes for `dateStr` (YYYY-MM-DD) are not yet all logged.
func (r *Repo) MedsDue(ctx context.Context, userID uuid.UUID, dateStr string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM medications m
			WHERE m.user_id=$1 AND m.active
			  AND (SELECT count(*) FROM medication_intakes i WHERE i.medication_id=m.id AND i.taken_on=$2::date) < m.times_per_day
		)`
	var due bool
	if err := r.db.Read().QueryRow(ctx, q, userID, dateStr).Scan(&due); err != nil {
		return false, err
	}
	return due, nil
}
