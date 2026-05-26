package repositories

import "testing"

func TestMigration000002_Tables(t *testing.T) {
	db := NewTestDB(t)

	present := []string{"projects", "provider_definitions", "provider_path_candidates", "project_providers", "installs"}
	for _, tbl := range present {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", tbl, err)
		}
	}

	var scanResultsName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='scan_results'").Scan(&scanResultsName)
	if err == nil {
		t.Error("table scan_results must NOT exist in migration 000002")
	}
}

func TestMigration000002_Seed(t *testing.T) {
	db := NewTestDB(t)

	var defCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM provider_definitions WHERE key='generic_agents'").Scan(&defCount); err != nil {
		t.Fatalf("provider_definitions query: %v", err)
	}
	if defCount != 1 {
		t.Errorf("generic_agents definition row count: got %d want 1", defCount)
	}

	var candCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'generic_agents'
	`).Scan(&candCount); err != nil {
		t.Fatalf("provider_path_candidates query: %v", err)
	}
	if candCount != 4 {
		t.Errorf("generic_agents candidate row count: got %d want 4 (2 project + 2 global from migration 009)", candCount)
	}

	var detectCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'generic_agents' AND ppc.purpose = 'detect' AND ppc.relative_path = '.agents'
	`).Scan(&detectCount); err != nil {
		t.Fatalf("detect candidate query: %v", err)
	}
	if detectCount != 1 {
		t.Errorf("detect/.agents candidate: got %d want 1", detectCount)
	}

	var skillsCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'generic_agents' AND ppc.purpose = 'skills' AND ppc.relative_path = '.agents/skills'
	`).Scan(&skillsCount); err != nil {
		t.Fatalf("skills candidate query: %v", err)
	}
	if skillsCount != 1 {
		t.Errorf("skills/.agents/skills candidate: got %d want 1", skillsCount)
	}
}
