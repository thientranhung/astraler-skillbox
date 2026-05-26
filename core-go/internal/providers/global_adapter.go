package providers

import (
	"strings"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// GlobalDetectResult holds the outcome of running a provider's global detection.
type GlobalDetectResult struct {
	Present          bool
	GlobalPath       string
	GlobalSkillsPath string
	Status           domain.GlobalLocationStatus
	Entries          []AdapterEntry
	Warnings         []AdapterWarning
}

// GlobalScopePaths holds the effective detect and skills paths for global-scope provider detection.
// DetectRel and SkillsRel are relative to homeDir (may start with ~/ or be plain relative);
// callers expand ~/ to the actual home directory before passing to DetectGlobal.
// Alternatively pass absolute paths — adapters join homeDir only when the path is relative.
type GlobalScopePaths struct {
	DetectRel string
	SkillsRel string
}

// GlobalProviderAdapter is implemented only by adapters that have a global level.
// Adapters must be pure: they read facts via FsReader and return structured results.
// They must not write to the filesystem or to the database.
type GlobalProviderAdapter interface {
	ProviderAdapter
	// DefaultGlobalPaths returns the adapter's built-in relative paths for global scope.
	DefaultGlobalPaths() GlobalScopePaths
	// DetectGlobal probes the global provider location rooted at homeDir using the resolved paths.
	// paths.DetectRel and paths.SkillsRel are relative to homeDir (~/... will be expanded by adapter).
	DetectGlobal(homeDir string, paths GlobalScopePaths, fs FsReader) (GlobalDetectResult, error)
}

// expandGlobalPath expands a global path relative to homeDir.
// If rel starts with "~/", strip the "~/" prefix and join with homeDir.
// If rel starts with "/", treat as absolute.
// Otherwise join rel with homeDir.
func expandGlobalPath(homeDir, rel string) string {
	if strings.HasPrefix(rel, "~/") {
		return homeDir + "/" + rel[2:]
	}
	if strings.HasPrefix(rel, "/") {
		return rel
	}
	return homeDir + "/" + rel
}
