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

func (a *ClaudeAdapter) Detect(projectRoot string, fs FsReader) (DetectResult, error) {
	detectPath := filepath.Join(projectRoot, ClaudeDetectPath)
	skillsPath := filepath.Join(projectRoot, ClaudeSkillsPath)

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
