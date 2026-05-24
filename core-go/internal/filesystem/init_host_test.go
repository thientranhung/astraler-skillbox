package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAgentsSkills_Creates(t *testing.T) {
	host := t.TempDir()
	created, err := EnsureAgentsSkills(host)
	if err != nil {
		t.Fatalf("EnsureAgentsSkills: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on first call")
	}
	expected := filepath.Join(host, ".agents", "skills")
	if info, err := os.Stat(expected); err != nil || !info.IsDir() {
		t.Fatalf(".agents/skills not created: %v", err)
	}
}

func TestEnsureAgentsSkills_Idempotent(t *testing.T) {
	host := t.TempDir()
	if _, err := EnsureAgentsSkills(host); err != nil {
		t.Fatal(err)
	}
	created, err := EnsureAgentsSkills(host)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created {
		t.Fatal("expected created=false on second call")
	}
}
