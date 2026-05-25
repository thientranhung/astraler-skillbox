package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
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
