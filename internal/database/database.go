package database

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// migrations contains versioned SQL files shipped with the application.
//
//go:embed migrations/*.sql
var migrations embed.FS

type migrationFile struct {
	version  int
	name     string
	script   []byte
	checksum string
}

type appliedMigration struct {
	version  int
	name     string
	checksum string
}

// Open initializes SQLite, applies migrations, and returns a database handle.
func Open(dataDir string) (*sql.DB, error) {
	dbPath := filepath.Join(dataDir, "doubao.db")
	existingDatabase, err := databaseFileExists(dbPath)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", dbPath, err)
	}
	dsn := (&url.URL{Scheme: "file", Path: dbPath}).String()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dbPath, err)
	}

	// This is a single-user application. A single connection keeps SQLite locking
	// behavior predictable while WAL still makes reads friendly to future features.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", dbPath, err)
	}
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure sqlite with %q: %w", pragma, err)
		}
	}

	migrationFiles, err := loadMigrations()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	if existingDatabase {
		needsBackup, err := migrationBackupNeeded(db, migrationFiles)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("inspect migration state: %w", err)
		}
		if needsBackup {
			if _, err := backupDatabase(db, dataDir); err != nil {
				db.Close()
				return nil, fmt.Errorf("backup database before migration: %w", err)
			}
		}
	}
	if err := migrate(db, migrationFiles); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB, migrationFiles []migrationFile) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	applied, err := readAppliedMigrations(db)
	if err != nil {
		return err
	}
	if err := validateAppliedMigrations(db, migrationFiles, applied); err != nil {
		return err
	}

	for _, migration := range migrationFiles {
		if _, ok := applied[migration.version]; ok {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(string(migration.script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply %s: %w", migration.name, err)
		}
		if _, err = tx.Exec(
			"INSERT INTO schema_migrations (version, name, checksum) VALUES (?, ?, ?)",
			migration.version,
			migration.name,
			migration.checksum,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record %s: %w", migration.name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", migration.name, err)
		}
		applied[migration.version] = appliedMigration{
			version:  migration.version,
			name:     migration.name,
			checksum: migration.checksum,
		}
	}
	return nil
}

func ensureMigrationTable(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	columns, err := tableColumns(db, "schema_migrations")
	if err != nil {
		return err
	}
	if !columns["checksum"] {
		if _, err := db.Exec(`ALTER TABLE schema_migrations ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add schema migration checksum column: %w", err)
		}
	}
	return nil
}

func loadMigrations() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, err
	}

	files := make([]migrationFile, 0, len(entries))
	versions := make(map[int]string, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		if previous, ok := versions[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", version, previous, entry.Name())
		}

		script, err := fs.ReadFile(migrations, path.Join("migrations", entry.Name()))
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(script)
		versions[version] = entry.Name()
		files = append(files, migrationFile{
			version:  version,
			name:     entry.Name(),
			script:   script,
			checksum: hex.EncodeToString(digest[:]),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}

func migrationBackupNeeded(db *sql.DB, migrationFiles []migrationFile) (bool, error) {
	exists, err := tableExists(db, "schema_migrations")
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}

	columns, err := tableColumns(db, "schema_migrations")
	if err != nil {
		return false, err
	}
	if !columns["checksum"] {
		return true, nil
	}

	applied, err := readAppliedMigrations(db)
	if err != nil {
		return false, err
	}
	expected := make(map[int]migrationFile, len(migrationFiles))
	for _, migration := range migrationFiles {
		expected[migration.version] = migration
		if _, ok := applied[migration.version]; !ok {
			return true, nil
		}
	}
	for version, record := range applied {
		migration, ok := expected[version]
		if !ok || record.name != migration.name || record.checksum == "" || record.checksum != migration.checksum {
			return true, nil
		}
	}
	return false, nil
}

func validateAppliedMigrations(db *sql.DB, migrationFiles []migrationFile, applied map[int]appliedMigration) error {
	expected := make(map[int]migrationFile, len(migrationFiles))
	for _, migration := range migrationFiles {
		expected[migration.version] = migration
	}

	for version, record := range applied {
		migration, ok := expected[version]
		if !ok {
			return fmt.Errorf("database contains unknown migration version %d; downgrade is not supported", version)
		}
		if record.name != migration.name {
			return fmt.Errorf("migration version %d was recorded as %s, but the application ships %s", version, record.name, migration.name)
		}
		if record.checksum != "" && record.checksum != migration.checksum {
			return fmt.Errorf("migration %s checksum mismatch; restore a backup or ship a new migration instead of editing an applied file", migration.name)
		}
	}

	pending := false
	for _, migration := range migrationFiles {
		if record, ok := applied[migration.version]; ok {
			if pending {
				return fmt.Errorf("migration %s is applied while an earlier migration is missing", migration.name)
			}
			if record.checksum == "" {
				if _, err := db.Exec("UPDATE schema_migrations SET checksum = ? WHERE version = ?", migration.checksum, migration.version); err != nil {
					return fmt.Errorf("record checksum for %s: %w", migration.name, err)
				}
			}
			continue
		}
		pending = true
	}
	return nil
}

func readAppliedMigrations(db *sql.DB) (map[int]appliedMigration, error) {
	rows, err := db.Query(`SELECT version, name, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]appliedMigration)
	for rows.Next() {
		var migration appliedMigration
		if err := rows.Scan(&migration.version, &migration.name, &migration.checksum); err != nil {
			return nil, err
		}
		applied[migration.version] = migration
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func tableExists(db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func tableColumns(db *sql.DB, name string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + name + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid        int
			column     string
			columnType string
			notNull    int
			defaultV   sql.NullString
			primary    int
		)
		if err := rows.Scan(&cid, &column, &columnType, &notNull, &defaultV, &primary); err != nil {
			return nil, err
		}
		columns[column] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

func databaseFileExists(dbPath string) (bool, error) {
	info, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("database path is a directory")
	}
	return info.Size() > 0, nil
}

func backupDatabase(db *sql.DB, dataDir string) (string, error) {
	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	for attempt := 0; attempt < 100; attempt++ {
		name := fmt.Sprintf("doubao-migration-%s.db", stamp)
		if attempt > 0 {
			name = fmt.Sprintf("doubao-migration-%s-%d.db", stamp, attempt)
		}
		backupPath := filepath.Join(backupDir, name)
		if _, err := os.Stat(backupPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect backup path %s: %w", backupPath, err)
		}

		if _, err := db.Exec("VACUUM INTO ?", backupPath); err != nil {
			return "", fmt.Errorf("write backup %s: %w", backupPath, err)
		}
		if err := os.Chmod(backupPath, 0o600); err != nil {
			return "", fmt.Errorf("secure backup %s: %w", backupPath, err)
		}
		return backupPath, nil
	}
	return "", fmt.Errorf("allocate a unique backup path in %s", backupDir)
}

func migrationVersion(name string) (int, error) {
	underscore := strings.IndexByte(name, '_')
	if underscore <= 0 {
		return 0, fmt.Errorf("migration filename must start with a numeric version: %s", name)
	}
	version, err := strconv.Atoi(name[:underscore])
	if err != nil {
		return 0, fmt.Errorf("parse migration version %s: %w", name, err)
	}
	if version <= 0 {
		return 0, fmt.Errorf("migration version must be positive: %s", name)
	}
	return version, nil
}
