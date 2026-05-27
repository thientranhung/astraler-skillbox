package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

// newFullScanSvc wires a ProjectService with all provider-scan deps for full-scan tests.
func newFullScanSvc(
	projRepo *mockProjectRepo,
	fs *mockProjectFS,
	runner *mockRunner,
	scanRepo *mockProjectScanCommitter,
	registry *mockProviderRegistry,
	pdRepo *mockProviderDefRepo,
	hostLister *mockHostLister,
	skillLister *mockSkillsByHostLister,
) *ProjectService {
	return NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		fs,
	).WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister)
}

// TestScanProjectInternal_ProviderDetected_PlainDirEntry_CommitsFullScan is the
// M3c2b2 happy-path: readable project root, one adapter returning a single plain
// directory entry → CommitProjectScan called once with the provider definition ID,
// detected paths, DetectionStatusDetected, and one direct/current install.
func TestScanProjectInternal_ProviderDetected_PlainDirEntry_CommitsFullScan(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "myproject", "/tmp/myproject") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/myproject/.agents",
			SkillsPath:      "/tmp/myproject/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
			Entries: []providers.AdapterEntry{
				{
					Name:  "my-tool",
					Path:  "/tmp/myproject/.agents/skills/my-tool",
					IsDir: true,
				},
			},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 42, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}

	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{},
	)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
	if scanRepo.terminalCallCount != 0 {
		t.Errorf("CommitProjectTerminal calls: got %d want 0", scanRepo.terminalCallCount)
	}

	if len(scanRepo.lastProviders) != 1 {
		t.Fatalf("provider results: got %d want 1", len(scanRepo.lastProviders))
	}
	p := scanRepo.lastProviders[0]
	if p.ProviderDefinitionID != 42 {
		t.Errorf("ProviderDefinitionID: got %d want 42", p.ProviderDefinitionID)
	}
	if p.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", p.DetectionStatus)
	}
	if p.DetectedPath == nil || *p.DetectedPath != "/tmp/myproject/.agents" {
		t.Errorf("DetectedPath: got %v want /tmp/myproject/.agents", p.DetectedPath)
	}
	if p.SkillsPath == nil || *p.SkillsPath != "/tmp/myproject/.agents/skills" {
		t.Errorf("SkillsPath: got %v want /tmp/myproject/.agents/skills", p.SkillsPath)
	}

	if len(p.Installs) != 1 {
		t.Fatalf("installs: got %d want 1", len(p.Installs))
	}
	inst := p.Installs[0]
	if inst.SkillName != "my-tool" {
		t.Errorf("SkillName: got %q want my-tool", inst.SkillName)
	}
	if inst.InstallMode != domain.InstallModeDirect {
		t.Errorf("InstallMode: got %q want direct", inst.InstallMode)
	}
	if inst.InstallStatus != domain.InstallStatusCurrent {
		t.Errorf("InstallStatus: got %q want current", inst.InstallStatus)
	}
	if inst.ProjectSkillPath != "/tmp/myproject/.agents/skills/my-tool" {
		t.Errorf("ProjectSkillPath: got %q want /tmp/myproject/.agents/skills/my-tool", inst.ProjectSkillPath)
	}
}

// --- install warning tests ---

// svcWithEntries returns a ProjectService whose fake adapter reports Present=true
// with the given entries under /tmp/p/.agents/skills. No skill hosts registered.
func svcWithEntries(entries []providers.AdapterEntry) (*ProjectService, *mockProjectRepo, *mockProjectScanCommitter) {
	projRepo := newMockProjectRepo()
	projRepo.UpsertByPath(context.Background(), "p", "/tmp/p") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.agents",
			SkillsPath:      "/tmp/p/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
			Entries:         entries,
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{},
	)
	return svc, projRepo, scanRepo
}

func firstInstall(t *testing.T, scanRepo *mockProjectScanCommitter) *repositories.InstallScanResult {
	t.Helper()
	if len(scanRepo.lastProviders) == 0 || len(scanRepo.lastProviders[0].Installs) == 0 {
		t.Fatal("expected at least one committed install")
	}
	inst := scanRepo.lastProviders[0].Installs[0]
	return &inst
}

