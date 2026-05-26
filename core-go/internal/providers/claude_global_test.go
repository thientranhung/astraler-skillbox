package providers_test

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

const testHomeForClaude = "/home/user"

func claudeDetectPath() string { return testHomeForClaude + "/.claude" }
func claudeSkillsPath() string { return testHomeForClaude + "/.claude/skills" }

func TestClaudeAdapter_DetectGlobal_ClaudeMissing(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS() // all paths missing

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Present {
		t.Error("Present: want false")
	}
	if result.Status != domain.GlobalLocationStatusMissing {
		t.Errorf("Status: got %q want missing", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("want at least one warning for missing ~/.claude")
	} else if result.Warnings[0].Code != "global_provider_location_missing" {
		t.Errorf("warning code: got %q", result.Warnings[0].Code)
	}
}

func TestClaudeAdapter_DetectGlobal_ClaudeIsFile(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setFile(claudeDetectPath())

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true (path exists)")
	}
	if result.Status != domain.GlobalLocationStatusInvalidStructure {
		t.Errorf("Status: got %q want invalid_structure", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("want warning for invalid_structure")
	}
}

func TestClaudeAdapter_DetectGlobal_SkillsMissing(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setDir(claudeDetectPath())
	// skills path NOT set → Exists=false

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusMissing {
		t.Errorf("Status: got %q want missing", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("want warning for missing ~/.claude/skills")
	} else if result.Warnings[0].Code != "global_provider_location_missing" {
		t.Errorf("warning code: got %q want global_provider_location_missing", result.Warnings[0].Code)
	}
}

func TestClaudeAdapter_DetectGlobal_SkillsEmpty(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setDir(claudeDetectPath())
	fs.setDir(claudeSkillsPath())

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusEmpty {
		t.Errorf("Status: got %q want empty", result.Status)
	}
	if len(result.Entries) != 0 {
		t.Errorf("entries: got %d want 0", len(result.Entries))
	}
}

func TestClaudeAdapter_DetectGlobal_Active(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setDir(claudeDetectPath())
	fs.setDir(claudeSkillsPath())
	fs.entries[claudeSkillsPath()] = []filesystem.ProjectEntry{
		{Name: "code-reviewer", Path: claudeSkillsPath() + "/code-reviewer", IsDir: true},
		{Name: "doc-writer", Path: claudeSkillsPath() + "/doc-writer", IsDir: true},
	}

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusActive {
		t.Errorf("Status: got %q want active", result.Status)
	}
	if len(result.Entries) != 2 {
		t.Errorf("entries: got %d want 2", len(result.Entries))
	}
}

func TestClaudeAdapter_DetectGlobal_SkillsUnreadable(t *testing.T) {
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	fs.setDir(claudeDetectPath())
	fs.setUnreadableDir(claudeSkillsPath())

	result, err := a.DetectGlobal(testHomeForClaude, a.DefaultGlobalPaths(), fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusUnreadable {
		t.Errorf("Status: got %q want unreadable", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("want warning for unreadable skills dir")
	}
}

func TestClaudeAdapter_DetectGlobal_DefaultPaths(t *testing.T) {
	a := providers.NewClaudeAdapter()
	paths := a.DefaultGlobalPaths()
	if paths.DetectRel != providers.ClaudeDetectPath {
		t.Errorf("DetectRel: got %q want %q", paths.DetectRel, providers.ClaudeDetectPath)
	}
	if paths.SkillsRel != providers.ClaudeSkillsPath {
		t.Errorf("SkillsRel: got %q want %q", paths.SkillsRel, providers.ClaudeSkillsPath)
	}
}

func TestClaudeAdapter_DetectGlobal_OverridePaths(t *testing.T) {
	// Test that explicit paths override the default.
	a := providers.NewClaudeAdapter()
	fs := newMockFS()
	customDetect := testHomeForClaude + "/custom"
	customSkills := testHomeForClaude + "/custom/skills"
	fs.setDir(customDetect)
	fs.setDir(customSkills)
	fs.entries[customSkills] = []filesystem.ProjectEntry{
		{Name: "custom-tool", Path: customSkills + "/custom-tool", IsDir: true},
	}

	// Use absolute paths (no tilde) for override.
	paths := providers.GlobalScopePaths{DetectRel: customDetect, SkillsRel: customSkills}
	result, err := a.DetectGlobal(testHomeForClaude, paths, fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusActive {
		t.Errorf("Status: got %q want active", result.Status)
	}
	if result.GlobalPath != customDetect {
		t.Errorf("GlobalPath: got %q want %q", result.GlobalPath, customDetect)
	}
}
