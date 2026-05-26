package providers

import (
	"fmt"
	"path/filepath"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

const (
	CodexKey              = "codex"
	CodexDetectPath       = ".codex"
	CodexSkillsPath       = ".codex/skills"
	GeminiKey             = "gemini"
	GeminiDetectPath      = ".gemini"
	GeminiSkillsPath      = ".gemini/skills"
	AntigravityCLIKey     = "antigravity_cli"
	AntigravityDetectPath = ".antigravity-cli"
	AntigravitySkillsPath = ".antigravity-cli/skills"
)

type conventionalProviderAdapter struct {
	key       string
	detectRel string
	skillsRel string
}

func NewCodexAdapter() ProviderAdapter {
	return newConventionalProviderAdapter(CodexKey, CodexDetectPath, CodexSkillsPath)
}

func NewGeminiAdapter() ProviderAdapter {
	return newConventionalProviderAdapter(GeminiKey, GeminiDetectPath, GeminiSkillsPath)
}

func NewAntigravityCLIAdapter() ProviderAdapter {
	return newConventionalProviderAdapter(AntigravityCLIKey, AntigravityDetectPath, AntigravitySkillsPath)
}

func newConventionalProviderAdapter(key, detectRel, skillsRel string) *conventionalProviderAdapter {
	return &conventionalProviderAdapter{key: key, detectRel: detectRel, skillsRel: skillsRel}
}

func (a *conventionalProviderAdapter) Key() string { return a.key }

func (a *conventionalProviderAdapter) Detect(projectRoot string, fs FsReader) (DetectResult, error) {
	detectPath := filepath.Join(projectRoot, a.detectRel)
	skillsPath := filepath.Join(projectRoot, a.skillsRel)

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
				Message:   fmt.Sprintf("%s exists but is not a readable directory", a.detectRel),
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
			Message:   fmt.Sprintf("Could not read %s directory", a.skillsRel),
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
