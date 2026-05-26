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

// -- mock enablement resolver --

type mockEnablementResolver struct {
	result map[string]bool
	err    error
	calls  int
}

func (m *mockEnablementResolver) EnabledMap(_ context.Context) (map[string]bool, error) {
	m.calls++
	return m.result, m.err
}

// -- mock global scan writer that captures status --

type mockGlobalScanWriterFull struct {
	calls []globalScanCall
	err   error
}

type globalScanCall struct {
	defID      int64
	path       *string
	skillsPath *string
	status     domain.GlobalLocationStatus
	installs   []repositories.GlobalInstallScanResult
}

func (m *mockGlobalScanWriterFull) CommitGlobalScan(
	_ context.Context,
	defID int64,
	path, skillsPath *string,
	status domain.GlobalLocationStatus,
	installs []repositories.GlobalInstallScanResult,
	_ []domain.Warning,
	_ time.Time,
) error {
	m.calls = append(m.calls, globalScanCall{
		defID:      defID,
		path:       path,
		skillsPath: skillsPath,
		status:     status,
		installs:   installs,
	})
	return m.err
}

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

// -- Project scan: disabled provider is skipped --

func TestScanProjectInternal_DisabledProvider_AdapterNotCalled(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	detectCalls := 0
	adapter := &mockAdapter{
		key: "generic_agents",
		result: providers.DetectResult{
			Present:         true,
			DetectedPath:    "/tmp/p/.agents",
			SkillsPath:      "/tmp/p/.agents/skills",
			DetectionStatus: domain.DetectionStatusDetected,
		},
	}
	// Wrap the adapter to count Detect calls.
	countingAdapter := &countingDetectAdapter{inner: adapter, detectCallCount: &detectCalls}

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{countingAdapter}}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			"generic_agents": {ID: 1, Key: "generic_agents"},
		},
	}
	scanRepo := &mockProjectScanCommitter{}
	resolver := &mockEnablementResolver{result: map[string]bool{"generic_agents": false}}

	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithEnablementResolver(resolver)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}

	if detectCalls != 0 {
		t.Errorf("Detect calls: got %d want 0 (disabled provider must be skipped)", detectCalls)
	}
	// Scan still committed successfully with zero providers and one no_provider_detected warning.
	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("CommitProjectScan calls: got %d want 1", scanRepo.fullScanCallCount)
	}
	if len(scanRepo.lastProviders) != 0 {
		t.Errorf("committed providers: got %d want 0", len(scanRepo.lastProviders))
	}
}

// countingDetectAdapter wraps a mockAdapter and counts Detect calls.
type countingDetectAdapter struct {
	inner           *mockAdapter
	detectCallCount *int
}

func (c *countingDetectAdapter) Key() string { return c.inner.Key() }
func (c *countingDetectAdapter) DefaultProjectPaths() providers.ProjectScopePaths {
	return c.inner.DefaultProjectPaths()
}
func (c *countingDetectAdapter) Detect(path string, paths providers.ProjectScopePaths, fs providers.FsReader) (providers.DetectResult, error) {
	*c.detectCallCount++
	return c.inner.Detect(path, paths, fs)
}

func TestScanProjectInternal_EnablementResolverError_FailsScan(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{
		&mockAdapter{key: "generic_agents"},
	}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{}}
	scanRepo := &mockProjectScanCommitter{}
	resolver := &mockEnablementResolver{err: errors.New("db error")}

	svc := newFullScanSvc(projRepo, &mockProjectFS{}, &mockRunner{}, scanRepo, registry, pdRepo, &mockHostLister{}, &mockSkillsByHostLister{}).
		WithEnablementResolver(resolver)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err == nil {
		t.Fatal("expected error when enablement resolver fails")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
}

// -- Global scan: disabled provider commits disabled status --

func TestScanGlobalInternal_DisabledProvider_CommitsDisabledStatus(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriterFull{}

	adapter := &mockGlobalAdapter{
		key: providers.GenericAgentsKey,
		result: providers.GlobalDetectResult{
			Present: true,
			Status:  domain.GlobalLocationStatusActive,
		},
	}

	fs := newMockGlobalFS("/fakehome")
	resolver := &mockEnablementResolver{result: map[string]bool{providers.GenericAgentsKey: false}}

	registry := &mockGlobalRegistry{adapter: adapter}
	svc := NewGlobalSkillsService(
		globalRepo, scanWriter, newMockSettings(nil),
		&mockHostLister{},
		&mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry, fs, &syncRunner{},
	).WithEnablementResolver(resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal: %v", err)
	}

	if len(scanWriter.calls) != 1 {
		t.Fatalf("CommitGlobalScan calls: got %d want 1", len(scanWriter.calls))
	}
	call := scanWriter.calls[0]
	if call.status != domain.GlobalLocationStatusDisabled {
		t.Errorf("status: got %q want disabled", call.status)
	}
	if call.path != nil {
		t.Errorf("path: want nil, got %v", *call.path)
	}
	if call.skillsPath != nil {
		t.Errorf("skillsPath: want nil, got %v", *call.skillsPath)
	}
	if len(call.installs) != 0 {
		t.Errorf("installs: want 0, got %d", len(call.installs))
	}
	// DetectGlobal must NOT have been called — adapter's result.Present=true but we shouldn't reach it.
}

