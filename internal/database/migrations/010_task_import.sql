ALTER TABLE tasks ADD COLUMN import_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_parent_import_key
ON tasks(parent_id, import_key)
WHERE import_key IS NOT NULL;
