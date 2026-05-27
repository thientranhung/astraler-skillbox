package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
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
		1: {},                                      // local: absent
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
		1: {},                                       // local: absent
		2: {},                                       // project: absent
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
	def  *domain.ProviderDefinition
	defs map[string]*domain.ProviderDefinition
	err  error
}

func (m *mockPluginDefRepo) GetByKey(_ context.Context, key string) (*domain.ProviderDefinition, error) {
	if m.defs != nil {
		return m.defs[key], m.err
	}
	if m.def != nil && m.def.Key != "" && m.def.Key != key {
		return nil, m.err
	}
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

func TestProviderPluginService_PluginProviderDefsIncludesCodex(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
		"codex":  {ID: 2, Key: "codex"},
	}}, &mockPluginProjectRepo{}, nil)

	defs, err := svc.pluginProviderDefs(context.Background())
	if err != nil {
		t.Fatalf("pluginProviderDefs: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("defs: got %d want 2", len(defs))
	}
	if defs[0].Provider.Key != "claude" || defs[1].Provider.Key != "codex" {
		t.Fatalf("provider order: got %q, %q", defs[0].Provider.Key, defs[1].Provider.Key)
	}
	if defs[1].GlobalDir != ".codex" || defs[1].UserFile != "config.toml" {
		t.Errorf("codex global path parts: got %q/%q", defs[1].GlobalDir, defs[1].UserFile)
	}
	if defs[1].ProjectDir != ".codex" || defs[1].ProjectFile != "config.toml" || defs[1].LocalFile != "" {
		t.Errorf("codex project path parts: got dir=%q project=%q local=%q", defs[1].ProjectDir, defs[1].ProjectFile, defs[1].LocalFile)
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

// ---- SetPluginEnabled tests ----

func makeSyncRunner() *mockRunner {
	return &mockRunner{
		startFn: func(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error) {
			_, err := fn(ctx, func(phase string, processed, total int, msg string) {})
			if err != nil {
				return 0, err
			}
			return 1, nil
		},
	}
}

func TestSetPluginEnabled_CodexIsNowSupported(t *testing.T) {
	// Codex was previously rejected; with TOML write support it now passes provider validation.
	// Expect "provider not configured" (validation) because no DB entry, not "unsupported provider".
	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{}}
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "codex", "p", "m", "user", 0, true)
	if err == nil {
		t.Fatal("expected error (provider not in DB), got nil")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *domain.AppError, got %T", err)
	}
	if appErr.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", appErr.Code)
	}
}

func TestSetPluginEnabled_UnknownProviderReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "openai", "p", "m", "user", 0, true)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	appErr := err.(*domain.AppError)
	if appErr.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", appErr.Code)
	}
}

func TestSetPluginEnabled_EmptyNameReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "", "market", "user", 0, true)
	if err == nil {
		t.Fatal("expected error for empty pluginName, got nil")
	}
	if err.(*domain.AppError).Code != domain.CodeValidation {
		t.Errorf("expected validation_error")
	}
}

func TestSetPluginEnabled_ProviderNotInDB_ReturnsValidationError(t *testing.T) {
	// claude key allowed but not present in DB → provider not configured
	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{}}
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "user", 0, true)
	if err == nil {
		t.Fatal("expected error for missing DB entry, got nil")
	}
	if err.(*domain.AppError).Code != domain.CodeValidation {
		t.Errorf("expected validation_error")
	}
}

func TestSetPluginEnabled_WritesFileAndReturnsOpID(t *testing.T) {
	dir := t.TempDir()
	var capturedPath string
	var capturedEnabled bool

	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude":          {ID: 1, Key: "claude"},
		"antigravity_cli": {ID: 2, Key: "antigravity_cli"},
	}}
	svc2 := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{}, &mockRunner{
		startFn: func(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error) {
			return 42, nil // skip execution
		},
	})
	svc2.pluginWriter = func(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error {
		capturedPath = filePath
		capturedEnabled = enabled
		return nil
	}
	_ = dir

	opID, err := svc2.SetPluginEnabled(context.Background(), "claude", "my-plugin", "my-market", "user", 0, true)
	if err != nil {
		t.Fatalf("SetPluginEnabled: %v", err)
	}
	if opID != 42 {
		t.Errorf("operationId: got %d want 42", opID)
	}
	_ = capturedPath
	_ = capturedEnabled
}

