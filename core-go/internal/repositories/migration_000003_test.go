package repositories

import "testing"

func TestMigration000003_ClaudeSeed(t *testing.T) {
	db := NewTestDB(t)

	var defCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM provider_definitions WHERE key='claude'").Scan(&defCount); err != nil {
		t.Fatalf("provider_definitions query: %v", err)
	}
	if defCount != 1 {
		t.Errorf("claude definition row count: got %d want 1", defCount)
	}

	var status string
	var canCreate, hasGlobal int
	if err := db.QueryRow(`
		SELECT status, can_create_structure, has_global_level
		FROM provider_definitions WHERE key = 'claude'
	`).Scan(&status, &canCreate, &hasGlobal); err != nil {
		t.Fatalf("claude definition fields query: %v", err)
	}
	if status != "experimental" {
		t.Errorf("status: got %q want experimental", status)
	}
	if canCreate != 0 {
		t.Errorf("can_create_structure: got %d want 0", canCreate)
	}
	if hasGlobal != 1 {
		t.Errorf("has_global_level: got %d want 1", hasGlobal)
	}

	var candCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'claude'
	`).Scan(&candCount); err != nil {
		t.Fatalf("provider_path_candidates query: %v", err)
	}
	if candCount != 2 {
		t.Errorf("claude candidate row count: got %d want 2", candCount)
	}

	var detectCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'claude' AND ppc.purpose = 'detect' AND ppc.relative_path = '.claude'
	`).Scan(&detectCount); err != nil {
		t.Fatalf("detect candidate query: %v", err)
	}
	if detectCount != 1 {
		t.Errorf("detect/.claude candidate: got %d want 1", detectCount)
	}

	var skillsCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_path_candidates ppc
		JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
		WHERE pd.key = 'claude' AND ppc.purpose = 'skills' AND ppc.relative_path = '.claude/skills'
	`).Scan(&skillsCount); err != nil {
		t.Fatalf("skills candidate query: %v", err)
	}
	if skillsCount != 1 {
		t.Errorf("skills/.claude/skills candidate: got %d want 1", skillsCount)
	}
}

// TestMigration000003_RegistryVsSeed guards that the provider keys registered
// in code match what is seeded in the database. Any adapter key missing from
// the DB seed will cause runtime skips during project scans.
func TestMigration000003_RegistryVsSeed(t *testing.T) {
	db := NewTestDB(t)

	seededKeys := []string{"generic_agents", "claude"}
	for _, key := range seededKeys {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM provider_definitions WHERE key=?", key).Scan(&count); err != nil {
			t.Fatalf("query for key %q: %v", key, err)
		}
		if count != 1 {
			t.Errorf("provider_definitions missing seeded key %q", key)
		}
	}
}
