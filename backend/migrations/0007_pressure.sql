-- +goose Up
-- Blood-pressure diary: an append-only time series of measurements (systolic /
-- diastolic / optional pulse). Unlike weight, several measurements per day are
-- normal (morning/evening), so entries are keyed by timestamp, not by day.

CREATE TABLE pressure_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    systolic    INT  NOT NULL CHECK (systolic BETWEEN 50 AND 300),
    diastolic   INT  NOT NULL CHECK (diastolic BETWEEN 30 AND 200),
    pulse       INT  CHECK (pulse IS NULL OR pulse BETWEEN 20 AND 300),
    note        TEXT NOT NULL DEFAULT '',
    measured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX pressure_entries_user_time_idx ON pressure_entries (user_id, measured_at);

-- +goose Down
DROP TABLE pressure_entries;
