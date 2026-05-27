package repositories

import "testing"

func TestMigration000015_OpencodeHasGlobalLevel(t *testing.T) {
	db := NewTestDB(t)

	var hasGlobal int
	if err := db.QueryRow(`SELECT has_global_level FROM provider_definitions WHERE key='opencode'`).Scan(&hasGlobal); err != nil {
		t.Fatalf("opencode query: %v", err)
	}
	if hasGlobal != 1 {
		t.Errorf("opencode has_global_level: got %d want 1", hasGlobal)
	}
}

func TestMigration000015_OpencodeGlobalPaths(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		scope   string
		path    string
		purpose string
	}{
		{"global", "~/.config/opencode", "detect"},
		{"global", "~/.config/opencode/skills", "skills"},
		{"global", "~/.config/opencode/config.json", "config"},
	}

	for _, c := range cases {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM provider_path_candidates ppc
			JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
			WHERE pd.key = 'opencode' AND ppc.scope = ? AND ppc.purpose = ? AND ppc.relative_path = ?
		`, c.scope, c.purpose, c.path).Scan(&count)
		if err != nil {
			t.Fatalf("opencode %s %s query: %v", c.scope, c.path, err)
		}
		if count != 1 {
			t.Errorf("opencode %s path %q: got %d want 1", c.scope, c.path, count)
		}
	}
}
