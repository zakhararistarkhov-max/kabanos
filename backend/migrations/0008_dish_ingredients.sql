-- +goose Up
-- A dish can be composed of other dishes (ingredients) with a gram/ml amount.
-- e.g. "Печёные овощи" contains "Кабачок" (150 г) + "Морковь" (100 г). The
-- parent's per-100g macros are computed from its ingredients at save time and
-- stored on the dish row, so all existing logic (add-to-diet, listing) keeps
-- working unchanged. Using each ingredient's *stored* per-100g (a snapshot)
-- keeps computation one level deep and immune to reference cycles.

CREATE TABLE dish_ingredients (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dish_id            UUID NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    ingredient_dish_id UUID NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    grams              NUMERIC(8,2) NOT NULL CHECK (grams > 0),
    position           INT NOT NULL DEFAULT 0,
    CHECK (dish_id <> ingredient_dish_id)
);
CREATE INDEX dish_ingredients_dish_idx ON dish_ingredients (dish_id, position);

-- +goose Down
DROP TABLE dish_ingredients;
