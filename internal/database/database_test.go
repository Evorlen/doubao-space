package database

import (
	"database/sql"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAppliesMigrations(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	var appName string
	if err := db.QueryRow(`SELECT value FROM app_settings WHERE key = 'app_name'`).Scan(&appName); err != nil {
		db.Close()
		t.Fatalf("read app setting: %v", err)
	}
	if appName != "豆包空间" {
		db.Close()
		t.Fatalf("app_name = %q, want %q", appName, "豆包空间")
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = 12`).Scan(&migrationCount); err != nil {
		db.Close()
		t.Fatalf("read schema migration: %v", err)
	}
	if migrationCount != 1 {
		db.Close()
		t.Fatalf("latest migration count = %d, want 1", migrationCount)
	}
	var taskTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('tasks', 'task_tags', 'task_links', 'task_attachments')`).Scan(&taskTableCount); err != nil {
		db.Close()
		t.Fatalf("read task tables: %v", err)
	}
	if taskTableCount != 4 {
		db.Close()
		t.Fatalf("task table count = %d, want 4", taskTableCount)
	}
	var recipeTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('recipe_foods', 'recipe_food_days', 'recipe_weekday_order')`).Scan(&recipeTableCount); err != nil {
		db.Close()
		t.Fatalf("read recipe tables: %v", err)
	}
	if recipeTableCount != 3 {
		db.Close()
		t.Fatalf("recipe table count = %d, want 3", recipeTableCount)
	}
	var scheduleTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('doubao_schedules', 'doubao_schedule_completions')`).Scan(&scheduleTableCount); err != nil {
		db.Close()
		t.Fatalf("read schedule tables: %v", err)
	}
	if scheduleTableCount != 2 {
		db.Close()
		t.Fatalf("schedule table count = %d, want 2", scheduleTableCount)
	}
	var personalPlanTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('personal_plans', 'personal_plan_completions')`).Scan(&personalPlanTableCount); err != nil {
		db.Close()
		t.Fatalf("read personal plan tables: %v", err)
	}
	if personalPlanTableCount != 2 {
		db.Close()
		t.Fatalf("personal plan table count = %d, want 2", personalPlanTableCount)
	}
	var entertainmentTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('entertainment_items', 'entertainment_tags', 'entertainment_item_tags')`).Scan(&entertainmentTableCount); err != nil {
		db.Close()
		t.Fatalf("read entertainment tables: %v", err)
	}
	if entertainmentTableCount != 3 {
		db.Close()
		t.Fatalf("entertainment table count = %d, want 3", entertainmentTableCount)
	}
	var memoTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'memos'`).Scan(&memoTableCount); err != nil {
		db.Close()
		t.Fatalf("read memo table: %v", err)
	}
	if memoTableCount != 1 {
		db.Close()
		t.Fatalf("memo table count = %d, want 1", memoTableCount)
	}
	var albumTableCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'doubao_album_items'`).Scan(&albumTableCount); err != nil {
		db.Close()
		t.Fatalf("read doubao album table: %v", err)
	}
	if albumTableCount != 1 {
		db.Close()
		t.Fatalf("doubao album table count = %d, want 1", albumTableCount)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	// Opening the same directory again must be idempotent.
	db, err = Open(filepath.Clean(dataDir))
	if err != nil {
		t.Fatalf("reopen() error = %v", err)
	}
	defer db.Close()
}

func TestOpenBacksUpBeforeUpgradeAndStoresChecksums(t *testing.T) {
	dataDir := t.TempDir()
	createLegacyDatabase(t, dataDir, 6)

	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	var taskTitle string
	if err := db.QueryRow(`SELECT title FROM tasks LIMIT 1`).Scan(&taskTitle); err != nil {
		db.Close()
		t.Fatalf("read preserved task: %v", err)
	}
	if taskTitle != "历史任务" {
		db.Close()
		t.Fatalf("preserved task title = %q, want %q", taskTitle, "历史任务")
	}

	var emptyChecksums int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE checksum = ''`).Scan(&emptyChecksums); err != nil {
		db.Close()
		t.Fatalf("count migration checksums: %v", err)
	}
	if emptyChecksums != 0 {
		db.Close()
		t.Fatalf("empty migration checksums = %d, want 0", emptyChecksums)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close upgraded database: %v", err)
	}

	backups, err := filepath.Glob(filepath.Join(dataDir, "backups", "doubao-migration-*.db"))
	if err != nil {
		t.Fatalf("find migration backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("migration backups = %d, want 1", len(backups))
	}

	backupDB, err := openRawDatabase(backups[0])
	if err != nil {
		t.Fatalf("open migration backup: %v", err)
	}
	defer backupDB.Close()

	var backupSchema string
	if err := backupDB.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&backupSchema); err != nil {
		t.Fatalf("read backup migration schema: %v", err)
	}
	if strings.Contains(backupSchema, "checksum") {
		t.Fatal("migration backup unexpectedly contains the post-upgrade checksum column")
	}
	var backupLatest int
	if err := backupDB.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&backupLatest); err != nil {
		t.Fatalf("read backup migration version: %v", err)
	}
	if backupLatest != 6 {
		t.Fatalf("backup latest migration = %d, want 6", backupLatest)
	}
}

func TestOpenRejectsChangedAppliedMigration(t *testing.T) {
	dataDir := t.TempDir()
	db, err := Open(dataDir)
	if err != nil {
		t.Fatalf("initial Open() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close initial database: %v", err)
	}

	rawDB, err := openRawDatabase(filepath.Join(dataDir, "doubao.db"))
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	if _, err := rawDB.Exec(`UPDATE schema_migrations SET checksum = 'tampered' WHERE version = 7`); err != nil {
		rawDB.Close()
		t.Fatalf("tamper migration checksum: %v", err)
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close tampered database: %v", err)
	}

	if _, err := Open(dataDir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Open() error = %v, want checksum mismatch", err)
	}
}

func createLegacyDatabase(t *testing.T, dataDir string, latestVersion int) {
	t.Helper()
	db, err := openRawDatabase(filepath.Join(dataDir, "doubao.db"))
	if err != nil {
		t.Fatalf("create legacy database: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin legacy database: %v", err)
	}
	if _, err := tx.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		tx.Rollback()
		t.Fatalf("create legacy migration table: %v", err)
	}
	migrationFiles, err := loadMigrations()
	if err != nil {
		tx.Rollback()
		t.Fatalf("load embedded migrations: %v", err)
	}
	for _, migration := range migrationFiles {
		if migration.version > latestVersion {
			break
		}
		if _, err := tx.Exec(string(migration.script)); err != nil {
			tx.Rollback()
			t.Fatalf("apply legacy migration %d: %v", migration.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, migration.version, migration.name); err != nil {
			tx.Rollback()
			t.Fatalf("record legacy migration %d: %v", migration.version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit legacy database: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tasks (title) VALUES ('历史任务')`); err != nil {
		t.Fatalf("seed legacy task: %v", err)
	}
}

func openRawDatabase(dbPath string) (*sql.DB, error) {
	dsn := (&url.URL{Scheme: "file", Path: dbPath}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
