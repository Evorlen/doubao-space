CREATE TABLE IF NOT EXISTS recipe_foods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    category TEXT NOT NULL DEFAULT '其他',
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS recipe_food_days (
    food_id INTEGER NOT NULL REFERENCES recipe_foods(id) ON DELETE CASCADE,
    weekday INTEGER NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    PRIMARY KEY (food_id, weekday)
);

CREATE INDEX IF NOT EXISTS idx_recipe_food_days_weekday ON recipe_food_days(weekday);
