package nutrition

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// LogRepo owns goals, the diet log and activities.
type LogRepo struct{ db *postgres.DB }

func NewLogRepo(db *postgres.DB) *LogRepo { return &LogRepo{db: db} }

// --- goals ---

func (r *LogRepo) GetGoal(ctx context.Context, userID uuid.UUID) (*Goal, error) {
	const q = `SELECT kcal, protein_g::float8, fat_g::float8, carbs_g::float8, updated_at
	           FROM nutrition_goals WHERE user_id=$1`
	var g Goal
	err := r.db.Read().QueryRow(ctx, q, userID).Scan(&g.Kcal, &g.Protein, &g.Fat, &g.Carbs, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *LogRepo) UpsertGoal(ctx context.Context, userID uuid.UUID, g Goal) (*Goal, error) {
	const q = `
		INSERT INTO nutrition_goals (user_id, kcal, protein_g, fat_g, carbs_g)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET kcal=EXCLUDED.kcal, protein_g=EXCLUDED.protein_g,
			fat_g=EXCLUDED.fat_g, carbs_g=EXCLUDED.carbs_g, updated_at=now()
		RETURNING kcal, protein_g::float8, fat_g::float8, carbs_g::float8, updated_at`
	var out Goal
	err := r.db.Pool.QueryRow(ctx, q, userID, g.Kcal, g.Protein, g.Fat, g.Carbs).
		Scan(&out.Kcal, &out.Protein, &out.Fat, &out.Carbs, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// --- diet entries ---

func (r *LogRepo) AddDietEntry(ctx context.Context, userID uuid.UUID, e DietEntry) (*DietEntry, error) {
	const q = `
		INSERT INTO diet_entries (user_id, dish_id, name, grams, meal, source, kcal, protein_g, fat_g, carbs_g, consumed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, dish_id, name, grams::float8, meal, source, kcal::float8, protein_g::float8, fat_g::float8, carbs_g::float8, consumed_at`
	var out DietEntry
	err := r.db.Pool.QueryRow(ctx, q,
		userID, e.DishID, e.Name, e.Grams, e.Meal, e.Source,
		e.Macros.Kcal, e.Macros.Protein, e.Macros.Fat, e.Macros.Carbs, e.ConsumedAt,
	).Scan(&out.ID, &out.DishID, &out.Name, &out.Grams, &out.Meal, &out.Source,
		&out.Macros.Kcal, &out.Macros.Protein, &out.Macros.Fat, &out.Macros.Carbs, &out.ConsumedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *LogRepo) DeleteDietEntry(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM diet_entries WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *LogRepo) ListDietEntries(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]DietEntry, error) {
	const q = `
		SELECT id, dish_id, name, grams::float8, meal, source, kcal::float8, protein_g::float8, fat_g::float8, carbs_g::float8, consumed_at
		FROM diet_entries WHERE user_id=$1 AND consumed_at >= $2 AND consumed_at < $3
		ORDER BY consumed_at`
	rows, err := r.db.Read().Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []DietEntry{}
	for rows.Next() {
		var e DietEntry
		if err := rows.Scan(&e.ID, &e.DishID, &e.Name, &e.Grams, &e.Meal, &e.Source,
			&e.Macros.Kcal, &e.Macros.Protein, &e.Macros.Fat, &e.Macros.Carbs, &e.ConsumedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- activities ---

func (r *LogRepo) AddActivity(ctx context.Context, userID uuid.UUID, a Activity) (*Activity, error) {
	const q = `
		INSERT INTO activities (user_id, type, kcal, duration_min, met, source, performed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, type, kcal::float8, duration_min, met::float8, source, performed_at`
	var out Activity
	err := r.db.Pool.QueryRow(ctx, q, userID, a.Type, a.Kcal, a.DurationMin, a.MET, a.Source, a.PerformedAt).
		Scan(&out.ID, &out.Type, &out.Kcal, &out.DurationMin, &out.MET, &out.Source, &out.PerformedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *LogRepo) DeleteActivity(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM activities WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *LogRepo) ListActivities(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]Activity, error) {
	const q = `
		SELECT id, type, kcal::float8, duration_min, met::float8, source, performed_at
		FROM activities WHERE user_id=$1 AND performed_at >= $2 AND performed_at < $3
		ORDER BY performed_at`
	rows, err := r.db.Read().Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	acts := []Activity{}
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.Type, &a.Kcal, &a.DurationMin, &a.MET, &a.Source, &a.PerformedAt); err != nil {
			return nil, err
		}
		acts = append(acts, a)
	}
	return acts, rows.Err()
}

// --- history aggregates (bucketed by local day) ---

// DailyTotals returns per-day consumed macros and burned kcal across [from,to).
func (r *LogRepo) DailyTotals(ctx context.Context, userID uuid.UUID, from, to time.Time, tz string) (map[string]DayTotals, error) {
	out := map[string]DayTotals{}

	const dietQ = `
		SELECT to_char(date_trunc('day', consumed_at AT TIME ZONE $4), 'YYYY-MM-DD') AS day,
		       SUM(kcal)::float8, SUM(protein_g)::float8, SUM(fat_g)::float8, SUM(carbs_g)::float8
		FROM diet_entries WHERE user_id=$1 AND consumed_at >= $2 AND consumed_at < $3
		GROUP BY day`
	rows, err := r.db.Read().Query(ctx, dietQ, userID, from, to, tz)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d DayTotals
		if err := rows.Scan(&d.Date, &d.Kcal, &d.Protein, &d.Fat, &d.Carbs); err != nil {
			rows.Close()
			return nil, err
		}
		out[d.Date] = d
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	const actQ = `
		SELECT to_char(date_trunc('day', performed_at AT TIME ZONE $4), 'YYYY-MM-DD') AS day, SUM(kcal)::float8
		FROM activities WHERE user_id=$1 AND performed_at >= $2 AND performed_at < $3
		GROUP BY day`
	arows, err := r.db.Read().Query(ctx, actQ, userID, from, to, tz)
	if err != nil {
		return nil, err
	}
	defer arows.Close()
	for arows.Next() {
		var day string
		var burned float64
		if err := arows.Scan(&day, &burned); err != nil {
			return nil, err
		}
		d := out[day]
		d.Date = day
		d.BurnedKcal = burned
		out[day] = d
	}
	return out, arows.Err()
}
