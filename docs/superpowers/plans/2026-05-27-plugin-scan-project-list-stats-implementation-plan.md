# Plugin Scan in Project Scan + Project-List Plugin Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `project.scan` also scan provider plugins (project + local layers) in one operation, and show an enabled/total Plugins column in the Projects list.

**Architecture:** Plugin scanning is folded into the existing project-scan operation by extracting a `ScanProjectLayers` method on `ProviderPluginService` that runs inside the caller's operation context (no nested operation lock). `ProjectService` gains two optional deps (a plugin scanner and a plugin counter) via a new `WithPluginDeps`. List stats reuse the tested Go effective-resolution (`ListAll` → pure aggregation) rather than SQL. The separate "Scan Plugins" button is removed; the project-detail plugin section refreshes when the unified scan completes.

**Tech Stack:** Go (modernc.org/sqlite, creachadair/jrpc2), React + TanStack Query, JSON-Schema contracts (generated TS), `go test`, Vitest + React Testing Library.

**Spec:** `docs/superpowers/specs/2026-05-27-plugin-scan-project-list-stats-design.md`

---

## File Structure

Backend (Go):
- `core-go/internal/domain/provider_plugin.go` — add `PluginCount` type.
- `core-go/internal/services/provider_plugin_service.go` — add `aggregatePluginCounts` (pure), `PluginCountsByProject`, `ScanProjectLayers`.
- `core-go/internal/services/project_service.go` — add `ProjectPluginScanner` + `ProjectPluginCounter` interfaces, `WithPluginDeps`, two `ProjectListItem` fields, wire `ListProjects` and `scanProjectInternal`.
- `core-go/internal/rpc/handlers/project_list.go` — map two new response fields.
- `core-go/cmd/skillbox-core/main.go` — call `projectSvc.WithPluginDeps(providerPluginSvc, providerPluginSvc)`.

Contract:
- `shared/api-contracts/methods/project.list.json` — add `pluginEnabledCount` + `pluginTotalCount`.
- `shared/generated/methods/project-list.ts` — regenerated (committed).

Frontend (React):
- `apps/desktop/renderer/src/screens/projects-screen.tsx` — add Plugins column header.
- `apps/desktop/renderer/src/features/projects/project-row.tsx` — add Plugins cell.
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — remove Scan Plugins button + hook usage.
- `apps/desktop/renderer/src/features/projects/use-scan-project.ts` — invalidate `providerPlugins.list` on scan success.

Tests:
- `core-go/internal/services/provider_plugin_service_test.go` — aggregation unit tests.
- `core-go/internal/services/project_service_test.go` — ListProjects with counter.
- `core-go/internal/services/project_scan_full_service_test.go` — scan invokes plugin scanner.
- `core-go/internal/rpc/handlers/project_handler_test.go` — handler maps new fields.
- `apps/desktop/renderer/src/features/projects/__tests__/project-row.test.tsx` — new.
- `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx` — update mocks/assertions.

---

## Task 1: PluginCount type + pure aggregation + PluginCountsByProject

**Files:**
- Modify: `core-go/internal/domain/provider_plugin.go` (append new type)
- Modify: `core-go/internal/services/provider_plugin_service.go` (add helper + method)
- Test: `core-go/internal/services/provider_plugin_service_test.go` (append tests)

- [ ] **Step 1: Write the failing test**

Append to `core-go/internal/services/provider_plugin_service_test.go`:

```go
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

	got := aggregatePluginCounts(views)

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
}

func TestAggregatePluginCounts_EmptyIsEmptyMap(t *testing.T) {
	got := aggregatePluginCounts(nil)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd core-go && go test ./internal/services/ -run TestAggregatePluginCounts -v`
Expected: compile error — `undefined: aggregatePluginCounts` and `undefined: domain.PluginCount`.

- [ ] **Step 3: Add the domain type**

Append to `core-go/internal/domain/provider_plugin.go`:

```go
// PluginCount is the per-project aggregate of effective plugins across all providers.
// Total counts effective entries that are not absent (enabled + disabled + unknown).
type PluginCount struct {
	Enabled int
	Total   int
}
```

- [ ] **Step 4: Add the helper and method**

In `core-go/internal/services/provider_plugin_service.go`, add (place near the other effective-state helpers, e.g. after `buildProjectPluginView`):

