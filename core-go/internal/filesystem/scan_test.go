package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func makeSkillsDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	skills := filepath.Join(root, ".agents", "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	return skills
}

func TestScanHostFolder_Empty(t *testing.T) {
	skills := makeSkillsDir(t)
	entries, err := ScanHostFolder(skills)
	if err != nil {
		t.Fatalf("ScanHostFolder: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestScanHostFolder_NormalDir(t *testing.T) {
	skills := makeSkillsDir(t)
	if err := os.Mkdir(filepath.Join(skills, "my-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanHostFolder(skills)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	e := entries[0]
	if e.Name != "my-skill" {
		t.Errorf("name: %q", e.Name)
	}
	if !e.IsDir {
		t.Error("expected IsDir=true")
	}
	if e.IsSymlink {
		t.Error("expected IsSymlink=false")
	}
}

func TestScanHostFolder_ValidSymlink(t *testing.T) {
	skills := makeSkillsDir(t)
	// Create a real skill dir inside the same skills dir.
	realSkill := filepath.Join(skills, "real-skill")
	if err := os.Mkdir(realSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(skills, "linked-skill")
	if err := os.Symlink(realSkill, link); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanHostFolder(skills)
	if err != nil {
		t.Fatal(err)
	}

	var found *HostEntry
	for i := range entries {
		if entries[i].Name == "linked-skill" {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatal("linked-skill not in entries")
	}
	if !found.IsSymlink {
		t.Error("expected IsSymlink=true")
	}
	if found.Broken {
		t.Error("expected Broken=false")
	}
	if found.External {
		t.Error("expected External=false for symlink inside skillsPath")
	}
}

func TestScanHostFolder_BrokenSymlink(t *testing.T) {
	skills := makeSkillsDir(t)
	link := filepath.Join(skills, "broken-skill")
	if err := os.Symlink("/tmp/skillbox-no-such-target-xyz", link); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanHostFolder(skills)
	if err != nil {
		t.Fatal(err)
	}

	var found *HostEntry
	for i := range entries {
		if entries[i].Name == "broken-skill" {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatal("broken-skill not in entries")
	}
	if !found.IsSymlink {
		t.Error("expected IsSymlink=true")
	}
	if !found.Broken {
		t.Error("expected Broken=true")
	}
}

func TestScanHostFolder_ExternalSymlink(t *testing.T) {
	skills := makeSkillsDir(t)
	// External target: outside skillsPath
	extDir := t.TempDir()
	link := filepath.Join(skills, "external-skill")
	if err := os.Symlink(extDir, link); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanHostFolder(skills)
	if err != nil {
		t.Fatal(err)
	}

	var found *HostEntry
	for i := range entries {
		if entries[i].Name == "external-skill" {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		t.Fatal("external-skill not in entries")
	}
	if !found.IsSymlink {
		t.Error("expected IsSymlink=true")
	}
	if found.Broken {
		t.Error("expected Broken=false (target exists)")
	}
	if !found.External {
		t.Error("expected External=true")
	}
}

func TestScanHostFolder_Missing(t *testing.T) {
	_, err := ScanHostFolder("/tmp/skillbox-no-such-skills-dir-xyz")
	if err == nil {
		t.Fatal("expected error for missing skills dir")
	}
}
