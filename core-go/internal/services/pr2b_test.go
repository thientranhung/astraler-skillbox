package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// makeTestEntryFull creates a registry entry with both detect and skills project-scope candidates.
func makeTestEntryFull(key string, provID int64, detectRel, skillsRel string) domain.ProviderRegistryEntry {
	iconKey := key
	return domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			ID:           provID,
			Key:          key,
			DisplayName:  key,
			ProviderType: key,
			IconKey:      &iconKey,
			Status:       domain.ProviderStatusSupported,
		},
		Candidates: []domain.ProviderPathCandidate{
			{ProviderDefinitionID: provID, RelativePath: detectRel, Scope: "project", Purpose: "detect", Priority: 10},
			{ProviderDefinitionID: provID, RelativePath: skillsRel, Scope: "project", Purpose: "skills", Priority: 10},
		},
	}
}

// -- ProviderRegistryService.ProjectPaths tests --

func TestProjectPaths_NoOverride_ReturnsBuiltinPaths(t *testing.T) {
	entry := makeTestEntryFull("claude", 1, ".claude", ".claude/skills")
	svc := makeSvc([]domain.ProviderRegistryEntry{entry}, nil)

	got, err := svc.ProjectPaths(context.Background())
	if err != nil {
		t.Fatalf("ProjectPaths: %v", err)
	}
	p, ok := got["claude"]
	if !ok {
		t.Fatal("expected entry for 'claude'")
	}
	if p.DetectRel != ".claude" {
		t.Errorf("DetectRel: got %q want .claude", p.DetectRel)
	}
	if p.SkillsRel != ".claude/skills" {
		t.Errorf("SkillsRel: got %q want .claude/skills", p.SkillsRel)
	}
}

func TestProjectPaths_DetectOverride_ReturnsOverridePath(t *testing.T) {
	entry := makeTestEntryFull("claude", 1, ".claude", ".claude/skills")
	override := domain.ProviderPathOverride{
		ProviderDefinitionID: 1,
		Scope:                "project",
		Purpose:              "detect",
		Paths:                []string{".config/claude"},
	}
	svc := makeSvc([]domain.ProviderRegistryEntry{entry}, []domain.ProviderPathOverride{override})

	got, err := svc.ProjectPaths(context.Background())
	if err != nil {
		t.Fatalf("ProjectPaths: %v", err)
	}
	p := got["claude"]
	if p.DetectRel != ".config/claude" {
		t.Errorf("DetectRel: got %q want .config/claude", p.DetectRel)
	}
	// skills not overridden — falls back to builtin
	if p.SkillsRel != ".claude/skills" {
		t.Errorf("SkillsRel: got %q want .claude/skills", p.SkillsRel)
	}
}

func TestProjectPaths_SkillsOverride_ReturnsOverridePath(t *testing.T) {
	entry := makeTestEntryFull("claude", 1, ".claude", ".claude/skills")
	override := domain.ProviderPathOverride{
		ProviderDefinitionID: 1,
		Scope:                "project",
		Purpose:              "skills",
		Paths:                []string{".config/claude/skills"},
	}
	svc := makeSvc([]domain.ProviderRegistryEntry{entry}, []domain.ProviderPathOverride{override})

	got, err := svc.ProjectPaths(context.Background())
	if err != nil {
		t.Fatalf("ProjectPaths: %v", err)
	}
	p := got["claude"]
	if p.DetectRel != ".claude" {
		t.Errorf("DetectRel: got %q want .claude", p.DetectRel)
	}
	if p.SkillsRel != ".config/claude/skills" {
		t.Errorf("SkillsRel: got %q want .config/claude/skills", p.SkillsRel)
	}
}

func TestProjectPaths_GlobalScopeOverride_DoesNotAffectProjectPaths(t *testing.T) {
	entry := makeTestEntryFull("generic_agents", 2, ".agents", ".agents/skills")
	// global-scope override for detect — must NOT affect project paths
	globalOverride := domain.ProviderPathOverride{
		ProviderDefinitionID: 2,
		Scope:                "global",
		Purpose:              "detect",
		Paths:                []string{"/custom/global/agents"},
	}
	svc := makeSvc([]domain.ProviderRegistryEntry{entry}, []domain.ProviderPathOverride{globalOverride})

	got, err := svc.ProjectPaths(context.Background())
	if err != nil {
		t.Fatalf("ProjectPaths: %v", err)
	}
	p := got["generic_agents"]
	if p.DetectRel != ".agents" {
		t.Errorf("DetectRel: global override leaked into project — got %q want .agents", p.DetectRel)
	}
	if p.SkillsRel != ".agents/skills" {
		t.Errorf("SkillsRel: got %q want .agents/skills", p.SkillsRel)
	}
}

