package repositories

import "testing"

func TestMigration000004_SharedAgentDisplayNames(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key  string
		name string
	}{
		{"generic_agents", "Shared Agent Skills"},
		{"claude", "Claude"},
	}

	for _, c := range cases {
		var got string
		if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key=?", c.key).Scan(&got); err != nil {
			t.Fatalf("query display_name for %s: %v", c.key, err)
		}
		if got != c.name {
			t.Errorf("%s display_name: got %q want %q", c.key, got, c.name)
		}
	}

	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id=1").Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 17 {
		t.Errorf("database_version: got %d want 17 (latest after all migrations)", dbVersion)
	}
}

// TestMigration000004_UpIsConditional verifies the WHERE guard in the up migration:
// rows whose display_name was already customised must not be overwritten.
func TestMigration000004_UpIsConditional(t *testing.T) {
	db := NewTestDB(t)

	// Reset to the old seeded name so we can re-exercise the guard.
	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'Generic Agents' WHERE key = 'generic_agents'`); err != nil {
		t.Fatalf("reset to old name: %v", err)
	}
	// Simulate a user-defined custom name.
	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'My Custom Label' WHERE key = 'generic_agents'`); err != nil {
		t.Fatalf("set custom name: %v", err)
	}

	// Re-run the 000004 up migration SQL. It must not touch rows where display_name != 'Generic Agents'.
	const upSQL = `UPDATE provider_definitions
SET display_name = 'Shared Agent Skills (.agents)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Generic Agents'`
	if _, err := db.Exec(upSQL); err != nil {
		t.Fatalf("up SQL: %v", err)
	}

	var got string
	if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key='generic_agents'").Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != "My Custom Label" {
		t.Errorf("custom display_name was overwritten: got %q want %q", got, "My Custom Label")
	}
}

// TestMigration000004_DownRestoresSeededNames verifies the down migration reverts
// the seeded display names and leaves custom names untouched.
func TestMigration000004_DownRestoresSeededNames(t *testing.T) {
	db := NewTestDB(t)

	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'Shared Agent Skills (.agents)' WHERE key = 'generic_agents'`); err != nil {
		t.Fatalf("set generic_agents migrated name: %v", err)
	}
	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'Claude (.claude)' WHERE key = 'claude'`); err != nil {
		t.Fatalf("set claude migrated name: %v", err)
	}
	// Simulate a custom name on claude so we can verify the down guard too.
	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'My Claude Label' WHERE key = 'claude'`); err != nil {
		t.Fatalf("set custom claude name: %v", err)
	}

	const downGenericAgents = `UPDATE provider_definitions
SET display_name = 'Generic Agents', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Shared Agent Skills (.agents)'`
	const downClaude = `UPDATE provider_definitions
SET display_name = 'Claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'claude' AND display_name = 'Claude (.claude)'`
	const downVersion = `UPDATE app_settings SET database_version = 3, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1`

	for _, stmt := range []string{downGenericAgents, downClaude, downVersion} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("down SQL: %v", err)
		}
	}

	var gaName string
	if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key='generic_agents'").Scan(&gaName); err != nil {
		t.Fatalf("query generic_agents: %v", err)
	}
	if gaName != "Generic Agents" {
		t.Errorf("generic_agents after down: got %q want %q", gaName, "Generic Agents")
	}

	// Custom claude label must be unchanged because the WHERE guard did not match.
	var claudeName string
	if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key='claude'").Scan(&claudeName); err != nil {
		t.Fatalf("query claude: %v", err)
	}
	if claudeName != "My Claude Label" {
		t.Errorf("custom claude name was overwritten: got %q want %q", claudeName, "My Claude Label")
	}

	var dbVersion int
	if err := db.QueryRow("SELECT database_version FROM app_settings WHERE id=1").Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 3 {
		t.Errorf("database_version after down: got %d want 3", dbVersion)
	}
}
