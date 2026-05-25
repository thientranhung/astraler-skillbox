package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// noProviderWarning returns a DetectResult with Present=false and a no_provider_detected warning,
// mirroring what GenericAgentsAdapter and ClaudeAdapter emit when their detect path is absent.
func noProviderWarning(key string) providers.DetectResult {
	return providers.DetectResult{
		Present:         false,
		DetectionStatus: domain.DetectionStatusMissing,
		Warnings: []providers.AdapterWarning{{
			Code:      "no_provider_detected",
			Message:   "No provider detected",
			Severity:  domain.WarningSeverityWarning,
			ScopeType: domain.WarningScopeProject,
		}},
	}
}

func detectedResult(key, detectPath, skillsPath string) providers.DetectResult {
	return providers.DetectResult{
		Present:         true,
		DetectedPath:    detectPath,
		SkillsPath:      skillsPath,
		DetectionStatus: domain.DetectionStatusDetected,
	}
}

// TestWarningAggregation_AgentsOnly: generic_agents detected, claude not → no no_provider_detected warning.
func TestWarningAggregation_AgentsOnly(t *testing.T) {
	ctx := context.Background()
	projRepo := newMockProjectRepo()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	agentsAdapter := &mockAdapter{
		key:    "generic_agents",
		result: detectedResult("generic_agents", "/tmp/p/.agents", "/tmp/p/.agents/skills"),
	}
	claudeAdapter := &mockAdapter{
		key:    "claude",
		result: noProviderWarning("claude"),
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{agentsAdapter, claudeAdapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
			"claude":         {ID: 2, Key: "claude"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if len(scanRepo.lastProviders) != 1 {
		t.Fatalf("providers: got %d want 1", len(scanRepo.lastProviders))
	}
	for _, w := range scanRepo.lastProjectWarnings {
		if w.Code == "no_provider_detected" {
			t.Errorf("unexpected no_provider_detected warning when .agents is detected")
		}
	}
}

// TestWarningAggregation_ClaudeOnly: claude detected, generic_agents not → no no_provider_detected warning.
func TestWarningAggregation_ClaudeOnly(t *testing.T) {
	ctx := context.Background()
	projRepo := newMockProjectRepo()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	agentsAdapter := &mockAdapter{
		key:    "generic_agents",
		result: noProviderWarning("generic_agents"),
	}
	claudeAdapter := &mockAdapter{
		key:    "claude",
		result: detectedResult("claude", "/tmp/p/.claude", "/tmp/p/.claude/skills"),
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{agentsAdapter, claudeAdapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
			"claude":         {ID: 2, Key: "claude"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if len(scanRepo.lastProviders) != 1 {
		t.Fatalf("providers: got %d want 1", len(scanRepo.lastProviders))
	}
	for _, w := range scanRepo.lastProjectWarnings {
		if w.Code == "no_provider_detected" {
			t.Errorf("unexpected no_provider_detected warning when .claude is detected")
		}
	}
}

// TestWarningAggregation_NoProvider: neither adapter detects → exactly one no_provider_detected warning.
func TestWarningAggregation_NoProvider(t *testing.T) {
	ctx := context.Background()
	projRepo := newMockProjectRepo()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{
		&mockAdapter{key: "generic_agents", result: noProviderWarning("generic_agents")},
		&mockAdapter{key: "claude", result: noProviderWarning("claude")},
	}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
			"claude":         {ID: 2, Key: "claude"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if len(scanRepo.lastProviders) != 0 {
		t.Errorf("providers: got %d want 0", len(scanRepo.lastProviders))
	}

	count := 0
	for _, w := range scanRepo.lastProjectWarnings {
		if w.Code == "no_provider_detected" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("no_provider_detected warning count: got %d want 1", count)
	}
}
