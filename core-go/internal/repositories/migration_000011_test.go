package repositories

import (
	"testing"
)

func TestMigration000011_TableExists(t *testing.T) {
	db := NewTestDB(t)

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='provider_user_settings'`,
	).Scan(&name)
	if err != nil {
		t.Fatalf("provider_user_settings table not created: %v", err)
	}
	if name != "provider_user_settings" {
		t.Errorf("table name: got %q want provider_user_settings", name)
	}
}

func TestMigration000011_FKConstraint(t *testing.T) {
	db := NewTestDB(t)

	_, err := db.Exec(`
		INSERT INTO provider_user_settings (provider_definition_id, enabled)
		VALUES (99999, 1)
	`)
	if err == nil {
		t.Error("expected FK constraint error for nonexistent provider_definition_id, got nil")
	}
}

func TestMigration000011_UniqueConstraint(t *testing.T) {
	db := NewTestDB(t)

	var providerID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&providerID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	_, err := db.Exec(`INSERT INTO provider_user_settings (provider_definition_id, enabled) VALUES (?, 1)`, providerID)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	_, err = db.Exec(`INSERT INTO provider_user_settings (provider_definition_id, enabled) VALUES (?, 0)`, providerID)
	if err == nil {
		t.Error("expected UNIQUE constraint error on second insert, got nil")
	}
}

func TestMigration000011_CheckConstraint(t *testing.T) {
	db := NewTestDB(t)

	var providerID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&providerID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	_, err := db.Exec(`INSERT INTO provider_user_settings (provider_definition_id, enabled) VALUES (?, 2)`, providerID)
	if err == nil {
		t.Error("expected CHECK constraint error for enabled=2, got nil")
	}
}

func TestMigration000011_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 24 {
		t.Errorf("database_version: got %d want 24", dbVersion)
	}
}
