package providers

import "github.com/astraler/skillbox/core-go/internal/domain"

// GlobalDetectResult holds the outcome of running a provider's global detection.
type GlobalDetectResult struct {
	Present          bool
	GlobalPath       string
	GlobalSkillsPath string
	Status           domain.GlobalLocationStatus
	Entries          []AdapterEntry
	Warnings         []AdapterWarning
}

// GlobalProviderAdapter is implemented only by adapters that have a global level.
// Adapters must be pure: they read facts via FsReader and return structured results.
// They must not write to the filesystem or to the database.
type GlobalProviderAdapter interface {
	ProviderAdapter
	DetectGlobal(homeDir string, fs FsReader) (GlobalDetectResult, error)
}
