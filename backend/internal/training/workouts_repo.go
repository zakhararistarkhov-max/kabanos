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

// WorkoutRepo owns workout persistence, the ordered exercise list, and the
// workout's social features (embedded).
type WorkoutRepo struct {
	db *postgres.DB
	*social
}

func NewWorkoutRepo(db *postgres.DB) *WorkoutRepo {
	return &WorkoutRepo{db: db, social: newSocial(db, "workout")}
}

// woColumns selects a workout plus viewer-specific flags. $1 = viewer id.
const woColumns = `
	w.id, w.created_by, w.name, w.description, w.difficulty, w.image_key,
	w.rating_count, w.rating_sum, w.created_at, w.updated_at,
	u.display_name,
	EXISTS(SELECT 1 FROM workout_favorites f WHERE f.workout_id = w.id AND f.user_id = $1) AS is_favorite,
	(SELECT r.rating FROM workout_ratings r WHERE r.workout_id = w.id AND r.user_id = $1) AS my_rating`

func scanWorkout(row pgx.Row) (*Workout, error) {
	var w Workout
	err := row.Scan(
		&w.ID, &w.CreatedBy, &w.Name, &w.Description, &w.Difficulty, &w.ImageKey,
		&w.RatingCount, &w.RatingSum, &w.CreatedAt, &w.UpdatedAt,
		&w.AuthorName, &w.IsFavorite, &w.MyRating,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WorkoutRepo) Create(ctx context.Context, w *Workout, items []ItemInput) (*Workout, error) {
	var id uuid.UUID
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		const q = `
			INSERT INTO workouts (created_by, name, description, difficulty, image_key)
			VALUES ($1,$2,$3,$4,$5) RETURNING id`
		if err := tx.QueryRow(ctx, q, w.CreatedBy, w.Name, w.Description, w.Difficulty, w.ImageKey).Scan(&id); err != nil {
			return err
		}
		return insertItems(ctx, tx, id, items)
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, w.CreatedBy)
}

func (r *WorkoutRepo) Update(ctx context.Context, w *Workout, items []ItemInput, ownerID uuid.UUID) (*Workout, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		// image_key uses COALESCE so an edit without a new photo keeps the old one.
		const q = `
			UPDATE workouts SET name=$3, description=$4, difficulty=$5,
				image_key=COALESCE($6, image_key), updated_at=now()
			WHERE id=$1 AND created_by=$2`
		ct, err := tx.Exec(ctx, q, w.ID, ownerID, w.Name, w.Description, w.Difficulty, w.ImageKey)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return postgres.ErrNotFound
		}
		// Replace the item list wholesale — simplest correct way to persist an
		// arbitrary reorder/insert/remove.
		if _, err := tx.Exec(ctx, `DELETE FROM workout_exercises WHERE workout_id=$1`, w.ID); err != nil {
			return err
		}
		return insertItems(ctx, tx, w.ID, items)
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, w.ID, ownerID)
}

