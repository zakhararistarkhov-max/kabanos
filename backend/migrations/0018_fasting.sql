-- +goose Up
-- Intermittent fasting: a per-user protocol (fasting/eating window lengths) plus
-- a log of fasts. An ongoing fast (ended_at IS NULL) means we're in the fasting
-- phase; after it ends the eating window runs for eating_hours. The live
-- countdown and clock ring are derived from these on the client.

CREATE TABLE fasting_settings (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    fasting_hours NUMERIC(4,1) NOT NULL DEFAULT 16 CHECK (fasting_hours > 0 AND fasting_hours <= 48),
    eating_hours  NUMERIC(4,1) NOT NULL DEFAULT 8  CHECK (eating_hours  > 0 AND eating_hours  <= 48),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE fasting_sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at   TIMESTAMPTZ,                              -- NULL = ongoing fast
    goal_hours NUMERIC(4,1) NOT NULL,                    -- target fast length at start
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);
CREATE INDEX fasting_sessions_user_idx ON fasting_sessions (user_id, started_at DESC);
-- At most one ongoing fast per user.
CREATE UNIQUE INDEX fasting_sessions_active_idx ON fasting_sessions (user_id) WHERE ended_at IS NULL;

-- +goose Down
DROP TABLE fasting_sessions;
DROP TABLE fasting_settings;
