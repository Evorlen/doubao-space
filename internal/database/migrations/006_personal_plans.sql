CREATE TABLE IF NOT EXISTS personal_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    note TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL,
    interval_days INTEGER NOT NULL CHECK (interval_days BETWEEN 1 AND 3650),
    last_completed_on TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_personal_plans_start_date ON personal_plans(start_date);
CREATE INDEX IF NOT EXISTS idx_personal_plans_last_completed ON personal_plans(last_completed_on);

CREATE TABLE IF NOT EXISTS personal_plan_completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL REFERENCES personal_plans(id) ON DELETE CASCADE,
    completed_on TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (plan_id, completed_on)
);

CREATE INDEX IF NOT EXISTS idx_personal_plan_completions_plan_id ON personal_plan_completions(plan_id);
