package services

import (
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// validateSkillSegment rejects any name that is unsafe as a final path segment
// under a skills directory. It prevents path traversal attacks by enforcing:
//   - non-empty
//   - not "." or ".."
//   - not an absolute path
//   - no slash, OS separator, or NUL byte
//   - filepath.Clean(name) == name  (catches trailing slashes, ./a, a/../b, etc.)
func validateSkillSegment(name string) error {
	invalid := func(detail string) error {
		return domain.NewValidationError(
			"Invalid skill name",
			detail,
		)
	}

	if name == "" {
		return invalid("skill name must not be empty")
	}
	if name == "." || name == ".." {
		return invalid("skill name must not be \".\" or \"..\"")
	}
	if filepath.IsAbs(name) {
		return invalid("skill name must not be an absolute path")
	}
	if strings.ContainsAny(name, "/\x00") {
		return invalid("skill name must not contain '/', or NUL bytes")
	}
	// On Windows filepath.Separator is '\\'; catch it when different from '/'.
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return invalid("skill name must not contain the OS path separator")
	}
	if filepath.Clean(name) != name {
		return invalid("skill name must be a clean path segment (no trailing slashes, dot components, etc.)")
	}
	return nil
}

// isWithin reports whether path is strictly inside root (i.e., path starts with
// root followed by the OS path separator). Both arguments should be absolute,
// clean paths; the function normalises them with filepath.Clean before comparing.
func isWithin(root, path string) bool {
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)
	prefix := cleanRoot + string(filepath.Separator)
	return strings.HasPrefix(cleanPath, prefix)
}
