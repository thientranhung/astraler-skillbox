package repositories

import "testing"

func TestMigration000009_IconKeysSeeded(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key     string
		iconKey string
	}{
		{"generic_agents", "generic_agents"},
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"antigravity_cli", "antigravity"},
	}

	for _, c := range cases {
		var got string
		err := db.QueryRow(`SELECT COALESCE(icon_key,'') FROM provider_definitions WHERE key=?`, c.key).Scan(&got)
		if err != nil {
			t.Fatalf("%s icon_key query: %v", c.key, err)
		}
		if got != c.iconKey {
			t.Errorf("%s icon_key: got %q want %q", c.key, got, c.iconKey)
		}
	}
}

func TestMigration000009_GlobalCandidatesAdded(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key  string
		path string
		want int
	}{
		{"generic_agents", "~/.agents", 1},
		{"generic_agents", "~/.agents/skills", 1},
		{"claude", "~/.claude", 1},
		{"claude", "~/.claude/skills", 1},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = ? AND ppc.relative_path = ? AND ppc.scope = 'global'
		`, c.key, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("%s / %s query: %v", c.key, c.path, err)
		}
		if count != c.want {
			t.Errorf("%s global candidate %q: got %d want %d", c.key, c.path, count, c.want)
		}
	}
}

func TestMigration000009_OpenCodeProvider(t *testing.T) {
	db := NewTestDB(t)

	var key, displayName, status, iconKey string
	var hasGlobal int
	err := db.QueryRow(`
		SELECT key, display_name, status, COALESCE(icon_key,''), has_global_level
		  FROM provider_definitions WHERE key = 'opencode'
	`).Scan(&key, &displayName, &status, &iconKey, &hasGlobal)
	if err != nil {
		t.Fatalf("opencode provider not found: %v", err)
	}
	if key != "opencode" {
		t.Errorf("key: got %q want opencode", key)
	}
	if status != "unsupported" {
		t.Errorf("status: got %q want unsupported", status)
	}
	if iconKey != "opencode" {
		t.Errorf("icon_key: got %q want opencode", iconKey)
	}
	if hasGlobal != 0 {
		t.Errorf("has_global_level: got %d want 0", hasGlobal)
	}

	var candCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'opencode'
	`).Scan(&candCount); err != nil {
		t.Fatalf("opencode candidates query: %v", err)
	}
	if candCount < 1 {
		t.Errorf("opencode should have at least 1 path candidate, got %d", candCount)
	}
}

func TestMigration000009_GenericAgentsHasGlobalLevel(t *testing.T) {
	db := NewTestDB(t)

	var hasGlobal int
	if err := db.QueryRow(`SELECT has_global_level FROM provider_definitions WHERE key='generic_agents'`).Scan(&hasGlobal); err != nil {
		t.Fatalf("query: %v", err)
	}
	if hasGlobal != 1 {
		t.Errorf("generic_agents.has_global_level: got %d want 1 (fixed in migration 009)", hasGlobal)
	}
}

func TestMigration000009_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 10 {
		t.Errorf("database_version: got %d want 10", dbVersion)
	}
}
