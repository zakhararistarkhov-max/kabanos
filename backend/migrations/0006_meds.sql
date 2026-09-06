-- +goose Up
-- Medications / vitamins: a per-user list of courses (name, dose per intake,
-- how many intakes per day, when the course starts and how long it lasts) plus
-- an append-only log of intakes used to fill the daily progress bar and compute
-- weekly adherence.

CREATE TABLE medications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    unit          TEXT NOT NULL DEFAULT 'таблетка',       -- таблетка / капсула / мг / мл …
    dose          NUMERIC(8,2) NOT NULL DEFAULT 1 CHECK (dose > 0),          -- amount per intake
    times_per_day INT  NOT NULL DEFAULT 1 CHECK (times_per_day BETWEEN 1 AND 24),
    start_date    DATE NOT NULL,
    duration_days INT  CHECK (duration_days IS NULL OR duration_days > 0),   -- NULL = бессрочно
    notes         TEXT NOT NULL DEFAULT '',
    active        BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX medications_user_idx ON medications (user_id);

CREATE TABLE medication_intakes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount        NUMERIC(8,2) NOT NULL DEFAULT 1,
    -- The user-local calendar day the intake counts towards (for daily/weekly
    -- aggregation without timezone ambiguity).
    taken_on      DATE NOT NULL,
    taken_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX medication_intakes_med_day_idx  ON medication_intakes (medication_id, taken_on);
CREATE INDEX medication_intakes_user_day_idx ON medication_intakes (user_id, taken_on);

-- +goose Down
DROP TABLE medication_intakes;
DROP TABLE medications;