func TestProjectPaths_MultipleProviders(t *testing.T) {
	claudeEntry := makeTestEntryFull("claude", 1, ".claude", ".claude/skills")
	agentsEntry := makeTestEntryFull("generic_agents", 2, ".agents", ".agents/skills")
	claudeOverride := domain.ProviderPathOverride{
		ProviderDefinitionID: 1,
		Scope:                "project",
		Purpose:              "skills",
		Paths:                []string{"custom/skills"},
	}
	svc := makeSvc(
		[]domain.ProviderRegistryEntry{claudeEntry, agentsEntry},
		[]domain.ProviderPathOverride{claudeOverride},
	)

	got, err := svc.ProjectPaths(context.Background())
	if err != nil {
		t.Fatalf("ProjectPaths: %v", err)
	}
	// claude: detect builtin, skills overridden
	cp := got["claude"]
	if cp.DetectRel != ".claude" {
		t.Errorf("claude DetectRel: got %q", cp.DetectRel)
	}
	if cp.SkillsRel != "custom/skills" {
		t.Errorf("claude SkillsRel: got %q want custom/skills", cp.SkillsRel)
	}
	// generic_agents: both builtin (unaffected by claude override)
	gp := got["generic_agents"]
	if gp.DetectRel != ".agents" {
		t.Errorf("generic_agents DetectRel: got %q", gp.DetectRel)
	}
	if gp.SkillsRel != ".agents/skills" {
		t.Errorf("generic_agents SkillsRel: got %q", gp.SkillsRel)
	}
}

// -- Scan integration: path resolver passes effective paths to adapter --

// pathCapturingAdapter records the ProjectScopePaths it was called with.
type pathCapturingAdapter struct {
	key          string
	capturedPath providers.ProjectScopePaths
	result       providers.DetectResult
}

func (m *pathCapturingAdapter) Key() string { return m.key }
func (m *pathCapturingAdapter) DefaultProjectPaths() providers.ProjectScopePaths {
	return providers.ProjectScopePaths{DetectRel: "." + m.key, SkillsRel: "." + m.key + "/skills"}
}
func (m *pathCapturingAdapter) Detect(_ string, paths providers.ProjectScopePaths, _ providers.FsReader) (providers.DetectResult, error) {
	m.capturedPath = paths
	return m.result, nil
}

// mockPathResolver is a stub PathResolver returning a fixed map.
type mockPathResolver struct {
	paths map[string]providers.ProjectScopePaths
	err   error
}

func (m *mockPathResolver) ProjectPaths(_ context.Context) (map[string]providers.ProjectScopePaths, error) {
	return m.paths, m.err
}

func TestScan_NoPathResolver_UsesAdapterDefault(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &pathCapturingAdapter{
		key:    "claude",
		result: providers.DetectResult{Present: false, DetectionStatus: domain.DetectionStatusMissing},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	scanRepo := &mockProjectScanCommitter{}

	// Wire WITHOUT a path resolver → must fall back to adapter.DefaultProjectPaths()
	svc := NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		&mockProjectFS{},
	).WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{})

	project, _ := projRepo.GetByID(ctx, 1)
	svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}) //nolint:errcheck

	if adapter.capturedPath.DetectRel != ".claude" {
		t.Errorf("DetectRel: got %q want .claude", adapter.capturedPath.DetectRel)
	}
	if adapter.capturedPath.SkillsRel != ".claude/skills" {
		t.Errorf("SkillsRel: got %q want .claude/skills", adapter.capturedPath.SkillsRel)
	}
}

func TestScan_WithPathResolver_UsesResolvedPaths(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &pathCapturingAdapter{
		key:    "claude",
		result: providers.DetectResult{Present: false, DetectionStatus: domain.DetectionStatusMissing},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	scanRepo := &mockProjectScanCommitter{}

	resolver := &mockPathResolver{paths: map[string]providers.ProjectScopePaths{
		"claude": {DetectRel: ".config/claude", SkillsRel: ".config/claude/skills"},
	}}

	svc := NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		&mockProjectFS{},
	).WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithPathResolver(resolver)

	project, _ := projRepo.GetByID(ctx, 1)
	svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}) //nolint:errcheck

	if adapter.capturedPath.DetectRel != ".config/claude" {
		t.Errorf("DetectRel: got %q want .config/claude", adapter.capturedPath.DetectRel)
	}
	if adapter.capturedPath.SkillsRel != ".config/claude/skills" {
		t.Errorf("SkillsRel: got %q want .config/claude/skills", adapter.capturedPath.SkillsRel)
	}
}

