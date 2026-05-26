package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ---- effective state resolution tests (pure logic, no DB needed) ----

func makeEntry(scanID int64, plugin, marketplace string, enabled bool) domain.PluginEntry {
	decl := domain.PluginDeclarationEnabled
	if !enabled {
		decl = domain.PluginDeclarationDisabled
	}
	return domain.PluginEntry{ID: 1, LayerScanID: scanID, PluginName: plugin, MarketplaceName: marketplace, Declaration: decl}
}

func okScan(id int64, layer domain.PluginSettingsLayer, projectID *int64) *domain.PluginLayerScan {
	return &domain.PluginLayerScan{ID: id, SettingsLayer: layer, ScanStatus: domain.PluginLayerScanOK, ProjectID: projectID}
}

func badScan(id int64, layer domain.PluginSettingsLayer, status domain.PluginLayerScanStatus, projectID *int64) *domain.PluginLayerScan {
	return &domain.PluginLayerScan{ID: id, SettingsLayer: layer, ScanStatus: status, ProjectID: projectID}
}

func TestResolveEffectivePlugin_LocalEnabledWins(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {makeEntry(1, "plugin-a", "npm", true)},  // local: enabled
		2: {makeEntry(2, "plugin-a", "npm", false)}, // project: disabled (should not win)
		3: {},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("expected enabled (local wins), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerLocal {
		t.Errorf("expected provenance=local, got %v", result.ProvenanceLayer)
	}
}

func TestResolveEffectivePlugin_LocalDisabledOverridesProjectEnabled(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))

	entryMap := map[int64][]domain.PluginEntry{
		1: {makeEntry(1, "plugin-a", "npm", false)}, // local: disabled
		2: {makeEntry(2, "plugin-a", "npm", true)},  // project: enabled
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, nil, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveDisabled {
		t.Errorf("expected disabled (local overrides project), got %s", result.EffectiveStatus)
	}
}

