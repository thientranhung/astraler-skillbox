package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEntry_Missing(t *testing.T) {
	facts, err := ResolveEntry(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if facts.Exists {
		t.Errorf("Exists: got true want false")
	}
}

func TestResolveEntry_RealDir(t *testing.T) {
	dir := t.TempDir()
	facts, err := ResolveEntry(dir)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || facts.IsSymlink {
		t.Errorf("got %+v want exists non-symlink", facts)
	}
}

func TestResolveEntry_GoodSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	// Resolve expected target to handle OS-level symlinks (e.g. /var -> /private/var on macOS).
	wantTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("EvalSymlinks on target: %v", err)
	}
	facts, err := ResolveEntry(link)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || !facts.IsSymlink || facts.Broken {
		t.Errorf("got %+v want resolving symlink", facts)
	}
	if facts.ResolvedTarget != wantTarget {
		t.Errorf("ResolvedTarget: got %q want %q", facts.ResolvedTarget, wantTarget)
	}
}

func TestResolveEntry_BrokenSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "gone"), link); err != nil {
		t.Fatal(err)
	}
	facts, err := ResolveEntry(link)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || !facts.IsSymlink || !facts.Broken {
		t.Errorf("got %+v want broken symlink", facts)
	}
	if facts.ResolvedTarget != "" {
		t.Errorf("ResolvedTarget: got %q want empty", facts.ResolvedTarget)
	}
}

func TestRemoveSymlink_UnlinksLinkNotTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSymlink(link); err != nil {
		t.Fatalf("RemoveSymlink: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("link still present: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("target content was destroyed: %v", err)
	}
}

func TestRemoveSymlink_NonEmptyDirErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSymlink(dir); err == nil {
		t.Errorf("expected error removing non-empty dir, got nil")
	}
}
