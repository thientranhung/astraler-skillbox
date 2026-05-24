package repositories

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// NewTestDB creates a temp SQLite DB with migrations applied; caller must close.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDatabase(path)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenDatabase_CreatesAndMigrates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skillbox.db")

	db, err := OpenDatabase(path)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer db.Close()

	// DB file created
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}

	// Tables exist
	tables := []string{"app_settings", "skill_host_folders", "skills", "skill_sources", "operations", "warnings"}
	for _, tbl := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", tbl, err)
		}
	}

	// Singleton app_settings row exists
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM app_settings").Scan(&count); err != nil {
		t.Fatalf("app_settings query: %v", err)
	}
	if count != 1 {
		t.Fatalf("app_settings row count: got %d want 1", count)
	}
}

func TestOpenDatabase_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skillbox.db")

	db1, err := OpenDatabase(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := OpenDatabase(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()
}

func TestOpenDatabase_PRAGMAs(t *testing.T) {
	db := NewTestDB(t)

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode: got %q want %q", journalMode, "wal")
	}

	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("foreign_keys: got %d want 1", fkEnabled)
	}
}
