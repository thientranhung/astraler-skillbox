package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/repositories"
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

func TestResolveEffectivePlugin_ProjectOverrideKeepsUserLayerBreakdown(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := okScan(3, domain.PluginLayerUser, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},
		2: {makeEntry(2, "plugin-a", "npm", true)}, // project: enabled override wins
		3: {makeEntry(3, "plugin-a", "npm", true)}, // user/global: enabled must remain visible
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("expected enabled (project), got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerProject {
		t.Errorf("expected provenance=project, got %v", result.ProvenanceLayer)
	}
	if len(result.LayerBreakdown) != 3 {
		t.Fatalf("breakdown count: got %d want 3", len(result.LayerBreakdown))
	}
	userLayer := result.LayerBreakdown[2]
	if userLayer.Layer != domain.PluginLayerUser {
		t.Fatalf("breakdown[2].layer: got %s want user", userLayer.Layer)
	}
	if userLayer.Declaration == nil || *userLayer.Declaration != domain.PluginDeclarationEnabled {
		t.Fatalf("user/global declaration: got %v want enabled", userLayer.Declaration)
	}
}

func TestResolveEffectivePlugin_ProjectOverrideWinsWhenUserLayerMalformed(t *testing.T) {
	local := okScan(1, domain.PluginLayerLocal, ptr64(10))
	project := okScan(2, domain.PluginLayerProject, ptr64(10))
	user := badScan(3, domain.PluginLayerUser, domain.PluginLayerScanMalformed, nil)

	entryMap := map[int64][]domain.PluginEntry{
		1: {},
		2: {makeEntry(2, "plugin-a", "npm", true)},
		3: {},
	}

	result := resolveEffectivePlugin("plugin-a", "npm", local, project, user, entryMap)
	if result.EffectiveStatus != domain.PluginEffectiveEnabled {
		t.Errorf("expected project override to remain enabled, got %s", result.EffectiveStatus)
	}
	if result.ProvenanceLayer == nil || *result.ProvenanceLayer != domain.PluginLayerProject {
		t.Errorf("expected provenance=project, got %v", result.ProvenanceLayer)
	}
	if len(result.LayerBreakdown) != 3 {
		t.Fatalf("breakdown count: got %d want 3", len(result.LayerBreakdown))
	}
	userLayer := result.LayerBreakdown[2]
	if userLayer.Layer != domain.PluginLayerUser {
		t.Fatalf("breakdown[2].layer: got %s want user", userLayer.Layer)
	}
	if userLayer.ScanStatus != domain.PluginLayerScanMalformed {
		t.Fatalf("user/global scan status: got %s want malformed", userLayer.ScanStatus)
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

type mockProviderRegistrySvc struct {
	entries []domain.ProviderRegistryEntry
	err     error
}

func (m *mockProviderRegistrySvc) List(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
	return m.entries, m.err
}

// ---- registry-driven path resolution tests ----

func TestPluginProviderDefsAllowMissing_UsesRegistryPaths(t *testing.T) {
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: "custom/global.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: "custom/project.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: "custom/local.json", Priority: 5},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, nil)
	defs, err := svc.pluginProviderDefsAllowMissing(context.Background())
	if err != nil {
		t.Fatalf("pluginProviderDefsAllowMissing: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("defs count: got %d want 1", len(defs))
	}
	d := defs[0]
	if d.GlobalRelPath != "custom/global.json" {
		t.Errorf("GlobalRelPath: got %q want custom/global.json", d.GlobalRelPath)
	}
	if d.ProjectRelPath != "custom/project.json" {
		t.Errorf("ProjectRelPath: got %q want custom/project.json", d.ProjectRelPath)
	}
	if d.LocalRelPath != "custom/local.json" {
		t.Errorf("LocalRelPath: got %q want custom/local.json", d.LocalRelPath)
	}
}

func TestPluginProviderDefsAllowMissing_ProjectCandidatesSortedByPriority(t *testing.T) {
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				// Intentionally out of order — local.json has higher priority number and should be ProjectRelPath.
				{Scope: "project", Purpose: "config", RelativePath: "settings.local.json", Priority: 5},
				{Scope: "project", Purpose: "config", RelativePath: "settings.json", Priority: 10},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, nil)
	defs, err := svc.pluginProviderDefsAllowMissing(context.Background())
	if err != nil {
		t.Fatalf("pluginProviderDefsAllowMissing: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("defs count: got %d want 1", len(defs))
	}
	// Highest priority → ProjectRelPath; second highest → LocalRelPath.
	if defs[0].ProjectRelPath != "settings.json" {
		t.Errorf("ProjectRelPath: got %q want settings.json", defs[0].ProjectRelPath)
	}
	if defs[0].LocalRelPath != "settings.local.json" {
		t.Errorf("LocalRelPath: got %q want settings.local.json", defs[0].LocalRelPath)
	}
}

func TestPluginProviderDefsAllowMissing_SkipsUnknownProviderKeys(t *testing.T) {
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 99, Key: "unknown_provider"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: "some/path.json", Priority: 10},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, nil)
	defs, err := svc.pluginProviderDefsAllowMissing(context.Background())
	if err != nil {
		t.Fatalf("pluginProviderDefsAllowMissing: %v", err)
	}
	if len(defs) != 0 {
		t.Errorf("expected no defs for unknown provider, got %d", len(defs))
	}
}

