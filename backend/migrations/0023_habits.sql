-- +goose Up
-- Habits: a list of habits to build ('good') or break ('bad'). Each habit has a
-- mini-diary (habit_logs) for status notes, and can attach push reminders — which
-- reuse the reminders table + worker (habit_id links them) rather than a second
-- delivery pipeline.

CREATE TABLE habits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('good','bad')),
    description TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0,
    archived    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX habits_user_idx ON habits (user_id, kind, sort_order, created_at);

CREATE TABLE habit_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id   UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    note       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT '' CHECK (status IN ('', 'positive', 'negative')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX habit_logs_habit_idx ON habit_logs (habit_id, created_at DESC);

-- Habit reminders live in the shared reminders table so the worker delivers them
-- like any other; habit_id NULL means a standalone reminder.
ALTER TABLE reminders ADD COLUMN habit_id UUID REFERENCES habits(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE reminders DROP COLUMN habit_id;
DROP TABLE habit_logs;
DROP TABLE habits;
