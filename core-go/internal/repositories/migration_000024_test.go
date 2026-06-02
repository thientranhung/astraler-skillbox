package repositories

import "testing"

func TestMigration000024_GenericAgentsCanCreateStructure(t *testing.T) {
	db := NewTestDB(t)

	var canCreate int
	if err := db.QueryRow(
		`SELECT can_create_structure FROM provider_definitions WHERE key='generic_agents'`,
	).Scan(&canCreate); err != nil {
		t.Fatalf("query generic_agents can_create_structure: %v", err)
	}
	if canCreate != 1 {
		t.Errorf("generic_agents can_create_structure: got %d want 1", canCreate)
	}

	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id = 1").Scan(&dbVersion); err != nil {
		t.Fatalf("database_version: %v", err)
	}
	if dbVersion != 24 {
		t.Errorf("database_version: got %d want 24", dbVersion)
	}
}
