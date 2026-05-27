# Plugin Scan in Project Scan + Project-List Plugin Stats — Design

- **Date:** 2026-05-27
- **Status:** Approved (pre-implementation)
- **Scope:** Fold provider-plugin scanning into `project.scan`; surface per-project plugin stats in the Projects list; enumerate remaining plugin-display gaps.

## Context

Skillbox already has a working provider-plugin subsystem:

- **Schema** (migration `000012` + `000013` warnings, `000014` config paths): `provider_plugin_layer_scans` (one row per `provider_definition_id` + `project_id` + `settings_layer`; `user` layer has null `project_id`), `provider_plugin_entries` (plugin_name, marketplace_name, declaration enabled/disabled), `provider_plugin_marketplaces`.
- **Service** (`core-go/internal/services/provider_plugin_service.go`): `ScanGlobal`, `ScanProject` (operation target `provider_plugin_project`), `List`/`ListAll`, `SetPluginEnabled`. Plugin-capable providers are `claude`, `codex`, `antigravity_cli`. Effective status is resolved per plugin across layers with precedence **local > project > user** (`buildProjectPluginView` / `resolveEffectivePlugin`); a `missing` file does not block inheritance, other non-ok statuses do.
- **RPC**: `providerPlugin.scanGlobal`, `providerPlugin.scanProject`, `providerPlugin.list`, `providerPlugin.setEnabled`.
- **UI**: `project-detail-screen.tsx` renders `ProjectPluginSection`, which reads `providerPlugin.list` and has its **own "Scan Plugins" button** (`useScanProviderPluginsProject`) separate from the project "Scan" button.

What is missing:

1. `project.scan` (`ProjectService.ScanProject`, operation target `project`) scans **skills/providers only** — it never scans plugins. Users must click a second button.
2. The Projects list (`project.list` + `projects-screen.tsx`) shows **no plugin information** at all.
3. No systematic record of where else plugin info should appear.

## Goals

- **R1:** A single `project.scan` also scans the project's plugins (project + local layers) and persists them, associated with the project.
- **R2:** The Projects list shows a **Plugins** column with **enabled/total** stats per project (e.g. `2/5`, or `—` when there is no plugin data).
- **R3:** Identify UI surfaces that should show plugin info but don't, and recommend additions (enumeration only — not designed in this spec).

## Non-Goals

- No new DB migration — existing plugin tables are sufficient.
- No re-implementation of effective-status resolution in SQL — reuse the tested Go resolution.
- No changes to plugin write/toggle (`setEnabled`) or global/user-layer scan flows.
- R3 surfaces are not designed or implemented here.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Stats meaning | **enabled / total** (e.g. `2/5`) | Chosen by user; mirrors task's "installed/total" framing. |
| `total` definition | distinct effective entries with status ≠ `absent` (enabled + disabled + unknown), summed across providers | `buildProjectPluginView` already excludes `absent`, so `total = len(view.Plugins)`. |
| `enabled` definition | effective entries with status `enabled`, summed across providers | Direct count. |
| Separate "Scan Plugins" button | **Removed** | Chosen by user — one "Scan" does skills + plugins. |
| Scan integration | **Single operation** — plugin scan folded into the project-scan operation, no nested `runner.Start` | Matches one-operation/one-lock model in CLAUDE.md. |
| Stats computation | Reuse Go effective-resolution via plugin service, not SQL | Resolution logic (missing vs malformed inheritance blocking, local>project>user) is subtle and already tested. |

## Architecture & Approach

### Chosen approach (scan): fold plugin layer-scan into the project-scan operation

Extract the project/local layer scan logic from `ProviderPluginService` into a method that runs **inside the caller's operation context** (no new operation lock). `ProjectService.scanProjectInternal` calls it after the provider/skill commit. Result: one operation, one lock (target `project`), one progress stream.

**Alternatives considered and rejected:**

- **Nested operation** — `scanProjectInternal` calls the existing public `ProviderPluginService.ScanProject`, which itself calls `runner.Start` with target `provider_plugin_project`. Rejected: two operation locks, two progress streams, awkward cancellation; violates the single-operation model.
- **Two operations triggered from the UI/handler** — Rejected: not atomic, and R1 explicitly makes scanning plugins a responsibility of `project.scan` (backend concern, not UI orchestration).

### Chosen approach (stats): reuse Go resolution

`ProjectService.ListProjects` asks the plugin service for a per-project count map computed from `ListAll` project views, rather than reimplementing precedence in SQL.

## Detailed Design

### 1. Backend — scan integration

**`core-go/internal/services/provider_plugin_service.go`**

Add an exported method that wraps the existing private `scanProjectInternal` without starting an operation:

```go
// ScanProjectLayers scans the project + local settings layers for all plugin-capable
// providers, committing results. It runs within the caller's operation context and does
// NOT start its own operation (used by ProjectService during a unified project scan).
func (s *ProviderPluginService) ScanProjectLayers(
    ctx context.Context,
    project *domain.Project,
    progress operations.ProgressFn,
) error {
    defs, err := s.pluginProviderDefs(ctx)
    if err != nil {
        return err
    }
    return s.scanProjectInternal(ctx, project, defs, progress)
}
```

