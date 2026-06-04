package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
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

// TestPR2C_ProviderNotInResolver_CommitsNotConfigured verifies that when the resolver map
// does not include a key for an otherwise global-capable adapter, the registry-driven scan
// commits a not_configured status rather than silently omitting the provider.
func TestPR2C_ProviderNotInResolver_CommitsNotConfigured(t *testing.T) {
	home := "/fakehome"

	// Adapter IS global-capable but resolver excludes it (has_global_level=false semantics).
	capAdapter := &capturingGlobalAdapter{
		key:          providers.GenericAgentsKey,
		result:       providers.GlobalDetectResult{Present: true, Status: domain.GlobalLocationStatusActive},
		capturedPath: &providers.GlobalScopePaths{},
	}

	// Resolver returns empty map — no providers with global level.
	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{},
	}

	// Registry lister returns one entry for generic_agents.
	regLister := &mockRegistryLister{
		entries: []domain.ProviderRegistryEntry{
			{Definition: domain.ProviderDefinition{ID: 1, Key: providers.GenericAgentsKey, DisplayName: "Shared Agent Skills", Status: domain.ProviderStatusSupported}},
		},
	}

	multiWriter := &multiCommitWriter{}
	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(&mockGlobalRepo{defID: 1}, multiWriter, fs, capAdapter, resolver).
		WithProviderRegistryLister(regLister)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	if len(multiWriter.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(multiWriter.commits))
	}
	if multiWriter.commits[0].status != domain.GlobalLocationStatusNotConfigured {
		t.Errorf("expected not_configured, got %q", multiWriter.commits[0].status)
	}
}

// TestPR2C_NoGlobalAdapter_CommitsNoGlobalSkills verifies that adapters not implementing
// GlobalProviderAdapter cause the registry-driven scan to commit a no_global_skills status
// rather than silently omitting the provider row.
func TestPR2C_NoGlobalAdapter_CommitsNoGlobalSkills(t *testing.T) {
	home := "/fakehome"

	// Non-global adapter (only implements ProviderAdapter, not GlobalProviderAdapter).
	nonGlobalAdapter := &nonGlobalProviderAdapter{key: "codex"}
	registry := &singleAdapterRegistry{adapter: nonGlobalAdapter}

	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{
			"codex": {DetectRel: "~/.codex", SkillsRel: "~/.codex/skills"},
		},
	}

	regLister := &mockRegistryLister{
		entries: []domain.ProviderRegistryEntry{
			{Definition: domain.ProviderDefinition{ID: 1, Key: "codex", DisplayName: "Codex", Status: domain.ProviderStatusSupported}},
		},
	}

	multiWriter := &multiCommitWriter{}
	fs := newMockGlobalFS(home)
	svc := NewGlobalSkillsService(
		&mockGlobalRepo{defID: 1}, multiWriter, newMockSettings(nil),
		&mockHostLister{}, &mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry, fs, &syncRunner{},
	).WithGlobalPathResolver(resolver).WithProviderRegistryLister(regLister)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}
	if len(multiWriter.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(multiWriter.commits))
	}
	if multiWriter.commits[0].status != domain.GlobalLocationStatusNoGlobalSkills {
		t.Errorf("expected no_global_skills, got %q", multiWriter.commits[0].status)
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

// -- Issue 1 regression tests: ProviderDefByKey DB error must fail the scan --

// TestPR2C_ProviderDefDBError_FailsScan verifies that a real DB error from ProviderDefByKey
// fails the scan rather than silently skipping the provider.
func TestPR2C_ProviderDefDBError_FailsScan(t *testing.T) {
	dbErr := errors.New("database is locked")
	// defID=0 with non-nil error simulates a real DB failure (not a not-found).
	globalRepo := &mockGlobalRepo{defID: 0, defErr: dbErr}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"

	capAdapter := &capturingGlobalAdapter{
		key:          providers.GenericAgentsKey,
		result:       providers.GlobalDetectResult{},
		capturedPath: &providers.GlobalScopePaths{},
	}

	// No resolver: adapter defaults are used, so the provider is not skipped by the resolver.
	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, nil)

	_, err := svc.ScanGlobal(context.Background())
	if err == nil {
		t.Fatal("expected scan to fail when ProviderDefByKey returns a DB error")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
	// No commit should have happened.
	if len(scanWriter.committed) != 0 {
		t.Errorf("expected no commits after DB error, got %d", len(scanWriter.committed))
	}
}

// TestPR2C_ProviderDefNotFound_SkipsSilently verifies that when ProviderDefByKey returns
// defID=0 with no error (provider not seeded), the provider is silently skipped and
// the scan succeeds with no commits.
func TestPR2C_ProviderDefNotFound_SkipsSilently(t *testing.T) {
	// defID=0 with nil error: the new contract for "not found".
	globalRepo := &mockGlobalRepo{defID: 0, defErr: nil}
	scanWriter := &mockGlobalScanWriter{}
	home := "/fakehome"

	capAdapter := &capturingGlobalAdapter{
		key:          providers.GenericAgentsKey,
		result:       providers.GlobalDetectResult{Present: true, Status: domain.GlobalLocationStatusActive},
		capturedPath: &providers.GlobalScopePaths{},
	}

	fs := newMockGlobalFS(home)
	svc := newGlobalServiceWithResolver(globalRepo, scanWriter, fs, capAdapter, nil)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("expected scan to succeed for not-found provider, got: %v", err)
	}
	if len(scanWriter.committed) != 0 {
		t.Errorf("expected 0 commits for not-found provider, got %d", len(scanWriter.committed))
	}
}

// -- mock registry lister --

