-- +goose Up
-- Deadlines colour graph nodes (green far off, yellow soon/just past, red well
-- past). The offsets are per-user in gtd_graph_settings.

ALTER TABLE gtd_graph_nodes
    ADD COLUMN deadline   DATE,
    ADD COLUMN note       TEXT NOT NULL DEFAULT '',
    ADD COLUMN image_keys TEXT[] NOT NULL DEFAULT '{}';

CREATE TABLE gtd_graph_settings (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    soon_days  INT NOT NULL DEFAULT 3 CHECK (soon_days >= 0),   -- yellow this many days BEFORE the deadline
    grace_days INT NOT NULL DEFAULT 1 CHECK (grace_days >= 0),  -- stay yellow this many days AFTER, then red
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE gtd_graph_settings;
ALTER TABLE gtd_graph_nodes
    DROP COLUMN deadline,
    DROP COLUMN note,
    DROP COLUMN image_keys;
