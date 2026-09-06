-- +goose Up
-- Water tracking: one goal per user plus an append-only log of intakes.
-- Amounts are stored in millilitres (integers) to avoid floating-point drift.

CREATE TABLE water_goals (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    daily_ml   INT NOT NULL CHECK (daily_ml > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE water_intakes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_ml   INT  NOT NULL CHECK (amount_ml > 0),
    -- Provenance of the pour, for nicer UI history and analytics.
    source      TEXT NOT NULL DEFAULT 'custom'
                CHECK (source IN ('glass','bottle_small','bottle_large','custom')),
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Daily summaries scan by (user, consumed_at); composite index serves them.
CREATE INDEX water_intakes_user_time_idx ON water_intakes (user_id, consumed_at);

-- +goose Down
DROP TABLE water_intakes;
DROP TABLE water_goals;
