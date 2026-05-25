package filesystem

import (
	"errors"
	"io/fs"
	"os"
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
