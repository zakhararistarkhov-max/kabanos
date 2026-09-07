-- +goose Up
-- User-configurable reminders delivered as Web Push. Two schedule modes:
--   * interval — every N minutes within a daily time window (e.g. "воду каждые
--     2 часа с 08:00 до 22:00");
--   * times — at fixed local clock times (e.g. завтрак/обед/ужин 09:00/14:00/20:00).
-- An optional condition suppresses the reminder when the goal is already met
-- (e.g. don't nag about water if the daily goal is reached).

CREATE TABLE reminders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    body             TEXT NOT NULL DEFAULT '',
    url              TEXT NOT NULL DEFAULT '/dashboard',
    mode             TEXT NOT NULL CHECK (mode IN ('interval','times')),
    interval_minutes INT  CHECK (interval_minutes IS NULL OR interval_minutes BETWEEN 5 AND 1440),
    window_start     TEXT NOT NULL DEFAULT '08:00',
    window_end       TEXT NOT NULL DEFAULT '22:00',
    times            TEXT[] NOT NULL DEFAULT '{}',
    days             INT[]  NOT NULL DEFAULT '{}',   -- empty = every day; else 0..6 (0=Sun)
    condition        TEXT NOT NULL DEFAULT '' CHECK (condition IN ('','water_below_goal','meds_due')),
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    enabled          BOOLEAN NOT NULL DEFAULT true,
    last_fired_at    TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX reminders_user_idx ON reminders (user_id);
CREATE INDEX reminders_enabled_idx ON reminders (enabled) WHERE enabled;

-- +goose Down
DROP TABLE reminders;