```go
// aggregatePluginCounts sums effective plugin counts per project across all providers.
// ListAll yields one ProjectPluginView per (provider, project), so a project with several
// plugin-capable providers produces several views sharing the same ProjectID; accumulating
// into counts[pv.ProjectID] across them is intentional — the column shows one project-wide
// enabled/total summed over all providers. Each view's Plugins already excludes absent
// entries, so len(Plugins) is the per-view non-absent total.
func aggregatePluginCounts(projects []domain.ProjectPluginView) map[int64]domain.PluginCount {
	counts := make(map[int64]domain.PluginCount)
	for _, pv := range projects {
		c := counts[pv.ProjectID]
		for _, p := range pv.Plugins {
			c.Total++
			if p.EffectiveStatus == domain.PluginEffectiveEnabled {
				c.Enabled++
			}
		}
		counts[pv.ProjectID] = c
	}
	return counts
}

// PluginCountsByProject returns per-project effective plugin counts (enabled/total)
// across all plugin-capable providers, derived from persisted scan data.
func (s *ProviderPluginService) PluginCountsByProject(ctx context.Context) (map[int64]domain.PluginCount, error) {
	_, projects, err := s.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return aggregatePluginCounts(projects), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/services/ -run TestAggregatePluginCounts -v`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add core-go/internal/domain/provider_plugin.go core-go/internal/services/provider_plugin_service.go core-go/internal/services/provider_plugin_service_test.go
git commit -m "Add PluginCount aggregation to ProviderPluginService"
```

---

## Task 2: ProjectService plugin deps + list stats

**Files:**
- Modify: `core-go/internal/services/project_service.go`
- Test: `core-go/internal/services/project_service_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `core-go/internal/services/project_service_test.go`:

```go
type fakePluginCounter struct {
	counts map[int64]domain.PluginCount
	err    error
}

func (f *fakePluginCounter) PluginCountsByProject(_ context.Context) (map[int64]domain.PluginCount, error) {
	return f.counts, f.err
}

func TestListProjects_PopulatesPluginCounts(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	counter := &fakePluginCounter{counts: map[int64]domain.PluginCount{1: {Enabled: 2, Total: 5}}}
	svc := NewProjectService(projRepo, &mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{}, &mockProjectInstallRepo{}, &mockProjectFS{}).
		WithPluginDeps(nil, counter)

	items, err := svc.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if items[0].PluginEnabledCount != 2 {
		t.Errorf("PluginEnabledCount: got %d want 2", items[0].PluginEnabledCount)
	}
	if items[0].PluginTotalCount != 5 {
		t.Errorf("PluginTotalCount: got %d want 5", items[0].PluginTotalCount)
	}
}

func TestListProjects_NoPluginCounter_CountsZero(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	svc := newProjectSvc(&mockProjectFS{}, projRepo) // no WithPluginDeps
	items, err := svc.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if items[0].PluginEnabledCount != 0 || items[0].PluginTotalCount != 0 {
		t.Errorf("expected zero counts, got %d/%d", items[0].PluginEnabledCount, items[0].PluginTotalCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd core-go && go test ./internal/services/ -run TestListProjects_PopulatesPluginCounts -v`
Expected: compile error — `WithPluginDeps` undefined and `PluginEnabledCount`/`PluginTotalCount` undefined.

- [ ] **Step 3: Add interfaces, struct fields, WithPluginDeps**

In `core-go/internal/services/project_service.go`:

(a) Add the two new fields to the `ProjectListItem` struct (after `WarningCount int`):

```go
	PluginEnabledCount int
	PluginTotalCount   int
```

(b) Add interfaces near the top of the file (after the import block, before `ProjectRemoveResult`):

```go
// ProjectPluginScanner scans a project's plugin settings layers within the caller's
// operation context (no new operation). Implemented by *ProviderPluginService.
type ProjectPluginScanner interface {
	ScanProjectLayers(ctx context.Context, project *domain.Project, progress operations.ProgressFn) error
}

// ProjectPluginCounter returns per-project effective plugin counts.
// Implemented by *ProviderPluginService.
type ProjectPluginCounter interface {
	PluginCountsByProject(ctx context.Context) (map[int64]domain.PluginCount, error)
}
```

(c) Add two fields to the `ProjectService` struct (after the remove deps block):

```go
	// plugin deps — nil until WithPluginDeps is called
	pluginScanner ProjectPluginScanner
	pluginCounter ProjectPluginCounter
```

(d) Add the builder method (after `WithRemoveDeps`):

