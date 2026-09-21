package habits

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

const habitCols = `id, user_id, name, kind, description, sort_order, archived, created_at, updated_at`

func scanHabit(row pgx.Row) (*Habit, error) {
	var h Habit
	if err := row.Scan(&h.ID, &h.UserID, &h.Name, &h.Kind, &h.Description, &h.SortOrder, &h.Archived, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &h, nil
}

// List returns the user's non-archived habits with derived counts and the most
// recent log's status/time, ordered for display.
func (r *Repo) List(ctx context.Context, userID uuid.UUID) ([]Habit, error) {
	const q = `
		SELECT h.id, h.user_id, h.name, h.kind, h.description, h.sort_order, h.archived, h.created_at, h.updated_at,
			(SELECT count(*) FROM reminders rm WHERE rm.habit_id=h.id) AS reminder_count,
			(SELECT count(*) FROM habit_logs l WHERE l.habit_id=h.id) AS log_count,
			(SELECT status FROM habit_logs l WHERE l.habit_id=h.id ORDER BY created_at DESC LIMIT 1) AS last_status,
			(SELECT created_at FROM habit_logs l WHERE l.habit_id=h.id ORDER BY created_at DESC LIMIT 1) AS last_log_at
		FROM habits h
		WHERE h.user_id=$1 AND NOT h.archived
		ORDER BY h.kind, h.sort_order, h.created_at`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Habit{}
	for rows.Next() {
		var h Habit
		var lastStatus *string
		if err := rows.Scan(&h.ID, &h.UserID, &h.Name, &h.Kind, &h.Description, &h.SortOrder, &h.Archived,
			&h.CreatedAt, &h.UpdatedAt, &h.ReminderCount, &h.LogCount, &lastStatus, &h.LastLogAt); err != nil {
			return nil, err
		}
		if lastStatus != nil {
			h.LastStatus = *lastStatus
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Get returns one habit (ownership-checked).
func (r *Repo) Get(ctx context.Context, id, userID uuid.UUID) (*Habit, error) {
	return scanHabit(r.db.Read().QueryRow(ctx, `SELECT `+habitCols+` FROM habits WHERE id=$1 AND user_id=$2`, id, userID))
}

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, in Input) (*Habit, error) {
	const q = `INSERT INTO habits (user_id, name, kind, description) VALUES ($1,$2,$3,$4) RETURNING ` + habitCols
	return scanHabit(r.db.Pool.QueryRow(ctx, q, userID, in.Name, in.Kind, in.Description))
}

func (r *Repo) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Habit, error) {
	const q = `UPDATE habits SET name=$3, description=$4, archived=$5, updated_at=now()
		WHERE id=$1 AND user_id=$2 RETURNING ` + habitCols
	return scanHabit(r.db.Pool.QueryRow(ctx, q, id, userID, in.Name, in.Description, in.Archived))
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM habits WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// --- logs (mini-diary) ---

func (r *Repo) ListLogs(ctx context.Context, userID, habitID uuid.UUID, limit int) ([]Log, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `SELECT id, habit_id, note, status, created_at FROM habit_logs
		WHERE user_id=$1 AND habit_id=$2 ORDER BY created_at DESC LIMIT $3`
	rows, err := r.db.Read().Query(ctx, q, userID, habitID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Log{}
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.HabitID, &l.Note, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AddLog inserts a mini-diary entry, verifying the habit belongs to the user.
func (r *Repo) AddLog(ctx context.Context, userID, habitID uuid.UUID, note, status string) (*Log, error) {
	const q = `
		INSERT INTO habit_logs (user_id, habit_id, note, status)
		SELECT $1, $2, $3, $4 WHERE EXISTS (SELECT 1 FROM habits WHERE id=$2 AND user_id=$1)
		RETURNING id, habit_id, note, status, created_at`
	var l Log
	err := r.db.Pool.QueryRow(ctx, q, userID, habitID, note, status).Scan(&l.ID, &l.HabitID, &l.Note, &l.Status, &l.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound // habit not found / not owned
		}
		return nil, err
	}
	return &l, nil
}

func (r *Repo) DeleteLog(ctx context.Context, userID, logID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM habit_logs WHERE id=$1 AND user_id=$2`, logID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// --- check-ins (daily tracker) ---

type checkinRow struct {
	HabitID uuid.UUID
	Day     string
	Success bool
}

// RecentCheckins returns every check-in on or after sinceDay across the user's
// habits (for computing dashboard health + preview strips in one query).
func (r *Repo) RecentCheckins(ctx context.Context, userID uuid.UUID, sinceDay string) ([]checkinRow, error) {
	const q = `SELECT habit_id, to_char(day,'YYYY-MM-DD'), success FROM habit_checkins
		WHERE user_id=$1 AND day >= $2::date`
	rows, err := r.db.Read().Query(ctx, q, userID, sinceDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []checkinRow{}
	for rows.Next() {
		var c checkinRow
		if err := rows.Scan(&c.HabitID, &c.Day, &c.Success); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CheckinsRange returns one habit's check-ins in [from, to] (for the tracker).
func (r *Repo) CheckinsRange(ctx context.Context, userID, habitID uuid.UUID, from, to string) (map[string]bool, error) {
	const q = `SELECT to_char(day,'YYYY-MM-DD'), success FROM habit_checkins
		WHERE user_id=$1 AND habit_id=$2 AND day BETWEEN $3::date AND $4::date`
	rows, err := r.db.Read().Query(ctx, q, userID, habitID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var day string
		var success bool
		if err := rows.Scan(&day, &success); err != nil {
			return nil, err
		}
		out[day] = success
	}
	return out, rows.Err()
}

// SetCheckin upserts a day's result, verifying the habit belongs to the user.
func (r *Repo) SetCheckin(ctx context.Context, userID, habitID uuid.UUID, day string, success bool) error {
	const q = `
		INSERT INTO habit_checkins (user_id, habit_id, day, success)
		SELECT $1, $2, $3::date, $4 WHERE EXISTS (SELECT 1 FROM habits WHERE id=$2 AND user_id=$1)
		ON CONFLICT (habit_id, day) DO UPDATE SET success=$4, updated_at=now()`
	ct, err := r.db.Pool.Exec(ctx, q, userID, habitID, day, success)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound // habit not owned
	}
	return nil
}

// ClearCheckin removes a day's mark (untracked).
func (r *Repo) ClearCheckin(ctx context.Context, userID, habitID uuid.UUID, day string) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM habit_checkins WHERE user_id=$1 AND habit_id=$2 AND day=$3::date`, userID, habitID, day)
	return err
}

// Owns reports whether the habit exists and belongs to the user.
func (r *Repo) Owns(ctx context.Context, userID, habitID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.Read().QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM habits WHERE id=$1 AND user_id=$2)`, habitID, userID).Scan(&ok)
	return ok, err
}