func TestScan_UnknownProviderInResolver_FallsBackToAdapterDefault(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &pathCapturingAdapter{
		key:    "claude",
		result: providers.DetectResult{Present: false, DetectionStatus: domain.DetectionStatusMissing},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	scanRepo := &mockProjectScanCommitter{}

	// Resolver returns empty map — provider not in map, must fall back to adapter default.
	resolver := &mockPathResolver{paths: map[string]providers.ProjectScopePaths{}}

	svc := NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		&mockProjectFS{},
	).WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithPathResolver(resolver)

	project, _ := projRepo.GetByID(ctx, 1)
	svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}) //nolint:errcheck

	if adapter.capturedPath.DetectRel != ".claude" {
		t.Errorf("DetectRel: got %q want .claude (fallback)", adapter.capturedPath.DetectRel)
	}
}

// TestScan_WithPathResolver_PresentTrue_CommitsDetectedAndSkillsPath verifies that when
// an adapter returns Present:true using the effective resolved paths, the committed
// ProviderScanResult carries those DetectedPath and SkillsPath values.
func TestScan_WithPathResolver_PresentTrue_CommitsDetectedAndSkillsPath(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &pathCapturingAdapter{
		key: "claude",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.config/claude",
			SkillsPath:      "/tmp/p/.config/claude/skills",
			DetectionStatus: domain.DetectionStatusDetected,
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	scanRepo := &mockProjectScanCommitter{}

	resolver := &mockPathResolver{paths: map[string]providers.ProjectScopePaths{
		"claude": {DetectRel: ".config/claude", SkillsRel: ".config/claude/skills"},
	}}

	svc := NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		&mockProjectFS{},
	).WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithPathResolver(resolver)

	project, _ := projRepo.GetByID(ctx, 1)
	if _, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("CommitProjectScan call count: got %d want 1", scanRepo.fullScanCallCount)
	}
	if len(scanRepo.lastProviders) != 1 {
		t.Fatalf("committed providers: got %d want 1", len(scanRepo.lastProviders))
	}
	psr := scanRepo.lastProviders[0]
	if psr.DetectedPath == nil || *psr.DetectedPath != "/tmp/p/.config/claude" {
		t.Errorf("DetectedPath: got %v want /tmp/p/.config/claude", psr.DetectedPath)
	}
	if psr.SkillsPath == nil || *psr.SkillsPath != "/tmp/p/.config/claude/skills" {
		t.Errorf("SkillsPath: got %v want /tmp/p/.config/claude/skills", psr.SkillsPath)
	}
	if psr.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %v want detected", psr.DetectionStatus)
	}
}

// -- Install integration: path resolver affects skills target path --

// mockInstallPathResolver records whether it was called and returns configured paths.
type mockInstallPathResolver struct {
	paths  map[string]providers.ProjectScopePaths
	called bool
	err    error
}

func (m *mockInstallPathResolver) ProjectPaths(_ context.Context) (map[string]providers.ProjectScopePaths, error) {
	m.called = true
	return m.paths, m.err
}

