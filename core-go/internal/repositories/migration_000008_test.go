package repositories

import "testing"

func TestMigration000008_CleansProviderDisplayNames(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key  string
		want string
	}{
		{"generic_agents", "Shared Agent Skills"},
		{"claude", "Claude"},
		{"codex", "Codex"},
		{"gemini", "Gemini"},
		{"antigravity_cli", "Antigravity CLI"},
	}

	for _, c := range cases {
		var got string
		if err := db.QueryRow(`SELECT display_name FROM provider_definitions WHERE key = ?`, c.key).Scan(&got); err != nil {
			t.Fatalf("%s provider display name: %v", c.key, err)
		}
		if got != c.want {
			t.Errorf("%s display_name: got %q want %q", c.key, got, c.want)
		}
	}

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id = 1`).Scan(&dbVersion); err != nil {
		t.Fatalf("select database version: %v", err)
	}
	if dbVersion != 15 {
		t.Errorf("database_version: got %d want 15", dbVersion)
	}
}

func TestMigration000008_UpPreservesCustomProviderNames(t *testing.T) {
	db := NewTestDB(t)

	if _, err := db.Exec(`UPDATE provider_definitions SET display_name = 'My Claude' WHERE key = 'claude'`); err != nil {
		t.Fatalf("set custom display name: %v", err)
	}

	upSQL := `
UPDATE provider_definitions
   SET display_name = 'Claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'claude'
   AND display_name = 'Claude (.claude)';`
	if _, err := db.Exec(upSQL); err != nil {
		t.Fatalf("re-run guarded update: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT display_name FROM provider_definitions WHERE key = 'claude'`).Scan(&got); err != nil {
		t.Fatalf("select display name: %v", err)
	}
	if got != "My Claude" {
		t.Errorf("custom display_name was overwritten: got %q want %q", got, "My Claude")
	}
}