type mockRegistryLister struct {
	entries []domain.ProviderRegistryEntry
	err     error
}

func (m *mockRegistryLister) ListAll(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
	return m.entries, m.err
}

// -- multi-commit writer for registry-driven tests --

type commitRecord struct {
	defID    int64
	status   domain.GlobalLocationStatus
	installs []repositories.GlobalInstallScanResult
}

type multiCommitWriter struct {
	commits []commitRecord
	err     error
}

func (w *multiCommitWriter) CommitGlobalScan(
	_ context.Context, defID int64, _, _ *string,
	status domain.GlobalLocationStatus, installs []repositories.GlobalInstallScanResult,
	_ []domain.Warning, _ time.Time,
) error {
	w.commits = append(w.commits, commitRecord{defID: defID, status: status, installs: installs})
	return w.err
}

// TestPR2C_RegistryDriven_FullCoverage verifies that when a registry lister is injected,
// every provider in the registry receives an explicit state:
//   - provider with a global adapter → scanned (active/missing/etc.)
//   - provider without a global adapter → no_global_skills
//   - disabled provider → disabled
//   - global-capable provider excluded from resolver → not_configured
func TestPR2C_RegistryDriven_FullCoverage(t *testing.T) {
	home := "/fakehome"
	skillsPath := home + "/.agents/skills"

	// Four providers in the registry:
	// 1. generic_agents — global-capable adapter, in resolver → normal scan
	// 2. codex         — no global adapter → no_global_skills
	// 3. claude        — global-capable adapter, excluded from resolver → not_configured
	// 4. disabled_prov — disabled → disabled
	regLister := &mockRegistryLister{
		entries: []domain.ProviderRegistryEntry{
			{Definition: domain.ProviderDefinition{ID: 1, Key: providers.GenericAgentsKey, Status: domain.ProviderStatusSupported}},
			{Definition: domain.ProviderDefinition{ID: 2, Key: "codex", Status: domain.ProviderStatusSupported}},
			{Definition: domain.ProviderDefinition{ID: 3, Key: "claude", Status: domain.ProviderStatusSupported}},
			{Definition: domain.ProviderDefinition{ID: 4, Key: "disabled_prov", Status: domain.ProviderStatusSupported}},
		},
	}

	// Adapter registry: generic_agents and claude have global adapters; codex does not.
	genericAdapter := &mockGlobalAdapter{
		key: providers.GenericAgentsKey,
		result: providers.GlobalDetectResult{
			Present:          true,
			GlobalPath:       home + "/.agents",
			GlobalSkillsPath: skillsPath,
			Status:           domain.GlobalLocationStatusActive,
		},
	}
	claudeAdapter := &mockGlobalAdapter{
		key:    "claude",
		result: providers.GlobalDetectResult{Present: false, Status: domain.GlobalLocationStatusMissing},
	}
	codexNonGlobal := &nonGlobalProviderAdapter{key: "codex"}

	type multiAdapter struct {
		adapters []providers.ProviderAdapter
	}
	multiReg := struct {
		adapters []providers.ProviderAdapter
	}{
		adapters: []providers.ProviderAdapter{genericAdapter, claudeAdapter, codexNonGlobal},
	}
	registry := &multiAdapterRegistry{adapters: multiReg.adapters}

	// Resolver: only generic_agents has a global path; claude is excluded.
	resolver := &mockGlobalPathResolver{
		paths: map[string]providers.GlobalScopePaths{
			providers.GenericAgentsKey: {DetectRel: "~/.agents", SkillsRel: "~/.agents/skills"},
		},
	}

	// Enabled reader: disabled_prov is disabled.
	enabledReader := &mockEnabledReader{m: map[string]bool{"disabled_prov": false}}

	multiWriter := &multiCommitWriter{}
	fs := newMockGlobalFS(home)

	svc := NewGlobalSkillsService(
		&mockGlobalRepo{defID: 1}, multiWriter, newMockSettings(nil),
		&mockHostLister{}, &mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry, fs, &syncRunner{},
	).WithGlobalPathResolver(resolver).WithEnabledReader(enabledReader).WithProviderRegistryLister(regLister)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}

	if len(multiWriter.commits) != 4 {
		t.Fatalf("expected 4 commits (one per registry row), got %d", len(multiWriter.commits))
	}

	statusByDefID := map[int64]domain.GlobalLocationStatus{}
	for _, c := range multiWriter.commits {
		statusByDefID[c.defID] = c.status
	}

	if statusByDefID[1] != domain.GlobalLocationStatusActive {
		t.Errorf("generic_agents (defID=1): expected active, got %q", statusByDefID[1])
	}
	if statusByDefID[2] != domain.GlobalLocationStatusNoGlobalSkills {
		t.Errorf("codex (defID=2): expected no_global_skills, got %q", statusByDefID[2])
	}
	if statusByDefID[3] != domain.GlobalLocationStatusNotConfigured {
		t.Errorf("claude (defID=3): expected not_configured, got %q", statusByDefID[3])
	}
	if statusByDefID[4] != domain.GlobalLocationStatusDisabled {
		t.Errorf("disabled_prov (defID=4): expected disabled, got %q", statusByDefID[4])
	}
}

// multiAdapterRegistry holds multiple adapters.
type multiAdapterRegistry struct {
	adapters []providers.ProviderAdapter
}

func (r *multiAdapterRegistry) All() []providers.ProviderAdapter { return r.adapters }
func (r *multiAdapterRegistry) Get(key string) (providers.ProviderAdapter, bool) {
	for _, a := range r.adapters {
		if a.Key() == key {
			return a, true
		}
	}
	return nil, false
}
