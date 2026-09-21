package decisions

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

const cols = `id, user_id, title, context, decision, expected, confidence,
	to_char(decided_on,'YYYY-MM-DD'), to_char(review_at,'YYYY-MM-DD'), status,
	result, rating, lessons, reviewed_at, created_at, updated_at,
	(status='open' AND review_at IS NOT NULL AND review_at <= current_date) AS due`

func scan(row pgx.Row) (*Decision, error) {
	var d Decision
	if err := row.Scan(&d.ID, &d.UserID, &d.Title, &d.Context, &d.Decision, &d.Expected, &d.Confidence,
		&d.DecidedOn, &d.ReviewAt, &d.Status, &d.Result, &d.Rating, &d.Lessons, &d.ReviewedAt,
		&d.CreatedAt, &d.UpdatedAt, &d.Due); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

// List returns the user's decisions, due-for-review first, then other open ones
// by soonest review date, then reviewed ones most-recent first.
func (r *Repo) List(ctx context.Context, userID uuid.UUID) ([]Decision, error) {
	const q = `SELECT ` + cols + ` FROM decisions WHERE user_id=$1
		ORDER BY (status='reviewed'),
		         (status='open' AND review_at IS NOT NULL AND review_at <= current_date) DESC,
		         COALESCE(review_at, decided_on),
		         created_at DESC`
	rows, err := r.db.Read().Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Decision{}
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, in Input) (*Decision, error) {
	const q = `INSERT INTO decisions (user_id, title, context, decision, expected, confidence, decided_on, review_at)
		VALUES ($1,$2,$3,$4,$5,$6, COALESCE($7::date, current_date), $8::date)
		RETURNING ` + cols
	var decidedOn any
	if in.DecidedOn != "" {
		decidedOn = in.DecidedOn
	}
	return scan(r.db.Pool.QueryRow(ctx, q, userID, in.Title, in.Context, in.Decision, in.Expected,
		in.Confidence, decidedOn, in.ReviewAt))
}

func (r *Repo) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Decision, error) {
	const q = `UPDATE decisions SET title=$3, context=$4, decision=$5, expected=$6, confidence=$7,
		decided_on=COALESCE($8::date, decided_on), review_at=$9::date, updated_at=now()
		WHERE id=$1 AND user_id=$2 RETURNING ` + cols
	var decidedOn any
	if in.DecidedOn != "" {
		decidedOn = in.DecidedOn
	}
	return scan(r.db.Pool.QueryRow(ctx, q, id, userID, in.Title, in.Context, in.Decision, in.Expected,
		in.Confidence, decidedOn, in.ReviewAt))
}

// Review records the outcome and marks the decision reviewed.
func (r *Repo) Review(ctx context.Context, id, userID uuid.UUID, in ReviewInput) (*Decision, error) {
	const q = `UPDATE decisions SET result=$3, rating=$4, lessons=$5, status='reviewed',
		reviewed_at=now(), updated_at=now()
		WHERE id=$1 AND user_id=$2 RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, id, userID, in.Result, in.Rating, in.Lessons))
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM decisions WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// --- worker: review nudges ---

// DueNudge is a minimal row for the review-reminder push.
type DueNudge struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Title  string
}

// DueForNudge lists open decisions whose review date has arrived and that have
// not yet been nudged.
func (r *Repo) DueForNudge(ctx context.Context) ([]DueNudge, error) {
	const q = `SELECT id, user_id, title FROM decisions
		WHERE status='open' AND review_at IS NOT NULL AND review_at <= current_date AND notified_at IS NULL`
	rows, err := r.db.Read().Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DueNudge{}
	for rows.Next() {
		var n DueNudge
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ClaimNudge marks a decision as nudged; returns true if this caller won (so the
// push is sent exactly once across worker replicas).
func (r *Repo) ClaimNudge(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := r.db.Pool.Exec(ctx, `UPDATE decisions SET notified_at=now() WHERE id=$1 AND notified_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
