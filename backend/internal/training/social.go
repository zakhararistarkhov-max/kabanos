package training

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// social implements ratings, comments and favorites for a catalog entity. The
// exercise and workout repos each hold one; the table/column names are derived
// from the entity constant (never from user input) so the interpolated SQL is
// safe. This keeps the two resources' social behaviour identical and in one
// place.
type social struct {
	db     *postgres.DB
	parent string // "exercises" | "workouts"
	rate   string // "<entity>_ratings"
	cmt    string // "<entity>_comments"
	fav    string // "<entity>_favorites"
	fk     string // "<entity>_id"
}

func newSocial(db *postgres.DB, entity string) *social {
	return &social{
		db:     db,
		parent: entity + "s",
		rate:   entity + "_ratings",
		cmt:    entity + "_comments",
		fav:    entity + "_favorites",
		fk:     entity + "_id",
	}
}

// --- ratings (denormalized aggregates kept consistent inside a transaction) ---

func (s *social) SetRating(ctx context.Context, entityID, userID uuid.UUID, rating int) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		q := fmt.Sprintf(`
			INSERT INTO %s (%s, user_id, rating) VALUES ($1,$2,$3)
			ON CONFLICT (%s, user_id) DO UPDATE SET rating=EXCLUDED.rating, updated_at=now()`,
			s.rate, s.fk, s.fk)
		if _, err := tx.Exec(ctx, q, entityID, userID, rating); err != nil {
			return err
		}
		return s.recompute(ctx, tx, entityID)
	})
}

func (s *social) DeleteRating(ctx context.Context, entityID, userID uuid.UUID) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		q := fmt.Sprintf(`DELETE FROM %s WHERE %s=$1 AND user_id=$2`, s.rate, s.fk)
		if _, err := tx.Exec(ctx, q, entityID, userID); err != nil {
			return err
		}
		return s.recompute(ctx, tx, entityID)
	})
}

func (s *social) recompute(ctx context.Context, tx pgx.Tx, entityID uuid.UUID) error {
	q := fmt.Sprintf(`
		UPDATE %s SET
			rating_count = (SELECT count(*) FROM %s WHERE %s=$1),
			rating_sum   = (SELECT COALESCE(SUM(rating),0) FROM %s WHERE %s=$1)
		WHERE id=$1`, s.parent, s.rate, s.fk, s.rate, s.fk)
	_, err := tx.Exec(ctx, q, entityID)
	return err
}

// --- comments ---

func (s *social) AddComment(ctx context.Context, entityID, userID uuid.UUID, body string) (*Comment, error) {
	q := fmt.Sprintf(`
		WITH ins AS (
			INSERT INTO %s (%s, user_id, body) VALUES ($1,$2,$3)
			RETURNING id, %s, user_id, body, created_at
		)
		SELECT ins.id, ins.%s, ins.user_id, u.display_name, ins.body, ins.created_at
		FROM ins JOIN users u ON u.id = ins.user_id`, s.cmt, s.fk, s.fk, s.fk)
	var c Comment
	err := s.db.Pool.QueryRow(ctx, q, entityID, userID, body).
		Scan(&c.ID, &c.EntityID, &c.UserID, &c.AuthorName, &c.Body, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *social) ListComments(ctx context.Context, entityID uuid.UUID, limit int) ([]Comment, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := fmt.Sprintf(`
		SELECT c.id, c.%s, c.user_id, u.display_name, c.body, c.created_at
		FROM %s c JOIN users u ON u.id = c.user_id
		WHERE c.%s = $1 ORDER BY c.created_at DESC LIMIT $2`, s.fk, s.cmt, s.fk)
	rows, err := s.db.Read().Query(ctx, q, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.EntityID, &c.UserID, &c.AuthorName, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *social) DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE id=$1 AND user_id=$2`, s.cmt)
	ct, err := s.db.Pool.Exec(ctx, q, commentID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// --- favorites ---

func (s *social) AddFavorite(ctx context.Context, userID, entityID uuid.UUID) error {
	q := fmt.Sprintf(`INSERT INTO %s (user_id, %s) VALUES ($1,$2) ON CONFLICT DO NOTHING`, s.fav, s.fk)
	_, err := s.db.Pool.Exec(ctx, q, userID, entityID)
	return err
}

func (s *social) RemoveFavorite(ctx context.Context, userID, entityID uuid.UUID) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE user_id=$1 AND %s=$2`, s.fav, s.fk)
	_, err := s.db.Pool.Exec(ctx, q, userID, entityID)
	return err
}

func (s *social) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
