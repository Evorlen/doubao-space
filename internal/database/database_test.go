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
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = 3`).Scan(&migrationCount); err != nil {
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
