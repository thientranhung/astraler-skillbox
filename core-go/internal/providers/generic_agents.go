package providers

import (
	"path/filepath"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// GenericAgentsKey is the stable provider key for the generic_agents adapter.
const GenericAgentsKey = "generic_agents"

// GenericAgentsAdapter detects the generic_agents provider by looking for
// <root>/.agents (detect candidate) and <root>/.agents/skills (skills path).
//
// Detection rules (from spec §6):
//  1. .agents missing  → Present=false, DetectionStatus=missing, warning no_provider_detected
//  2. .agents dir      → Present=true, detected; read .agents/skills for entries
//  3. .agents/skills missing → detected, 0 entries, no error
//  4. .agents exists but is unreadable dir → invalid_structure + warning
//  5. .agents exists but is a file → invalid_structure + warning
//
// Adapter is read-only: no writes to filesystem or database.
type GenericAgentsAdapter struct{}

// NewGenericAgentsAdapter constructs a GenericAgentsAdapter.
func NewGenericAgentsAdapter() *GenericAgentsAdapter {
	return &GenericAgentsAdapter{}
}

func (a *GenericAgentsAdapter) Key() string { return GenericAgentsKey }

func (a *GenericAgentsAdapter) Detect(projectRoot string, fs FsReader) (DetectResult, error) {
	agentsPath := filepath.Join(projectRoot, ".agents")
	skillsPath := filepath.Join(projectRoot, ".agents", "skills")

	pi, err := fs.PathInfo(agentsPath)
	if err != nil {
		return DetectResult{}, err
	}

	// Rule 1: .agents does not exist.
	if !pi.Exists {
		return DetectResult{
			Present:         false,
			DetectionStatus: domain.DetectionStatusMissing,
			Warnings: []AdapterWarning{{
				Code:      "no_provider_detected",
				Message:   "No generic agents provider detected (.agents directory not found)",
				Severity:  domain.WarningSeverityWarning,
				ScopeType: domain.WarningScopeProject,
			}},
		}, nil
	}

	// Rules 4+5: .agents exists but is not a readable directory.
	if !pi.IsDir || !pi.Readable {
		return DetectResult{
			Present:         false,
			DetectedPath:    agentsPath,
			DetectionStatus: domain.DetectionStatusInvalidStructure,
			Warnings: []AdapterWarning{{
				Code:      "invalid_structure",
				Message:   ".agents exists but is not a readable directory",
				Severity:  domain.WarningSeverityWarning,
				ScopeType: domain.WarningScopeProjectProvider,
			}},
		}, nil
	}

	// .agents is a readable directory → provider detected.
	result := DetectResult{
		Present:         true,
		DetectedPath:    agentsPath,
		SkillsPath:      skillsPath,
		DetectionStatus: domain.DetectionStatusDetected,
	}

	// Rule 3: .agents/skills does not exist → detected with 0 entries.
	skillsPi, err := fs.PathInfo(skillsPath)
	if err != nil {
		return result, err
	}
	if !skillsPi.Exists {
		return result, nil
	}

	// .agents/skills exists → read top-level entries.
	rawEntries, err := fs.ListSkillEntries(skillsPath)
	if err != nil {
		result.Warnings = append(result.Warnings, AdapterWarning{
			Code:      "invalid_structure",
			Message:   "Could not read .agents/skills directory",
			Severity:  domain.WarningSeverityWarning,
			ScopeType: domain.WarningScopeProjectProvider,
		})
		return result, nil
	}

	result.Entries = make([]AdapterEntry, 0, len(rawEntries))
	for _, e := range rawEntries {
		result.Entries = append(result.Entries, entryFromProjectEntry(e))
	}
	return result, nil
}

func entryFromProjectEntry(e filesystem.ProjectEntry) AdapterEntry {
	return AdapterEntry{
		Name:             e.Name,
		Path:             e.Path,
		IsDir:            e.IsDir,
		IsSymlink:        e.IsSymlink,
		SymlinkTargetRaw: e.SymlinkTargetRaw,
		ResolvedTarget:   e.ResolvedTarget,
		Broken:           e.Broken,
		ResolveError:     e.ResolveError,
	}
}
