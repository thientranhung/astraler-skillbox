package repositories

import (
	"database/sql"
	"testing"
)

func TestMigration000022_UpDown(t *testing.T) {
	db := NewTestDB(t)

	// Up: tables should exist after migration.
	tables := []string{"plugin_update_check_cache", "network_settings"}
	for _, tbl := range tables {
		var name string
		if err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name); err != nil {
			t.Errorf("table %q missing after up: %v", tbl, err)
		}
	}

	// network_settings row id=1 should exist with defaults.
	var enabled, ttl int
	if err := db.QueryRow(
		"SELECT update_check_enabled, cache_ttl_hours FROM network_settings WHERE id = 1",
	).Scan(&enabled, &ttl); err != nil {
		t.Fatalf("network_settings default row: %v", err)
	}
	if enabled != 0 {
		t.Errorf("update_check_enabled default: got %d want 0", enabled)
	}
	if ttl != 6 {
		t.Errorf("cache_ttl_hours default: got %d want 6", ttl)
	}

	// database_version should be 22.
	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id = 1").Scan(&dbVersion); err != nil {
		t.Fatalf("database_version: %v", err)
	}
	if dbVersion != 22 {
		t.Errorf("database_version: got %d want 22", dbVersion)
	}

	// Verify UNIQUE constraint on plugin_update_check_cache.
	_, err := db.Exec(`INSERT INTO plugin_update_check_cache
		(provider_key, plugin_name, marketplace_name, source_url, checked_at)
		VALUES ('claude', 'testplugin', 'testmarket', 'https://example.com', '2026-05-29T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert first row: %v", err)
	}
	_, err = db.Exec(`INSERT INTO plugin_update_check_cache
		(provider_key, plugin_name, marketplace_name, source_url, checked_at)
		VALUES ('claude', 'testplugin', 'testmarket', 'https://example.com', '2026-05-29T00:00:00Z')`)
	if err == nil {
		t.Error("expected UNIQUE constraint violation on duplicate (provider_key, plugin_name, marketplace_name)")
	}

	// Down: manually apply the down migration.
	if _, err := db.Exec("DROP TABLE IF EXISTS network_settings"); err != nil {
		t.Fatalf("drop network_settings: %v", err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS plugin_update_check_cache"); err != nil {
		t.Fatalf("drop plugin_update_check_cache: %v", err)
	}
	if _, err := db.Exec(
		"UPDATE app_settings SET database_version = 21, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = 1",
	); err != nil {
		t.Fatalf("restore database_version: %v", err)
	}

	// After down: tables should be gone.
	for _, tbl := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		).Scan(&name)
		if err != sql.ErrNoRows {
			t.Errorf("table %q still exists after down (got %q, err %v)", tbl, name, err)
		}
	}

	// database_version should be 21 again.
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id = 1").Scan(&dbVersion); err != nil {
		t.Fatalf("database_version after down: %v", err)
	}
	if dbVersion != 21 {
		t.Errorf("database_version after down: got %d want 21", dbVersion)
	}
}