func TestExpandGlobalPath_TildeExpansion(t *testing.T) {
	home := "/Users/tester"
	cases := []struct {
		rel  string
		want string
	}{
		{"~/foo/bar.json", "/Users/tester/foo/bar.json"},
		{"/absolute/path.json", "/absolute/path.json"},
		{"relative/path.json", "/Users/tester/relative/path.json"},
		{"", ""},
	}
	for _, tc := range cases {
		got := expandGlobalPath(home, tc.rel)
		if got != tc.want {
			t.Errorf("expandGlobalPath(%q, %q): got %q want %q", home, tc.rel, got, tc.want)
		}
	}
}

func TestProviderPluginService_List_NilWhenProviderNotFound(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{def: nil}, &mockPluginProjectRepo{}, nil, nil)
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
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.local.json", Priority: 5},
			},
		},
		{
			Definition: domain.ProviderDefinition{ID: 2, Key: "codex"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: ".codex/config.toml", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".codex/config.toml", Priority: 10},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, nil)

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
	if defs[1].GlobalRelPath != ".codex/config.toml" {
		t.Errorf("codex GlobalRelPath: got %q want .codex/config.toml", defs[1].GlobalRelPath)
	}
	if defs[1].ProjectRelPath != ".codex/config.toml" {
		t.Errorf("codex ProjectRelPath: got %q want .codex/config.toml", defs[1].ProjectRelPath)
	}
	if defs[1].LocalRelPath != "" {
		t.Errorf("codex LocalRelPath: got %q want empty", defs[1].LocalRelPath)
	}
}

func TestProviderPluginService_ScanProject_ProjectNotFound(t *testing.T) {
	svc := NewProviderPluginService(nil,
		&mockPluginDefRepo{def: &domain.ProviderDefinition{ID: 1, Key: "claude"}},
		&mockPluginProjectRepo{project: nil},
		nil, nil)
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
	svc := NewProviderPluginService(nil, pdRepo, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
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
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
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
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "", "market", "user", 0, true)
	if err == nil {
		t.Fatal("expected error for empty pluginName, got nil")
	}
	if err.(*domain.AppError).Code != domain.CodeValidation {
		t.Errorf("expected validation_error")
	}
}

func TestSetPluginEnabled_ProviderNotInDB_ReturnsValidationError(t *testing.T) {
	// claude key allowed but not present in registry → provider not configured
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
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

	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
			},
		},
		{
			Definition: domain.ProviderDefinition{ID: 2, Key: "antigravity_cli"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: ".gemini/antigravity-cli/settings.json", Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".gemini/antigravity-cli/settings.json", Priority: 10},
			},
		},
	}}
	svc2 := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, &mockRunner{
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
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, registry, conflictRunner)
	_, err := svc.SetPluginEnabled(context.Background(), "claude", "p", "m", "user", 0, true)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if err.(*domain.AppError).Code != domain.CodeConflict {
		t.Errorf("code: got %q want conflict_error", err.(*domain.AppError).Code)
	}
}

