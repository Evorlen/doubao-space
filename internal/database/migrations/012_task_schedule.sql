ALTER TABLE tasks ADD COLUMN scheduled INTEGER NOT NULL DEFAULT 0 CHECK (scheduled IN (0, 1));

CREATE INDEX IF NOT EXISTS idx_tasks_scheduled
ON tasks(scheduled, completed);
