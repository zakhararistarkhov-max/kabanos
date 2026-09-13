package gtd

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

// ---------- projects ----------

const projCols = `id, user_id, title, outcome, notes, status, created_at, updated_at, completed_at`

func scanProject(row pgx.Row, withCounts bool) (*Project, error) {
	var p Project
	dst := []any{&p.ID, &p.UserID, &p.Title, &p.Outcome, &p.Notes, &p.Status, &p.CreatedAt, &p.UpdatedAt, &p.CompletedAt}
	if withCounts {
		dst = append(dst, &p.OpenActions, &p.NextActions)
	}
	if err := row.Scan(dst...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repo) CreateProject(ctx context.Context, userID uuid.UUID, in ProjectInput) (*Project, error) {
	const q = `INSERT INTO gtd_projects (user_id, title, outcome, notes, status)
		VALUES ($1,$2,$3,$4,$5) RETURNING ` + projCols
	return scanProject(r.db.Pool.QueryRow(ctx, q, userID, in.Title, in.Outcome, in.Notes, in.Status), false)
}

func (r *Repo) UpdateProject(ctx context.Context, id, userID uuid.UUID, in ProjectInput) (*Project, error) {
	const q = `UPDATE gtd_projects
		SET title=$3, outcome=$4, notes=$5, status=$6,
			completed_at = CASE WHEN $6='done' AND completed_at IS NULL THEN now()
			                    WHEN $6<>'done' THEN NULL ELSE completed_at END,
			updated_at=now()
		WHERE id=$1 AND user_id=$2 RETURNING ` + projCols
	return scanProject(r.db.Pool.QueryRow(ctx, q, id, userID, in.Title, in.Outcome, in.Notes, in.Status), false)
}

func (r *Repo) DeleteProject(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM gtd_projects WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ListProjects returns the user's projects with open/next action counts.
func (r *Repo) ListProjects(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	const q = `
		SELECT ` + projCols + `,
			(SELECT count(*) FROM gtd_items i WHERE i.project_id=p.id AND NOT i.done) AS open_actions,
			(SELECT count(*) FROM gtd_items i WHERE i.project_id=p.id AND i.bucket='next' AND NOT i.done) AS next_actions
		FROM gtd_projects p
		WHERE p.user_id=$1
		ORDER BY (p.status='active') DESC, p.created_at DESC`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// StalledProjects returns active projects that have no open next action — the
// weekly review flags these because a stalled project is invisible progress.
func (r *Repo) StalledProjects(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	const q = `
		SELECT ` + projCols + `, 0, 0
		FROM gtd_projects p
		WHERE p.user_id=$1 AND p.status='active'
		  AND NOT EXISTS (
		      SELECT 1 FROM gtd_items i
		      WHERE i.project_id=p.id AND i.bucket='next' AND NOT i.done)
		ORDER BY p.created_at`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ---------- items ----------

const itemCols = `i.id, i.user_id, i.project_id, i.title, i.notes, i.bucket, i.context, i.waiting_for,
	i.scheduled_at, i.end_at, i.all_day, to_char(i.due_on,'YYYY-MM-DD'), i.energy, i.time_minutes,
	i.priority, i.done, i.completed_at, i.created_at, i.updated_at, COALESCE(p.title,'')`

const itemFrom = ` FROM gtd_items i LEFT JOIN gtd_projects p ON p.id=i.project_id `

func scanItem(row pgx.Row) (*Item, error) {
	var it Item
	if err := row.Scan(&it.ID, &it.UserID, &it.ProjectID, &it.Title, &it.Notes, &it.Bucket, &it.Context,
		&it.WaitingFor, &it.ScheduledAt, &it.EndAt, &it.AllDay, &it.DueOn, &it.Energy, &it.TimeMinutes,
		&it.Priority, &it.Done, &it.CompletedAt, &it.CreatedAt, &it.UpdatedAt, &it.ProjectTitle); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &it, nil
}

func (r *Repo) GetItem(ctx context.Context, id, userID uuid.UUID) (*Item, error) {
	q := `SELECT ` + itemCols + itemFrom + ` WHERE i.id=$1 AND i.user_id=$2`
	return scanItem(r.db.Read().QueryRow(ctx, q, id, userID))
}

func (r *Repo) CreateItem(ctx context.Context, userID uuid.UUID, in ItemInput) (*Item, error) {
	const q = `
		INSERT INTO gtd_items (user_id, project_id, title, notes, bucket, context, waiting_for,
			scheduled_at, end_at, all_day, due_on, energy, time_minutes, priority)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::date,$12,$13,$14)
		RETURNING id`
	var id uuid.UUID
	if err := r.db.Pool.QueryRow(ctx, q, userID, in.ProjectID, in.Title, in.Notes, in.Bucket, in.Context,
		in.WaitingFor, in.ScheduledAt, in.EndAt, in.AllDay, in.DueOn, in.Energy, in.TimeMinutes, in.Priority).
		Scan(&id); err != nil {
		return nil, err
	}
	return r.GetItem(ctx, id, userID)
}

func (r *Repo) UpdateItem(ctx context.Context, id, userID uuid.UUID, in ItemInput) (*Item, error) {
	const q = `
		UPDATE gtd_items SET project_id=$3, title=$4, notes=$5, bucket=$6, context=$7, waiting_for=$8,
			scheduled_at=$9, end_at=$10, all_day=$11, due_on=$12::date, energy=$13, time_minutes=$14,
			priority=$15, updated_at=now()
		WHERE id=$1 AND user_id=$2`
	ct, err := r.db.Pool.Exec(ctx, q, id, userID, in.ProjectID, in.Title, in.Notes, in.Bucket, in.Context,
		in.WaitingFor, in.ScheduledAt, in.EndAt, in.AllDay, in.DueOn, in.Energy, in.TimeMinutes, in.Priority)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, postgres.ErrNotFound
	}
	return r.GetItem(ctx, id, userID)
}

func (r *Repo) SetDone(ctx context.Context, id, userID uuid.UUID, done bool) (*Item, error) {
	const q = `UPDATE gtd_items
		SET done=$3, completed_at = CASE WHEN $3 THEN now() ELSE NULL END, updated_at=now()
		WHERE id=$1 AND user_id=$2`
	ct, err := r.db.Pool.Exec(ctx, q, id, userID, done)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, postgres.ErrNotFound
	}
	return r.GetItem(ctx, id, userID)
}

func (r *Repo) DeleteItem(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM gtd_items WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ListItems returns items matching the filter, ordered for the relevant view:
// calendar by time, everything else by priority then recency.
func (r *Repo) ListItems(ctx context.Context, userID uuid.UUID, f ItemFilter) ([]Item, error) {
	var sb strings.Builder
	sb.WriteString(`SELECT ` + itemCols + itemFrom + ` WHERE i.user_id=$1`)
	args := []any{userID}
	add := func(clause string, val any) {
		args = append(args, val)
		sb.WriteString(clause)
		sb.WriteString(itoa(len(args)))
	}
	if f.Bucket != "" {
		add(` AND i.bucket=$`, f.Bucket)
	}
	if f.Context != "" {
		add(` AND i.context=$`, f.Context)
	}
	if f.ProjectID != nil {
		add(` AND i.project_id=$`, *f.ProjectID)
	}
	if f.Done != nil {
		add(` AND i.done=$`, *f.Done)
	}
	if f.Energy != "" {
		add(` AND i.energy=$`, f.Energy)
	}
	if f.MaxTime != nil {
		// Items with no estimate still qualify (unknown effort shouldn't hide them).
		add(` AND (i.time_minutes IS NULL OR i.time_minutes<=$`, *f.MaxTime)
		sb.WriteString(`)`)
	}
	if f.Bucket == "calendar" {
		sb.WriteString(` ORDER BY i.scheduled_at NULLS LAST, i.created_at`)
	} else {
		sb.WriteString(` ORDER BY i.priority DESC, i.due_on NULLS LAST, i.created_at DESC`)
	}

	rows, err := r.db.Read().Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// ---------- review aggregates ----------

// OpenCountsByBucket returns the number of not-done items in each bucket.
func (r *Repo) OpenCountsByBucket(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Read().Query(ctx,
		`SELECT bucket, count(*) FROM gtd_items WHERE user_id=$1 AND NOT done GROUP BY bucket`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var b string
		var c int
		if err := rows.Scan(&b, &c); err != nil {
			return nil, err
		}
		out[b] = c
	}
	return out, rows.Err()
}

// ContextCounts groups open next actions by context.
func (r *Repo) ContextCounts(ctx context.Context, userID uuid.UUID) ([]ContextCount, error) {
	const q = `SELECT COALESCE(NULLIF(context,''),'(без контекста)'), count(*)
		FROM gtd_items WHERE user_id=$1 AND bucket='next' AND NOT done
		GROUP BY 1 ORDER BY 2 DESC`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContextCount{}
	for rows.Next() {
		var cc ContextCount
		if err := rows.Scan(&cc.Context, &cc.Count); err != nil {
			return nil, err
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

// Contexts returns the distinct non-empty contexts the user has used, for filters.
func (r *Repo) Contexts(ctx context.Context, userID uuid.UUID) ([]string, error) {
	const q = `SELECT DISTINCT context FROM gtd_items
		WHERE user_id=$1 AND context<>'' AND NOT done ORDER BY context`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// OverdueCalendar returns not-done calendar items whose start is before `now`.
func (r *Repo) OverdueCalendar(ctx context.Context, userID uuid.UUID, now time.Time) ([]Item, error) {
	q := `SELECT ` + itemCols + itemFrom +
		` WHERE i.user_id=$1 AND i.bucket='calendar' AND NOT i.done
		    AND i.scheduled_at IS NOT NULL AND i.scheduled_at < $2
		  ORDER BY i.scheduled_at`
	rows, err := r.db.Read().Query(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// UpcomingCalendarCount counts not-done calendar items at or after `now`.
func (r *Repo) UpcomingCalendarCount(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	var n int
	err := r.db.Read().QueryRow(ctx,
		`SELECT count(*) FROM gtd_items WHERE user_id=$1 AND bucket='calendar' AND NOT done
		   AND scheduled_at IS NOT NULL AND scheduled_at >= $2`, userID, now).Scan(&n)
	return n, err
}

// CompletedSince counts items marked done at or after `since`.
func (r *Repo) CompletedSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := r.db.Read().QueryRow(ctx,
		`SELECT count(*) FROM gtd_items WHERE user_id=$1 AND done AND completed_at >= $2`, userID, since).Scan(&n)
	return n, err
}

// itoa avoids importing strconv for a single positive int in the query builder.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
