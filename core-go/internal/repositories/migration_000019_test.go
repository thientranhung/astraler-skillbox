package repositories

import "testing"

func TestMigration000019_OpenCodeProjectConfig(t *testing.T) {
	db := NewTestDB(t)

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'opencode' AND ppc.scope = 'project' AND ppc.purpose = 'config'
		  AND ppc.relative_path = '.opencode/config.json'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("opencode project config: got %d want 1", count)
	}
}

func TestMigration000019_PiConfigPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		scope string
		path  string
	}{
		{"project", ".pi/settings.json"},
		{"global", "~/.pi/agent/settings.json"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'pi' AND ppc.scope = ? AND ppc.purpose = 'config' AND ppc.relative_path = ?
		`, c.scope, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("pi %s config %q: %v", c.scope, c.path, err)
		}
		if count != 1 {
			t.Errorf("pi %s config %q: got %d want 1", c.scope, c.path, count)
		}
	}

	// Old wrong config should be gone
	var oldCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'pi' AND ppc.relative_path = '~/.config/opencode/config.json'
	`).Scan(&oldCount)
	if err != nil {
		t.Fatalf("query old config: %v", err)
	}
	if oldCount != 0 {
		t.Errorf("pi old global config should be gone: got %d", oldCount)
	}
}

func TestMigration000019_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 20 {
		t.Errorf("database_version: got %d want 20", dbVersion)
	}
}
