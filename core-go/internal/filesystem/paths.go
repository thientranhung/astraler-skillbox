package filesystem

import (
	"fmt"
	"path/filepath"
)

// NormalizeAbs cleans a path and ensures it is absolute.
func NormalizeAbs(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("path %q is not absolute", path)
	}
	return clean, nil
}

// Realpath resolves symlinks and returns the canonical absolute path.
func Realpath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
