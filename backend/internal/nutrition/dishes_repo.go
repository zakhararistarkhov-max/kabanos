package nutrition

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type DishRepo struct{ db *postgres.DB }

func NewDishRepo(db *postgres.DB) *DishRepo { return &DishRepo{db: db} }

// dishColumns selects a dish plus viewer-specific flags. $1 must be the viewer id.
const dishColumns = `
	d.id, d.created_by, d.name, d.description, d.recipe, d.image_key,
	d.kcal_per_100g::float8, d.protein_per_100g::float8, d.fat_per_100g::float8, d.carbs_per_100g::float8,
	d.serving_grams::float8, d.rating_count, d.rating_sum, d.created_at, d.updated_at,
	u.display_name,
	EXISTS(SELECT 1 FROM dish_favorites f WHERE f.dish_id = d.id AND f.user_id = $1) AS is_favorite,
	(SELECT r.rating FROM dish_ratings r WHERE r.dish_id = d.id AND r.user_id = $1) AS my_rating`

func scanDish(row pgx.Row) (*Dish, error) {
	var d Dish
	err := row.Scan(
		&d.ID, &d.CreatedBy, &d.Name, &d.Description, &d.Recipe, &d.ImageKey,
		&d.KcalPer100, &d.ProteinPer100, &d.FatPer100, &d.CarbsPer100,
		&d.ServingGrams, &d.RatingCount, &d.RatingSum, &d.CreatedAt, &d.UpdatedAt,
		&d.AuthorName, &d.IsFavorite, &d.MyRating,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DishRepo) Create(ctx context.Context, d *Dish) (*Dish, error) {
	const q = `
		INSERT INTO dishes (created_by, name, description, recipe, image_key,
			kcal_per_100g, protein_per_100g, fat_per_100g, carbs_per_100g, serving_grams)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx, q,
		d.CreatedBy, d.Name, d.Description, d.Recipe, d.ImageKey,
		d.KcalPer100, d.ProteinPer100, d.FatPer100, d.CarbsPer100, d.ServingGrams,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, d.CreatedBy)
}

func (r *DishRepo) GetByID(ctx context.Context, id, viewerID uuid.UUID) (*Dish, error) {
	q := `SELECT ` + dishColumns + ` FROM dishes d JOIN users u ON u.id = d.created_by WHERE d.id = $2`
	return scanDish(r.db.Read().QueryRow(ctx, q, viewerID, id))
}

// Update modifies a dish owned by ownerID. Returns ErrNotFound if the dish does
// not exist or is not owned by the caller.
func (r *DishRepo) Update(ctx context.Context, d *Dish, ownerID uuid.UUID) (*Dish, error) {
	const q = `
		UPDATE dishes SET name=$3, description=$4, recipe=$5, image_key=$6,
			kcal_per_100g=$7, protein_per_100g=$8, fat_per_100g=$9, carbs_per_100g=$10,
			serving_grams=$11, updated_at=now()
		WHERE id=$1 AND created_by=$2`
	ct, err := r.db.Pool.Exec(ctx, q, d.ID, ownerID, d.Name, d.Description, d.Recipe, d.ImageKey,
		d.KcalPer100, d.ProteinPer100, d.FatPer100, d.CarbsPer100, d.ServingGrams)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, postgres.ErrNotFound
	}
	return r.GetByID(ctx, d.ID, ownerID)
}

func (r *DishRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM dishes WHERE id=$1 AND created_by=$2`, id, ownerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ListParams controls the dish listing.
type ListParams struct {
	ViewerID uuid.UUID
	Scope    string // all | mine | favorites
	Query    string
	Sort     string // new | rating | name
	Limit    int
	Offset   int
}

func (r *DishRepo) List(ctx context.Context, p ListParams) ([]Dish, int, error) {
	// buildWhere appends filter args to the given slice (using its running
	// length for placeholder numbers) and returns the WHERE clause. It is called
	// separately for the count and list queries because they start numbering at
	// different offsets (the list query reserves $1 for the viewer id used in
	// dishColumns).
	buildWhere := func(args *[]any) string {
		conds := []string{}
		switch p.Scope {
		case "mine":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("d.created_by = $%d", len(*args)))
		case "favorites":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("EXISTS(SELECT 1 FROM dish_favorites ff WHERE ff.dish_id = d.id AND ff.user_id = $%d)", len(*args)))
		}
		if q := strings.TrimSpace(p.Query); q != "" {
			*args = append(*args, "%"+q+"%")
			conds = append(conds, fmt.Sprintf("d.name ILIKE $%d", len(*args)))
		}
		if len(conds) == 0 {
			return ""
		}
		return " WHERE " + strings.Join(conds, " AND ")
	}

	order := "d.created_at DESC"
	switch p.Sort {
	case "rating":
		order = "(CASE WHEN d.rating_count=0 THEN -1 ELSE d.rating_sum::float8/d.rating_count END) DESC, d.rating_count DESC"
	case "name":
		order = "lower(d.name) ASC"
	}

	// total count — its own arg numbering starting at $1.
	countArgs := []any{}
	countSQL := `SELECT count(*) FROM dishes d` + buildWhere(&countArgs)
	var total int
	if err := r.db.Read().QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 24
	}
	// list — $1 is the viewer id (dishColumns), filters follow.
	listArgs := []any{p.ViewerID}
	whereSQL := buildWhere(&listArgs)
	listArgs = append(listArgs, p.Limit)
	limitIdx := len(listArgs)
	listArgs = append(listArgs, p.Offset)
	offsetIdx := len(listArgs)

	listSQL := fmt.Sprintf(`SELECT %s FROM dishes d JOIN users u ON u.id = d.created_by%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		dishColumns, whereSQL, order, limitIdx, offsetIdx)

	rows, err := r.db.Read().Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	dishes := []Dish{}
	for rows.Next() {
		d, err := scanDish(rows)
		if err != nil {
			return nil, 0, err
		}
		dishes = append(dishes, *d)
	}
	return dishes, total, rows.Err()
}

// --- ratings (denormalized aggregates kept consistent in a transaction) ---

func (r *DishRepo) SetRating(ctx context.Context, dishID, userID uuid.UUID, rating int) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO dish_ratings (dish_id, user_id, rating) VALUES ($1,$2,$3)
			ON CONFLICT (dish_id, user_id) DO UPDATE SET rating=EXCLUDED.rating, updated_at=now()`,
			dishID, userID, rating); err != nil {
			return err
		}
		return recomputeRating(ctx, tx, dishID)
	})
}

