package repositories

import "testing"

func TestMigration000017_GeminiRemoved(t *testing.T) {
	db := NewTestDB(t)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM provider_definitions WHERE key='gemini'`).Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Errorf("gemini provider should be deleted: got %d rows", count)
	}

	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'gemini'
	`).Scan(&count); err != nil {
		t.Fatalf("query paths: %v", err)
	}
	if count != 0 {
		t.Errorf("gemini path candidates should be deleted: got %d", count)
	}
}

func TestMigration000017_AntigravitySkillsPath(t *testing.T) {
	db := NewTestDB(t)

	// Project skills should be .agents/skills (not .antigravity-cli/skills)
	var path string
	err := db.QueryRow(`
		SELECT ppc.relative_path FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'antigravity_cli' AND ppc.scope = 'project' AND ppc.purpose = 'skills'
	`).Scan(&path)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if path != ".agents/skills" {
		t.Errorf("antigravity_cli project skills: got %q want .agents/skills", path)
	}

	// Global skills should exist
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'antigravity_cli' AND ppc.scope = 'global' AND ppc.purpose = 'skills'
		  AND ppc.relative_path = '~/.gemini/antigravity-cli/skills/'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query global skills: %v", err)
	}
	if count != 1 {
		t.Errorf("antigravity_cli global skills: got %d want 1", count)
	}
}

func TestMigration000017_DatabaseVersion(t *testing.T) {
	db := NewTestDB(t)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
		t.Fatalf("query database_version: %v", err)
	}
	if dbVersion != 20 {
		t.Errorf("database_version: got %d want 20", dbVersion)
	}
}
