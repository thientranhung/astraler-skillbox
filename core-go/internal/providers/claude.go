package providers

import (
	"path/filepath"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

const ClaudeKey = "claude"

// ClaudeDetectPath and ClaudeSkillsPath are the relative paths used by ClaudeAdapter.
// Exported so drift tests can verify they match the seeded path candidates.
const ClaudeDetectPath = ".claude"
const ClaudeSkillsPath = ".claude/skills"

// ClaudeAdapter detects the Claude provider by looking for
// <root>/.claude (detect candidate) and <root>/.claude/skills (skills path).
//
// Detection rules:
//  1. .claude missing         → Present=false, DetectionStatus=missing, no warning
//                               (differs from GenericAgents: service aggregates the
//                               project-level no_provider_detected warning after all
//                               adapters run, so adapters must not emit it themselves)
//  2. .claude dir             → Present=true, detected; read .claude/skills for entries
//  3. .claude/skills missing  → detected, 0 entries, no error
//  4. .claude unreadable dir  → Present=true, invalid_structure + provider warning
//  5. .claude is a file       → Present=true, invalid_structure + provider warning
//
// Rules 2-5 mirror GenericAgentsAdapter semantics.
// Adapter is read-only: no writes to filesystem or database.
type ClaudeAdapter struct{}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{}
}

func (a *ClaudeAdapter) Key() string { return ClaudeKey }

func (a *ClaudeAdapter) DefaultProjectPaths() ProjectScopePaths {
	return ProjectScopePaths{DetectRel: ClaudeDetectPath, SkillsRel: ClaudeSkillsPath}
}

func (a *ClaudeAdapter) Detect(projectRoot string, paths ProjectScopePaths, fs FsReader) (DetectResult, error) {
	detectPath := filepath.Join(projectRoot, paths.DetectRel)
	skillsPath := filepath.Join(projectRoot, paths.SkillsRel)

	pi, err := fs.PathInfo(detectPath)
	if err != nil {
		return DetectResult{}, err
	}

	if !pi.Exists {
		return DetectResult{
			Present:         false,
			DetectionStatus: domain.DetectionStatusMissing,
		}, nil
	}

	if !pi.IsDir || !pi.Readable {
		return DetectResult{
			Present:         true,
			DetectedPath:    detectPath,
			SkillsPath:      skillsPath,
			DetectionStatus: domain.DetectionStatusInvalidStructure,
			Warnings: []AdapterWarning{{
				Code:      "invalid_structure",
				Message:   ".claude exists but is not a readable directory",
				Severity:  domain.WarningSeverityWarning,
				ScopeType: domain.WarningScopeProjectProvider,
			}},
		}, nil
	}

	result := DetectResult{
		Present:         true,
		DetectedPath:    detectPath,
		SkillsPath:      skillsPath,
		DetectionStatus: domain.DetectionStatusDetected,
	}

	skillsPi, err := fs.PathInfo(skillsPath)
	if err != nil {
		return result, err
	}
	if !skillsPi.Exists {
		return result, nil
	}

	rawEntries, err := fs.ListSkillEntries(skillsPath)
	if err != nil {
		result.Warnings = append(result.Warnings, AdapterWarning{
			Code:      "invalid_structure",
			Message:   "Could not read .claude/skills directory",
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

// DefaultGlobalPaths returns the adapter's built-in global-scope paths.
// Claude's global convention: ~/.claude (detect), ~/.claude/skills (skills).
func (a *ClaudeAdapter) DefaultGlobalPaths() GlobalScopePaths {
	return GlobalScopePaths{DetectRel: ClaudeDetectPath, SkillsRel: ClaudeSkillsPath}
}

// DetectGlobal detects the global Claude provider rooted at homeDir using the resolved paths.
// It is read-only: no folder creation occurs. Logic mirrors GenericAgentsAdapter.DetectGlobal.
func (a *ClaudeAdapter) DetectGlobal(homeDir string, paths GlobalScopePaths, fs FsReader) (GlobalDetectResult, error) {
	claudePath := expandGlobalPath(homeDir, paths.DetectRel)
	skillsPath := expandGlobalPath(homeDir, paths.SkillsRel)

	pi, err := fs.PathInfo(claudePath)
	if err != nil {
		return GlobalDetectResult{}, err
	}

	// ~/.claude missing.
	if !pi.Exists {
		return GlobalDetectResult{
			Present: false,
			Status:  domain.GlobalLocationStatusMissing,
			Warnings: []AdapterWarning{{
				Code:      "global_provider_location_missing",
				Message:   "~/.claude directory not found",
				Severity:  domain.WarningSeverityWarning,
				ScopeType: domain.WarningScopeGlobalProviderLocation,
			}},
		}, nil
	}

	// ~/.claude exists but is not a readable directory (or is a file).
	if !pi.IsDir || !pi.Readable {
		return GlobalDetectResult{
			Present:          true,
			GlobalPath:       claudePath,
			GlobalSkillsPath: skillsPath,
			Status:           domain.GlobalLocationStatusInvalidStructure,
			Warnings: []AdapterWarning{{
				Code:      "invalid_structure",
				Message:   "~/.claude exists but is not a readable directory",
				Severity:  domain.WarningSeverityWarning,
				ScopeType: domain.WarningScopeGlobalProviderLocation,
			}},
		}, nil
	}

	// ~/.claude is a readable directory.
	result := GlobalDetectResult{
		Present:          true,
		GlobalPath:       claudePath,
		GlobalSkillsPath: skillsPath,
	}

	// Check ~/.claude/skills.
	skillsPi, err := fs.PathInfo(skillsPath)
	if err != nil {
		return result, err
	}

	if !skillsPi.Exists {
		// skills root absent — do NOT create the folder.
		result.Status = domain.GlobalLocationStatusMissing
		result.Warnings = append(result.Warnings, AdapterWarning{
			Code:      "global_provider_location_missing",
			Message:   "~/.claude/skills directory not found",
			Severity:  domain.WarningSeverityWarning,
			ScopeType: domain.WarningScopeGlobalProviderLocation,
		})
		return result, nil
	}

	if !skillsPi.IsDir || !skillsPi.Readable {
		result.Status = domain.GlobalLocationStatusUnreadable
		result.Warnings = append(result.Warnings, AdapterWarning{
			Code:      "unreadable",
			Message:   "~/.claude/skills is not readable",
			Severity:  domain.WarningSeverityWarning,
			ScopeType: domain.WarningScopeGlobalProviderLocation,
		})
		return result, nil
	}

	rawEntries, err := fs.ListSkillEntries(skillsPath)
	if err != nil {
		result.Status = domain.GlobalLocationStatusUnreadable
		result.Warnings = append(result.Warnings, AdapterWarning{
			Code:      "unreadable",
			Message:   "Could not read ~/.claude/skills directory",
			Severity:  domain.WarningSeverityWarning,
			ScopeType: domain.WarningScopeGlobalProviderLocation,
		})
		return result, nil
	}

	if len(rawEntries) == 0 {
		result.Status = domain.GlobalLocationStatusEmpty
		return result, nil
	}

	result.Status = domain.GlobalLocationStatusActive
	result.Entries = make([]AdapterEntry, 0, len(rawEntries))
	for _, e := range rawEntries {
		result.Entries = append(result.Entries, entryFromProjectEntry(e))
	}
	return result, nil
}

// ensure ClaudeAdapter implements GlobalProviderAdapter at compile time.
var _ GlobalProviderAdapter = (*ClaudeAdapter)(nil)
