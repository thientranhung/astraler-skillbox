package repositories

import (
	"testing"
)

func TestMigration000010_TableExists(t *testing.T) {
	db := NewTestDB(t)

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='provider_path_overrides'`,
	).Scan(&name)
	if err != nil {
		t.Fatalf("provider_path_overrides table not created: %v", err)
	}
	if name != "provider_path_overrides" {
		t.Errorf("table name: got %q want provider_path_overrides", name)
	}
}

func TestMigration000010_FKConstraint(t *testing.T) {
	db := NewTestDB(t)

	_, err := db.Exec(`
		INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
		VALUES (99999, 'project', 'detect', '[]')
	`)
	if err == nil {
		t.Error("expected FK constraint error for nonexistent provider_definition_id, got nil")
	}
}

func TestMigration000010_UniqueConstraint(t *testing.T) {
	db := NewTestDB(t)

	var providerID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&providerID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
		VALUES (?, 'project', 'detect', '["custom"]')
	`, providerID)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
		VALUES (?, 'project', 'detect', '["another"]')
	`, providerID)
	if err == nil {
		t.Error("expected UNIQUE constraint error on second insert, got nil")
	}
}

func TestMigration000010_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 10 {
		t.Errorf("database_version: got %d want 10", dbVersion)
	}
}