func TestResolveEffectivePlugin_UnknownLocalBlocksInheritance(t *testing.T) {
	local := badScan(1, domain.PluginLayerLocal, domain.PluginLayerScanMalformed, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},
		2: {makeEntry(2, "plugin-a", "npm", true)},
		3: {makeEntry(3, "plugin-a", "npm", true)},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveUnknown {
		t.Errorf("expected unknown (bad local blocks), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer != nil {
		t.Errorf("expected no provenance when unknown, got %v", result.ProvenanceLayer)
	}
}

func TestResolveEffectivePlugin_UnknownProjectBlocksUserInheritance(t *testing.T) {
	project := badScan(2, domain.PluginLayerProject, domain.PluginLayerScanUnreadable, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		2: {},
		3: {makeEntry(3, "plugin-a", "npm", true)},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", nil, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveUnknown {
		t.Errorf("expected unknown (bad project blocks user), got %s", result.EffectiveStatus)
	}
}

// ---- missing-status tests (Issue 1 fix) ----

// Missing local layer must not block project/user inheritance.
func TestResolveEffectivePlugin_MissingLocalDoesNotBlock(t *testing.T) {
	local := badScan(1, domain.PluginLayerLocal, domain.PluginLayerScanMissing, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},
		2: {makeEntry(2, "plugin-a", "npm", true)},
		3: {},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("missing local must not block; expected enabled (project), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerProject {
		t.Errorf("provenance: expected project, got %v", result.ProvenanceLayer)
	}
}

// Missing project layer must not block user inheritance.
func TestResolveEffectivePlugin_MissingProjectDoesNotBlock(t *testing.T) {
	project := badScan(2, domain.PluginLayerProject, domain.PluginLayerScanMissing, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		2: {},
		3: {makeEntry(3, "plugin-a", "npm", false)},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", nil, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveDisabled {
		t.Errorf("missing project must not block; expected disabled (user), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerUser {
		t.Errorf("provenance: expected user, got %v", result.ProvenanceLayer)
	}
}

// Missing all layers → absent (not unknown).
func TestResolveEffectivePlugin_AllMissing_IsAbsent(t *testing.T) {
	local := badScan(1, domain.PluginLayerLocal, domain.PluginLayerScanMissing, ptr64(10))
	project := badScan(2, domain.PluginLayerProject, domain.PluginLayerScanMissing, ptr64(10))
	user := badScan(3, domain.PluginLayerUser, domain.PluginLayerScanMissing, nil)

	entryMap := map[int64][]domain.PluginEntry{}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveAbsent {
		t.Errorf("all missing → absent, got %s", result.EffectiveStatus)
	}
}

// Malformed local (non-missing error) must still block.
func TestResolveEffectivePlugin_MalformedLocalStillBlocks(t *testing.T) {
	local := badScan(1, domain.PluginLayerLocal, domain.PluginLayerScanMalformed, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},
		2: {makeEntry(2, "plugin-a", "npm", true)},
		3: {makeEntry(3, "plugin-a", "npm", true)},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveUnknown {
		t.Errorf("malformed local must block; expected unknown, got %s", result.EffectiveStatus)
	}
}

// Missing local + missing project + ok user with entry → falls all the way to user.
func TestResolveEffectivePlugin_MissingLocal_MissingProject_UserEnabled(t *testing.T) {
	local := badScan(1, domain.PluginLayerLocal, domain.PluginLayerScanMissing, ptr64(10))
	project := badScan(2, domain.PluginLayerProject, domain.PluginLayerScanMissing, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		3: {makeEntry(3, "plugin-a", "npm", true)},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("expected enabled (user), got %s", result.EffectiveStatus)
	}
}

// ---- sanitizeWarnings tests (Issue 2 fix) ----

func TestSanitizeWarnings_Empty(t *testing.T) {
	if got := sanitizeWarnings(nil); got != nil {
		t.Errorf("nil input: expected nil, got %v", got)
	}
	if got := sanitizeWarnings([]string{}); got != nil {
		t.Errorf("empty input: expected nil, got %v", got)
	}
}

func TestSanitizeWarnings_CapsAt20(t *testing.T) {
	in := make([]string, 30)
	for i := range in {
		in[i] = "warning"
	}
	got := sanitizeWarnings(in)
	if len(got) != maxScanWarnings {
		t.Errorf("cap: got %d want %d", len(got), maxScanWarnings)
	}
}

func TestSanitizeWarnings_TruncatesLongStrings(t *testing.T) {
	long := make([]byte, maxScanWarningLen+100)
	for i := range long {
		long[i] = 'x'
	}
	got := sanitizeWarnings([]string{string(long)})
	if len(got[0]) != maxScanWarningLen {
		t.Errorf("truncation: got len %d want %d", len(got[0]), maxScanWarningLen)
	}
}

func TestResolveEffectivePlugin_AbsentAcrossAllLayers(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{1: {}, 2: {}, 3: {}}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveAbsent {
		t.Errorf("expected absent (not in any layer), got %s", result.EffectiveStatus)
	}
}

func TestResolveEffectivePlugin_FallsThrough_LocalAbsent_ProjectEnabled(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))

	entryMap := map[int64][]domain.PluginEntry{
		1: {},                                       // local: absent
		2: {makeEntry(2, "plugin-a", "npm", true)}, // project: enabled
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, nil, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("expected enabled (local absent, falls through to project), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerProject {
		t.Errorf("expected provenance=project, got %v", result.ProvenanceLayer)
	}
}

func TestResolveEffectivePlugin_LayerBreakdownPopulated(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},                                        // local: absent
		2: {},                                        // project: absent
		3: {makeEntry(3, "plugin-a", "npm", false)}, // user: disabled
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveDisabled {
		t.Errorf("status: got %s want disabled", result.EffectiveStatus)
	}
	if len(result.LayerBreakdown) != 3 {
		t.Fatalf("breakdown count: got %d want 3", len(result.LayerBreakdown))
	}
	if result.LayerBreakdown[0].Layer != domain.PluginLayerLocal {
		t.Errorf("breakdown[0].layer: got %s want local", result.LayerBreakdown[0].Layer)
	}
	if result.LayerBreakdown[2].Declaration == nil {
		t.Error("breakdown[2].declaration: expected non-nil for user layer with entry")
	}
}

