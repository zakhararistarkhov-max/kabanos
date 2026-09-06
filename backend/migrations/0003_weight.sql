-- +goose Up
-- Weight tracking: a target weight per user plus a time series of measurements.
-- Weight is stored in kilograms with two decimals.

CREATE TABLE weight_goals (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    target_kg  NUMERIC(5,2) NOT NULL CHECK (target_kg > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE weight_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight_kg   NUMERIC(5,2) NOT NULL CHECK (weight_kg > 0 AND weight_kg < 700),
    note        TEXT NOT NULL DEFAULT '',
    -- The calendar day the measurement belongs to (user-local). One canonical
    -- measurement per day keeps the chart clean; re-submitting upserts it.
    measured_on DATE NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, measured_on)
);
CREATE INDEX weight_entries_user_day_idx ON weight_entries (user_id, measured_on);

-- +goose Down
DROP TABLE weight_entries;
DROP TABLE weight_goals;
