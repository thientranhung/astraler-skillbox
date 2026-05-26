package repositories

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

// TestRegistrySeedDrift verifies that every adapter key in NewDefaultRegistry()
// has a seeded row in provider_definitions, and that each seeded row has a non-null
// icon_key. This prevents the adapter registry and the DB seed from drifting apart.
func TestRegistrySeedDrift_AllAdapterKeysSeeded(t *testing.T) {
	db := NewTestDB(t)
	reg := providers.NewDefaultRegistry()

	for _, adapter := range reg.All() {
		key := adapter.Key()

		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM provider_definitions WHERE key=?`, key).Scan(&count); err != nil {
			t.Fatalf("query provider_definitions for key %q: %v", key, err)
		}
		if count != 1 {
			t.Errorf("adapter key %q: expected 1 provider_definitions row, got %d", key, count)
		}
	}
}

func TestRegistrySeedDrift_SeededRowsHaveIconKey(t *testing.T) {
	db := NewTestDB(t)
	reg := providers.NewDefaultRegistry()

	for _, adapter := range reg.All() {
		key := adapter.Key()

		var iconKey string
		err := db.QueryRow(`SELECT COALESCE(icon_key,'') FROM provider_definitions WHERE key=?`, key).Scan(&iconKey)
		if err != nil {
			t.Fatalf("query icon_key for %q: %v", key, err)
		}
		if iconKey == "" {
			t.Errorf("adapter key %q: icon_key is null/empty in provider_definitions (seed it in a migration)", key)
		}
	}
}

func TestRegistrySeedDrift_SeededRowsHavePathCandidates(t *testing.T) {
	db := NewTestDB(t)
	reg := providers.NewDefaultRegistry()

	for _, adapter := range reg.All() {
		key := adapter.Key()

		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = ?
		`, key).Scan(&count)
		if err != nil {
			t.Fatalf("query path_candidates for %q: %v", key, err)
		}
		if count < 1 {
			t.Errorf("adapter key %q: no path candidates seeded in provider_path_candidates", key)
		}
	}
}