func TestScanProjectInternal_BrokenSymlink_InstallHasRescanWarning(t *testing.T) {
	svc, projRepo, scanRepo := svcWithEntries([]providers.AdapterEntry{
		{Name: "sk", Path: "/tmp/p/.agents/skills/sk", IsSymlink: true, Broken: true, SymlinkTargetRaw: "/missing"},
	})
	project, _ := projRepo.GetByID(context.Background(), 1)
	if _, err := svc.scanProjectInternal(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	inst := firstInstall(t, scanRepo)
	if inst.Warning == nil {
		t.Fatal("expected warning for broken_symlink install, got nil")
	}
	if inst.Warning.Code != "broken_symlink" {
		t.Errorf("code: got %q want broken_symlink", inst.Warning.Code)
	}
	if inst.Warning.Severity != domain.WarningSeverityWarning {
		t.Errorf("severity: got %q want warning", inst.Warning.Severity)
	}
	if inst.Warning.ActionKey == nil || *inst.Warning.ActionKey != "rescan" {
		t.Errorf("actionKey: got %v want rescan", inst.Warning.ActionKey)
	}
	if inst.Warning.ScopeType != domain.WarningScopeInstall {
		t.Errorf("scopeType: got %q want install", inst.Warning.ScopeType)
	}
}

func TestScanProjectInternal_ExternalSymlink_InstallHasOpenFolderWarning(t *testing.T) {
	svc, projRepo, scanRepo := svcWithEntries([]providers.AdapterEntry{
		{Name: "sk", Path: "/tmp/p/.agents/skills/sk", IsSymlink: true, ResolvedTarget: "/outside/path/sk"},
	})
	project, _ := projRepo.GetByID(context.Background(), 1)
	if _, err := svc.scanProjectInternal(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	inst := firstInstall(t, scanRepo)
	if inst.Warning == nil {
		t.Fatal("expected warning for external_symlink install, got nil")
	}
	if inst.Warning.Code != "external_symlink" {
		t.Errorf("code: got %q want external_symlink", inst.Warning.Code)
	}
	if inst.Warning.ActionKey == nil || *inst.Warning.ActionKey != "open_folder" {
		t.Errorf("actionKey: got %v want open_folder", inst.Warning.ActionKey)
	}
}

func TestScanProjectInternal_OldHostSymlink_InstallHasRescanWarning(t *testing.T) {
	inactiveHost := domain.SkillHostFolder{
		ID: 5, SkillsPath: "/hosts/old/.agents/skills",
		Status: domain.SkillHostStatusInactive,
	}
	projRepo := newMockProjectRepo()
	projRepo.UpsertByPath(context.Background(), "p", "/tmp/p") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.agents",
			SkillsPath:      "/tmp/p/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
			Entries: []providers.AdapterEntry{
				{
					Name:           "sk",
					Path:           "/tmp/p/.agents/skills/sk",
					IsSymlink:      true,
					ResolvedTarget: "/hosts/old/.agents/skills/sk",
				},
			},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo,
		&mockHostLister{hosts: []domain.SkillHostFolder{inactiveHost}},
		&mockSkillsByHostLister{},
	)

	project, _ := projRepo.GetByID(context.Background(), 1)
	if _, err := svc.scanProjectInternal(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	inst := firstInstall(t, scanRepo)
	if inst.Warning == nil {
		t.Fatal("expected warning for old_host install, got nil")
	}
	if inst.Warning.Code != "old_host_symlink" {
		t.Errorf("code: got %q want old_host_symlink", inst.Warning.Code)
	}
	if inst.Warning.ActionKey == nil || *inst.Warning.ActionKey != "rescan" {
		t.Errorf("actionKey: got %v want rescan", inst.Warning.ActionKey)
	}
}

func TestScanProjectInternal_ErrorEntry_InstallHasInfoWarning(t *testing.T) {
	svc, projRepo, scanRepo := svcWithEntries([]providers.AdapterEntry{
		// Regular file (not dir, not symlink) → direct/error
		{Name: "bad-file", Path: "/tmp/p/.agents/skills/bad-file", IsDir: false, IsSymlink: false},
	})
	project, _ := projRepo.GetByID(context.Background(), 1)
	if _, err := svc.scanProjectInternal(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	inst := firstInstall(t, scanRepo)
	if inst.Warning == nil {
		t.Fatal("expected warning for error install, got nil")
	}
	if inst.Warning.Code != "entry_error" {
		t.Errorf("code: got %q want entry_error", inst.Warning.Code)
	}
	if inst.Warning.Severity != domain.WarningSeverityInfo {
		t.Errorf("severity: got %q want info", inst.Warning.Severity)
	}
	if inst.Warning.ActionKey == nil || *inst.Warning.ActionKey != "open_folder" {
		t.Errorf("actionKey: got %v want open_folder", inst.Warning.ActionKey)
	}
}

func TestScanProjectInternal_DirectCurrentEntry_NoInstallWarning(t *testing.T) {
	svc, projRepo, scanRepo := svcWithEntries([]providers.AdapterEntry{
		{Name: "tool", Path: "/tmp/p/.agents/skills/tool", IsDir: true},
	})
	project, _ := projRepo.GetByID(context.Background(), 1)
	if _, err := svc.scanProjectInternal(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	inst := firstInstall(t, scanRepo)
	if inst.Warning != nil {
		t.Errorf("expected nil warning for direct/current install, got %+v", inst.Warning)
	}
}

// TestScanProjectInternal_ProviderNotPresent_ZeroProviderResults verifies that when the
// adapter reports Present=false (no .agents directory found), CommitProjectScan receives
// zero ProviderScanResults and one project-scoped no_provider_detected warning.
func TestScanProjectInternal_ProviderNotPresent_ZeroProviderResults(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "myproject", "/tmp/myproject") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         false,
			DetectionStatus: domain.DetectionStatusMissing,
			Warnings: []providers.AdapterWarning{
				{
					Code:      "no_provider_detected",
					Message:   "No generic agents provider detected (.agents directory not found)",
					Severity:  domain.WarningSeverityWarning,
					ScopeType: domain.WarningScopeProject,
				},
			},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 42, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}

	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{},
	)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
	if len(scanRepo.lastProviders) != 0 {
		t.Errorf("provider results: got %d want 0 (provider not present)", len(scanRepo.lastProviders))
	}
	if len(scanRepo.lastProjectWarnings) != 1 {
		t.Fatalf("project warnings: got %d want 1", len(scanRepo.lastProjectWarnings))
	}
	w := scanRepo.lastProjectWarnings[0]
	if w.Code != "no_provider_detected" {
		t.Errorf("warning code: got %q want no_provider_detected", w.Code)
	}
	if w.ScopeType != domain.WarningScopeProject {
		t.Errorf("warning scope: got %q want project", w.ScopeType)
	}
}

// --- metadata summary tests ---

// TestScanProjectInternal_MetadataCounts verifies that scanProjectInternal returns a
// non-nil summary with correct providersFound, entriesClassified, and warningsCreated.
// Setup: 1 present provider, 2 entries (plain dir + broken symlink), 1 provider warning.
// Expected counts: providers=1, entries=2, warnings=2 (1 provider + 1 install warning).
func TestScanProjectInternal_MetadataCounts(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.agents",
			SkillsPath:      "/tmp/p/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
			Entries: []providers.AdapterEntry{
				{Name: "tool", Path: "/tmp/p/.agents/skills/tool", IsDir: true},
				{Name: "sk", Path: "/tmp/p/.agents/skills/sk", IsSymlink: true, Broken: true},
			},
			Warnings: []providers.AdapterWarning{
				{Code: "some_warning", Severity: domain.WarningSeverityWarning, ScopeType: domain.WarningScopeProjectProvider},
			},
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{},
	)

	project, _ := projRepo.GetByID(ctx, 1)
	got, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil metadata summary, got nil")
	}
	summary, ok := got.(*projectScanSummary)
	if !ok {
		t.Fatalf("expected *projectScanSummary, got %T", got)
	}
	if summary.ProvidersFound != 1 {
		t.Errorf("ProvidersFound: got %d want 1", summary.ProvidersFound)
	}
	if summary.EntriesClassified != 2 {
		t.Errorf("EntriesClassified: got %d want 2", summary.EntriesClassified)
	}
	if summary.WarningsCreated != 2 {
		t.Errorf("WarningsCreated: got %d want 2 (1 provider + 1 install)", summary.WarningsCreated)
	}
}