func (r *DishRepo) DeleteRating(ctx context.Context, dishID, userID uuid.UUID) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM dish_ratings WHERE dish_id=$1 AND user_id=$2`, dishID, userID); err != nil {
			return err
		}
		return recomputeRating(ctx, tx, dishID)
	})
}

func recomputeRating(ctx context.Context, tx pgx.Tx, dishID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		UPDATE dishes SET
			rating_count = (SELECT count(*) FROM dish_ratings WHERE dish_id=$1),
			rating_sum   = (SELECT COALESCE(SUM(rating),0) FROM dish_ratings WHERE dish_id=$1)
		WHERE id=$1`, dishID)
	return err
}

// --- comments ---

func (r *DishRepo) AddComment(ctx context.Context, dishID, userID uuid.UUID, body string) (*Comment, error) {
	const q = `
		WITH ins AS (
			INSERT INTO dish_comments (dish_id, user_id, body) VALUES ($1,$2,$3)
			RETURNING id, dish_id, user_id, body, created_at
		)
		SELECT ins.id, ins.dish_id, ins.user_id, u.display_name, ins.body, ins.created_at
		FROM ins JOIN users u ON u.id = ins.user_id`
	var c Comment
	err := r.db.Pool.QueryRow(ctx, q, dishID, userID, body).
		Scan(&c.ID, &c.DishID, &c.UserID, &c.AuthorName, &c.Body, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *DishRepo) ListComments(ctx context.Context, dishID uuid.UUID, limit int) ([]Comment, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	const q = `
		SELECT c.id, c.dish_id, c.user_id, u.display_name, c.body, c.created_at
		FROM dish_comments c JOIN users u ON u.id = c.user_id
		WHERE c.dish_id = $1 ORDER BY c.created_at DESC LIMIT $2`
	rows, err := r.db.Read().Query(ctx, q, dishID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.DishID, &c.UserID, &c.AuthorName, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// DeleteComment removes a comment authored by userID.
func (r *DishRepo) DeleteComment(ctx context.Context, commentID, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM dish_comments WHERE id=$1 AND user_id=$2`, commentID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// --- favorites ---

func (r *DishRepo) AddFavorite(ctx context.Context, userID, dishID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO dish_favorites (user_id, dish_id) VALUES ($1,$2)
		ON CONFLICT DO NOTHING`, userID, dishID)
	return err
}

func (r *DishRepo) RemoveFavorite(ctx context.Context, userID, dishID uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM dish_favorites WHERE user_id=$1 AND dish_id=$2`, userID, dishID)
	return err
}

func (r *DishRepo) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