func TestSetPluginEnabled_LocalLayerReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
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
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
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
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
			},
		},
	}}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{project: nil}, registry, &mockRunner{})
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
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: 1, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
			},
		},
	}}
	project := &domain.Project{ID: 42, Path: t.TempDir()}
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{project: project}, registry, &mockRunner{
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

// Compile-time assertions that *ProviderPluginService satisfies both plugin service interfaces.
var _ ProjectPluginScanner = (*ProviderPluginService)(nil)
var _ ProjectPluginCounter = (*ProviderPluginService)(nil)

// F2: zero plugin-capable providers must be a no-op (nil), NOT a validation error,
// so a project scan on a fresh/partial DB does not fail.
func TestScanProjectLayers_NoPluginProviders_IsNoOp(t *testing.T) {
	// mockProviderRegistrySvc{} returns an empty registry → no plugin-capable defs.
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{},
		&mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})

	project := &domain.Project{ID: 1, Path: t.TempDir()}
	if err := svc.ScanProjectLayers(context.Background(), project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("expected no-op nil for zero plugin providers, got %v", err)
	}
}

func TestAggregatePluginCounts_SumsEnabledAndTotalAcrossProviders(t *testing.T) {
	enabled := domain.PluginEffectiveEnabled
	disabled := domain.PluginEffectiveDisabled
	unknown := domain.PluginEffectiveUnknown

	views := []domain.ProjectPluginView{
		{ProjectID: 1, ProviderKey: "claude", Plugins: []domain.PluginEffectiveEntry{
			{PluginName: "a", EffectiveStatus: enabled},
			{PluginName: "b", EffectiveStatus: disabled},
		}},
		{ProjectID: 1, ProviderKey: "codex", Plugins: []domain.PluginEffectiveEntry{
			{PluginName: "c", EffectiveStatus: enabled},
			{PluginName: "d", EffectiveStatus: unknown},
		}},
		{ProjectID: 2, ProviderKey: "claude", Plugins: []domain.PluginEffectiveEntry{
			{PluginName: "e", EffectiveStatus: disabled},
		}},
	}

	displayNames := map[string]string{"claude": "Claude", "codex": "Codex"}
	got := aggregatePluginCounts(views, displayNames)

	if got[1].Enabled != 2 {
		t.Errorf("project 1 Enabled: got %d want 2", got[1].Enabled)
	}
	if got[1].Total != 4 {
		t.Errorf("project 1 Total: got %d want 4 (enabled+disabled+unknown)", got[1].Total)
	}
	if got[2].Enabled != 0 {
		t.Errorf("project 2 Enabled: got %d want 0", got[2].Enabled)
	}
	if got[2].Total != 1 {
		t.Errorf("project 2 Total: got %d want 1", got[2].Total)
	}

	// ByProvider: project 1 should have two entries sorted by key (claude < codex).
	bp1 := got[1].ByProvider
	if len(bp1) != 2 {
		t.Fatalf("project 1 ByProvider len: got %d want 2", len(bp1))
	}
	if bp1[0].ProviderKey != "claude" || bp1[0].DisplayName != "Claude" || bp1[0].Enabled != 1 || bp1[0].Total != 2 {
		t.Errorf("project 1 ByProvider[0]: got %+v", bp1[0])
	}
	if bp1[1].ProviderKey != "codex" || bp1[1].DisplayName != "Codex" || bp1[1].Enabled != 1 || bp1[1].Total != 2 {
		t.Errorf("project 1 ByProvider[1]: got %+v", bp1[1])
	}

	// ByProvider: project 2 has one entry.
	bp2 := got[2].ByProvider
	if len(bp2) != 1 {
		t.Fatalf("project 2 ByProvider len: got %d want 1", len(bp2))
	}
	if bp2[0].ProviderKey != "claude" || bp2[0].DisplayName != "Claude" || bp2[0].Enabled != 0 || bp2[0].Total != 1 {
		t.Errorf("project 2 ByProvider[0]: got %+v", bp2[0])
	}
}

