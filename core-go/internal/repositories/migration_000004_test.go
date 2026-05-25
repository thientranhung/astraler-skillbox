package repositories

import "testing"

func TestMigration000004_SharedAgentDisplayNames(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key  string
		name string
	}{
		{"generic_agents", "Shared Agent Skills (.agents)"},
		{"claude", "Claude (.claude)"},
	}

	for _, c := range cases {
		var got string
		if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key=?", c.key).Scan(&got); err != nil {
			t.Fatalf("query display_name for %s: %v", c.key, err)
		}
		if got != c.name {
			t.Errorf("%s display_name: got %q want %q", c.key, got, c.name)
		}
	}
}
