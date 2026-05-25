package repositories

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

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
}

// TestMigration000003_RegistryVsSeed guards that every adapter key in
// NewDefaultRegistry has a corresponding seeded row in provider_definitions.
// Any adapter registered in code but missing from the DB seed will silently
// skip provider detection at runtime.
func TestMigration000003_RegistryVsSeed(t *testing.T) {
	db := NewTestDB(t)
	reg := providers.NewDefaultRegistry()

	for _, adapter := range reg.All() {
		key := adapter.Key()
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM provider_definitions WHERE key=?", key).Scan(&count); err != nil {
			t.Fatalf("query for adapter key %q: %v", key, err)
		}
		if count != 1 {
			t.Errorf("adapter key %q is registered but has no seeded provider_definitions row", key)
		}
	}
}

// TestMigration000003_SeedVsAdapterPaths_Claude queries the migrated DB and
// verifies that the seeded provider_path_candidates rows for 'claude' match
// the path constants hardcoded in ClaudeAdapter.
func TestMigration000003_SeedVsAdapterPaths_Claude(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		purpose string
		want    string
	}{
		{"detect", providers.ClaudeDetectPath},
		{"skills", providers.ClaudeSkillsPath},
	}
	for _, c := range cases {
		var got string
		err := db.QueryRow(`
			SELECT ppc.relative_path
			FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'claude' AND ppc.purpose = ?
		`, c.purpose).Scan(&got)
		if err != nil {
			t.Fatalf("claude %s candidate query: %v", c.purpose, err)
		}
		if got != c.want {
			t.Errorf("claude %s: DB has %q but adapter constant is %q", c.purpose, got, c.want)
		}
	}
}

// TestMigration000003_SeedVsAdapterPaths_GenericAgents queries the migrated DB
// and verifies that the seeded provider_path_candidates rows for 'generic_agents'
// match the path constants hardcoded in GenericAgentsAdapter.
func TestMigration000003_SeedVsAdapterPaths_GenericAgents(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		purpose string
		want    string
	}{
		{"detect", providers.GenericAgentsDetectPath},
		{"skills", providers.GenericAgentsSkillsPath},
	}
	for _, c := range cases {
		var got string
		err := db.QueryRow(`
			SELECT ppc.relative_path
			FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'generic_agents' AND ppc.purpose = ?
		`, c.purpose).Scan(&got)
		if err != nil {
			t.Fatalf("generic_agents %s candidate query: %v", c.purpose, err)
		}
		if got != c.want {
			t.Errorf("generic_agents %s: DB has %q but adapter constant is %q", c.purpose, got, c.want)
		}
	}
}
