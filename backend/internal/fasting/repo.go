package fasting

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

func defaultSettings() Settings { return Settings{FastingHours: 16, EatingHours: 8} }

func (r *Repo) GetSettings(ctx context.Context, userID uuid.UUID) (Settings, error) {
	var s Settings
	err := r.db.Read().QueryRow(ctx,
		`SELECT fasting_hours::float8, eating_hours::float8 FROM fasting_settings WHERE user_id=$1`, userID).
		Scan(&s.FastingHours, &s.EatingHours)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSettings(), nil
		}
		return Settings{}, err
	}
	return s, nil
}

func (r *Repo) UpsertSettings(ctx context.Context, userID uuid.UUID, s Settings) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO fasting_settings (user_id, fasting_hours, eating_hours) VALUES ($1,$2,$3)
		 ON CONFLICT (user_id) DO UPDATE SET fasting_hours=$2, eating_hours=$3, updated_at=now()`,
		userID, s.FastingHours, s.EatingHours)
	return err
}

func defaultSchedule() Schedule {
	return Schedule{Enabled: false, StartHour: 20, StartMinute: 0, Timezone: "UTC", AutoStart: true, NotifyStart: true, NotifyHourly: true}
}

// GetSchedule returns the user's daily schedule (defaults when no row yet).
func (r *Repo) GetSchedule(ctx context.Context, userID uuid.UUID) (Schedule, error) {
	var s Schedule
	err := r.db.Read().QueryRow(ctx,
		`SELECT schedule_enabled, start_hour, start_minute, timezone, auto_start, notify_start, notify_hourly
		 FROM fasting_settings WHERE user_id=$1`, userID).
		Scan(&s.Enabled, &s.StartHour, &s.StartMinute, &s.Timezone, &s.AutoStart, &s.NotifyStart, &s.NotifyHourly)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSchedule(), nil
		}
		return Schedule{}, err
	}
	return s, nil
}

// UpsertSchedule writes the schedule columns, leaving the protocol untouched.
// Changing the schedule resets the notification bookkeeping so a re-enable does
// not immediately fire a stale hourly push.
func (r *Repo) UpsertSchedule(ctx context.Context, userID uuid.UUID, s Schedule) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO fasting_settings (user_id, schedule_enabled, start_hour, start_minute, timezone, auto_start, notify_start, notify_hourly)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (user_id) DO UPDATE SET
		   schedule_enabled=$2, start_hour=$3, start_minute=$4, timezone=$5,
		   auto_start=$6, notify_start=$7, notify_hourly=$8, updated_at=now()`,
		userID, s.Enabled, s.StartHour, s.StartMinute, s.Timezone, s.AutoStart, s.NotifyStart, s.NotifyHourly)
	return err
}

