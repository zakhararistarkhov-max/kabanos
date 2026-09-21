-- +goose Up
-- Daily habit check-ins: one row per habit per day recording whether the habit
-- was kept that day (success = true) or broken/missed (false). This backs the
-- day-by-day tracker heatmap and the dashboard health colour.

CREATE TABLE habit_checkins (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    habit_id   UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    day        DATE NOT NULL,
    success    BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (habit_id, day)
);
CREATE INDEX habit_checkins_user_idx ON habit_checkins (user_id, habit_id, day DESC);

-- +goose Down
DROP TABLE habit_checkins;
