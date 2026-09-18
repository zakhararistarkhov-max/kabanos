-- +goose Up
-- Five priority levels for GTD. Graphs (projects) get their own priority so the
-- graph panel can sort within a colour band; task items are widened from 0..3 to
-- 0..5 (0 = unset, 1..5 = the five levels, 5 = highest) so tasks share the scale.

ALTER TABLE gtd_projects
    ADD COLUMN priority SMALLINT NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5);

ALTER TABLE gtd_items DROP CONSTRAINT IF EXISTS gtd_items_priority_check;
ALTER TABLE gtd_items ADD CONSTRAINT gtd_items_priority_check CHECK (priority BETWEEN 0 AND 5);

-- +goose Down
ALTER TABLE gtd_items DROP CONSTRAINT IF EXISTS gtd_items_priority_check;
UPDATE gtd_items SET priority = LEAST(priority, 3);
ALTER TABLE gtd_items ADD CONSTRAINT gtd_items_priority_check CHECK (priority BETWEEN 0 AND 3);
ALTER TABLE gtd_projects DROP COLUMN priority;