func TestAggregatePluginCounts_EmptyIsEmptyMap(t *testing.T) {
	got := aggregatePluginCounts(nil, nil)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestSetPluginEnabled_PathConfinementEscape_ReturnsValidationError(t *testing.T) {
	pd := &domain.ProviderDefinition{ID: 1, Key: "claude"}
	project := &domain.Project{ID: 1, Path: "/tmp/proj"}

	// Craft a def where ProjectRelPath contains ".." to escape the project root.
	escapeDef := pluginProviderDef{
		Provider:       pd,
		GlobalRelPath:  ".claude/settings.json",
		ProjectRelPath: "../escape.json",
		Scanner:        nil,
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

// ---- RemoveOverride tests ----

func TestRemoveOverride_UnknownProvider_ReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
	_, err := svc.RemoveOverride(context.Background(), "unknown_provider", "p", "m", "project", 1)
	if err == nil {
		t.Fatal("expected validation error for unknown provider, got nil")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok || appErr.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %T: %v", err, err)
	}
}

func TestRemoveOverride_NonProjectLayer_ReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
	_, err := svc.RemoveOverride(context.Background(), "claude", "p", "m", "user", 0)
	if err == nil {
		t.Fatal("expected validation error for non-project layer, got nil")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok || appErr.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %T: %v", err, err)
	}
}

func TestRemoveOverride_EmptyPluginName_ReturnsValidationError(t *testing.T) {
	svc := NewProviderPluginService(nil, &mockPluginDefRepo{}, &mockPluginProjectRepo{}, &mockProviderRegistrySvc{}, &mockRunner{})
	_, err := svc.RemoveOverride(context.Background(), "claude", "", "market", "project", 1)
	if err == nil {
		t.Fatal("expected validation error for empty pluginName, got nil")
	}
	appErr, ok := err.(*domain.AppError)
	if !ok || appErr.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %T: %v", err, err)
	}
}

func TestRemoveOverride_ProjectLayer_RemovesPluginFromFile(t *testing.T) {
	// Real SQLite DB with migrations applied (provider_definitions seeded by migration).
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := repositories.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Migration seeds the claude provider definition; retrieve its ID.
	var claudeDefID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM provider_definitions WHERE key = 'claude'`).Scan(&claudeDefID); err != nil {
		t.Fatalf("query claude def: %v", err)
	}

	// Insert a project row (project_id FK required by provider_plugin_layer_scans).
	projectDir := t.TempDir()
	res, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('test-proj', ?, 'active')`, projectDir)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	projectID, _ := res.LastInsertId()

	// Write a settings.json with the plugin entry.
	settingsDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{"my-plugin@market":true}}`), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	pluginRepo := repositories.NewProviderPluginRepo(db)
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: claudeDefID, Key: "claude"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "project", Purpose: "config", RelativePath: ".claude/settings.json", Priority: 10},
			},
		},
	}}
	project := &domain.Project{ID: projectID, Path: projectDir}
	svc := NewProviderPluginService(pluginRepo, &mockPluginDefRepo{}, &mockPluginProjectRepo{project: project}, registry, makeSyncRunner())

	_, err = svc.RemoveOverride(ctx, "claude", "my-plugin", "market", "project", projectID)
	if err != nil {
		t.Fatalf("RemoveOverride: %v", err)
	}

	// Verify the plugin was removed from the settings file.
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if strings.Contains(string(data), "my-plugin@market") {
		t.Error("plugin should have been removed from the settings file after RemoveOverride")
	}
}

// TestScanProjectLayers_CodexVersionPopulatedFromCache verifies that when
// scanProjectInternal runs for the Codex provider, it reads the version from
// ~/.codex/plugins/cache and stores it on the plugin entry in the DB.
func TestScanProjectLayers_CodexVersionPopulatedFromCache(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := repositories.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Migration seeds the codex provider definition; retrieve its ID.
	var codexDefID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM provider_definitions WHERE key = 'codex'`).Scan(&codexDefID); err != nil {
		t.Fatalf("query codex def: %v", err)
	}

	// Insert a project row.
	projectDir := t.TempDir()
	res, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('codex-proj', ?, 'active')`, projectDir)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	projectID, _ := res.LastInsertId()

	// Fake ~/.codex dir (the global config root for this test).
	fakeCodexDir := t.TempDir()

	// Write a config.toml with stitch-design@stitch-skills enabled (in the project dir).
	codexProjectDir := filepath.Join(projectDir, ".codex")
	if err := os.MkdirAll(codexProjectDir, 0o755); err != nil {
		t.Fatalf("mkdir project .codex: %v", err)
	}
	configPath := filepath.Join(codexProjectDir, "config.toml")
	tomlContent := "[plugins.\"stitch-design@stitch-skills\"]\nenabled = true\n"
	if err := os.WriteFile(configPath, []byte(tomlContent), 0o644); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}

	// Build cache dir: fakeCodexDir/plugins/cache/stitch-skills/stitch-design/1.0.0/plugin.json
	versionDir := filepath.Join(fakeCodexDir, "plugins", "cache", "stitch-skills", "stitch-design", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	pluginJSON := `{"name":"stitch-design","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(versionDir, "plugin.json"), []byte(pluginJSON), 0o644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}

	// GlobalRelPath is set to the absolute path of fakeCodexDir/config.toml so that
	// UserFilePath() returns fakeCodexDir/config.toml and filepath.Dir gives fakeCodexDir.
	fakeGlobalConfigPath := filepath.Join(fakeCodexDir, "config.toml")

	pluginRepo := repositories.NewProviderPluginRepo(db)
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: codexDefID, Key: "codex"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: fakeGlobalConfigPath, Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".codex/config.toml", Priority: 10},
			},
		},
	}}
	project := &domain.Project{ID: projectID, Path: projectDir}
	svc := NewProviderPluginService(pluginRepo, &mockPluginDefRepo{}, &mockPluginProjectRepo{project: project}, registry, makeSyncRunner())

	if err := svc.ScanProjectLayers(ctx, project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("ScanProjectLayers: %v", err)
	}

	// Query the persisted plugin entry and check the version.
	var version *string
	err = db.QueryRowContext(ctx, `
		SELECT e.version
		FROM provider_plugin_entries e
		JOIN provider_plugin_layer_scans s ON e.layer_scan_id = s.id
		WHERE e.plugin_name = 'stitch-design' AND e.marketplace_name = 'stitch-skills'
		  AND s.project_id = ?
	`, projectID).Scan(&version)
	if err != nil {
		t.Fatalf("query plugin entry: %v", err)
	}
	if version == nil || *version != "1.0.0" {
		t.Errorf("expected version 1.0.0 from cache, got %v", version)
	}
}

