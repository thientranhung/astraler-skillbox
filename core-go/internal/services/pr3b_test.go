package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// -- EnabledMap tests --

func TestEnableMap_DefaultsFromStatus(t *testing.T) {
	cases := []struct {
		status  string
		want    bool
		wantKey string
	}{
		{"supported", true, "prov_s"},
		{"experimental", true, "prov_e"},
		{"unsupported", false, "prov_u"},
		{"disabled", false, "prov_d"},
	}

	entries := make([]domain.ProviderRegistryEntry, len(cases))
	for i, tc := range cases {
		entries[i] = makeEntryWithID(int64(i+1), tc.wantKey, tc.status)
	}

	repo := &mockProviderRegistryRepo{entries: entries}
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{}, &mockProviderUserSettingsRepo{})

	got, err := svc.EnabledMap(context.Background())
	if err != nil {
		t.Fatalf("EnabledMap: %v", err)
	}
	for _, tc := range cases {
		v, ok := got[tc.wantKey]
		if !ok {
			t.Errorf("key %q missing from EnabledMap", tc.wantKey)
			continue
		}
		if v != tc.want {
			t.Errorf("key %q: got %v want %v (status=%s)", tc.wantKey, v, tc.want, tc.status)
		}
	}
}

func TestEnableMap_UserSettingOverride(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(42, "prov", "supported")}
	repo := &mockProviderRegistryRepo{entries: entries}
	us := &mockProviderUserSettingsRepo{
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 42, Enabled: false}},
	}
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{}, us)

	got, err := svc.EnabledMap(context.Background())
	if err != nil {
		t.Fatalf("EnabledMap: %v", err)
	}
	if got["prov"] {
		t.Error("EnabledMap: prov should be false (user setting overrides supported default)")
	}
}

func TestEnableMap_ClampedForUnsupported(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "prov", "unsupported")}
	repo := &mockProviderRegistryRepo{entries: entries}
	us := &mockProviderUserSettingsRepo{
		// stale enabled=true stored, but status=unsupported must clamp to false
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 1, Enabled: true}},
	}
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{}, us)

	got, err := svc.EnabledMap(context.Background())
	if err != nil {
		t.Fatalf("EnabledMap: %v", err)
	}
	if got["prov"] {
		t.Error("EnabledMap: prov must be false for unsupported even with stale enabled=true")
	}
}

func TestEnableMap_DBError(t *testing.T) {
	repo := &mockProviderRegistryRepo{err: errors.New("db gone")}
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{}, &mockProviderUserSettingsRepo{})

	_, err := svc.EnabledMap(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
}

// -- Regression: unsupported provider detected when folder present --

// TestScanProjectInternal_UnsupportedProvider_Detected verifies that a provider
// with status='unsupported' (e.g. codex, gemini) is now detected during project scan
// when its detect folder is present on disk. Previously, the enablement gate forced
// enabled=false for unsupported providers and skipped the adapter before Detect ran.
func TestScanProjectInternal_UnsupportedProvider_Detected(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	adapter := &mockAdapter{
		key: "codex",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.codex",
			SkillsPath:      "/tmp/p/.codex/skills",
			DetectionStatus: domain.DetectionStatusDetected,
		},
	}
	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{adapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"codex": {ID: 1, Key: "codex", Status: domain.ProviderStatusUnsupported},
		},
	}
	scanRepo := &mockProjectScanCommitter{}

	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
	if len(scanRepo.lastProviders) != 1 {
		t.Errorf("committed providers: got %d want 1 (codex must be detected regardless of status)", len(scanRepo.lastProviders))
	}
	if len(scanRepo.lastProviders) == 1 {
		p := scanRepo.lastProviders[0]
		if p.ProviderDefinitionID != 1 {
			t.Errorf("ProviderDefinitionID: got %d want 1", p.ProviderDefinitionID)
		}
		if p.DetectionStatus != domain.DetectionStatusDetected {
			t.Errorf("DetectionStatus: got %q want detected", p.DetectionStatus)
		}
	}
}
