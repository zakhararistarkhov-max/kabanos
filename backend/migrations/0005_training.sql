-- +goose Up
-- Training domain: a shared catalog of exercises and workouts (created by any
-- user), each with the same social signals as dishes (ratings, comments,
-- favorites). A workout is an ordered list of exercises with a per-item
-- prescription (sets/reps/duration/rest/weight).
--
-- Mirrors the nutrition domain's shape on purpose so the two read the same.

-- ---------- exercises ----------

CREATE TABLE exercises (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT 'strength' CHECK (category IN ('strength','cardio','mobility')),
    difficulty   TEXT NOT NULL DEFAULT 'medium'   CHECK (difficulty IN ('easy','medium','hard')),
    joint_impact TEXT NOT NULL DEFAULT 'medium'   CHECK (joint_impact IN ('low','medium','high')),
    -- Free-form tag arrays; validated against a suggested list in the app layer
    -- but kept open so users can add their own.
    equipment    TEXT[] NOT NULL DEFAULT '{}',
    muscles      TEXT[] NOT NULL DEFAULT '{}',
    image_key    TEXT,                       -- uploaded photo (S3/MinIO object key)
    video_url    TEXT NOT NULL DEFAULT '',   -- external demo video link (YouTube, …)
    rating_count INT  NOT NULL DEFAULT 0,
    rating_sum   INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX exercises_created_by_idx ON exercises (created_by);
CREATE INDEX exercises_name_idx       ON exercises (lower(name));
CREATE INDEX exercises_category_idx   ON exercises (category);
CREATE INDEX exercises_equipment_idx  ON exercises USING GIN (equipment);

CREATE TABLE exercise_ratings (
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating      INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (exercise_id, user_id)
);

CREATE TABLE exercise_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX exercise_comments_idx ON exercise_comments (exercise_id, created_at DESC);

CREATE TABLE exercise_favorites (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, exercise_id)
);

-- ---------- workouts ----------

CREATE TABLE workouts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    difficulty   TEXT NOT NULL DEFAULT 'medium' CHECK (difficulty IN ('easy','medium','hard')),
    image_key    TEXT,
    rating_count INT  NOT NULL DEFAULT 0,
    rating_sum   INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX workouts_created_by_idx ON workouts (created_by);
CREATE INDEX workouts_name_idx       ON workouts (lower(name));

CREATE TABLE workout_ratings (
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating     INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workout_id, user_id)
);

CREATE TABLE workout_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX workout_comments_idx ON workout_comments (workout_id, created_at DESC);

CREATE TABLE workout_favorites (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workout_id UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, workout_id)
);

-- Ordered exercises inside a workout, with the per-item prescription. Deleting
-- an exercise removes it from every workout that referenced it (the list is
-- renumbered on the next edit).
CREATE TABLE workout_exercises (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_id   UUID NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    exercise_id  UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    position     INT  NOT NULL,
    sets         INT  CHECK (sets IS NULL OR sets > 0),
    reps         INT  CHECK (reps IS NULL OR reps > 0),
    duration_sec INT  CHECK (duration_sec IS NULL OR duration_sec > 0),
    rest_sec     INT  CHECK (rest_sec IS NULL OR rest_sec >= 0),
    weight_kg    NUMERIC(6,2) CHECK (weight_kg IS NULL OR weight_kg >= 0),
    note         TEXT NOT NULL DEFAULT ''
);
CREATE INDEX workout_exercises_workout_idx ON workout_exercises (workout_id, position);

-- +goose Down
DROP TABLE workout_exercises;
DROP TABLE workout_favorites;
DROP TABLE workout_comments;
DROP TABLE workout_ratings;
DROP TABLE workouts;
DROP TABLE exercise_favorites;
DROP TABLE exercise_comments;
DROP TABLE exercise_ratings;
DROP TABLE exercises;
