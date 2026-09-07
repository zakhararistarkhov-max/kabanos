package diary

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

func scan(row pgx.Row) (*Entry, error) {
	var e Entry
	var atts []byte
	err := row.Scan(&e.ID, &e.UserID, &e.Date, &e.Body, &atts, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	if len(atts) > 0 {
		if err := json.Unmarshal(atts, &e.Attachments); err != nil {
			return nil, err
		}
	}
	if e.Attachments == nil {
		e.Attachments = []Attachment{}
	}
	return &e, nil
}

const cols = `id, user_id, to_char(entry_date, 'YYYY-MM-DD'), body, attachments, created_at, updated_at`

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, date, body string, atts []Attachment) (*Entry, error) {
	payload, err := json.Marshal(atts)
	if err != nil {
		return nil, err
	}
	const q = `
		INSERT INTO diary_entries (user_id, entry_date, body, attachments)
		VALUES ($1, $2::date, $3, $4::jsonb)
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, userID, date, body, string(payload)))
}

func (r *Repo) Update(ctx context.Context, id, userID uuid.UUID, body string, atts []Attachment) (*Entry, error) {
	payload, err := json.Marshal(atts)
	if err != nil {
		return nil, err
	}
	const q = `
		UPDATE diary_entries SET body=$3, attachments=$4::jsonb, updated_at=now()
		WHERE id=$1 AND user_id=$2
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, id, userID, body, string(payload)))
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM diary_entries WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ListByDate returns a day's entries, oldest first.
func (r *Repo) ListByDate(ctx context.Context, userID uuid.UUID, date string) ([]Entry, error) {
	const q = `SELECT ` + cols + ` FROM diary_entries WHERE user_id=$1 AND entry_date=$2::date ORDER BY created_at`
	rows, err := r.db.Read().Query(ctx, q, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Entry{}
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// DatesWithCounts lists days in [from, to] that have entries, for browsing.
func (r *Repo) DatesWithCounts(ctx context.Context, userID uuid.UUID, from, to string) ([]DateCount, error) {
	const q = `
		SELECT to_char(entry_date, 'YYYY-MM-DD'), count(*)
		FROM diary_entries WHERE user_id=$1 AND entry_date BETWEEN $2::date AND $3::date
		GROUP BY entry_date ORDER BY entry_date DESC`
	rows, err := r.db.Read().Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DateCount{}
	for rows.Next() {
		var dc DateCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		out = append(out, dc)
	}
	return out, rows.Err()
}
