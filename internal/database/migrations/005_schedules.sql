CREATE TABLE IF NOT EXISTS doubao_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    note TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL,
    interval_days INTEGER NOT NULL CHECK (interval_days BETWEEN 1 AND 3650),
    last_completed_on TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doubao_schedules_start_date ON doubao_schedules(start_date);
CREATE INDEX IF NOT EXISTS idx_doubao_schedules_last_completed ON doubao_schedules(last_completed_on);

CREATE TABLE IF NOT EXISTS doubao_schedule_completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL REFERENCES doubao_schedules(id) ON DELETE CASCADE,
    completed_on TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (schedule_id, completed_on)
);

CREATE INDEX IF NOT EXISTS idx_doubao_schedule_completions_schedule_id ON doubao_schedule_completions(schedule_id);
