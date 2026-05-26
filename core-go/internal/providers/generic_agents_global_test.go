package providers_test

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

const testHome = "/home/user"

// agentsPath returns the expected ~/.agents path for testHome.
func agentsPath() string { return testHome + "/.agents" }
func skillsPath() string { return testHome + "/.agents/skills" }

func TestGenericAgentsAdapter_DetectGlobal_AgentsMissing(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS() // all paths missing

	result, err := a.DetectGlobal(testHome, fs)
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
		t.Error("want at least one warning for missing ~/.agents")
	} else if result.Warnings[0].Code != "global_provider_location_missing" {
		t.Errorf("warning code: got %q", result.Warnings[0].Code)
	}
	if len(result.Entries) != 0 {
		t.Errorf("entries: got %d want 0", len(result.Entries))
	}
}

func TestGenericAgentsAdapter_DetectGlobal_AgentsIsFile(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setFile(agentsPath())

	result, err := a.DetectGlobal(testHome, fs)
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

func TestGenericAgentsAdapter_DetectGlobal_AgentsUnreadableDir(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setUnreadableDir(agentsPath())

	result, err := a.DetectGlobal(testHome, fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusInvalidStructure {
		t.Errorf("Status: got %q want invalid_structure", result.Status)
	}
}

func TestGenericAgentsAdapter_DetectGlobal_SkillsMissing_NoFolderCreated(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setDir(agentsPath())
	// skills path is NOT set → Exists=false

	result, err := a.DetectGlobal(testHome, fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusMissing {
		t.Errorf("Status: got %q want missing", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("want warning for missing ~/.agents/skills")
	} else if result.Warnings[0].Code != "global_provider_location_missing" {
		t.Errorf("warning code: got %q want global_provider_location_missing", result.Warnings[0].Code)
	}
	if len(result.Entries) != 0 {
		t.Errorf("entries: got %d want 0", len(result.Entries))
	}
	// Verify no write occurred (mockFS has no write surface — compile-time guarantee).
}

func TestGenericAgentsAdapter_DetectGlobal_SkillsEmpty(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setDir(agentsPath())
	fs.setDir(skillsPath())
	// No entries in skills dir.

	result, err := a.DetectGlobal(testHome, fs)
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

func TestGenericAgentsAdapter_DetectGlobal_Active(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setDir(agentsPath())
	fs.setDir(skillsPath())
	fs.entries[skillsPath()] = []filesystem.ProjectEntry{
		{Name: "research-writer", Path: skillsPath() + "/research-writer", IsDir: true},
		{Name: "adr-helper", Path: skillsPath() + "/adr-helper", IsDir: true, IsSymlink: true,
			SymlinkTargetRaw: "/host/.agents/skills/adr-helper",
			ResolvedTarget:   "/host/.agents/skills/adr-helper"},
	}

	result, err := a.DetectGlobal(testHome, fs)
	if err != nil {
		t.Fatalf("DetectGlobal: %v", err)
	}
	if result.Status != domain.GlobalLocationStatusActive {
		t.Errorf("Status: got %q want active", result.Status)
	}
	if len(result.Entries) != 2 {
		t.Errorf("entries: got %d want 2", len(result.Entries))
	}
	// Adapter returns raw entries — classification is the service's job.
	if result.Entries[0].Name != "research-writer" && result.Entries[1].Name != "research-writer" {
		t.Error("research-writer entry missing")
	}
}

func TestGenericAgentsAdapter_DetectGlobal_SkillsUnreadable(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setDir(agentsPath())
	fs.setUnreadableDir(skillsPath())

	result, err := a.DetectGlobal(testHome, fs)
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