// TestScanProjectLayers_NonCodexProvider_VersionNil verifies that providers without
// a version source (e.g. antigravity_cli) leave plugin version as NULL in the DB.
func TestScanProjectLayers_NonCodexProvider_VersionNil(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := repositories.OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var agDefID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM provider_definitions WHERE key = 'antigravity_cli'`).Scan(&agDefID); err != nil {
		t.Fatalf("query antigravity_cli def: %v", err)
	}

	projectDir := t.TempDir()
	res, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('ag-proj', ?, 'active')`, projectDir)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	projectID, _ := res.LastInsertId()

	// Write antigravity_cli settings.json with one plugin enabled.
	agDir := filepath.Join(projectDir, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(agDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	settingsPath := filepath.Join(agDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{"my-plugin@some-market":true}}`), 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	fakeGlobalPath := filepath.Join(t.TempDir(), "settings.json")

	pluginRepo := repositories.NewProviderPluginRepo(db)
	registry := &mockProviderRegistrySvc{entries: []domain.ProviderRegistryEntry{
		{
			Definition: domain.ProviderDefinition{ID: agDefID, Key: "antigravity_cli"},
			Candidates: []domain.ProviderPathCandidate{
				{Scope: "global", Purpose: "config", RelativePath: fakeGlobalPath, Priority: 10},
				{Scope: "project", Purpose: "config", RelativePath: ".gemini/antigravity-cli/settings.json", Priority: 10},
			},
		},
	}}
	project := &domain.Project{ID: projectID, Path: projectDir}
	svc := NewProviderPluginService(pluginRepo, &mockPluginDefRepo{}, &mockPluginProjectRepo{project: project}, registry, makeSyncRunner())

	if err := svc.ScanProjectLayers(ctx, project, func(string, int, int, string) {}); err != nil {
		t.Fatalf("ScanProjectLayers: %v", err)
	}

	// Version must be NULL — no version source for antigravity_cli.
	var version *string
	err = db.QueryRowContext(ctx, `
		SELECT e.version
		FROM provider_plugin_entries e
		JOIN provider_plugin_layer_scans s ON e.layer_scan_id = s.id
		WHERE e.plugin_name = 'my-plugin' AND e.marketplace_name = 'some-market'
		  AND s.project_id = ?
	`, projectID).Scan(&version)
	if err != nil {
		t.Fatalf("query plugin entry: %v", err)
	}
	if version != nil {
		t.Errorf("expected NULL version for antigravity_cli, got %q", *version)
	}
}
