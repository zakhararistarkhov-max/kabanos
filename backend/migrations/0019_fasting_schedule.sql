-- +goose Up
-- Daily fasting schedule + push notifications. The user picks a clock time to
-- start fasting each day; the worker auto-starts the fast at that time (opt-in)
-- and sends Web Push notifications: one at the start and one every hour with the
-- remaining time. The notify_* columns are the worker's per-user delivery state,
-- guarded with an IS NOT DISTINCT FROM claim so multiple worker replicas never
-- double-send (mirrors reminders.ClaimFire).

ALTER TABLE fasting_settings
    ADD COLUMN schedule_enabled BOOLEAN  NOT NULL DEFAULT false,
    ADD COLUMN start_hour       SMALLINT NOT NULL DEFAULT 20 CHECK (start_hour   BETWEEN 0 AND 23),
    ADD COLUMN start_minute     SMALLINT NOT NULL DEFAULT 0  CHECK (start_minute BETWEEN 0 AND 59),
    ADD COLUMN timezone         TEXT     NOT NULL DEFAULT 'UTC',
    ADD COLUMN auto_start       BOOLEAN  NOT NULL DEFAULT true,  -- create the fast automatically at the scheduled time
    ADD COLUMN notify_start     BOOLEAN  NOT NULL DEFAULT true,  -- push when a fast begins
    ADD COLUMN notify_hourly    BOOLEAN  NOT NULL DEFAULT true,  -- push every elapsed hour with time remaining
    ADD COLUMN auto_started_on  DATE,                            -- last local date we auto-started (dedupe per day)
    ADD COLUMN notify_anchor    TIMESTAMPTZ,                     -- the active fast's start we've been notifying about
    ADD COLUMN notify_last_hour SMALLINT NOT NULL DEFAULT 0;     -- last elapsed-hour index pushed for that anchor

-- Worker scans only scheduled users each minute.
CREATE INDEX fasting_settings_scheduled_idx ON fasting_settings (user_id) WHERE schedule_enabled;

-- +goose Down
DROP INDEX IF EXISTS fasting_settings_scheduled_idx;
ALTER TABLE fasting_settings
    DROP COLUMN schedule_enabled,
    DROP COLUMN start_hour,
    DROP COLUMN start_minute,
    DROP COLUMN timezone,
    DROP COLUMN auto_start,
    DROP COLUMN notify_start,
    DROP COLUMN notify_hourly,
    DROP COLUMN auto_started_on,
    DROP COLUMN notify_anchor,
    DROP COLUMN notify_last_hour;
