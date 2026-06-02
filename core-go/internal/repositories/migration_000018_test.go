package repositories

import "testing"

func TestMigration000018_PiProviderExists(t *testing.T) {
	db := NewTestDB(t)

	var key, displayName, status, iconKey string
	var hasGlobal int
	err := db.QueryRow(`
		SELECT key, display_name, status, COALESCE(icon_key,''), has_global_level
		FROM provider_definitions WHERE key = 'pi'
	`).Scan(&key, &displayName, &status, &iconKey, &hasGlobal)
	if err != nil {
		t.Fatalf("pi provider not found: %v", err)
	}
	if status != "unsupported" {
		t.Errorf("status: got %q want unsupported", status)
	}
	if iconKey != "pi" {
		t.Errorf("icon_key: got %q want pi", iconKey)
	}
	if hasGlobal != 1 {
		t.Errorf("has_global_level: got %d want 1", hasGlobal)
	}
}

func TestMigration000018_PiPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		scope   string
		purpose string
		path    string
	}{
		{"project", "detect", ".opencode"},
		{"project", "skills", ".opencode/skills"},
		{"project", "skills", ".claude/skills"},
		{"project", "skills", ".agents/skills"},
		{"global", "detect", "~/.config/opencode"},
		{"global", "skills", "~/.config/opencode/skills"},
		{"global", "skills", "~/.claude/skills"},
		{"global", "skills", "~/.agents/skills"},
		// global config changed to ~/.pi/agent/settings.json by migration 019
		{"global", "config", "~/.pi/agent/settings.json"},
		{"project", "config", ".pi/settings.json"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'pi' AND ppc.scope = ? AND ppc.purpose = ? AND ppc.relative_path = ?
		`, c.scope, c.purpose, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("pi %s %s %q query: %v", c.scope, c.purpose, c.path, err)
		}
		if count != 1 {
			t.Errorf("pi %s %s %q: got %d want 1", c.scope, c.purpose, c.path, count)
		}
	}
}

func TestMigration000018_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 24 {
		t.Errorf("database_version: got %d want 24", dbVersion)
	}
}
