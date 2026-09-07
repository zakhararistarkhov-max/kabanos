-- +goose Up
-- Catalog items (dishes, exercises, workouts) are private drafts by default and
-- appear in the shared "all" catalog only after the author publishes them.
-- Existing items are marked public so nothing disappears from the catalog.

ALTER TABLE dishes    ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE exercises ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE workouts  ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;

UPDATE dishes    SET is_public = true;
UPDATE exercises SET is_public = true;
UPDATE workouts  SET is_public = true;

CREATE INDEX dishes_public_idx    ON dishes (is_public) WHERE is_public;
CREATE INDEX exercises_public_idx ON exercises (is_public) WHERE is_public;
CREATE INDEX workouts_public_idx  ON workouts (is_public) WHERE is_public;

-- +goose Down
DROP INDEX IF EXISTS workouts_public_idx;
DROP INDEX IF EXISTS exercises_public_idx;
DROP INDEX IF EXISTS dishes_public_idx;
ALTER TABLE workouts  DROP COLUMN is_public;
ALTER TABLE exercises DROP COLUMN is_public;
ALTER TABLE dishes    DROP COLUMN is_public;
