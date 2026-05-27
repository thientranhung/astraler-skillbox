package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LstatExists reports whether a filesystem entry exists at path.
// It uses os.Lstat so it does not follow symlinks — broken or external symlinks
// at that path return (true, nil). A missing path returns (false, nil).
// Any other OS error returns (false, err).
func LstatExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// EnsureDir creates path and all parent directories with mode 0755.
// It is idempotent: calling it on an existing directory is not an error.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// CreateSymlink creates a symbolic link at linkPath pointing to source.
// It returns an error if linkPath already exists.
func CreateSymlink(source, linkPath string) error {
	return os.Symlink(source, linkPath)
}

// RemoveSymlink unlinks the entry at path using os.Remove. On a symlink it
// removes the link itself WITHOUT following it (the target is untouched). On a
// non-empty real directory os.Remove returns an error rather than recursing —
// defense in depth so a regression in the caller's checks cannot destroy real
// content.
func RemoveSymlink(path string) error {
	return os.Remove(path)
}

// EnsureDirSafe creates path and all parent directories with mode 0755, then
// verifies the resulting entry is a real directory (not a symlink). Returns
// an error if any pre-existing entry at path is a symlink.
func EnsureDirSafe(path string) error {
	// Check existing entry before creating.
	if lfi, err := os.Lstat(path); err == nil {
		if lfi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %q is a symlink", path)
		}
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	// Verify the created path is a real directory (not a symlink).
	lfi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if lfi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a symlink", path)
	}
	return nil
}

// WriteFileAtomic writes data to path atomically by first writing to a temp
// file in the same directory, then renaming. The rename is atomic on the same
// filesystem. perm is applied to the temp file before rename.
// The parent directory must already exist.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".skillbox-write-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Clean up on any error after this point.
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	// Rename succeeded — clear tmpPath so the defer no-ops.
	tmpPath = ""
	return nil
}
