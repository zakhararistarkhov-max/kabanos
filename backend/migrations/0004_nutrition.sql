-- +goose Up
-- Nutrition domain: a shared catalog of dishes (created by any user), social
-- signals (ratings, comments, favorites), per-user daily macro goals, the diet
-- log, and calorie-burning activities.
--
-- Macro values on dishes are stored PER 100 g (plus an optional serving size);
-- diet_entries store the ALREADY-COMPUTED macros for what was actually eaten, as
-- a snapshot, so later edits to a dish never rewrite history.

CREATE TABLE dishes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    recipe          TEXT NOT NULL DEFAULT '',
    image_key       TEXT,                       -- object key in the S3/MinIO bucket
    kcal_per_100g   NUMERIC(8,2) NOT NULL CHECK (kcal_per_100g >= 0),
    protein_per_100g NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (protein_per_100g >= 0),
    fat_per_100g     NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (fat_per_100g >= 0),
    carbs_per_100g   NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (carbs_per_100g >= 0),
    serving_grams   NUMERIC(7,2) CHECK (serving_grams IS NULL OR serving_grams > 0),
    -- Denormalized rating aggregates for cheap sorting/listing.
    rating_count    INT NOT NULL DEFAULT 0,
    rating_sum      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX dishes_created_by_idx ON dishes (created_by);
CREATE INDEX dishes_name_idx ON dishes (lower(name));

CREATE TABLE dish_ratings (
    dish_id    UUID NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating     INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (dish_id, user_id)
);

CREATE TABLE dish_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dish_id    UUID NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX dish_comments_dish_idx ON dish_comments (dish_id, created_at DESC);

CREATE TABLE dish_favorites (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dish_id    UUID NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, dish_id)
);

CREATE TABLE nutrition_goals (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    kcal       INT NOT NULL CHECK (kcal > 0),
    protein_g  NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (protein_g >= 0),
    fat_g      NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (fat_g >= 0),
    carbs_g    NUMERIC(7,2) NOT NULL DEFAULT 0 CHECK (carbs_g >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE diet_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Optional link to the source dish; kept for reference, nulled if the dish
    -- is deleted. The macro snapshot below is authoritative.
    dish_id     UUID REFERENCES dishes(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    grams       NUMERIC(8,2) CHECK (grams IS NULL OR grams > 0),
    meal        TEXT CHECK (meal IS NULL OR meal IN ('breakfast','lunch','dinner','snack')),
    source      TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','dish')),
    kcal        NUMERIC(8,2) NOT NULL CHECK (kcal >= 0),
    protein_g   NUMERIC(8,2) NOT NULL DEFAULT 0 CHECK (protein_g >= 0),
    fat_g       NUMERIC(8,2) NOT NULL DEFAULT 0 CHECK (fat_g >= 0),
    carbs_g     NUMERIC(8,2) NOT NULL DEFAULT 0 CHECK (carbs_g >= 0),
    consumed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX diet_entries_user_time_idx ON diet_entries (user_id, consumed_at);

CREATE TABLE activities (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    kcal         NUMERIC(8,2) NOT NULL CHECK (kcal >= 0),
    duration_min INT CHECK (duration_min IS NULL OR duration_min >= 0),
    met          NUMERIC(5,2),
    source       TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','met')),
    performed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX activities_user_time_idx ON activities (user_id, performed_at);

-- +goose Down
DROP TABLE activities;
DROP TABLE diet_entries;
DROP TABLE nutrition_goals;
DROP TABLE dish_favorites;
DROP TABLE dish_comments;
DROP TABLE dish_ratings;
DROP TABLE dishes;