// ListEnabledSchedules returns every user with an enabled schedule (for the worker).
func (r *Repo) ListEnabledSchedules(ctx context.Context) ([]ScheduleRow, error) {
	rows, err := r.db.Read().Query(ctx,
		`SELECT user_id, fasting_hours::float8, schedule_enabled, start_hour, start_minute, timezone,
		        auto_start, notify_start, notify_hourly, auto_started_on, notify_anchor, notify_last_hour
		 FROM fasting_settings WHERE schedule_enabled`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScheduleRow{}
	for rows.Next() {
		var sr ScheduleRow
		if err := rows.Scan(&sr.UserID, &sr.FastingHours, &sr.Schedule.Enabled, &sr.Schedule.StartHour,
			&sr.Schedule.StartMinute, &sr.Schedule.Timezone, &sr.Schedule.AutoStart, &sr.Schedule.NotifyStart,
			&sr.Schedule.NotifyHourly, &sr.AutoStartedOn, &sr.NotifyAnchor, &sr.NotifyLastHour); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}

// ClaimAutoStart marks today's local date as auto-started, but only if it was
// not already — the winning worker replica then creates the session.
func (r *Repo) ClaimAutoStart(ctx context.Context, userID uuid.UUID, localDate string) (bool, error) {
	ct, err := r.db.Pool.Exec(ctx,
		`UPDATE fasting_settings SET auto_started_on=$2::date, updated_at=now()
		 WHERE user_id=$1 AND auto_started_on IS DISTINCT FROM $2::date`, userID, localDate)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// ClaimNotify atomically advances the notification cursor from (prevAnchor,
// prevHour) to (anchor, hour). Only the caller that matches the current cursor
// wins, so exactly one replica sends each notification.
func (r *Repo) ClaimNotify(ctx context.Context, userID uuid.UUID, prevAnchor *time.Time, prevHour int, anchor time.Time, hour int) (bool, error) {
	ct, err := r.db.Pool.Exec(ctx,
		`UPDATE fasting_settings SET notify_anchor=$4, notify_last_hour=$5, updated_at=now()
		 WHERE user_id=$1 AND schedule_enabled
		   AND notify_anchor IS NOT DISTINCT FROM $2 AND notify_last_hour=$3`,
		userID, prevAnchor, prevHour, anchor, hour)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

const sessionCols = `id, user_id, started_at, ended_at, goal_hours::float8, created_at`

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	if err := row.Scan(&s.ID, &s.UserID, &s.StartedAt, &s.EndedAt, &s.GoalHours, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// GetActive returns the ongoing fast, or ErrNotFound.
func (r *Repo) GetActive(ctx context.Context, userID uuid.UUID) (*Session, error) {
	return scanSession(r.db.Read().QueryRow(ctx,
		`SELECT `+sessionCols+` FROM fasting_sessions WHERE user_id=$1 AND ended_at IS NULL`, userID))
}

// GetLast returns the most recently started fast (active or ended), or ErrNotFound.
func (r *Repo) GetLast(ctx context.Context, userID uuid.UUID) (*Session, error) {
	return scanSession(r.db.Read().QueryRow(ctx,
		`SELECT `+sessionCols+` FROM fasting_sessions WHERE user_id=$1 ORDER BY started_at DESC LIMIT 1`, userID))
}

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, startedAt any, goalHours float64) (*Session, error) {
	return scanSession(r.db.Pool.QueryRow(ctx,
		`INSERT INTO fasting_sessions (user_id, started_at, goal_hours) VALUES ($1,$2,$3) RETURNING `+sessionCols,
		userID, startedAt, goalHours))
}

func (r *Repo) End(ctx context.Context, id, userID uuid.UUID, endedAt any) (*Session, error) {
	return scanSession(r.db.Pool.QueryRow(ctx,
		`UPDATE fasting_sessions SET ended_at=$3 WHERE id=$1 AND user_id=$2 AND ended_at IS NULL RETURNING `+sessionCols,
		id, userID, endedAt))
}

func (r *Repo) UpdateActiveStart(ctx context.Context, userID uuid.UUID, startedAt any) (*Session, error) {
	return scanSession(r.db.Pool.QueryRow(ctx,
		`UPDATE fasting_sessions SET started_at=$2 WHERE user_id=$1 AND ended_at IS NULL RETURNING `+sessionCols,
		userID, startedAt))
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM fasting_sessions WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) List(ctx context.Context, userID uuid.UUID, limit int) ([]Session, error) {
	if limit <= 0 || limit > 200 {
		limit = 30
	}
	rows, err := r.db.Read().Query(ctx,
		`SELECT `+sessionCols+` FROM fasting_sessions WHERE user_id=$1 ORDER BY started_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// Stats aggregates completed (ended) fasts.
func (r *Repo) Stats(ctx context.Context, userID uuid.UUID) (Stats, error) {
	var st Stats
	err := r.db.Read().QueryRow(ctx,
		`SELECT count(*),
			COALESCE(max(EXTRACT(EPOCH FROM (ended_at - started_at)) / 3600.0), 0),
			COALESCE(avg(EXTRACT(EPOCH FROM (ended_at - started_at)) / 3600.0), 0)
		 FROM fasting_sessions WHERE user_id=$1 AND ended_at IS NOT NULL`, userID).
		Scan(&st.TotalFasts, &st.LongestHours, &st.AvgHours)
	return st, err
}