```go
// WithPluginDeps attaches the plugin scanner (folded into project scan) and the
// plugin counter (used by ListProjects). Either may be nil. Returns the receiver.
func (s *ProjectService) WithPluginDeps(
	scanner ProjectPluginScanner,
	counter ProjectPluginCounter,
) *ProjectService {
	s.pluginScanner = scanner
	s.pluginCounter = counter
	return s
}
```

- [ ] **Step 4: Wire ListProjects**

In `ListProjects`, after the `projects, err := s.projectRepo.List(ctx)` error check and before the `items := make(...)` line, add:

```go
	var pluginCounts map[int64]domain.PluginCount
	if s.pluginCounter != nil {
		pluginCounts, err = s.pluginCounter.PluginCountsByProject(ctx)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not count plugins", err.Error())
		}
	}
```

Then inside the loop, in the `items = append(items, ProjectListItem{...})` literal, add the two fields:

```go
			PluginEnabledCount: pluginCounts[p.ID].Enabled,
			PluginTotalCount:   pluginCounts[p.ID].Total,
```

(Indexing a nil map returns the zero `PluginCount{}`, so the no-counter path yields 0/0.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/services/ -run TestListProjects -v`
Expected: PASS (new tests plus existing ListProjects tests still green).

- [ ] **Step 6: Commit**

```bash
git add core-go/internal/services/project_service.go core-go/internal/services/project_service_test.go
git commit -m "Add plugin deps and list plugin counts to ProjectService"
```

---

## Task 3: Fold plugin scan into project scan operation

**Files:**
- Modify: `core-go/internal/services/project_service.go` (`scanProjectInternal`)
- Test: `core-go/internal/services/project_scan_full_service_test.go` (append)

- [ ] **Step 1: Write the failing test**

First ensure `core-go/internal/services/project_scan_full_service_test.go` imports the operations package (the fake scanner's signature uses `operations.ProgressFn`). Its current import block is `context`, `testing`, `domain`, `providers`, `repositories`; add:

```go
	"github.com/astraler/skillbox/core-go/internal/operations"
```

Then append to the same file:

```go
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

// F3: a plugin-step error must NOT discard the committed skill-scan summary. The runner
// persists returned metadata on the failure path too (operations/runner.go ~106-138), so
// scanProjectInternal must return buildScanSummary(...) alongside the error, not nil.
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/services/ -run TestScanProjectInternal_InvokesPluginScanner -v`
Expected: FAIL — `scanner.called` is 0 (the production code does not call the scanner yet).

Run: `cd core-go && go test ./internal/services/ -run TestScanProjectInternal_PluginError -v`
Expected: FAIL — without the wiring the scanner is never called, so no error propagates (`expected plugin-step error to propagate`).

- [ ] **Step 3: Wire the scanner into scanProjectInternal (F3-safe return)**

In `core-go/internal/services/project_service.go`, in `scanProjectInternal`, replace the final success block:

```go
	if err := s.scanRepo.CommitProjectScan(ctx, project.ID, providerResults, projectWarnings, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not commit project scan", err.Error())
	}

	progress("done", 0, 0, "")
	return buildScanSummary(providerResults, projectWarnings), nil
```

with:

```go
	if err := s.scanRepo.CommitProjectScan(ctx, project.ID, providerResults, projectWarnings, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not commit project scan", err.Error())
	}

	if s.pluginScanner != nil {
		progress("scanning_plugins", 0, 0, "")
		if err := s.pluginScanner.ScanProjectLayers(ctx, project, progress); err != nil {
			// F3: skills already committed — return the summary WITH the error so the runner
			// persists it as operation metadata (partial failure), instead of discarding it.
			return buildScanSummary(providerResults, projectWarnings), err
		}
	}

	progress("done", 0, 0, "")
	return buildScanSummary(providerResults, projectWarnings), nil
```

Note: `operations.Runner.run` marshals the returned metadata once and writes it on both the success and failure `UpdateStatus` calls (runner.go ~106–138), so returning the summary alongside a non-nil error is the supported partial-failure pattern.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/services/ -run TestScanProjectInternal -v`
Expected: PASS (the three new tests plus the existing `TestScanProjectInternal_*` tests still green).

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/services/project_service.go core-go/internal/services/project_scan_full_service_test.go
git commit -m "Fold plugin scan into the project scan operation"
```

---

## Task 4: Implement ScanProjectLayers on ProviderPluginService

**Files:**
- Modify: `core-go/internal/services/provider_plugin_service.go`
- Test: `core-go/internal/services/provider_plugin_service_test.go` (append)

This method makes `*ProviderPluginService` satisfy `services.ProjectPluginScanner`. It delegates to the already-tested private `scanProjectInternal` but skips `runner.Start` so it runs inside the project-scan operation. **F2:** it must use `pluginProviderDefsAllowMissing` (not the strict `pluginProviderDefs`) and treat zero plugin-capable providers as a no-op — otherwise a fresh/partial DB with no seeded plugin providers would make the whole `project.scan` fail with a validation error.

- [ ] **Step 1: Write the failing test**

Append to `core-go/internal/services/provider_plugin_service_test.go`:

```go
// Compile-time assertion that *ProviderPluginService satisfies ProjectPluginScanner.
var _ ProjectPluginScanner = (*ProviderPluginService)(nil)

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
```

Note: `mockProviderRegistrySvc{}` returns an empty registry (no plugin-capable keys) — confirmed by existing `TestPluginProviderDefsAllowMissing_*` usage in this file. With no matching defs, `pluginProviderDefsAllowMissing` returns `(nil, nil)`, and `ScanProjectLayers` returns `nil` before touching the (nil) repo. If the mock's zero value returns plugin-capable defs, construct it with an explicitly empty entries slice matching the existing pattern.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd core-go && go test ./internal/services/ -run TestScanProjectLayers -v`
Expected: compile error — `svc.ScanProjectLayers undefined` (and the `var _ ProjectPluginScanner` assertion fails to compile).

- [ ] **Step 3: Implement ScanProjectLayers**

In `core-go/internal/services/provider_plugin_service.go`, add (right after the `ScanProject` method):

```go
// ScanProjectLayers scans the project + local settings layers for all plugin-capable
// providers and commits the results. Unlike ScanProject, it runs within the caller's
// operation context and does NOT start its own operation — used by ProjectService so a
// single project.scan covers skills and plugins together.
//
// It uses pluginProviderDefsAllowMissing (not the strict pluginProviderDefs): zero
// plugin-capable providers is a legitimate no-op, not an error. The strict variant returns
// a validation_error on zero defs, which — propagated through scanProjectInternal — would
// fail the entire project scan on a fresh/partial DB (F2).
func (s *ProviderPluginService) ScanProjectLayers(
	ctx context.Context,
	project *domain.Project,
	progress operations.ProgressFn,
) error {
	defs, err := s.pluginProviderDefsAllowMissing(ctx)
	if err != nil {
		return err
	}
	if len(defs) == 0 {
		return nil // no plugin-capable providers configured — nothing to scan
	}
	return s.scanProjectInternal(ctx, project, defs, progress)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/services/ -run TestScanProjectLayers -v`
Expected: PASS, and the package compiles (the `var _ ProjectPluginScanner` assertion holds).

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/services/provider_plugin_service.go core-go/internal/services/provider_plugin_service_test.go
git commit -m "Add ScanProjectLayers for in-operation plugin scanning"
```

---

## Task 5: Wire WithPluginDeps in main.go

**Files:**
- Modify: `core-go/cmd/skillbox-core/main.go`

`providerPluginSvc` is constructed after `projectSvc`. Since `WithPluginDeps` mutates the receiver, call it as a standalone statement after `providerPluginSvc` exists.

- [ ] **Step 1: Add the wiring call**

In `core-go/cmd/skillbox-core/main.go`, immediately after the line:

```go
	providerPluginSvc := services.NewProviderPluginService(providerPluginRepo, pdRepo, projectRepo, providerRegistrySvc, runner)
```

add:

```go
	projectSvc.WithPluginDeps(providerPluginSvc, providerPluginSvc)
```

- [ ] **Step 2: Build to verify it compiles**

Run: `cd core-go && go build ./...`
Expected: no output (success). `*ProviderPluginService` satisfies both `ProjectPluginScanner` and `ProjectPluginCounter`.

- [ ] **Step 3: Run the full Go suite**

Run: `cd core-go && go test ./...`
Expected: PASS across all packages.

- [ ] **Step 4: Commit**

```bash
git add core-go/cmd/skillbox-core/main.go
git commit -m "Wire plugin scanner and counter into ProjectService"
```

---

## Task 6: Expose plugin counts in contract + project.list handler

**Files:**
- Modify: `shared/api-contracts/methods/project.list.json`
- Modify: `core-go/internal/rpc/handlers/project_list.go`
- Regenerate: `shared/generated/methods/project-list.ts`
- Test: `core-go/internal/rpc/handlers/project_handler_test.go` (append)

- [ ] **Step 1: Write the failing handler test**

Append to `core-go/internal/rpc/handlers/project_handler_test.go`:

```go
func TestProjectListHandler_IncludesPluginCounts(t *testing.T) {
	svc := &stubProjectList{items: []services.ProjectListItem{
		{
			ID: 1, Name: "p", Path: "/tmp/p", Status: domain.ProjectStatusActive,
			PluginEnabledCount: 2, PluginTotalCount: 5,
		},
	}}
	cli := startServer(t, handler.Map{"project.list": handlers.NewProjectListHandler(svc)})

	var resp struct {
		Projects []struct {
			PluginEnabledCount int `json:"pluginEnabledCount"`
			PluginTotalCount   int `json:"pluginTotalCount"`
		} `json:"projects"`
	}
	if err := cli.CallResult(context.Background(), "project.list", map[string]interface{}{}, &resp); err != nil {
		t.Fatalf("project.list: %v", err)
	}
	if len(resp.Projects) != 1 {
		t.Fatalf("projects: got %d want 1", len(resp.Projects))
	}
	if resp.Projects[0].PluginEnabledCount != 2 {
		t.Errorf("pluginEnabledCount: got %d want 2", resp.Projects[0].PluginEnabledCount)
	}
	if resp.Projects[0].PluginTotalCount != 5 {
		t.Errorf("pluginTotalCount: got %d want 5", resp.Projects[0].PluginTotalCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run TestProjectListHandler_IncludesPluginCounts -v`
Expected: FAIL — both counts are 0 (handler does not map the fields yet).

- [ ] **Step 3: Map the fields in the handler**

In `core-go/internal/rpc/handlers/project_list.go`:

(a) Add to the `projectListItem` struct (after `WarningCount int ...`):

```go
	PluginEnabledCount int     `json:"pluginEnabledCount"`
	PluginTotalCount   int     `json:"pluginTotalCount"`
```

(b) In the `resp.Projects = append(resp.Projects, projectListItem{...})` literal, add:

```go
			PluginEnabledCount: item.PluginEnabledCount,
			PluginTotalCount:   item.PluginTotalCount,
```

- [ ] **Step 4: Run handler test to verify it passes**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run TestProjectListHandler_IncludesPluginCounts -v`
Expected: PASS.

- [ ] **Step 5: Update the contract schema**

In `shared/api-contracts/methods/project.list.json`, inside `definitions.ProjectListItem.properties`, after the `warningCount` property, add:

```json
        "pluginEnabledCount": {
          "type": "integer",
          "description": "Count of effectively-enabled plugins across all providers for this project"
        },
        "pluginTotalCount": {
          "type": "integer",
          "description": "Count of distinct effective plugins (enabled + disabled + unknown) across all providers; 0 when no plugin scan data"
        },
```

and add both names to the `required` array of `ProjectListItem`:

```json
      "required": ["id", "name", "path", "status", "providers", "skillCount", "warningCount", "lastScannedAt", "pluginEnabledCount", "pluginTotalCount"],
```

- [ ] **Step 6: Regenerate contracts and typecheck**

Run:
```bash
cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck
```
Expected: generation writes `shared/generated/methods/project-list.ts` with the two new fields; drift check passes; typecheck passes (the renderer does not yet read the fields, which is fine).

- [ ] **Step 7: Commit (atomically)**

The schema (`project.list.json`), the regenerated `project-list.ts`, and the handler struct change must land in **one commit** — the fields are `required`, so a regenerated type without the handler emitting them (or vice versa) yields contract drift / a response that fails schema validation. Commit all four files together:

```bash
git add core-go/internal/rpc/handlers/project_list.go core-go/internal/rpc/handlers/project_handler_test.go shared/api-contracts/methods/project.list.json shared/generated/methods/project-list.ts
git commit -m "Expose plugin counts in project.list contract and handler"
```

---

## Task 7: Projects-list Plugins column (UI)

**Files:**
- Modify: `apps/desktop/renderer/src/screens/projects-screen.tsx`
- Modify: `apps/desktop/renderer/src/features/projects/project-row.tsx`
- Test: `apps/desktop/renderer/src/features/projects/__tests__/project-row.test.tsx` (create)

- [ ] **Step 1: Write the failing test**

Create `apps/desktop/renderer/src/features/projects/__tests__/project-row.test.tsx`:

```tsx
// @vitest-environment happy-dom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("@tanstack/react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("../use-scan-project.js", () => ({ useScanProject: () => ({ mutate: vi.fn(), isPending: false, operationId: null }) }));
vi.mock("../use-open-project-folder.js", () => ({ useOpenProjectFolder: () => ({ mutate: vi.fn(), isPending: false }) }));
vi.mock("../use-open-project-terminal.js", () => ({ useOpenProjectTerminal: () => ({ mutate: vi.fn(), isPending: false }) }));
vi.mock("../use-remove-project.js", () => ({ useRemoveProject: () => ({ mutate: vi.fn(), isPending: false }) }));

import { ProjectRow } from "../project-row.js";
import type { ProjectListItem } from "@contracts/index.js";

const base: ProjectListItem = {
  id: 1, name: "proj", path: "/tmp/proj", status: "active",
  providers: [], skillCount: 0, warningCount: 0, lastScannedAt: null,
  pluginEnabledCount: 0, pluginTotalCount: 0,
};

function renderRow(item: ProjectListItem) {
  return render(<table><tbody><ProjectRow project={item} /></tbody></table>);
}

afterEach(cleanup);

describe("ProjectRow plugin stats", () => {
  it("renders enabled/total when plugins present", () => {
    renderRow({ ...base, pluginEnabledCount: 2, pluginTotalCount: 5 });
    expect(screen.getByText("2/5")).toBeTruthy();
  });

  it("renders an em dash when no plugins", () => {
    renderRow({ ...base, pluginTotalCount: 0 });
    expect(screen.getByText("—")).toBeTruthy();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/desktop && pnpm test -- project-row`
Expected: FAIL — `2/5` not found (no plugin cell rendered yet).

- [ ] **Step 3: Add the Plugins cell to project-row**

In `apps/desktop/renderer/src/features/projects/project-row.tsx`, add a new `<td>` immediately after the Skills cell (the `<td>` containing `<ProjectProviderSkillStats .../>`) and before the Last Scanned `<td>`:

```tsx
      <td className="px-3 py-2">
        {project.pluginTotalCount > 0 ? (
          <span
            className="inline-flex items-center gap-1 rounded bg-zinc-100 px-1.5 py-0.5 text-xs font-medium text-zinc-600"
            title={`${project.pluginEnabledCount} enabled of ${project.pluginTotalCount} plugin${project.pluginTotalCount === 1 ? "" : "s"}`}
          >
            <span className="font-mono text-[11px]">
              {project.pluginEnabledCount}/{project.pluginTotalCount}
            </span>
          </span>
        ) : (
          <span className="text-xs text-zinc-400">—</span>
        )}
      </td>
```

- [ ] **Step 4: Add the Plugins column header**

In `apps/desktop/renderer/src/screens/projects-screen.tsx`, in the `<thead>` row, add a header `<th>` between the `Skills` header and the `Last Scanned` header:

```tsx
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Plugins</th>
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd apps/desktop && pnpm test -- project-row`
Expected: PASS (both cases).

- [ ] **Step 6: Typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/desktop/renderer/src/screens/projects-screen.tsx apps/desktop/renderer/src/features/projects/project-row.tsx apps/desktop/renderer/src/features/projects/__tests__/project-row.test.tsx
git commit -m "Add Plugins column to the Projects list"
```

---

## Task 8: Remove separate Scan Plugins button + refresh plugins on scan

**Files:**
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx`
- Modify: `apps/desktop/renderer/src/features/projects/use-scan-project.ts`
- Modify: `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx`

- [ ] **Step 1: Update the test to assert the button is gone**

In `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx`:

(a) Remove the mock for the project plugin scan hook (it is being removed from the component):

```tsx
vi.mock("../../features/provider-plugins/use-scan-provider-plugins-project.js", () => ({
  useScanProviderPluginsProject: vi.fn(),
}));
```
and its corresponding `import { useScanProviderPluginsProject } from ...` line, plus any `(useScanProviderPluginsProject as ...).mockReturnValue(...)` setup in `beforeEach`.

(b) Add assertions (in the describe block that renders a populated project detail) that the standalone button is gone while the section header remains, and that toggles are disabled while the unified scan is in flight (F1). For the F1 case, set the `useScanProject` mock to report an in-flight scan and render a project that has at least one toggleable Claude plugin (so an Enable/Disable button exists):

```tsx
  it("does not render a separate Scan Plugins button", () => {
    // ...existing render of a loaded project detail...
    expect(screen.queryByRole("button", { name: /scan plugins/i })).toBeNull();
    expect(screen.getByText(/provider plugins/i)).toBeTruthy();
  });

  it("disables plugin toggle while the unified scan is in flight (F1)", () => {
    // useScanProject mock returns an in-flight scan:
    (useScanProject as ReturnType<typeof vi.fn>).mockReturnValue({
      mutate: vi.fn(), isPending: true, operationId: 1,
    });
    // useProviderPluginList mock returns a project view for this projectId with a
    // claude plugin whose provenanceLayer !== "local" (so an Enable/Disable button renders).
    // ...render the loaded project detail...
    const toggle = screen.getByRole("button", { name: /enable|disable/i });
    expect(toggle).toHaveProperty("disabled", true);
  });
```

Note: this requires the `useProviderPluginList` mock to return `{ projects: [{ projectId, providerKey: "claude", layerStatuses: [], plugins: [{ pluginName: "p", marketplaceName: "m", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: [] }], marketplaces: [], managedOutOfScope: false }] }` for the rendered project. Mirror the existing populated-detail setup in this test file.

If the existing test file sets `useScanProviderPluginsProject` return values, removing those lines is required for compilation.

- [ ] **Step 2: Run test to verify it fails (or fails to compile)**

Run: `cd apps/desktop && pnpm test -- project-detail-screen`
Expected: FAIL — either compile error (removed mock still imported by component) or the button query still finds a button.

- [ ] **Step 3: Remove the button and hook usage from the component**

In `apps/desktop/renderer/src/screens/project-detail-screen.tsx`:

(a) Remove the import:

```tsx
import { useScanProviderPluginsProject } from "../features/provider-plugins/use-scan-provider-plugins-project.js";
```

(b) **Preserve the toggle guard via a `scanInFlight` prop (F1).** In `ProjectPluginSection`, remove the plugin-scan hook and its `isScanning`, but do NOT drop the in-flight guard — the unified scan now writes plugin rows, so toggling during it must stay disabled. Delete:

```tsx
  const scanMutation = useScanProviderPluginsProject();
  const isScanning = scanMutation.operationId != null || scanMutation.isPending;
```

Change the component signature to accept the unified scan's in-flight state:

```tsx
function ProjectPluginSection({ projectId, scanInFlight }: { projectId: number; scanInFlight: boolean }): React.JSX.Element {
```

and fold it into `isOperationInFlight` (keep `isTogglingPlugin` as-is):

```tsx
  const isTogglingPlugin = setEnabledMutation.isPending || setEnabledMutation.operationId != null;
  const isOperationInFlight = isTogglingPlugin || scanInFlight;
```

Update the render site in `ProjectDetailScreen` to pass the existing `isScanning` (derived there from `useScanProject()` as `scan.operationId != null || scan.isPending`):

```tsx
            {/* Provider Plugins */}
            <ProjectPluginSection projectId={validId} scanInFlight={isScanning} />
```

(c) Replace the header block that contains the Scan Plugins button:

```tsx
      <div className="mb-2 flex items-center justify-between">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Provider Plugins
        </h3>
        <button
          onClick={() => scanMutation.mutate(projectId)}
          disabled={isScanning}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-2 py-1 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
          <RefreshCw size={11} className={isScanning ? "animate-spin" : ""} />
          {isScanning ? "Scanning…" : "Scan Plugins"}
        </button>
      </div>
```

with just the heading:

```tsx
      <div className="mb-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Provider Plugins
        </h3>
      </div>
```

(d) Update the empty-state copy from `"No plugin data. Run Scan Plugins to populate."` to:

```tsx
        <p className="text-xs text-zinc-400">No plugin data. Run a scan to populate.</p>
```

(e) If `RefreshCw` is now unused in this file, remove it from the lucide-react import. (It is still used by the main project "Scan" button in `ProjectDetailScreen`, so keep it.)

- [ ] **Step 4: Invalidate the plugin list when a project scan completes**

In `apps/desktop/renderer/src/features/projects/use-scan-project.ts`, add a `providerPlugins.list` invalidation everywhere the scan-success handler invalidates `projects.detail`/`projects.list`. There are two such places: the buffered-terminal branch in `onSuccess`, and the live `subscribeOperationProgress` success branch. In each, after the existing two `invalidateQueries` calls, add:

```ts
        void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
```

(`queryKeys.providerPlugins.list()` already exists in `apps/desktop/renderer/src/lib/query-keys.ts`.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/desktop && pnpm test -- project-detail-screen`
Expected: PASS — no Scan Plugins button, Provider Plugins heading present.

- [ ] **Step 6: Typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/desktop/renderer/src/screens/project-detail-screen.tsx apps/desktop/renderer/src/features/projects/use-scan-project.ts apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx
git commit -m "Remove separate Scan Plugins button; refresh plugins on project scan"
```

---

## Task 9: End-to-end verification

**Files:** none (verification only); add an optional integration assertion if a project-scan DB test harness exists.

- [ ] **Step 1: Run the full Go suite with race detector on concurrency-sensitive packages**

Run:
```bash
cd core-go && go test ./... && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...
```
Expected: all PASS.

- [ ] **Step 2: Run the full frontend suite + typecheck + contract drift**

Run:
```bash
cd apps/desktop && pnpm test && pnpm typecheck && pnpm check:contracts-drift
```
Expected: all PASS; no contract drift.

- [ ] **Step 3: Manual full-stack smoke (UI verification)**

Run: `cd apps/desktop && pnpm dev`

Verify in the running app:
1. Projects list shows a **Plugins** column. A project with plugin declarations shows `enabled/total` (e.g. `2/5`); a project with none shows `—`.
2. Open a project, click the single **Scan** button. After it completes, the **Provider Plugins** section is populated **without** a separate "Scan Plugins" click, and there is **no** "Scan Plugins" button.
3. Return to the Projects list — the Plugins column reflects the latest scan (the list query was invalidated).

If the UI cannot be exercised in the current environment, state that explicitly rather than marking this step done.

- [ ] **Step 4: Final commit (if any verification fixes were needed)**

```bash
git add -A
git commit -m "Verify plugin-scan integration and list stats end-to-end"
```

---

## Spec Coverage Check

- **R1 (project.scan scans plugins):** Tasks 3 (fold into operation), 4 (`ScanProjectLayers`), 5 (wiring). Verified in Task 9 smoke step 2.
- **R2 (Projects list enabled/total column):** Tasks 1 (counts), 2 (service fields), 6 (contract+handler), 7 (UI). Verified in Task 9 smoke step 1.
- **UX decision (remove Scan Plugins button + refresh):** Task 8.
- **R3 (enumerate plugin-display gaps):** Documented in the spec; **no code**. Note for implementers: a dedicated **Plugins screen** (`apps/desktop/renderer/src/screens/plugins-screen.tsx`) already exists and surfaces global/user-layer plugins, so the spec's "global plugins overview" gap is already covered. Remaining recommendations (Dashboard plugin aggregate; optional per-provider plugin counts in the project detail providers table or list tooltip) stay as follow-up slices — do not implement here.

## Notes on Decisions Already Made (do not re-litigate)

- **Total** = effective entries with status ≠ absent (enabled + disabled + unknown). Project views already exclude absent, so `len(view.Plugins)` is the per-provider total.
- Plugin scan runs **after** the skill/provider commit in the same operation. On a plugin-step error the operation still fails, but `scanProjectInternal` returns the skill summary **alongside** the error (F3) so the runner persists it as metadata; per-file issues (missing/malformed) are recorded as scan-status rows, not errors.
- No new migration; effective resolution is reused from Go, not reimplemented in SQL.

## Review Fixes Incorporated (lead review of spec 8b1b4fc)

- **F1 — toggle race:** removing the plugin-scan hook would drop the in-flight guard on plugin toggles. Fixed in Task 8: `ProjectPluginSection` takes a `scanInFlight` prop (the unified scan's state) and keeps `isOperationInFlight = isTogglingPlugin || scanInFlight`.
- **F2 — fresh-install crash:** `ScanProjectLayers` uses `pluginProviderDefsAllowMissing` and no-ops (returns `nil`) on zero defs instead of propagating a validation error (Task 4).
- **F3 — silent data loss:** plugin-step error returns `buildScanSummary(...), err`, not `nil, err`, so the committed skill summary survives as operation metadata (Task 3). Verified against `operations/runner.go` ~106–138 (metadata persisted on the failure path).
- **F4 — wiring location:** `WithPluginDeps` is wired in `cmd/skillbox-core/main.go` (the `ProjectService` builder chain), not `wire.go` (which only registers handlers from already-built services) — Task 5.
- **Contract atomicity:** schema + generated TS + handler change land in one commit (Task 6 Step 7).
- **`providerPlugin.scanProject` left live (dormant):** after the Task 8 button removal it has no UI caller but stays registered. Its operation target (`provider_plugin_project`) differs from the unified scan's (`project`), so the per-target lock does NOT serialize them — concurrent runs could race on the same `provider_plugin_layer_scans` rows (transactional upsert → last-writer-wins, not corruption). Removing/retargeting it is out of scope for this slice; flagged for a future one.
