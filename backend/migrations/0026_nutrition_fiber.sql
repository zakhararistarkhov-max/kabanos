-- +goose Up
-- Add dietary fiber (клетчатка) to the macro set (КБЖУ) everywhere macros are
-- stored: dishes (per 100 g), the daily goal, logged diet entries, and cached
-- barcode products. Defaults to 0 so existing rows stay valid.

ALTER TABLE dishes          ADD COLUMN fiber_per_100g NUMERIC(7,2)  NOT NULL DEFAULT 0 CHECK (fiber_per_100g >= 0);
ALTER TABLE nutrition_goals ADD COLUMN fiber_g        NUMERIC(7,2)  NOT NULL DEFAULT 0 CHECK (fiber_g >= 0);
ALTER TABLE diet_entries    ADD COLUMN fiber_g        NUMERIC(8,2)  NOT NULL DEFAULT 0 CHECK (fiber_g >= 0);
ALTER TABLE food_barcodes   ADD COLUMN fiber_per_100g NUMERIC(10,2) NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE food_barcodes   DROP COLUMN fiber_per_100g;
ALTER TABLE diet_entries    DROP COLUMN fiber_g;
ALTER TABLE nutrition_goals DROP COLUMN fiber_g;
ALTER TABLE dishes          DROP COLUMN fiber_per_100g;