type fakeProjectPluginScanner struct {
	called     int
	gotProject *domain.Project
	err        error
}

func (f *fakeProjectPluginScanner) ScanProjectLayers(_ context.Context, project *domain.Project, progress operations.ProgressFn) error {
	f.called++
	f.gotProject = project
	progress("scanning_plugins", 1, 1, "")
	return f.err
}

func TestScanProjectInternal_InvokesPluginScannerAfterCommit(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "myproject", "/tmp/myproject") //nolint:errcheck

	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/myproject/.agents",
			SkillsPath:      "/tmp/myproject/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 42, Key: "generic_agents"},
	}}
	scanRepo := &mockProjectScanCommitter{}
	scanner := &fakeProjectPluginScanner{}

	svc := newFullScanSvc(
		projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{},
	).WithPluginDeps(scanner, nil)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if scanner.called != 1 {
		t.Fatalf("plugin scanner calls: got %d want 1", scanner.called)
	}
	if scanRepo.fullScanCallCount != 1 {
		t.Errorf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
	if scanner.gotProject == nil || scanner.gotProject.ID != project.ID {
		t.Errorf("scanner got wrong project: %v", scanner.gotProject)
	}
}

func TestScanProjectInternal_NoPluginScanner_Succeeds(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "myproject", "/tmp/myproject") //nolint:errcheck

	adapter := &mockAdapter{key: "generic_agents", result: providers.DetectResult{
		Present: true, DetectedPath: "/tmp/myproject/.agents",
		SkillsPath: "/tmp/myproject/.agents/skills", DetectionStatus: domain.DetectionStatusDetected,
	}}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 42, Key: "generic_agents"},
	}}
	scanRepo := &mockProjectScanCommitter{}

	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}) // no WithPluginDeps

	project, _ := projRepo.GetByID(ctx, 1)
	if _, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("scanProjectInternal without plugin scanner: %v", err)
	}
	if scanRepo.fullScanCallCount != 1 {
		t.Errorf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
}

