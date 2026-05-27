package repositories

import (
	"database/sql"
	"testing"
)

func TestMigration000005_GlobalSkills(t *testing.T) {
	db := NewTestDB(t)

	// Both tables must be creatable (i.e. they exist after migration).
	var gaDefID int64
	if err := db.QueryRow("SELECT id FROM provider_definitions WHERE key='generic_agents'").Scan(&gaDefID); err != nil {
		t.Fatalf("query generic_agents id: %v", err)
	}

	// Insert a global_provider_locations row to verify the table and FK exist.
	if _, err := db.Exec(
		`INSERT INTO global_provider_locations (provider_definition_id, status) VALUES (?, 'active')`,
		gaDefID,
	); err != nil {
		t.Fatalf("insert global_provider_locations: %v", err)
	}

	var locID int64
	if err := db.QueryRow("SELECT id FROM global_provider_locations WHERE provider_definition_id=?", gaDefID).Scan(&locID); err != nil {
		t.Fatalf("select location id: %v", err)
	}

	// Insert a global_installs row.
	if _, err := db.Exec(
		`INSERT INTO global_installs
		  (global_provider_location_id, skill_name, install_mode, install_status, global_skill_path)
		 VALUES (?, 'test-skill', 'direct', 'current', '/home/.agents/skills/test-skill')`,
		locID,
	); err != nil {
		t.Fatalf("insert global_installs: %v", err)
	}

	// generic_agents.has_global_level must be 1.
	var gaGlobalLevel int
	if err := db.QueryRow("SELECT has_global_level FROM provider_definitions WHERE key='generic_agents'").Scan(&gaGlobalLevel); err != nil {
		t.Fatalf("query generic_agents has_global_level: %v", err)
	}
	if gaGlobalLevel != 1 {
		t.Errorf("generic_agents.has_global_level: got %d want 1", gaGlobalLevel)
	}

	// claude.has_global_level must stay 1 (unchanged by this migration).
	var claudeGlobalLevel int
	if err := db.QueryRow("SELECT has_global_level FROM provider_definitions WHERE key='claude'").Scan(&claudeGlobalLevel); err != nil {
		t.Fatalf("query claude has_global_level: %v", err)
	}
	if claudeGlobalLevel != 1 {
		t.Errorf("claude.has_global_level: got %d want 1 (must be unchanged)", claudeGlobalLevel)
	}

	// database_version reflects the latest migration applied by NewTestDB.
	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id=1").Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 14 {
		t.Errorf("database_version: got %d want 14", dbVersion)
	}
}

func TestMigration000005_DownRevertsGlobalSkills(t *testing.T) {
	db := NewTestDB(t)

	const downSQL = `
DROP TABLE IF EXISTS global_installs;
DROP TABLE IF EXISTS global_provider_locations;

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 4, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;`

	if _, err := db.Exec(downSQL); err != nil {
		t.Fatalf("down SQL: %v", err)
	}

	for _, table := range []string{"global_installs", "global_provider_locations"} {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != sql.ErrNoRows {
			t.Fatalf("table %s still exists or unexpected error: name=%q err=%v", table, name, err)
		}
	}

	var gaGlobalLevel int
	if err := db.QueryRow("SELECT has_global_level FROM provider_definitions WHERE key='generic_agents'").Scan(&gaGlobalLevel); err != nil {
		t.Fatalf("query generic_agents has_global_level: %v", err)
	}
	if gaGlobalLevel != 0 {
		t.Errorf("generic_agents.has_global_level after down: got %d want 0", gaGlobalLevel)
	}

	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id=1").Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 4 {
		t.Errorf("database_version after down: got %d want 4", dbVersion)
	}
}