(The existing public `ScanProject` is unchanged and still wired to `providerPlugin.scanProject`.)

**`core-go/internal/services/project_service.go`**

Add an optional dependency, consistent with the existing `WithScanDeps` / `WithProviderDeps` / `WithInstallDeps` nil-until-set pattern:

```go
type ProjectPluginScanner interface {
    ScanProjectLayers(ctx context.Context, project *domain.Project, progress operations.ProgressFn) error
}

type ProjectPluginCounter interface {
    PluginCountsByProject(ctx context.Context) (map[int64]domain.PluginCount, error)
}

// on ProjectService struct:
//   pluginScanner ProjectPluginScanner // nil until WithPluginDeps
//   pluginCounter ProjectPluginCounter // nil until WithPluginDeps

func (s *ProjectService) WithPluginDeps(
    scanner ProjectPluginScanner,
    counter ProjectPluginCounter,
) *ProjectService {
    s.pluginScanner = scanner
    s.pluginCounter = counter
    return s
}
```

In `scanProjectInternal`, after `s.scanRepo.CommitProjectScan(...)` succeeds and before `progress("done", …)`:

```go
if s.pluginScanner != nil {
    progress("scanning_plugins", 0, 0, "")
    if err := s.pluginScanner.ScanProjectLayers(ctx, project, progress); err != nil {
        return nil, err // DB-level failure fails the op; committed skill data persists
    }
}
```

Notes:
- Per-file problems (missing/malformed/etc.) are recorded as `scan_status` rows by the plugin scan, **not** returned as errors — same as today's standalone plugin scan. Only DB-level failures surface as operation errors.
- Plugin scan runs after the skill/provider commit. They write disjoint tables, so ordering is not correctness-critical; running last keeps skill results committed even if the plugin step errors.
- The terminal paths (`commitTerminalPath` / `commitTerminalDirect` for missing/unreadable projects) do **not** scan plugins — an unreadable project has no readable settings files.

### 2. Backend — list stats

**New domain type** (`core-go/internal/domain/provider_plugin.go`):

```go
// PluginCount is the per-project aggregate of effective plugins across all providers.
type PluginCount struct {
    Enabled int
    Total   int // effective entries with status != absent (enabled + disabled + unknown)
}
```

**New method** on `ProviderPluginService`:

```go
// PluginCountsByProject aggregates effective plugin counts per project across all
// plugin-capable providers, derived from persisted scan data.
func (s *ProviderPluginService) PluginCountsByProject(ctx context.Context) (map[int64]domain.PluginCount, error) {
    _, projects, err := s.ListAll(ctx)
    if err != nil {
        return nil, err
    }
    counts := make(map[int64]domain.PluginCount)
    for _, pv := range projects { // pv is domain.ProjectPluginView (already excludes absent)
        c := counts[pv.ProjectID]
        for _, p := range pv.Plugins {
            c.Total++
            if p.EffectiveStatus == domain.PluginEffectiveEnabled {
                c.Enabled++
            }
        }
        counts[pv.ProjectID] = c
    }
    return counts, nil
}
```

**`ProjectListItem`** (`project_service.go`) gains two fields:

```go
PluginEnabledCount int
PluginTotalCount   int
```

**`ListProjects`** fetches the map once before the loop (nil-safe) and sets the fields:

```go
var pluginCounts map[int64]domain.PluginCount
if s.pluginCounter != nil {
    pluginCounts, err = s.pluginCounter.PluginCountsByProject(ctx)
    if err != nil {
        return nil, domain.NewDatabaseError("Could not count plugins", err.Error())
    }
}
// inside loop:
pc := pluginCounts[p.ID] // zero value {0,0} when absent
item.PluginEnabledCount = pc.Enabled
item.PluginTotalCount = pc.Total
```

### 3. Contract + handler

**`shared/api-contracts/methods/project.list.json`** — add to `ProjectListItem.properties` and `required`:

```json
"pluginEnabledCount": {
  "type": "integer",
  "description": "Count of effectively-enabled plugins across all providers for this project"
},
"pluginTotalCount": {
  "type": "integer",
  "description": "Count of distinct effective plugins (enabled + disabled + unknown) across all providers; 0 when no plugin scan data"
}
```

Regenerate types: `pnpm generate:contracts` → updates `shared/generated/methods/project-list.ts`. Verify with `pnpm check:contracts-drift`.

**`core-go/internal/rpc/handlers/project_list.go`** — add `PluginEnabledCount`/`PluginTotalCount` to the `projectListItem` JSON struct (`pluginEnabledCount`, `pluginTotalCount`) and map them from the service item.

### 4. UI

**`apps/desktop/renderer/src/screens/projects-screen.tsx`** — add a `Plugins` column header between `Skills` and `Last Scanned`:

```
Project | Status | Providers | Skills | Plugins | Last Scanned | (actions)
```

**`apps/desktop/renderer/src/features/projects/project-row.tsx`** — add a cell rendering plugin stats:

