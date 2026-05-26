package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// -- mock GlobalProviderPathResolver --

type mockGlobalPathResolver struct {
	paths map[string]providers.GlobalScopePaths
	err   error
}

func (m *mockGlobalPathResolver) GlobalPaths(_ context.Context) (map[string]providers.GlobalScopePaths, error) {
	return m.paths, m.err
}

// -- helpers --

// newGlobalServiceWithResolver builds a GlobalSkillsService with a path resolver and an adapter in the registry.
func newGlobalServiceWithResolver(
	globalRepo GlobalRepo,
	scanWriter GlobalScanWriter,
	fs GlobalFilesystem,
	adapter providers.GlobalProviderAdapter,
	resolver GlobalProviderPathResolver,
) *GlobalSkillsService {
	registry := &mockGlobalRegistry{adapter: adapter}
	svc := NewGlobalSkillsService(
		globalRepo,
		scanWriter,
		newMockSettings(nil),
		&mockHostLister{},
		&mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry,
		fs,
		&syncRunner{},
	)
	if resolver != nil {
		svc.WithGlobalPathResolver(resolver)
	}
	return svc
}

// -- tests --

// TestPR2C_NoOverride_UsesBuiltinPaths verifies that when the resolver returns global paths for
// generic_agents, the scan uses those paths (builtin in this case).
func TestPR2C_NoOverride_UsesBuiltinPaths(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"
	skillsPath := home + "/.agents/skills"

	capturedPaths := providers.GlobalScopePaths{}
	adapterResult := providers.GlobalDetectResult{
		Present:          true,
		GlobalPath:       home + "/.agents",
		GlobalSkillsPath: skillsPath,
		Status:           domain.GlobalLocationStatusActive,
		Entries: []providers.AdapterEntry{
			{Name: "tool-a", Path: skillsPath + "/tool-a", IsDir: true},
		},
	}

	// Use a capturing adapter variant to record the paths passed in.
	capAdapter := &capturingGlobalAdapter{
		key:          providers.GenericAgentsKey,
		result:       adapterResult,
		capturedPath: &capturedPaths,
	}

	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{
			providers.GenericAgentsKey: {DetectRel: "~/.agents", SkillsRel: "~/.agents/skills"},
		},
	}

	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	if capturedPaths.DetectRel != "~/.agents" {
		t.Errorf("DetectRel: got %q want ~/.agents", capturedPaths.DetectRel)
	}
	if capturedPaths.SkillsRel != "~/.agents/skills" {
		t.Errorf("SkillsRel: got %q want ~/.agents/skills", capturedPaths.SkillsRel)
	}
	if len(scanWriter.committed) != 1 {
		t.Errorf("committed installs: got %d want 1", len(scanWriter.committed))
	}
}

// TestPR2C_GlobalOverride_ChangesPath verifies that a global override path is passed to DetectGlobal.
func TestPR2C_GlobalOverride_ChangesPath(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"
	overrideDetect := "/custom/agents"
	overrideSkills := "/custom/agents/skills"

	capturedPaths := providers.GlobalScopePaths{}
	capAdapter := &capturingGlobalAdapter{
		key: providers.GenericAgentsKey,
		result: providers.GlobalDetectResult{
			Present: false,
			Status:  domain.GlobalLocationStatusMissing,
		},
		capturedPath: &capturedPaths,
	}

	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{
			providers.GenericAgentsKey: {DetectRel: overrideDetect, SkillsRel: overrideSkills},
		},
	}

	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	if capturedPaths.DetectRel != overrideDetect {
		t.Errorf("DetectRel: got %q want %q", capturedPaths.DetectRel, overrideDetect)
	}
	if capturedPaths.SkillsRel != overrideSkills {
		t.Errorf("SkillsRel: got %q want %q", capturedPaths.SkillsRel, overrideSkills)
	}
}

// TestPR2C_ProviderNotInResolver_Skipped verifies that when the resolver map does not include
// a key for the adapter, that adapter is skipped (has_global_level=false semantics).
func TestPR2C_ProviderNotInResolver_Skipped(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"

	// Adapter IS global-capable but resolver excludes it (simulates has_global_level=false).
	capAdapter := &capturingGlobalAdapter{
		key: providers.GenericAgentsKey,
		result: providers.GlobalDetectResult{
			Present: true,
			Status:  domain.GlobalLocationStatusActive,
		},
		capturedPath: &providers.GlobalScopePaths{},
	}

	// Resolver returns empty map — no providers with global level.
	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{},
	}

	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	// CommitGlobalScan should not have been called since the provider was skipped.
	if len(scanWriter.committed) != 0 {
		t.Errorf("committed installs: got %d want 0 (adapter should be skipped)", len(scanWriter.committed))
	}
}

