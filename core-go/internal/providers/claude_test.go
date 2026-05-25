package providers_test

import (
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

func TestClaudeAdapter_Key(t *testing.T) {
	a := providers.NewClaudeAdapter()
	if a.Key() != providers.ClaudeKey {
		t.Errorf("Key: got %q want %q", a.Key(), providers.ClaudeKey)
	}
}

func TestClaudeAdapter_ClaudeMissing(t *testing.T) {
	a := providers.NewClaudeAdapter()
	result, err := a.Detect("/project", newMockFS())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Present {
		t.Error("Present: want false")
	}
	if result.DetectionStatus != domain.DetectionStatusMissing {
		t.Errorf("DetectionStatus: got %q want missing", result.DetectionStatus)
	}
	// Missing .claude is not a project-level warning (differs from GenericAgents).
	if len(result.Warnings) != 0 {
		t.Errorf("Warnings: want 0, got %d", len(result.Warnings))
	}
}

func TestClaudeAdapter_ClaudeIsFile_InvalidStructure(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setFile("/project/.claude")

	result, err := a.Detect("/project", fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true (.claude exists even as a file)")
	}
	if result.DetectionStatus != domain.DetectionStatusInvalidStructure {
		t.Errorf("DetectionStatus: got %q want invalid_structure", result.DetectionStatus)
	}
	if result.DetectedPath != "/project/.claude" {
		t.Errorf("DetectedPath: got %q", result.DetectedPath)
	}
	if len(result.Warnings) == 0 || result.Warnings[0].Code != "invalid_structure" {
		t.Errorf("expected invalid_structure warning, got %v", result.Warnings)
	}
}

func TestClaudeAdapter_ClaudeUnreadable_InvalidStructure(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setUnreadableDir("/project/.claude")

	result, err := a.Detect("/project", fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true")
	}
	if result.DetectionStatus != domain.DetectionStatusInvalidStructure {
		t.Errorf("DetectionStatus: got %q want invalid_structure", result.DetectionStatus)
	}
}

func TestClaudeAdapter_SkillsMissing_DetectedZeroEntries(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setDir("/project/.claude")

	result, err := a.Detect("/project", fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true")
	}
	if result.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", result.DetectionStatus)
	}
	if result.DetectedPath != "/project/.claude" {
		t.Errorf("DetectedPath: got %q", result.DetectedPath)
	}
	if result.SkillsPath != "/project/.claude/skills" {
		t.Errorf("SkillsPath: got %q", result.SkillsPath)
	}
	if len(result.Entries) != 0 || len(result.Warnings) != 0 {
		t.Errorf("want 0 entries and 0 warnings, got %d entries %d warnings", len(result.Entries), len(result.Warnings))
	}
}

func TestClaudeAdapter_WithEntries(t *testing.T) {
	a := providers.NewClaudeAdapter()
	mfs := newMockFS()
	mfs.setDir("/project/.claude")
	mfs.setDir("/project/.claude/skills")
	mfs.entries["/project/.claude/skills"] = []filesystem.ProjectEntry{
		{Name: "skill-a", Path: "/project/.claude/skills/skill-a", IsDir: true},
		{Name: "skill-b", Path: "/project/.claude/skills/skill-b", IsSymlink: true, Broken: true},
	}

	result, err := a.Detect("/project", mfs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", result.DetectionStatus)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("Entries: want 2, got %d", len(result.Entries))
	}
	if result.Entries[0].Name != "skill-a" {
		t.Errorf("entry[0].Name: %q", result.Entries[0].Name)
	}
	if !result.Entries[1].Broken {
		t.Error("entry[1].Broken: want true")
	}
}

// listErrFS wraps mockFS but overrides ListSkillEntries to return an error
// for a specific path, letting PathInfo succeed normally.
type listErrFS struct {
	*mockFS
	errForPath string
}

func (f *listErrFS) ListSkillEntries(path string) ([]filesystem.ProjectEntry, error) {
	if path == f.errForPath {
		return nil, errors.New("permission denied")
	}
	return f.mockFS.ListSkillEntries(path)
}

func TestClaudeAdapter_UnreadableSkillsDir_Warning(t *testing.T) {
	a := providers.NewClaudeAdapter()
	base := newMockFS()
	base.setDir("/project/.claude")
	base.setDir("/project/.claude/skills")
	mfs := &listErrFS{mockFS: base, errForPath: "/project/.claude/skills"}

	result, err := a.Detect("/project", mfs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", result.DetectionStatus)
	}
	if len(result.Warnings) == 0 || result.Warnings[0].Code != "invalid_structure" {
		t.Errorf("expected invalid_structure warning for unreadable skills dir, got %v", result.Warnings)
	}
}

// TestSeedVsAdapterPaths_Claude guards that the paths hardcoded in ClaudeAdapter
// match the seeded provider_path_candidates rows.
func TestSeedVsAdapterPaths_Claude(t *testing.T) {
	if providers.ClaudeDetectPath != ".claude" {
		t.Errorf("ClaudeDetectPath: got %q want .claude", providers.ClaudeDetectPath)
	}
	if providers.ClaudeSkillsPath != ".claude/skills" {
		t.Errorf("ClaudeSkillsPath: got %q want .claude/skills", providers.ClaudeSkillsPath)
	}
}

// TestSeedVsAdapterPaths_GenericAgents guards that the paths hardcoded in
// GenericAgentsAdapter match the seeded provider_path_candidates rows.
func TestSeedVsAdapterPaths_GenericAgents(t *testing.T) {
	if providers.GenericAgentsDetectPath != ".agents" {
		t.Errorf("GenericAgentsDetectPath: got %q want .agents", providers.GenericAgentsDetectPath)
	}
	if providers.GenericAgentsSkillsPath != ".agents/skills" {
		t.Errorf("GenericAgentsSkillsPath: got %q want .agents/skills", providers.GenericAgentsSkillsPath)
	}
}