// ---- buildProjectPluginView tests ----

func TestBuildProjectPluginView_ManagedOutOfScope(t *testing.T) {
	view := buildProjectPluginView(1, "claude", nil, nil,
		map[int64][]domain.PluginEntry{},
		map[int64][]domain.PluginMarketplace{})
	if !view.ManagedOutOfScope {
		t.Error("expected ManagedOutOfScope=true")
	}
}

func TestBuildProjectPluginView_AbsentPluginsNotSurfaced(t *testing.T) {
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	entryMap := map[int64][]domain.PluginEntry{2: {}} // no plugins
	mpMap := map[int64][]domain.PluginMarketplace{}

	view := buildProjectPluginView(10, "claude", []domain.PluginLayerScan{*project}, nil, entryMap, mpMap)
	if len(view.Plugins) != 0 {
		t.Errorf("absent plugins should not be surfaced, got %d", len(view.Plugins))
	}
}

func TestBuildProjectPluginView_MarketplacesDeduped(t *testing.T) {
	pid := int64(10)
	project := okScan(2, domain.PluginLayerProject, &pid)
	local := okScan(1, domain.PluginLayerLocal, &pid)
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{1: {}, 2: {}, 3: {}}
	mpMap := map[int64][]domain.PluginMarketplace{
		1: {{LayerScanID: 1, MarketplaceName: "npm", SourceType: "github", SourceSummary: "a/b"}},
		2: {{LayerScanID: 2, MarketplaceName: "npm", SourceType: "github", SourceSummary: "a/b"}}, // duplicate
		3: {{LayerScanID: 3, MarketplaceName: "custom", SourceType: "git", SourceSummary: ""}},
	}

	view := buildProjectPluginView(10, "claude", []domain.PluginLayerScan{*project, *local}, user, entryMap, mpMap)
	if len(view.Marketplaces) != 2 {
		t.Errorf("marketplaces after dedup: got %d want 2", len(view.Marketplaces))
	}
}

// ---- mock types for service tests ----

type mockPluginDefRepo struct {
	def *domain.ProviderDefinition
	err error
}

func (m *mockPluginDefRepo) GetByKey(_ context.Context, key string) (*domain.ProviderDefinition, error) {
	return m.def, m.err
}

type mockPluginProjectRepo struct {
	project *domain.Project
	err     error
}

func (m *mockPluginProjectRepo) GetByID(_ context.Context, _ int64) (*domain.Project, error) {
	return m.project, m.err
}

func TestProviderPluginService_List_NilWhenProviderNotFound(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{def: nil}, &mockPluginProjectRepo{}, nil)
	global, projects, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if global.ProviderKey != "claude" {
		t.Errorf("providerKey: got %q want claude", global.ProviderKey)
	}
	if !global.ManagedOutOfScope {
		t.Error("expected ManagedOutOfScope=true when no provider definition")
	}
	if len(projects) != 0 {
		t.Errorf("projects: got %d want 0", len(projects))
	}
}

func TestProviderPluginService_ScanProject_ProjectNotFound(t *testing.T) {
	svc := NewProviderPluginService(nil,
		&mockPluginDefRepo{def: &domain.ProviderDefinition{ID: 1, Key: "claude"}},
		&mockPluginProjectRepo{project: nil},
		nil)
	_, err := svc.ScanProject(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *domain.AppError, got %T", err)
	}
	if appErr.Code != "validation_error" {
		t.Errorf("error code: got %q want validation_error", appErr.Code)
	}
}

func ptr64(v int64) *int64 { return &v }
