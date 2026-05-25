# Slice 2H: Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use `- [ ]` tracking. TDD throughout: test → red → implement → green → commit.

**Goal:** Add a read-only Dashboard surface backed by one aggregate RPC `dashboard.get`, and make `/dashboard` the post-setup landing screen.

**Architecture:** New Go `DashboardService` composes existing repos (settings, host, skill, project, install, warning) into a `DashboardView`. New read-only repo aggregate queries; no migration. Renderer adds `/dashboard` route, `use-dashboard` hook, screen, sidebar item, and changes the index redirect.

**Tech Stack:** Go (jrpc2, modernc sqlite), JSON Schema → generated TS, React + TanStack Router/Query, Vitest/RTL.

Spec: `docs/superpowers/specs/2026-05-25-skillbox-slice-2h-dashboard-design.md` (commit c5c534d).

## PM decisions (fixed)
Dedicated `dashboard.get`; backend omits global-skills & updates counts (UI muted placeholders only); install-mode aggregate across non-removed projects; `/dashboard` route+sidebar+redirect; Recent Operations deferred; warning list cap 50; removed-project warning exclusion predicate (project/project_provider/install scopes).

## Files
- Create: `core-go/internal/domain/dashboard.go` (+`_test.go`)
- Modify: `core-go/internal/repositories/{skill_repo,project_repo,install_repo,warning_repo}.go` (+ tests) — add aggregate queries
- Create: `core-go/internal/services/dashboard_service.go` (+`dashboard_service_test.go`)
- Create: `core-go/internal/rpc/handlers/dashboard_get.go`; Modify `contract_test.go` (add case)
- Modify: `core-go/internal/app/wire.go`, `wire_test.go`; `core-go/cmd/skillbox-core/main.go`
- Create: `shared/api-contracts/methods/dashboard.get.json`; Modify `shared/api-contracts/index.json`; regen `shared/generated/**`
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts`
- Modify: `apps/desktop/renderer/src/lib/core-client/methods.ts` (+`__tests__/methods.test.ts`)
- Modify: `apps/desktop/renderer/src/lib/query-keys.ts`
- Create: `apps/desktop/renderer/src/features/dashboard/use-dashboard.ts` (+`__tests__`)
- Create: `apps/desktop/renderer/src/screens/dashboard-screen.tsx` (+`__tests__`)
- Modify: `apps/desktop/renderer/src/app/router.tsx` (export `IndexRedirector`, add route, redirect→`/dashboard`) (+`__tests__`)
- Modify: `apps/desktop/renderer/src/components/sidebar.tsx` (export `NAV_ITEMS`, add Dashboard first) (+`__tests__`)

## Backend tasks (TDD; commit after each task)

### Task 1 — Domain aggregate types
- [ ] `domain/dashboard.go`: `InstallModeCounts{Symlink,RsyncCopy,Direct int}`; `WarningSeverityCounts{Info,Warning,Error,Blocking int}` with `Total() int` summing the four.
- [ ] Test `Total()` returns sum; commit.

### Task 2 — Repo aggregate queries
- [ ] `SkillRepo.CountByHost(ctx,hostID)(int,error)` → `SELECT COUNT(*) FROM skills WHERE skill_host_folder_id=?`.
- [ ] `ProjectRepo.CountActive(ctx)(int,error)` → `… FROM projects WHERE status<>'removed'`.
- [ ] `InstallRepo.CountByModeActive(ctx)(domain.InstallModeCounts,error)` → `SELECT install_mode,COUNT(*) FROM installs i JOIN project_providers pp ON pp.id=i.project_provider_id JOIN projects p ON p.id=pp.project_id WHERE p.status<>'removed' GROUP BY install_mode`; map rows into fixed struct (unknown modes ignored).
- [ ] `WarningRepo.CountActiveBySeverity(ctx)(domain.WarningSeverityCounts,error)` and `ListActive(ctx,limit)([]domain.Warning,error)`. Both share package-level const predicate; `ListActive` uses `ORDER BY id DESC LIMIT ?` and reuses `scanWarning`. Severity switch ignores unrecognized values (still listed by `ListActive`).
  - Predicate (DRY const): `WHERE is_resolved=0 AND NOT ( (scope_type='project' AND scope_id IN (SELECT id FROM projects WHERE status='removed')) OR (scope_type='project_provider' AND scope_id IN (SELECT pp.id FROM project_providers pp JOIN projects p ON p.id=pp.project_id WHERE p.status='removed')) OR (scope_type='install' AND scope_id IN (SELECT i.id FROM installs i JOIN project_providers pp ON pp.id=i.project_provider_id JOIN projects p ON p.id=pp.project_id WHERE p.status='removed')) )`. NULL `scope_id` never matches → app/db warnings retained.
- [ ] Tests (`NewTestDB`, seed via direct SQL):
  - CountByHost: 2 hosts, count per host.
  - CountActive: excludes `removed`.
  - CountByModeActive: symlink/direct counted; installs of removed project excluded; absent mode = 0.
  - Warning exclusion: seed active project (id 1, pp 10, install 100) + removed project (id 2, pp 20, install 200). Warnings: project/1(warning,keep), project/2(exclude), project_provider/20(exclude), install/200(exclude), skill_host_folder/1(error,keep), app/NULL(info,keep), install/100(blocking,keep), resolved row(exclude), app/`critical`(unrecognized,exclude). Assert `CountActiveBySeverity`={info1,warning1,error1,blocking1} (critical not bucketed; Total 4); `ListActive(50)` returns the 4 kept rows in id-desc order.
  - Limit: seed >limit, assert len==limit.
- [ ] Commit.

### Task 3 — DashboardService
- [ ] `dashboard_service.go`: local narrow interfaces (`dashboardSettingsRepo.Get`, `dashboardHostRepo.GetByID`, `dashboardSkillRepo.CountByHost`, `dashboardProjectRepo.CountActive`, `dashboardInstallRepo.CountByModeActive`, `dashboardWarningRepo.{CountActiveBySeverity,ListActive}`) — deliberately NOT widening shared `services/interfaces.go` (would break unrelated mocks). `const dashboardWarningLimit = 50`.
- [ ] View structs: `DashboardActiveHost{HostID,Path,SkillsPath,Status domain.SkillHostStatus,LastScannedAt *string}`, `DashboardSummary{Skills,Projects,Warnings int}`, `DashboardWarningItem{Code,Message string,Severity domain.WarningSeverity,ScopeType domain.WarningScopeType,ScopeID *int64,ActionKey *string}`, `DashboardView{ActiveHost *…,Summary,InstallsByMode domain.InstallModeCounts,WarningsBySeverity domain.WarningSeverityCounts,Warnings []DashboardWarningItem}`.
- [ ] `NewDashboardService(settings,host,skill,project,install,warning)` + `Get(ctx)`:
  1. `settingsRepo.Get`; err→`NewDatabaseError`.
  2. If `ActiveSkillHostFolderID!=nil`: `hostRepo.GetByID`; err→db error; if host!=nil set ActiveHost (format `LastScannedAt` UTC `2006-01-02T15:04:05Z`) and `Summary.Skills=CountByHost`. Missing host row → ActiveHost nil, Skills 0.
  3. `CountActive`→Summary.Projects; `CountByModeActive`→InstallsByMode; `CountActiveBySeverity`→WarningsBySeverity, `Summary.Warnings=Total()`; `ListActive(50)`→map to items. Each err→db error.
- [ ] Test with one fake implementing all six interfaces (configurable values+errs, captures limit). Cases: no active host (ActiveHost nil, Skills 0, others set); active host (Skills set, fields mapped); host-missing (ActiveHost nil); `Summary.Warnings==sev.Total()` & limit==50; settings err → `errors.As(*domain.AppError)` Code==`domain.CodeDatabase`, view nil.
- [ ] Commit.

### Task 4 — Contract + generated types (do before handler/renderer)
- [ ] `methods/dashboard.get.json` (draft-07, `oneOf` Request/Response, matching `settings.get.json` style). Request empty object `additionalProperties:false`. Definitions: ActiveHost (hostId,path,skillsPath,status enum[7 host states],lastScanAt string|null), Summary (skills,projects,warnings int), InstallsByMode (symlink,rsyncCopy,direct int), WarningsBySeverity (info,warning,error,blocking int), Warning (code,message string; severity enum[info,warning,error,blocking]; scopeType enum[app,skill_host_folder,skill,project,project_provider,install]; scopeId int|null; actionKey string|null). Response: `activeHost` `oneOf [ref,null]`, summary, installsByMode, warningsBySeverity, warnings array; all required; `additionalProperties:false`. **Omit globalSkills/updatesAvailable entirely.**
- [ ] Add manifest entry to `index.json` → `methods/dashboard-get.ts`.
- [ ] `cd apps/desktop && pnpm generate:contracts`; commit schema + generated `DashboardGetResponse` etc.

### Task 5 — Handler + wiring + allowlist
- [ ] `dashboard_get.go`: narrow `dashboardService{Get(ctx)(*services.DashboardView,error)}`; response structs camelCase json tags mirroring contract; assign `lastScanAt` directly from `view.ActiveHost.LastScannedAt` (already `*string`, formatted in the service — consistent with `settings_get.go`; no `formatTimePtr`); init `warnings` as `make([]…,0,len)` so JSON emits `[]`; map enums via `string(...)`; `wrapError` on error. `NewDashboardGetHandler`.
- [ ] `contract_test.go`: add `TestContract_DashboardGet_Response` (populated + empty-warnings) validating against schema; assert struct round-trips (no global/update fields exist on struct).
- [ ] `wire.go`: register `"dashboard.get": rpchandlers.NewDashboardGetHandler(dashboardSvc)`; add `dashboardSvc *services.DashboardService` as new last param to `New(...)`.
- [ ] `wire_test.go`: `New(nil,nil,nil,nil,nil,nil)`; add `"dashboard.get"` to expected set.
- [ ] `main.go`: `dashboardSvc := services.NewDashboardService(appSettingsRepo, hostRepo, skillRepo, projectRepo, installRepo, warningRepo)`; pass to `app.New(...)`; append `"dashboard.get"` to `server.ready` capabilities.
- [ ] `method-allowlist.ts`: add `"dashboard.get"`.
- [ ] `go test ./...`; commit.

## Renderer tasks (TDD; commit per task)

### Task 6 — core-client + query key
- [ ] `methods.ts`: import `DashboardGetResponse`; add `getDashboard: () => invoke<DashboardGetResponse>("dashboard.get", {})`.
- [ ] `query-keys.ts`: add `dashboard: { root: () => ["dashboard"] as const }`.
- [ ] `methods.test.ts`: add case → invoked with `"dashboard.get", {}`. Commit.

### Task 7 — use-dashboard hook
- [ ] `features/dashboard/use-dashboard.ts`: `useQuery({queryKey: queryKeys.dashboard.root(), queryFn: () => methods.getDashboard()})`.
- [ ] Hook test (mirror `use-app-settings.test.tsx`): success returns data; reject → isError. Commit.

### Task 8 — Dashboard screen
- [ ] `dashboard-screen.tsx` (operational style, no SaaS cards). States: pending→spinner; error→`<ErrorDisplay>`+Retry(`refetch`); `activeHost==null`→"No Skill Host Folder configured" + button `navigate({to:"/setup"})`; else render: Host block (path, status badge, last scan); Summary (Skills/Projects/Warnings) + muted static "Global skills — Not in this slice" and "Updates — Not in this slice"; Installs by mode (symlink/rsync-copy/direct); Warnings list or `<EmptyState>` "No active warnings"; when `summary.projects===0` show CTA buttons to `/projects` and `/skills`. Warning rows display-only EXCEPT `scopeType==="project" && scopeId!=null` → clickable `navigate({to:"/projects/$projectId", params:{projectId:String(scopeId)}})`. Use `useNavigate()` (not Link) for testability.
- [ ] Render tests (mock `use-dashboard`; mock `@tanstack/react-router` `useNavigate`): loading, error+retry, loaded counts, muted placeholders present, zero-data CTA, empty-warnings, project-scoped warning click → navigate called. Commit.

### Task 9 — Router + sidebar
- [ ] `router.tsx`: import+export `DashboardScreen` route under `shellRoute` at `/dashboard`; add to `shellRoute.addChildren([...])`; export `IndexRedirector`; change success redirect from `/skills` to `/dashboard`.
- [ ] Router test: mock `useAppSettings` + `useNavigate`; render `<IndexRedirector/>`; activeHost set → navigate `{to:"/dashboard",replace:true}`; null → `{to:"/setup",replace:true}`.
- [ ] `sidebar.tsx`: import `LayoutDashboard`; export `NAV_ITEMS`; add `{to:"/dashboard",label:"Dashboard",icon:LayoutDashboard}` first.
- [ ] Sidebar test: `NAV_ITEMS[0]` is Dashboard `/dashboard`. Commit.

## Verification commands
- Go: `cd core-go && go test ./...` and `go test -race ./internal/repositories/... ./internal/services/...`
- Front: `cd apps/desktop && pnpm typecheck && pnpm test && pnpm check:contracts-drift && pnpm build`

## Manual smoke checklist (`pnpm dev`)
1. Boot with active host + projects + ≥1 symlink install → lands on `/dashboard`; sidebar Dashboard active.
2. Counts match Skills/Projects screens; installs show symlink=N, rsync-copy=0, direct=M.
3. Global skills + Updates render muted "Not in this slice" (not 0).
4. Existing warning shows correct severity; a `project`-scoped row navigates to its Project Detail.
5. Soft-remove a project that has a warning → its counts AND its warning drop from list/summary/severity; active-project warning stays.
6. Kill Go sidecar mid-session → error state + Retry; shell stays navigable.

## Risks
- Shared `services/interfaces.go` widening would break other mocks → use local narrow interfaces (Task 3).
- `warnings` serialized as `null` instead of `[]` → init empty slice in handler.
- Contract drift if generated TS not committed → run `generate:contracts`, commit `shared/generated/**`.
- Removed-project predicate correctness → covered by dedicated repo test (all 3 scopes + NULL + control).
- Unrecognized severity leaking outside the response contract → count switch ignores it and `ListActive` filters to recognized severities; defensive test asserts.

## Commit checkpoints
1. domain types · 2. repo queries+tests · 3. service+tests · 4. contract+generated · 5. handler+wire+allowlist+main · 6. core-client+keys · 7. hook · 8. screen · 9. router+sidebar. Final: full verification green.

## Out of scope (do not implement)
Recent Operations panel; global-skills/updates backend fields or counts; rsync/copy install support; provider install changes; warning quick-fix remediation actions; schema migration.
