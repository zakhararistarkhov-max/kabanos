package training

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// ExerciseRepo owns exercise persistence plus its social features (embedded).
type ExerciseRepo struct {
	db *postgres.DB
	*social
}

func NewExerciseRepo(db *postgres.DB) *ExerciseRepo {
	return &ExerciseRepo{db: db, social: newSocial(db, "exercise")}
}

// exColumns selects an exercise plus viewer-specific flags. $1 = viewer id.
const exColumns = `
	e.id, e.created_by, e.name, e.description, e.category, e.difficulty, e.joint_impact,
	e.equipment, e.muscles, e.image_key, e.video_url, e.is_public,
	e.rating_count, e.rating_sum, e.created_at, e.updated_at,
	u.display_name,
	EXISTS(SELECT 1 FROM exercise_favorites f WHERE f.exercise_id = e.id AND f.user_id = $1) AS is_favorite,
	(SELECT r.rating FROM exercise_ratings r WHERE r.exercise_id = e.id AND r.user_id = $1) AS my_rating`

func scanExercise(row pgx.Row) (*Exercise, error) {
	var e Exercise
	err := row.Scan(
		&e.ID, &e.CreatedBy, &e.Name, &e.Description, &e.Category, &e.Difficulty, &e.JointImpact,
		&e.Equipment, &e.Muscles, &e.ImageKey, &e.VideoURL, &e.IsPublic,
		&e.RatingCount, &e.RatingSum, &e.CreatedAt, &e.UpdatedAt,
		&e.AuthorName, &e.IsFavorite, &e.MyRating,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *ExerciseRepo) Create(ctx context.Context, e *Exercise) (*Exercise, error) {
	const q = `
		INSERT INTO exercises (created_by, name, description, category, difficulty, joint_impact,
			equipment, muscles, image_key, video_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx, q,
		e.CreatedBy, e.Name, e.Description, e.Category, e.Difficulty, e.JointImpact,
		e.Equipment, e.Muscles, e.ImageKey, e.VideoURL,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, e.CreatedBy)
}

func (r *ExerciseRepo) GetByID(ctx context.Context, id, viewerID uuid.UUID) (*Exercise, error) {
	q := `SELECT ` + exColumns + ` FROM exercises e JOIN users u ON u.id = e.created_by WHERE e.id = $2`
	return scanExercise(r.db.Read().QueryRow(ctx, q, viewerID, id))
}

func (r *ExerciseRepo) Update(ctx context.Context, e *Exercise, ownerID uuid.UUID) (*Exercise, error) {
	// image_key uses COALESCE so an edit that doesn't re-upload a photo (nil key)
	// preserves the existing one rather than clearing it.
	const q = `
		UPDATE exercises SET name=$3, description=$4, category=$5, difficulty=$6, joint_impact=$7,
			equipment=$8, muscles=$9, image_key=COALESCE($10, image_key), video_url=$11, updated_at=now()
		WHERE id=$1 AND created_by=$2`
	ct, err := r.db.Pool.Exec(ctx, q, e.ID, ownerID, e.Name, e.Description, e.Category, e.Difficulty,
		e.JointImpact, e.Equipment, e.Muscles, e.ImageKey, e.VideoURL)
	if err != nil {
		return nil, err
	}
	if ct.RowsAffected() == 0 {
		return nil, postgres.ErrNotFound
	}
	return r.GetByID(ctx, e.ID, ownerID)
}

func (r *ExerciseRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM exercises WHERE id=$1 AND created_by=$2`, id, ownerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *ExerciseRepo) SetPublic(ctx context.Context, id, ownerID uuid.UUID, public bool) error {
	ct, err := r.db.Pool.Exec(ctx, `UPDATE exercises SET is_public=$3, updated_at=now() WHERE id=$1 AND created_by=$2`, id, ownerID, public)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *ExerciseRepo) List(ctx context.Context, p ListParams) ([]Exercise, int, error) {
	// buildWhere appends filter args to the given slice (numbering from its
	// current length) and returns the WHERE clause, so the count and list
	// queries can share the filter logic despite starting at different offsets.
	buildWhere := func(args *[]any) string {
		conds := []string{}
		switch p.Scope {
		case "mine":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("e.created_by = $%d", len(*args)))
		case "favorites":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("EXISTS(SELECT 1 FROM exercise_favorites ff WHERE ff.exercise_id = e.id AND ff.user_id = $%d)", len(*args)))
		default: // "all": only published exercises in the shared catalog
			conds = append(conds, "e.is_public")
		}
		if q := strings.TrimSpace(p.Query); q != "" {
			*args = append(*args, "%"+q+"%")
			conds = append(conds, fmt.Sprintf("e.name ILIKE $%d", len(*args)))
		}
		if p.Category != "" {
			*args = append(*args, p.Category)
			conds = append(conds, fmt.Sprintf("e.category = $%d", len(*args)))
		}
		if p.Difficulty != "" {
			*args = append(*args, p.Difficulty)
			conds = append(conds, fmt.Sprintf("e.difficulty = $%d", len(*args)))
		}
		if p.Equipment != "" {
			*args = append(*args, []string{p.Equipment})
			conds = append(conds, fmt.Sprintf("e.equipment @> $%d", len(*args)))
		}
		if len(conds) == 0 {
			return ""
		}
		return " WHERE " + strings.Join(conds, " AND ")
	}

	order := "e.created_at DESC"
	switch p.Sort {
	case "rating":
		order = "(CASE WHEN e.rating_count=0 THEN -1 ELSE e.rating_sum::float8/e.rating_count END) DESC, e.rating_count DESC"
	case "name":
		order = "lower(e.name) ASC"
	}

	countArgs := []any{}
	countSQL := `SELECT count(*) FROM exercises e` + buildWhere(&countArgs)
	var total int
	if err := r.db.Read().QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 24
	}
	listArgs := []any{p.ViewerID} // $1 reserved for viewer id in exColumns
	whereSQL := buildWhere(&listArgs)
	listArgs = append(listArgs, p.Limit)
	limitIdx := len(listArgs)
	listArgs = append(listArgs, p.Offset)
	offsetIdx := len(listArgs)

	listSQL := fmt.Sprintf(`SELECT %s FROM exercises e JOIN users u ON u.id = e.created_by%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		exColumns, whereSQL, order, limitIdx, offsetIdx)

	rows, err := r.db.Read().Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []Exercise{}
	for rows.Next() {
		e, err := scanExercise(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *e)
	}
	return items, total, rows.Err()
}

// existAll reports whether every id in the set exists (used to validate a
// workout's exercise references at save time). Any user's exercise may be used.
func (r *ExerciseRepo) existAll(ctx context.Context, ids []uuid.UUID) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var count int
	if err := r.db.Read().QueryRow(ctx, `SELECT count(*) FROM exercises WHERE id = ANY($1)`, ids).Scan(&count); err != nil {
		return false, err
	}
	return count == len(dedupeIDs(ids)), nil
}

func dedupeIDs(ids []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
