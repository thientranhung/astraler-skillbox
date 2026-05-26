package providers_test

import (
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// ---------------------------------------------------------------------------
// mock FsReader
// ---------------------------------------------------------------------------

type mockFS struct {
	paths   map[string]filesystem.PathInfo
	entries map[string][]filesystem.ProjectEntry
	errPath map[string]error
}

func newMockFS() *mockFS {
	return &mockFS{
		paths:   make(map[string]filesystem.PathInfo),
		entries: make(map[string][]filesystem.ProjectEntry),
		errPath: make(map[string]error),
	}
}

func (m *mockFS) PathInfo(path string) (filesystem.PathInfo, error) {
	if err, ok := m.errPath[path]; ok {
		return filesystem.PathInfo{}, err
	}
	pi, ok := m.paths[path]
	if !ok {
		return filesystem.PathInfo{Exists: false}, nil
	}
	return pi, nil
}

func (m *mockFS) ListSkillEntries(skillsPath string) ([]filesystem.ProjectEntry, error) {
	if err, ok := m.errPath[skillsPath]; ok {
		return nil, err
	}
	return m.entries[skillsPath], nil
}

// convenience helpers

func (m *mockFS) setDir(path string) {
	m.paths[path] = filesystem.PathInfo{Exists: true, IsDir: true, Readable: true}
}

func (m *mockFS) setFile(path string) {
	m.paths[path] = filesystem.PathInfo{Exists: true, IsDir: false, Readable: true}
}

func (m *mockFS) setUnreadableDir(path string) {
	m.paths[path] = filesystem.PathInfo{Exists: true, IsDir: true, Readable: false}
}

// ---------------------------------------------------------------------------
// GenericAgentsAdapter.Key
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_Key(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	if a.Key() != providers.GenericAgentsKey {
		t.Errorf("Key: got %q want %q", a.Key(), providers.GenericAgentsKey)
	}
}

// ---------------------------------------------------------------------------
// Rule 1: .agents missing → Present=false, DetectionStatus=missing, warning no_provider_detected
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_AgentsMissing(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS() // all paths missing

	result, err := a.Detect("/project", a.DefaultProjectPaths(), fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Present {
		t.Error("Present: want false")
	}
	if result.DetectionStatus != domain.DetectionStatusMissing {
		t.Errorf("DetectionStatus: got %q want missing", result.DetectionStatus)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Warnings: want 1, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Code != "no_provider_detected" {
		t.Errorf("Warning code: got %q want no_provider_detected", result.Warnings[0].Code)
	}
	if result.Warnings[0].ScopeType != domain.WarningScopeProject {
		t.Errorf("Warning scope: got %q want project", result.Warnings[0].ScopeType)
	}
}

// ---------------------------------------------------------------------------
// Rule 4/5: .agents exists as file or unreadable dir → invalid_structure
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_AgentsIsFile(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setFile("/project/.agents")

	result, err := a.Detect("/project", a.DefaultProjectPaths(), fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// .agents was found (Present=true); structure is invalid.
	if !result.Present {
		t.Error("Present: want true (.agents exists even if it is a file)")
	}
	if result.DetectionStatus != domain.DetectionStatusInvalidStructure {
		t.Errorf("DetectionStatus: got %q want invalid_structure", result.DetectionStatus)
	}
	if result.DetectedPath != "/project/.agents" {
		t.Errorf("DetectedPath: got %q want /project/.agents", result.DetectedPath)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected at least one warning")
	}
	if result.Warnings[0].Code != "invalid_structure" {
		t.Errorf("Warning code: got %q want invalid_structure", result.Warnings[0].Code)
	}
}

func TestGenericAgentsAdapter_AgentsUnreadable(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setUnreadableDir("/project/.agents")

	result, err := a.Detect("/project", a.DefaultProjectPaths(), fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// .agents was found (Present=true); unreadable dir is invalid_structure.
	if !result.Present {
		t.Error("Present: want true (.agents exists even if unreadable)")
	}
	if result.DetectionStatus != domain.DetectionStatusInvalidStructure {
		t.Errorf("DetectionStatus: got %q want invalid_structure", result.DetectionStatus)
	}
}

// ---------------------------------------------------------------------------
// Rule 2+3: .agents present & dir, .agents/skills missing → detected, 0 entries
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_SkillsMissing(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	fs := newMockFS()
	fs.setDir("/project/.agents")
	// .agents/skills not set → PathInfo returns Exists:false

	result, err := a.Detect("/project", a.DefaultProjectPaths(), fs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true (provider detected via .agents)")
	}
	if result.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", result.DetectionStatus)
	}
	if result.DetectedPath != "/project/.agents" {
		t.Errorf("DetectedPath: got %q", result.DetectedPath)
	}
	if result.SkillsPath != "/project/.agents/skills" {
		t.Errorf("SkillsPath: got %q", result.SkillsPath)
	}
	if len(result.Entries) != 0 {
		t.Errorf("Entries: want 0, got %d", len(result.Entries))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Warnings: want 0, got %d", len(result.Warnings))
	}
}

// ---------------------------------------------------------------------------
// Rule 2: .agents and .agents/skills present, entries detected
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_Detected_WithEntries(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	mfs := newMockFS()
	mfs.setDir("/project/.agents")
	mfs.setDir("/project/.agents/skills")
	mfs.entries["/project/.agents/skills"] = []filesystem.ProjectEntry{
		{Name: "skill-a", Path: "/project/.agents/skills/skill-a", IsDir: true},
		{Name: "skill-b", Path: "/project/.agents/skills/skill-b", IsDir: false, IsSymlink: true, Broken: true},
	}

	result, err := a.Detect("/project", a.DefaultProjectPaths(), mfs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true")
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

// ---------------------------------------------------------------------------
// Adapter does NOT write — verify no write calls are possible via interface
// (structural: FsReader exposes only read methods)
// ---------------------------------------------------------------------------

func TestGenericAgentsAdapter_Empty_Skills_Not_Error(t *testing.T) {
	// Empty .agents/skills is detected/0 entries, not an error.
	a := providers.NewGenericAgentsAdapter()
	mfs := newMockFS()
	mfs.setDir("/project/.agents")
	mfs.setDir("/project/.agents/skills")
	// entries left empty

	result, err := a.Detect("/project", a.DefaultProjectPaths(), mfs)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !result.Present {
		t.Error("Present: want true")
	}
	if len(result.Entries) != 0 {
		t.Errorf("Entries: want 0, got %d", len(result.Entries))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Warnings: want 0, got %d", len(result.Warnings))
	}
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

func TestRegistry_GetAndAll(t *testing.T) {
	a := providers.NewGenericAgentsAdapter()
	reg := providers.NewRegistry(a)

	got, ok := reg.Get(providers.GenericAgentsKey)
	if !ok {
		t.Fatal("Get: expected ok=true for generic_agents")
	}
	if got.Key() != providers.GenericAgentsKey {
		t.Errorf("adapter key: got %q", got.Key())
	}

	all := reg.All()
	if len(all) != 1 {
		t.Fatalf("All: want 1 adapter, got %d", len(all))
	}

	_, ok2 := reg.Get("nonexistent_provider")
	if ok2 {
		t.Error("Get: expected ok=false for unknown key")
	}
}
