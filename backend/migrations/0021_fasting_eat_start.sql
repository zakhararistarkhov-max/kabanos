-- +goose Up
-- The daily schedule now anchors on when the EATING window starts (more natural
-- for users). The fast begins when that window ends — eat_start + eating_hours —
-- which the worker derives. Rename the columns to reflect the new meaning and
-- default to noon (a common eating-window start).

ALTER TABLE fasting_settings RENAME COLUMN start_hour TO eat_start_hour;
ALTER TABLE fasting_settings RENAME COLUMN start_minute TO eat_start_minute;
ALTER TABLE fasting_settings ALTER COLUMN eat_start_hour SET DEFAULT 12;

-- +goose Down
ALTER TABLE fasting_settings ALTER COLUMN eat_start_hour SET DEFAULT 20;
ALTER TABLE fasting_settings RENAME COLUMN eat_start_minute TO start_minute;
ALTER TABLE fasting_settings RENAME COLUMN eat_start_hour TO start_hour;
