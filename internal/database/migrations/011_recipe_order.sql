ALTER TABLE recipe_foods ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE recipe_foods
SET sort_order = id
WHERE sort_order = 0;

CREATE INDEX IF NOT EXISTS idx_recipe_foods_sort_order ON recipe_foods(sort_order, id);

CREATE TABLE IF NOT EXISTS recipe_weekday_order (
    weekday INTEGER PRIMARY KEY CHECK (weekday BETWEEN 1 AND 7),
    sort_order INTEGER NOT NULL CHECK (sort_order BETWEEN 1 AND 7)
);

INSERT OR IGNORE INTO recipe_weekday_order (weekday, sort_order) VALUES
    (1, 1),
    (2, 2),
    (3, 3),
    (4, 4),
    (5, 5),
    (6, 6),
    (7, 7);
