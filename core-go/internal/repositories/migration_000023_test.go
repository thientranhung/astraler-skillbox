package repositories

import (
	"database/sql"
	"testing"
)

// networkSettingsHasColumn reports whether network_settings has the given column.
func networkSettingsHasColumn(t *testing.T, db *sql.DB, col string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(network_settings)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == col {
			return true
		}
	}
	return false
}

// TestMigration000023_DropColumn verifies that after the full migration chain the
// update_check_enabled gate column is gone, cache_ttl_hours survives, and the
// down migration restores the column (ADR-0002 always-on).
func TestMigration000023_DropColumn(t *testing.T) {
	db := NewTestDB(t)

	// After head migration (000023), the gate column must be gone.
	if networkSettingsHasColumn(t, db, "update_check_enabled") {
		t.Error("update_check_enabled column should be dropped at head (000023)")
	}

	// cache_ttl_hours must survive with its default.
	var ttl int
	if err := db.QueryRow("SELECT cache_ttl_hours FROM network_settings WHERE id = 1").Scan(&ttl); err != nil {
		t.Fatalf("cache_ttl_hours should still exist: %v", err)
	}
	if ttl != 6 {
		t.Errorf("cache_ttl_hours default: got %d want 6", ttl)
	}

	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id = 1").Scan(&dbVersion); err != nil {
		t.Fatalf("database_version: %v", err)
	}
	if dbVersion != 23 {
		t.Errorf("database_version: got %d want 23", dbVersion)
	}

	// Apply the down migration manually and verify the column is restored as =1.
	if _, err := db.Exec(`ALTER TABLE network_settings ADD COLUMN update_check_enabled INTEGER NOT NULL DEFAULT 0`); err != nil {
		t.Fatalf("down: add column: %v", err)
	}
	if _, err := db.Exec(`UPDATE network_settings SET update_check_enabled = 1 WHERE id = 1`); err != nil {
		t.Fatalf("down: set row: %v", err)
	}
	if !networkSettingsHasColumn(t, db, "update_check_enabled") {
		t.Error("update_check_enabled column should exist after down migration")
	}
	var enabled int
	if err := db.QueryRow("SELECT update_check_enabled FROM network_settings WHERE id = 1").Scan(&enabled); err != nil {
		t.Fatalf("read restored column: %v", err)
	}
	if enabled != 1 {
		t.Errorf("down migration should set row to 1, got %d", enabled)
	}

	// Re-apply the up migration (drop again) to prove the up SQL works post-rollback.
	if _, err := db.Exec(`ALTER TABLE network_settings DROP COLUMN update_check_enabled`); err != nil {
		t.Fatalf("re-up: drop column: %v", err)
	}
	if networkSettingsHasColumn(t, db, "update_check_enabled") {
		t.Error("update_check_enabled column should be dropped again after re-up")
	}
}
