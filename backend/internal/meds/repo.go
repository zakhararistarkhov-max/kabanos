package meds

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

type Repo struct{ db *postgres.DB }

func NewRepo(db *postgres.DB) *Repo { return &Repo{db: db} }

const cols = `id, user_id, name, unit, dose::float8, times_per_day, start_date, duration_days, notes, active, created_at, updated_at`

func scan(row pgx.Row) (*Medication, error) {
	var m Medication
	err := row.Scan(&m.ID, &m.UserID, &m.Name, &m.Unit, &m.Dose, &m.TimesPerDay,
		&m.StartDate, &m.DurationDays, &m.Notes, &m.Active, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *Repo) Create(ctx context.Context, userID uuid.UUID, in Input) (*Medication, error) {
	const q = `
		INSERT INTO medications (user_id, name, unit, dose, times_per_day, start_date, duration_days, notes, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING ` + cols
	return scan(r.db.Pool.QueryRow(ctx, q, userID, in.Name, in.Unit, in.Dose, in.TimesPerDay,
		in.StartDate, in.DurationDays, in.Notes, in.Active))
}

func (r *Repo) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Medication, error) {
	const q = `
		UPDATE medications SET name=$3, unit=$4, dose=$5, times_per_day=$6, start_date=$7,
			duration_days=$8, notes=$9, active=$10, updated_at=now()
		WHERE id=$1 AND user_id=$2
		RETURNING ` + cols
	m, err := scan(r.db.Pool.QueryRow(ctx, q, id, userID, in.Name, in.Unit, in.Dose, in.TimesPerDay,
		in.StartDate, in.DurationDays, in.Notes, in.Active))
	return m, err
}

func (r *Repo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM medications WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// List returns the user's medications with intake counts for `today` and the
// trailing week [weekStart, today]. Dates are YYYY-MM-DD strings compared
// against the DATE columns.
func (r *Repo) List(ctx context.Context, userID uuid.UUID, today, weekStart string) ([]Progress, error) {
	q := `
		SELECT ` + prefixed("m") + `,
			(SELECT count(*) FROM medication_intakes i WHERE i.medication_id=m.id AND i.taken_on=$2::date) AS taken_today,
			(SELECT count(*) FROM medication_intakes i WHERE i.medication_id=m.id AND i.taken_on>=$3::date AND i.taken_on<=$2::date) AS week_taken
		FROM medications m
		WHERE m.user_id=$1
		ORDER BY m.active DESC, m.created_at DESC`
	rows, err := r.db.Read().Query(ctx, q, userID, today, weekStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Progress{}
	for rows.Next() {
		var p Progress
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Unit, &p.Dose, &p.TimesPerDay,
			&p.StartDate, &p.DurationDays, &p.Notes, &p.Active, &p.CreatedAt, &p.UpdatedAt,
			&p.TakenToday, &p.WeekTaken); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddIntake logs one intake for the given day, taking the dose from the
// medication. Returns ErrNotFound if the medication is not owned by the user.
func (r *Repo) AddIntake(ctx context.Context, medID, userID uuid.UUID, takenOn string) error {
	const q = `
		INSERT INTO medication_intakes (medication_id, user_id, amount, taken_on)
		SELECT m.id, m.user_id, m.dose, $3::date FROM medications m WHERE m.id=$1 AND m.user_id=$2`
	ct, err := r.db.Pool.Exec(ctx, q, medID, userID, takenOn)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// UndoIntake removes the most recent intake logged for the given day (used when
// the user taps a filled segment to correct a mistake).
func (r *Repo) UndoIntake(ctx context.Context, medID, userID uuid.UUID, takenOn string) error {
	const q = `
		DELETE FROM medication_intakes WHERE id = (
			SELECT id FROM medication_intakes
			WHERE medication_id=$1 AND user_id=$2 AND taken_on=$3::date
			ORDER BY taken_at DESC LIMIT 1
		)`
	ct, err := r.db.Pool.Exec(ctx, q, medID, userID, takenOn)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// prefixed returns the column list aliased to the given table name.
func prefixed(alias string) string {
	return alias + ".id, " + alias + ".user_id, " + alias + ".name, " + alias + ".unit, " +
		alias + ".dose::float8, " + alias + ".times_per_day, " + alias + ".start_date, " +
		alias + ".duration_days, " + alias + ".notes, " + alias + ".active, " +
		alias + ".created_at, " + alias + ".updated_at"
}
