package nutrition

import (
	"context"
	"errors"
	"fmt"
	"math"
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

func (r *DishRepo) Create(ctx context.Context, d *Dish, ingredients []IngredientInput) (*Dish, error) {
	var id uuid.UUID
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		// When composed of ingredients, the parent's per-100g macros and serving
		// (total yield) are derived from them.
		if len(ingredients) > 0 {
			if err := applyIngredientMacros(ctx, tx, d, ingredients); err != nil {
				return err
			}
		}
		const q = `
			INSERT INTO dishes (created_by, name, description, recipe, image_key,
				kcal_per_100g, protein_per_100g, fat_per_100g, carbs_per_100g, serving_grams)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id`
		if err := tx.QueryRow(ctx, q,
			d.CreatedBy, d.Name, d.Description, d.Recipe, d.ImageKey,
			d.KcalPer100, d.ProteinPer100, d.FatPer100, d.CarbsPer100, d.ServingGrams,
		).Scan(&id); err != nil {
			return err
		}
		return insertIngredients(ctx, tx, id, ingredients)
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, d.CreatedBy)
}

func (r *DishRepo) GetByID(ctx context.Context, id, viewerID uuid.UUID) (*Dish, error) {
	q := `SELECT ` + dishColumns + ` FROM dishes d JOIN users u ON u.id = d.created_by WHERE d.id = $2`
	dish, err := scanDish(r.db.Read().QueryRow(ctx, q, viewerID, id))
	if err != nil {
		return nil, err
	}
	ings, err := r.listIngredients(ctx, id)
	if err != nil {
		return nil, err
	}
	dish.Ingredients = ings
	return dish, nil
}

// listIngredients returns a dish's ingredient lines with each ingredient dish's
// name and snapshot per-100g macros.
func (r *DishRepo) listIngredients(ctx context.Context, dishID uuid.UUID) ([]Ingredient, error) {
	const q = `
		SELECT di.ingredient_dish_id, d.name, di.grams::float8,
			d.kcal_per_100g::float8, d.protein_per_100g::float8, d.fat_per_100g::float8, d.carbs_per_100g::float8
		FROM dish_ingredients di JOIN dishes d ON d.id = di.ingredient_dish_id
		WHERE di.dish_id = $1 ORDER BY di.position`
	rows, err := r.db.Read().Query(ctx, q, dishID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ingredient{}
	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.DishID, &ing.Name, &ing.Grams,
			&ing.Per100.Kcal, &ing.Per100.Protein, &ing.Per100.Fat, &ing.Per100.Carbs); err != nil {
			return nil, err
		}
		out = append(out, ing)
	}
	return out, rows.Err()
}

// applyIngredientMacros fills d's per-100g macros and serving (total grams) from
// the given ingredients' stored per-100g values. Returns ErrIngredientNotFound
// if any referenced dish is missing.
func applyIngredientMacros(ctx context.Context, tx pgx.Tx, d *Dish, ingredients []IngredientInput) error {
	ids := make([]uuid.UUID, 0, len(ingredients))
	for _, ing := range ingredients {
		ids = append(ids, ing.DishID)
	}
	per100 := map[uuid.UUID]Macros{}
	rows, err := tx.Query(ctx, `
		SELECT id, kcal_per_100g::float8, protein_per_100g::float8, fat_per_100g::float8, carbs_per_100g::float8
		FROM dishes WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var m Macros
		if err := rows.Scan(&id, &m.Kcal, &m.Protein, &m.Fat, &m.Carbs); err != nil {
			return err
		}
		per100[id] = m
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var total Macros
	var totalG float64
	for _, ing := range ingredients {
		m, ok := per100[ing.DishID]
		if !ok {
			return ErrIngredientNotFound
		}
		f := ing.Grams / 100.0
		total.Kcal += m.Kcal * f
		total.Protein += m.Protein * f
		total.Fat += m.Fat * f
		total.Carbs += m.Carbs * f
		totalG += ing.Grams
	}
	if totalG <= 0 {
		return ErrIngredientNotFound
	}
	scale := 100.0 / totalG
	d.KcalPer100 = round2(total.Kcal * scale)
	d.ProteinPer100 = round2(total.Protein * scale)
	d.FatPer100 = round2(total.Fat * scale)
	d.CarbsPer100 = round2(total.Carbs * scale)
	g := round2(totalG)
	d.ServingGrams = &g
	return nil
}

func insertIngredients(ctx context.Context, tx pgx.Tx, dishID uuid.UUID, ingredients []IngredientInput) error {
	const q = `INSERT INTO dish_ingredients (dish_id, ingredient_dish_id, grams, position) VALUES ($1,$2,$3,$4)`
	for i, ing := range ingredients {
		if _, err := tx.Exec(ctx, q, dishID, ing.DishID, ing.Grams, i); err != nil {
			return err
		}
	}
	return nil
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// Update modifies a dish owned by ownerID and replaces its ingredient list.
// Returns ErrNotFound if the dish does not exist or is not owned by the caller.
func (r *DishRepo) Update(ctx context.Context, d *Dish, ingredients []IngredientInput, ownerID uuid.UUID) (*Dish, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		if len(ingredients) > 0 {
			if err := applyIngredientMacros(ctx, tx, d, ingredients); err != nil {
				return err
			}
		}
		const q = `
			UPDATE dishes SET name=$3, description=$4, recipe=$5, image_key=$6,
				kcal_per_100g=$7, protein_per_100g=$8, fat_per_100g=$9, carbs_per_100g=$10,
				serving_grams=$11, updated_at=now()
			WHERE id=$1 AND created_by=$2`
		ct, err := tx.Exec(ctx, q, d.ID, ownerID, d.Name, d.Description, d.Recipe, d.ImageKey,
			d.KcalPer100, d.ProteinPer100, d.FatPer100, d.CarbsPer100, d.ServingGrams)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return postgres.ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM dish_ingredients WHERE dish_id=$1`, d.ID); err != nil {
			return err
		}
		return insertIngredients(ctx, tx, d.ID, ingredients)
	})
	if err != nil {
		return nil, err
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
