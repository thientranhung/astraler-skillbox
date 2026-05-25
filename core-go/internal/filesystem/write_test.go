package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLstatExists(t *testing.T) {
	t.Run("missing path returns false nil", func(t *testing.T) {
		dir := t.TempDir()
		got, err := LstatExists(filepath.Join(dir, "nonexistent"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Error("expected false for missing path")
		}
	})

	t.Run("real directory returns true nil", func(t *testing.T) {
		dir := t.TempDir()
		got, err := LstatExists(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for existing directory")
		}
	})

	t.Run("regular file returns true nil", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "file.txt")
		if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := LstatExists(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for regular file")
		}
	})

	t.Run("broken symlink returns true nil", func(t *testing.T) {
		dir := t.TempDir()
		link := filepath.Join(dir, "broken-link")
		target := filepath.Join(dir, "nowhere")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		got, err := LstatExists(link)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for broken symlink")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates directory successfully", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "sub", "nested")
		if err := EnsureDir(target); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("directory not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected a directory")
		}
	})

	t.Run("idempotent: calling twice returns no error", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "idempotent")
		if err := EnsureDir(target); err != nil {
			t.Fatalf("first call error: %v", err)
		}
		if err := EnsureDir(target); err != nil {
			t.Fatalf("second call error: %v", err)
		}
	})
}

func TestCreateSymlink(t *testing.T) {
	t.Run("creates symlink successfully", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source-dir")
		if err := os.MkdirAll(source, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := CreateSymlink(source, link); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		dest, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("readlink error: %v", err)
		}
		if dest != source {
			t.Errorf("expected symlink target %q, got %q", source, dest)
		}
	})

	t.Run("existing linkPath returns error", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source-dir")
		if err := os.MkdirAll(source, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := CreateSymlink(source, link); err != nil {
			t.Fatalf("first call error: %v", err)
		}
		if err := CreateSymlink(source, link); err == nil {
			t.Error("expected error when linkPath already exists, got nil")
		}
	})
}
