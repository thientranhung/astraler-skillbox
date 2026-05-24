package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateHostPath_Happy(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateHostPath(dir); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateHostPath_NotAbsolute(t *testing.T) {
	err := ValidateHostPath("relative/path")
	if err == nil {
		t.Fatal("expected error for relative path")
	}
	fe, ok := err.(*FilesystemError)
	if !ok || fe.Code != ErrNotAbsolute {
		t.Fatalf("expected ErrNotAbsolute, got %v", err)
	}
}

func TestValidateHostPath_NotExist(t *testing.T) {
	err := ValidateHostPath("/tmp/skillbox-no-such-dir-xyz")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	fe, ok := err.(*FilesystemError)
	if !ok || fe.Code != ErrPathNotFound {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}

func TestValidateHostPath_NotDir(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "file")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	verr := ValidateHostPath(f.Name())
	if verr == nil {
		t.Fatal("expected error for file path")
	}
	fe, ok := verr.(*FilesystemError)
	if !ok || fe.Code != ErrNotADirectory {
		t.Fatalf("expected ErrNotADirectory, got %v", verr)
	}
}

func TestValidateHostPath_NotWritable(t *testing.T) {
	dir := t.TempDir()
	ro := filepath.Join(dir, "readonly")
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ro, 0o755)

	if os.Getuid() == 0 {
		t.Skip("running as root; chmod check not meaningful")
	}

	err := ValidateHostPath(ro)
	if err == nil {
		t.Fatal("expected error for non-writable dir")
	}
	fe, ok := err.(*FilesystemError)
	if !ok || fe.Code != ErrNotWritable {
		t.Fatalf("expected ErrNotWritable, got %v", err)
	}
}
