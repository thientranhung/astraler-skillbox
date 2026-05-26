package providers

import (
	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// FsReader is the read-only filesystem surface used by provider adapters.
// filesystem.Gateway satisfies this interface.
type FsReader interface {
	PathInfo(path string) (filesystem.PathInfo, error)
	ListSkillEntries(skillsPath string) ([]filesystem.ProjectEntry, error)
}

// AdapterEntry is a raw filesystem entry returned by a provider adapter.
// The adapter records facts only; classification vs known hosts is done by the service.
type AdapterEntry struct {
	Name             string
	Path             string
	IsDir            bool
	IsSymlink        bool
	SymlinkTargetRaw string
	ResolvedTarget   string
	Broken           bool
	ResolveError     error
}

// AdapterWarning is a provider-level diagnostic emitted by an adapter.
type AdapterWarning struct {
	Code      string
	Message   string
	Severity  domain.WarningSeverity
	ScopeType domain.WarningScopeType
}

// DetectResult holds the outcome of running a provider adapter against a project root.
type DetectResult struct {
	Present         bool
	DetectedPath    string
	SkillsPath      string
	DetectionStatus domain.DetectionStatus
	Entries         []AdapterEntry
	Warnings        []AdapterWarning
}

// ProjectScopePaths holds the effective relative paths for project-scope provider detection.
// Callers resolve override ?? builtin before passing here; adapters use these verbatim.
type ProjectScopePaths struct {
	DetectRel string
	SkillsRel string
}

// ProviderAdapter detects a specific agent provider in a project directory.
// Adapters must be pure: they read facts via FsReader and return structured
// results. They must not write to the filesystem or to the database.
type ProviderAdapter interface {
	Key() string
	// Detect inspects projectRoot using the resolved paths. Call DefaultProjectPaths()
	// or a resolver to produce paths; adapters must not hard-code them.
	Detect(projectRoot string, paths ProjectScopePaths, fs FsReader) (DetectResult, error)
	// DefaultProjectPaths returns the adapter's built-in relative paths for project scope.
	DefaultProjectPaths() ProjectScopePaths
}