func TestScanProjectInternal_PluginError_StillReturnsSkillSummary(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "myproject", "/tmp/myproject") //nolint:errcheck

	adapter := &mockAdapter{key: "generic_agents", result: providers.DetectResult{
		Present: true, DetectedPath: "/tmp/myproject/.agents",
		SkillsPath: "/tmp/myproject/.agents/skills", DetectionStatus: domain.DetectionStatusDetected,
	}}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 42, Key: "generic_agents"},
	}}
	scanRepo := &mockProjectScanCommitter{}
	scanner := &fakeProjectPluginScanner{err: domain.NewDatabaseError("boom", "plugin commit failed")}

	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo,
		registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithPluginDeps(scanner, nil)

	project, _ := projRepo.GetByID(ctx, 1)
	meta, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err == nil {
		t.Fatal("expected plugin-step error to propagate")
	}
	if meta == nil {
		t.Fatal("expected skill-scan summary metadata alongside the error, got nil (F3)")
	}
	if _, ok := meta.(*projectScanSummary); !ok {
		t.Errorf("metadata type: got %T want *projectScanSummary", meta)
	}
	if scanRepo.fullScanCallCount != 1 {
		t.Errorf("CommitProjectScan calls: got %d want 1 (skills committed before plugin step)", scanRepo.fullScanCallCount)
	}
}
