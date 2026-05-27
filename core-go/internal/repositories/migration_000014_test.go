package repositories

import "testing"

func TestMigration000014_ProviderPluginConfigPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key     string
		scope   string
		path    string
		purpose string
	}{
		{"claude", "global", "~/.claude/settings.json", "config"},
		{"claude", "project", ".claude/settings.json", "config"},
		{"claude", "project", ".claude/settings.local.json", "config"},
		// codex config paths removed by migration 016 (codex uses .agents/ natively)
		{"antigravity_cli", "global", "~/.gemini/antigravity-cli/settings.json", "config"},
		{"antigravity_cli", "project", ".gemini/antigravity-cli/settings.json", "config"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = ? AND ppc.scope = ? AND ppc.purpose = ? AND ppc.relative_path = ?
		`, c.key, c.scope, c.purpose, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("%s %s %s query: %v", c.key, c.scope, c.path, err)
		}
		if count != 1 {
			t.Errorf("%s %s config path %q: got %d want 1", c.key, c.scope, c.path, count)
		}
	}
}

func TestMigration000014_CodexAndAntigravityHaveGlobalLevel(t *testing.T) {
	db := NewTestDB(t)

	for _, key := range []string{"codex", "antigravity_cli"} {
		var hasGlobal int
		if err := db.QueryRow(`SELECT has_global_level FROM provider_definitions WHERE key=?`, key).Scan(&hasGlobal); err != nil {
			t.Fatalf("%s query: %v", key, err)
		}
		if hasGlobal != 1 {
			t.Errorf("%s has_global_level: got %d want 1", key, hasGlobal)
		}
	}
}
