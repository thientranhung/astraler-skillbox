package repositories

import "testing"

func TestMigration000016_OpenCodeSkillsPath(t *testing.T) {
	db := NewTestDB(t)

	// .opencode/rules should have been renamed to .opencode/skills
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'opencode' AND ppc.scope = 'project' AND ppc.purpose = 'skills'
		  AND ppc.relative_path = '.opencode/skills'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("opencode project skills '.opencode/skills': got %d want 1", count)
	}

	// Old path should be gone
	err = db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'opencode' AND ppc.relative_path = '.opencode/rules'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query old path: %v", err)
	}
	if count != 0 {
		t.Errorf("opencode old path '.opencode/rules' should be gone: got %d", count)
	}
}

func TestMigration000016_CodexUsesAgentsPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		scope   string
		purpose string
		path    string
	}{
		{"project", "detect", ".agents"},
		{"project", "skills", ".agents/skills"},
		{"global", "detect", "~/.agents"},
		{"global", "skills", "~/.agents/skills"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'codex' AND ppc.scope = ? AND ppc.purpose = ? AND ppc.relative_path = ?
		`, c.scope, c.purpose, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("codex %s %s query: %v", c.scope, c.purpose, err)
		}
		if count != 1 {
			t.Errorf("codex %s %s %q: got %d want 1", c.scope, c.purpose, c.path, count)
		}
	}

	// Old .codex detect/skills paths should be gone (config paths restored by migration 020)
	var oldCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'codex' AND ppc.relative_path LIKE '.codex%'
		  AND ppc.purpose IN ('detect', 'skills')
	`).Scan(&oldCount)
	if err != nil {
		t.Fatalf("query old codex paths: %v", err)
	}
	if oldCount != 0 {
		t.Errorf("old .codex detect/skills paths should be gone: got %d", oldCount)
	}
}

// TestMigration000016_GeminiGlobalPaths removed — gemini provider deleted by migration 017.

func TestMigration000016_OpenCodeCompatPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		scope string
		path  string
	}{
		{"project", ".claude/skills"},
		{"project", ".agents/skills"},
		{"global", "~/.claude/skills"},
		{"global", "~/.agents/skills"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'opencode' AND ppc.scope = ? AND ppc.purpose = 'skills' AND ppc.relative_path = ?
		`, c.scope, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("opencode compat %s %q query: %v", c.scope, c.path, err)
		}
		if count != 1 {
			t.Errorf("opencode compat %s %q: got %d want 1", c.scope, c.path, count)
		}
	}
}

func TestMigration000016_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 23 {
		t.Errorf("database_version: got %d want 23", dbVersion)
	}
}