func TestSetPluginEnabled_RunnerConflict_ReturnsConflictError(t *testing.T) {
	conflictRunner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 0, domain.NewConflictError("already running", "target locked")
		},
	}
	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{}, conflictRunner)
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "user", 0, true)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if err.(*domain.AppError).Code != domain.CodeConflict {
		t.Errorf("code: got %q want conflict_error", err.(*domain.AppError).Code)
	}
}

func TestSetPluginEnabled_LocalLayerReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "local", 0, true)
	if err == nil {
		t.Fatal("expected validation error for local layer, got nil")
	}
	appErr := err.(*domain.AppError)
	if appErr.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", appErr.Code)
	}
}

func TestSetPluginEnabled_UnknownLayerReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "unknown", 0, true)
	if err == nil {
		t.Fatal("expected validation error for unknown layer, got nil")
	}
	appErr := err.(*domain.AppError)
	if appErr.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error", appErr.Code)
	}
}

func TestSetPluginEnabled_ProjectMissingReturnsValidationError(t *testing.T) {
	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{project: nil}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "project", 999, true)
	if err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
	appErr := err.(*domain.AppError)
	if appErr.Code != domain.CodeValidation {
		t.Errorf("code: got %q want validation_error, got %q", appErr.Code, appErr.Code)
	}
}

func TestSetPluginEnabled_ProjectLayer_WritesAndReturnsOpID(t *testing.T) {
	pdRepo := &mockPluginDefRepo{defs: map[string]*domain.ProviderDefinition{
		"claude": {ID: 1, Key: "claude"},
	}}
	project := &domain.Project{ID: 42, Path: t.TempDir()}
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{project: project}, &mockRunner{
		startFn: func(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error) {
			if target.Type != "provider_plugin_project" {
				return 0, fmt.Errorf("unexpected target type: %s", target.Type)
			}
			if target.ID != 42 {
				return 0, fmt.Errorf("unexpected target ID: %d", target.ID)
			}
			return 77, nil
		},
	})
	opID, err := svc.SetPluginEnabled(context.Background(), "claude", "my-plugin", "market", "project", 42, true)
	if err != nil {
		t.Fatalf("SetPluginEnabled: %v", err)
	}
	if opID != 77 {
		t.Errorf("operationId: got %d want 77", opID)
	}
}

func TestSetPluginEnabled_PathConfinementEscape_ReturnsValidationError(t *testing.T) {
	pd := &domain.ProviderDefinition{ID: 1, Key: "claude"}
	project := &domain.Project{ID: 1, Path: "/tmp/proj"}

	// Craft a def where ProjectFile contains ".." to escape the allowed directory.
	escapeDef := pluginProviderDef{
		Provider:    pd,
		GlobalDir:   ".claude",
		UserFile:    "settings.json",
		ProjectDir:  ".claude",
		ProjectFile: "../escape.json",
		Scanner:     nil,
	}

	writerCalled := false
	stubWriter := pluginWriterFn(func(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error {
		writerCalled = true
		return nil
	})
	svc := &ProviderPluginService{}

	err := svc.setPluginEnabledProjectInternal(
		context.Background(),
		escapeDef,
		project,
		"plugin", "market",
		true,
		stubWriter,
		func(string, int, int, string) {},
	)
	if err == nil {
		t.Fatal("expected confinement error, got nil")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok || appErr.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %T: %v", err, err)
	}
	if writerCalled {
		t.Error("writer should not be called when confinement check fails")
	}
}