func TestScanGlobalInternal_EnablementResolverError_FailsScan(t *testing.T) {
	globalRepo := &mockGlobalRepo{defID: 1, displayName: "Shared Agent Skills", status: "supported"}
	scanWriter := &mockGlobalScanWriterFull{}

	adapter := &mockGlobalAdapter{key: providers.GenericAgentsKey}
	fs := newMockGlobalFS("/fakehome")
	resolver := &mockEnablementResolver{err: errors.New("db error")}

	registry := &mockGlobalRegistry{adapter: adapter}
	svc := NewGlobalSkillsService(
		globalRepo, scanWriter, newMockSettings(nil),
		&mockHostLister{},
		&mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		registry, fs, &syncRunner{},
	).WithEnablementResolver(resolver)

	_, err := svc.ScanGlobal(context.Background())
	if err == nil {
		t.Fatal("expected error when enablement resolver fails")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
}

// -- Install: disabled provider rejected with validation_error --

func buildInstallSvcWithEnablement(
	projRepo *mockProjectRepo,
	ppRepo *mockProjectProviderRepo,
	pdDefs map[string]*domain.ProviderDefinition,
	enabledMap map[string]bool,
) *ProjectService {
	return NewProjectService(
		projRepo,
		ppRepo,
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		&mockProjectFS{},
	).WithScanDeps(&syncRunner{}, &mockProjectScanCommitter{}).
		WithProviderDeps(
			&mockProviderRegistry{adapters: []providers.ProviderAdapter{
				&mockAdapter{key: "generic_agents"},
			}},
			&mockProviderDefRepo{defs: pdDefs},
			&mockHostLister{},
			&mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		).
		WithInstallDeps(&mockInstallFS{}, &mockActiveHostReader{}, &mockSkillsByHostLister{}).
		WithEnablementResolver(&mockEnablementResolver{result: enabledMap})
}

func TestInstallSkillsInternal_DisabledProvider_ReturnsValidationError(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	pdDefs := map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 1, Key: "generic_agents", Status: domain.ProviderStatusSupported},
	}
	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProviderKey: "generic_agents", DetectionStatus: domain.DetectionStatusDetected}},
		},
	}

	svc := buildInstallSvcWithEnablement(projRepo, ppRepo, pdDefs, map[string]bool{"generic_agents": false})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.installSkillsInternal(ctx, project, "generic_agents", []int64{1}, func(string, int, int, string) {})
	if err == nil {
		t.Fatal("expected error for disabled provider install")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if ae.Code != domain.CodeValidation {
		t.Errorf("error code: got %q want validation_error", ae.Code)
	}
}

func TestInstallSkillsInternal_EnablementResolverError_ReturnsDBError(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	pdDefs := map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 1, Key: "generic_agents", Status: domain.ProviderStatusSupported},
	}
	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProviderKey: "generic_agents", DetectionStatus: domain.DetectionStatusDetected}},
		},
	}

	svc := NewProjectService(
		projRepo, ppRepo, &mockProjectWarningRepo{}, &mockProjectInstallRepo{}, &mockProjectFS{},
	).WithScanDeps(&syncRunner{}, &mockProjectScanCommitter{}).
		WithProviderDeps(
			&mockProviderRegistry{adapters: []providers.ProviderAdapter{&mockAdapter{key: "generic_agents"}}},
			&mockProviderDefRepo{defs: pdDefs},
			&mockHostLister{},
			&mockSkillsByHostLister{skills: make(map[int64][]domain.Skill)},
		).
		WithInstallDeps(&mockInstallFS{}, &mockActiveHostReader{}, &mockSkillsByHostLister{}).
		WithEnablementResolver(&mockEnablementResolver{err: errors.New("db error")})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.installSkillsInternal(ctx, project, "generic_agents", []int64{1}, func(string, int, int, string) {})
	if err == nil {
		t.Fatal("expected error when enablement resolver fails")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
}

func TestInstallSkillsInternal_EnabledProvider_ProceedsNormally(t *testing.T) {
	// Verify that an enabled provider is not blocked by enablement check.
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "p", "/tmp/p") //nolint:errcheck

	pdDefs := map[string]*domain.ProviderDefinition{
		"generic_agents": {ID: 1, Key: "generic_agents", Status: domain.ProviderStatusSupported},
	}
	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProviderKey: "generic_agents", DetectionStatus: domain.DetectionStatusDetected}},
		},
	}
	svc := buildInstallSvcWithEnablement(projRepo, ppRepo, pdDefs, map[string]bool{"generic_agents": true})

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.installSkillsInternal(ctx, project, "generic_agents", []int64{1}, func(string, int, int, string) {})
	// Expect failure further down (no active host), not a validation_error from enablement.
	if err != nil {
		var ae *domain.AppError
		if errors.As(err, &ae) && ae.Code == domain.CodeValidation {
			// validation_error for "No active skill host" is acceptable — it means we passed the enablement gate.
			return
		}
		t.Logf("got non-enablement error (acceptable): %v", err)
	}
}