```tsx
<td className="px-3 py-2">
  {project.pluginTotalCount > 0 ? (
    <span
      className="inline-flex items-center gap-1 rounded bg-zinc-100 px-1.5 py-0.5 text-xs font-medium text-zinc-600"
      title={`${project.pluginEnabledCount} enabled of ${project.pluginTotalCount} plugin${project.pluginTotalCount === 1 ? "" : "s"}`}
    >
      <span className="font-mono text-[11px]">{project.pluginEnabledCount}/{project.pluginTotalCount}</span>
    </span>
  ) : (
    <span className="text-xs text-zinc-400">—</span>
  )}
</td>
```

**`apps/desktop/renderer/src/screens/project-detail-screen.tsx`** — in `ProjectPluginSection`:
- Remove the **"Scan Plugins"** button and the `useScanProviderPluginsProject` usage (and the import if unused elsewhere). The section becomes read-only display fed by the unified scan.
- Keep the `useProviderPluginList` query for display.
- Ensure the plugin list refreshes after a unified scan: in the project-scan success handler (`use-scan-project.ts`), invalidate the `providerPlugin.list` query key in addition to project detail, so `ProjectPluginSection` refetches once the scan completes.
- Empty-state copy update: "No plugin data. Run a scan to populate." (drop the "Scan Plugins" wording).

### 5. Testing

**Go:**
- Extend project scan/handler tests (`core-go/internal/rpc/handlers/project_handler_test.go` and/or service tests): after `project.scan`, assert `provider_plugin_layer_scans` rows exist for the project (project + local layers) and that `project.list` returns expected `pluginEnabledCount`/`pluginTotalCount`.
- Unit test `ProviderPluginService.PluginCountsByProject` with a fixture mixing enabled/disabled/absent across user/project/local layers (verify `absent` excluded from total, `unknown` counted in total but not enabled, multiple providers summed).
- Nil-safety: a `ProjectService` built without `WithPluginDeps` still scans skills and lists projects with plugin counts `0`.
- Run `go test ./...` and `go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...`.

**Frontend (Vitest + RTL):**
- `project-row` renders `2/5` when counts present and `—` when `pluginTotalCount === 0`.
- `projects-screen` renders the `Plugins` column header.
- `project-detail-screen` `ProjectPluginSection` no longer renders a "Scan Plugins" button.

### 6. Requirement 3 — plugin-display gaps (enumerate + recommend)

Identified surfaces where plugin info is absent but arguably useful. **Recommendations only — not implemented in this slice.**

- **Dashboard** (`dashboard.get` / dashboard screen): no plugin aggregate. *Recommend* a headline stat such as "N plugins enabled across M projects", reusing `PluginCountsByProject`.
- **Settings / global (user-layer) plugins:** `providerPlugin.scanGlobal` and `GlobalPluginView` exist in the backend/contract, but there is no confirmed Settings surface exposing user-layer plugins and marketplaces. *Recommend* a "Global Plugins" overview in Settings (status per provider, enabled/disabled list, scan button) — this is the highest-value gap.
- **Project-list provider badges:** the new aggregate column covers the headline need; a per-provider plugin breakdown (e.g. `claude 2/4`, `codex 0/1`) could live in the column tooltip later. *Deferred.*
- **Project detail providers table:** currently shows skill `Entries` per provider but not plugin counts. *Recommend* an optional per-provider plugin count column once the global overview lands.

## Risks & Mitigations

- **`ListAll` cost on the list path:** `PluginCountsByProject` loads all layer scans + entries for all plugin-capable providers on every `project.list`. For the current scale (local desktop, few projects/providers) this is acceptable. *Mitigation if needed later:* a dedicated aggregate COUNT query, accepting the resolution-in-SQL cost — explicitly out of scope now.
- **Stale plugin section after unified scan:** mitigated by invalidating `providerPlugin.list` in the project-scan success handler.
- **User-layer never scanned:** effective resolution treats the user layer as nil and resolves from project/local only — existing, accepted behavior.

## Files Touched (implementation checklist)

Backend:
- `core-go/internal/domain/provider_plugin.go` — add `PluginCount`.
- `core-go/internal/services/provider_plugin_service.go` — add `ScanProjectLayers`, `PluginCountsByProject`.
- `core-go/internal/services/project_service.go` — add `WithPluginDeps`, scanner/counter deps, `scanning_plugins` phase, `PluginEnabledCount`/`PluginTotalCount` on `ProjectListItem` + `ListProjects` wiring.
- `core-go/internal/app/wire.go` — call `projectSvc.WithPluginDeps(providerPluginSvc, providerPluginSvc)` (and `cmd/skillbox-core/main.go` if construction lives there).
- `core-go/internal/rpc/handlers/project_list.go` — map new fields.

Contract:
- `shared/api-contracts/methods/project.list.json` — add two fields. Regenerate `shared/generated/`.

Frontend:
- `apps/desktop/renderer/src/screens/projects-screen.tsx` — Plugins column header.
- `apps/desktop/renderer/src/features/projects/project-row.tsx` — Plugins cell.
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — remove Scan Plugins button.
- `apps/desktop/renderer/src/features/projects/use-scan-project.ts` — invalidate `providerPlugin.list` on scan success.

Tests: as listed in §5.