// TestInstall_WithSkillsPathOverride_LinksIntoOverriddenDir verifies that when a
// project-scope skills override is active, installSkillsInternal creates the symlink
// in the overridden directory, not the default one.
func TestInstall_WithSkillsPathOverride_LinksIntoOverriddenDir(t *testing.T) {
	ctx := context.Background()

	// Project dir: create the custom override skills dir, not the builtin .agents/skills.
	projectDir := t.TempDir()
	overrideSkillsDir := filepath.Join(projectDir, "custom", "skills")
	if err := os.MkdirAll(overrideSkillsDir, 0o755); err != nil {
		t.Fatalf("mkdir override skills: %v", err)
	}

	// Host dir with one skill folder.
	hostSkillsDir := t.TempDir()
	hostSkillPath := filepath.Join(hostSkillsDir, "test-skill")
	if err := os.MkdirAll(hostSkillPath, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}

	gw := filesystem.NewGateway()
	project := &domain.Project{
		ID:     1,
		Name:   "override-test",
		Path:   projectDir,
		Status: domain.ProjectStatusActive,
	}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{
				ProviderKey:     providers.GenericAgentsKey,
				DetectionStatus: domain.DetectionStatusDetected,
			}},
		},
	}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			providers.GenericAgentsKey: {
				ID:                 10,
				Key:                providers.GenericAgentsKey,
				Status:             domain.ProviderStatusSupported,
				CanCreateStructure: true,
			},
		},
	}
	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: hostSkillsDir, Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}
	skillLister := &mockSkillsByHostLister{
		skills: map[int64][]domain.Skill{
			1: {{ID: 1, Name: "test-skill", AbsolutePath: hostSkillPath, Status: domain.SkillStatusAvailable}},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	scanRepo := &mockProjectScanCommitter{}
	runner := &mockRunner{}

	resolver := &mockInstallPathResolver{paths: map[string]providers.ProjectScopePaths{
		providers.GenericAgentsKey: {
			DetectRel: ".agents",
			SkillsRel: "custom/skills",
		},
	}}

	svc := NewProjectService(projRepo, ppRepo, &mockProjectWarningRepo{}, &mockProjectInstallRepo{}, gw).
		WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister).
		WithPathResolver(resolver)

	_, err := svc.installSkillsInternal(ctx, project, providers.GenericAgentsKey, []int64{1}, noopProgress)
	if err != nil {
		t.Fatalf("installSkillsInternal: %v", err)
	}

	// Symlink must be in the overridden dir, not the builtin .agents/skills.
	linkPath := filepath.Join(overrideSkillsDir, "test-skill")
	fi, lerr := os.Lstat(linkPath)
	if lerr != nil {
		t.Fatalf("expected symlink at overridden path %q: %v", linkPath, lerr)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got %v", fi.Mode())
	}

	// Default builtin dir must NOT contain the skill.
	defaultLinkPath := filepath.Join(projectDir, providers.GenericAgentsSkillsPath, "test-skill")
	if _, err := os.Lstat(defaultLinkPath); err == nil {
		t.Errorf("skill unexpectedly found in builtin dir %q", defaultLinkPath)
	}

	if !resolver.called {
		t.Error("expected path resolver to be called during install")
	}
}

// TestInstall_GlobalScopeOverride_UsesBuiltinPath verifies that when the path resolver
// returns an empty map (simulating global-scope-only overrides being filtered out by
// ProjectPaths), installSkillsInternal falls back to the builtin skills path and does
// NOT write to any global override location.
func TestInstall_GlobalScopeOverride_UsesBuiltinPath(t *testing.T) {
	ctx := context.Background()

	// Project dir: create the builtin .agents/skills dir.
	projectDir := t.TempDir()
	builtinSkillsDir := filepath.Join(projectDir, providers.GenericAgentsSkillsPath)
	if err := os.MkdirAll(builtinSkillsDir, 0o755); err != nil {
		t.Fatalf("mkdir builtin skills: %v", err)
	}

	// Host dir with one skill folder.
	hostSkillsDir := t.TempDir()
	hostSkillPath := filepath.Join(hostSkillsDir, "test-skill")
	if err := os.MkdirAll(hostSkillPath, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}

	gw := filesystem.NewGateway()
	project := &domain.Project{
		ID:     1,
		Name:   "global-override-test",
		Path:   projectDir,
		Status: domain.ProjectStatusActive,
	}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{
				ProviderKey:     providers.GenericAgentsKey,
				DetectionStatus: domain.DetectionStatusDetected,
			}},
		},
	}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			providers.GenericAgentsKey: {
				ID:                 10,
				Key:                providers.GenericAgentsKey,
				Status:             domain.ProviderStatusSupported,
				CanCreateStructure: true,
			},
		},
	}
	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: hostSkillsDir, Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}
	skillLister := &mockSkillsByHostLister{
		skills: map[int64][]domain.Skill{
			1: {{ID: 1, Name: "test-skill", AbsolutePath: hostSkillPath, Status: domain.SkillStatusAvailable}},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	scanRepo := &mockProjectScanCommitter{}
	runner := &mockRunner{}

	// Resolver returns empty map: simulates global-scope-only override being filtered out.
	resolver := &mockInstallPathResolver{paths: map[string]providers.ProjectScopePaths{}}

	svc := NewProjectService(projRepo, ppRepo, &mockProjectWarningRepo{}, &mockProjectInstallRepo{}, gw).
		WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister).
		WithPathResolver(resolver)

	_, err := svc.installSkillsInternal(ctx, project, providers.GenericAgentsKey, []int64{1}, noopProgress)
	if err != nil {
		t.Fatalf("installSkillsInternal: %v", err)
	}

	// Symlink must be in the builtin dir.
	linkPath := filepath.Join(builtinSkillsDir, "test-skill")
	fi, lerr := os.Lstat(linkPath)
	if lerr != nil {
		t.Fatalf("expected symlink at builtin path %q: %v", linkPath, lerr)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got %v", fi.Mode())
	}

	if !resolver.called {
		t.Error("expected path resolver to be called during install")
	}
}
