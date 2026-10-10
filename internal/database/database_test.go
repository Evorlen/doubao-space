package database

import (
	"path/filepath"
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
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = 6`).Scan(&migrationCount); err != nil {
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
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name IN ('recipe_foods', 'recipe_food_days')`).Scan(&recipeTableCount); err != nil {
		db.Close()
		t.Fatalf("read recipe tables: %v", err)
	}
	if recipeTableCount != 2 {
		db.Close()
		t.Fatalf("recipe table count = %d, want 2", recipeTableCount)
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