// TestPR2C_NoGlobalAdapter_Skipped verifies that adapters not implementing GlobalProviderAdapter
// are skipped even if they are in the resolver map.
func TestPR2C_NoGlobalAdapter_Skipped(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Codex", status: "supported"}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"

	// Non-global adapter (only implements ProviderAdapter, not GlobalProviderAdapter).
	nonGlobalAdapter := &nonGlobalProviderAdapter{key: "codex"}

	// Registry with the non-global adapter.
	registry := &singleAdapterRegistry{adapter: nonGlobalAdapter}

	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{
			"codex": {DetectRel: "~/.codex", SkillsRel: "~/.codex/skills"},
		},
	}

	fs := newMockGlobalFS(home)
	svc := NewGlobalSkillsService(
		globalRepo, scanWriter, newMockSettings(nil),
		&mockHostLister{}, &mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry, fs, &syncRunner{},
	).WithGlobalPathResolver(resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	// Must not commit any scan since codex is not a GlobalProviderAdapter.
	if len(scanWriter.committed) != 0 {
		t.Errorf("committed installs: got %d want 0 (non-global adapter should be skipped)", len(scanWriter.committed))
	}
}

// TestPR2C_NoResolver_UsesAdapterDefaults verifies that without a path resolver,
// the adapter's DefaultGlobalPaths are used.
func TestPR2C_NoResolver_UsesAdapterDefaults(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"
	skillsPath := home + "/.agents/skills"

	capturedPaths := providers.GlobalScopePaths{}
	capAdapter := &capturingGlobalAdapter{
		key: providers.GenericAgentsKey,
		result: providers.GlobalDetectResult{
			Present:          true,
			GlobalPath:       home + "/.agents",
			GlobalSkillsPath: skillsPath,
			Status:           domain.GlobalLocationStatusEmpty,
		},
		capturedPath: &capturedPaths,
		defaultPaths: providers.GlobalScopePaths{DetectRel: ".agents", SkillsRel: ".agents/skills"},
	}

	fs := newMockGlobalFS(home)
	// No resolver passed.
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, nil)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	if capturedPaths.DetectRel != ".agents" {
		t.Errorf("DetectRel: got %q want .agents", capturedPaths.DetectRel)
	}
}

// TestPR2C_ClaudeGlobalAdapter_DefaultPaths verifies ClaudeAdapter implements GlobalProviderAdapter
// and has the expected default paths.
func TestPR2C_ClaudeGlobalAdapter_DefaultPaths(t *testing.T) {
	a := providers.NewClaudeAdapter()
	paths := a.DefaultGlobalPaths()
	if paths.DetectRel != providers.ClaudeDetectPath {
		t.Errorf("DetectRel: got %q want %q", paths.DetectRel, providers.ClaudeDetectPath)
	}
	if paths.SkillsRel != providers.ClaudeSkillsPath {
		t.Errorf("SkillsRel: got %q want %q", paths.SkillsRel, providers.ClaudeSkillsPath)
	}
}

// -- helper adapter types --

// capturingGlobalAdapter records the GlobalScopePaths passed to DetectGlobal.
type capturingGlobalAdapter struct {
	key          string
	result       providers.GlobalDetectResult
	err          error
	capturedPath *providers.GlobalScopePaths
	defaultPaths providers.GlobalScopePaths
}

func (m *capturingGlobalAdapter) Key() string { return m.key }
func (m *capturingGlobalAdapter) DefaultProjectPaths() providers.ProjectScopePaths {
	return providers.ProjectScopePaths{}
}
func (m *capturingGlobalAdapter) DefaultGlobalPaths() providers.GlobalScopePaths {
	return m.defaultPaths
}
func (m *capturingGlobalAdapter) Detect(_ string, _ providers.ProjectScopePaths, _ providers.FsReader) (providers.DetectResult, error) {
	return providers.DetectResult{}, nil
}
func (m *capturingGlobalAdapter) DetectGlobal(_ string, paths providers.GlobalScopePaths, _ providers.FsReader) (providers.GlobalDetectResult, error) {
	*m.capturedPath = paths
	return m.result, m.err
}

// nonGlobalProviderAdapter only implements ProviderAdapter, not GlobalProviderAdapter.
type nonGlobalProviderAdapter struct{ key string }

func (n *nonGlobalProviderAdapter) Key() string { return n.key }
func (n *nonGlobalProviderAdapter) DefaultProjectPaths() providers.ProjectScopePaths {
	return providers.ProjectScopePaths{}
}
func (n *nonGlobalProviderAdapter) Detect(_ string, _ providers.ProjectScopePaths, _ providers.FsReader) (providers.DetectResult, error) {
	return providers.DetectResult{}, nil
}

// singleAdapterRegistry holds one ProviderAdapter (non-global).
type singleAdapterRegistry struct{ adapter providers.ProviderAdapter }

func (r *singleAdapterRegistry) All() []providers.ProviderAdapter {
	if r.adapter == nil {
		return nil
	}
	return []providers.ProviderAdapter{r.adapter}
}
func (r *singleAdapterRegistry) Get(key string) (providers.ProviderAdapter, bool) {
	if r.adapter != nil && r.adapter.Key() == key {
		return r.adapter, true
	}
	return nil, false
}