func insertItems(ctx context.Context, tx pgx.Tx, workoutID uuid.UUID, items []ItemInput) error {
	const q = `
		INSERT INTO workout_exercises
			(workout_id, exercise_id, position, sets, reps, duration_sec, rest_sec, weight_kg, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	for i, it := range items {
		if _, err := tx.Exec(ctx, q, workoutID, it.ExerciseID, i+1,
			it.Sets, it.Reps, it.DurationSec, it.RestSec, it.WeightKg, it.Note); err != nil {
			return err
		}
	}
	return nil
}

func (r *WorkoutRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM workouts WHERE id=$1 AND created_by=$2`, id, ownerID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// GetByID returns a workout with its ordered item list hydrated.
func (r *WorkoutRepo) GetByID(ctx context.Context, id, viewerID uuid.UUID) (*Workout, error) {
	q := `SELECT ` + woColumns + ` FROM workouts w JOIN users u ON u.id = w.created_by WHERE w.id = $2`
	w, err := scanWorkout(r.db.Read().QueryRow(ctx, q, viewerID, id))
	if err != nil {
		return nil, err
	}
	items, err := r.listItems(ctx, id)
	if err != nil {
		return nil, err
	}
	w.Items = items
	return w, nil
}

func (r *WorkoutRepo) listItems(ctx context.Context, workoutID uuid.UUID) ([]WorkoutItem, error) {
	const q = `
		SELECT wi.id, wi.exercise_id, wi.position, wi.sets, wi.reps, wi.duration_sec, wi.rest_sec,
			wi.weight_kg::float8, wi.note,
			ex.name, ex.category, ex.difficulty, ex.joint_impact, ex.equipment, ex.image_key,
			ex.rating_count, ex.rating_sum
		FROM workout_exercises wi JOIN exercises ex ON ex.id = wi.exercise_id
		WHERE wi.workout_id = $1 ORDER BY wi.position ASC`
	rows, err := r.db.Read().Query(ctx, q, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WorkoutItem{}
	for rows.Next() {
		var it WorkoutItem
		var ref ExerciseRef
		if err := rows.Scan(&it.ID, &it.ExerciseID, &it.Position, &it.Sets, &it.Reps, &it.DurationSec,
			&it.RestSec, &it.WeightKg, &it.Note,
			&ref.Name, &ref.Category, &ref.Difficulty, &ref.JointImpact, &ref.Equipment, &ref.ImageKey,
			&ref.RatingCount, &ref.RatingSum); err != nil {
			return nil, err
		}
		ref.ID = it.ExerciseID
		it.Exercise = ref
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *WorkoutRepo) List(ctx context.Context, p ListParams) ([]Workout, int, error) {
	buildWhere := func(args *[]any) string {
		conds := []string{}
		switch p.Scope {
		case "mine":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("w.created_by = $%d", len(*args)))
		case "favorites":
			*args = append(*args, p.ViewerID)
			conds = append(conds, fmt.Sprintf("EXISTS(SELECT 1 FROM workout_favorites ff WHERE ff.workout_id = w.id AND ff.user_id = $%d)", len(*args)))
		}
		if q := strings.TrimSpace(p.Query); q != "" {
			*args = append(*args, "%"+q+"%")
			conds = append(conds, fmt.Sprintf("w.name ILIKE $%d", len(*args)))
		}
		if p.Difficulty != "" {
			*args = append(*args, p.Difficulty)
			conds = append(conds, fmt.Sprintf("w.difficulty = $%d", len(*args)))
		}
		if len(conds) == 0 {
			return ""
		}
		return " WHERE " + strings.Join(conds, " AND ")
	}

	order := "w.created_at DESC"
	switch p.Sort {
	case "rating":
		order = "(CASE WHEN w.rating_count=0 THEN -1 ELSE w.rating_sum::float8/w.rating_count END) DESC, w.rating_count DESC"
	case "name":
		order = "lower(w.name) ASC"
	}

	countArgs := []any{}
	countSQL := `SELECT count(*) FROM workouts w` + buildWhere(&countArgs)
	var total int
	if err := r.db.Read().QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 24
	}
	listArgs := []any{p.ViewerID}
	whereSQL := buildWhere(&listArgs)
	listArgs = append(listArgs, p.Limit)
	limitIdx := len(listArgs)
	listArgs = append(listArgs, p.Offset)
	offsetIdx := len(listArgs)

	listSQL := fmt.Sprintf(`SELECT %s FROM workouts w JOIN users u ON u.id = w.created_by%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		woColumns, whereSQL, order, limitIdx, offsetIdx)

	rows, err := r.db.Read().Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// exercise_count for cards, fetched in a single follow-up query to avoid an
	// N+1: collect ids first.
	items := []Workout{}
	for rows.Next() {
		w, err := scanWorkout(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *w)
	}
	return items, total, rows.Err()
}

// ItemCounts returns the number of exercises per workout id, for list cards.
func (r *WorkoutRepo) ItemCounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	out := map[uuid.UUID]int{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Read().Query(ctx,
		`SELECT workout_id, count(*) FROM workout_exercises WHERE workout_id = ANY($1) GROUP BY workout_id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}
