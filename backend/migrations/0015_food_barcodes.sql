-- +goose Up
-- Shared cache of products looked up by barcode (Open Food Facts). Barcodes are
-- universal, so the cache is global (not per-user): the first scan of a product
-- warms it for everyone, keeps lookups fast, and spares the upstream API.

CREATE TABLE food_barcodes (
    barcode          TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    brand            TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    kcal_per_100g    NUMERIC(10,2) NOT NULL DEFAULT 0,
    protein_per_100g NUMERIC(10,2) NOT NULL DEFAULT 0,
    fat_per_100g     NUMERIC(10,2) NOT NULL DEFAULT 0,
    carbs_per_100g   NUMERIC(10,2) NOT NULL DEFAULT 0,
    serving_grams    NUMERIC(10,2),
    source           TEXT NOT NULL DEFAULT 'openfoodfacts',
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE food_barcodes;
