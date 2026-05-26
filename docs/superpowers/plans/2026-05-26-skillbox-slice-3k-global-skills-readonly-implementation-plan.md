# Slice 3K — Global Skills Read-Only Visibility Implementation Plan

> **For agentic workers (tmux team):** Implemented by `agent-tech-skillbox` in the current `main`
> workspace. **No git worktrees, no subagents.** **Do not commit** — the PM/orchestrator commits after
> lead review and verification. Work task-by-task; each task ends at a **PM checkpoint** (stop, report,
> wait). Steps use checkbox (`- [ ]`) syntax. Test-first where practical.

**Goal:** Add a read-only Global Skills surface that scans `~/.agents/skills` (Shared Agent Skills only),
persists `global_provider_locations` + `global_installs`, and renders them in a new screen — no global
write actions.

**Architecture:** A new `GlobalProviderAdapter` (implemented only by `GenericAgentsAdapter`) reports the
home-relative `.agents/skills` location and its raw entries. `GlobalSkillsService.ScanGlobal` runs as an
async operation under a stable singleton lock (`global_scan/0`), classifies entries by **reusing
`ClassifyAdapterEntry`** against known Skill Host Folders, and commits atomically via a new
`GlobalScanRepo`. `global.list` reads the persisted state. The renderer adds a sidebar item, `/global`
route, a scan hook (mirroring `useScanHost`), and a read-only screen.

**Tech Stack:** Go (`modernc.org/sqlite`, `golang-migrate`, `creachadair/jrpc2`), JSON Schema →
generated TS, Electron preload allowlist, React + TanStack Router/Query + sonner, `go test`, Vitest + RTL.

**Source spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3k-global-skills-readonly-design.md`

---

## Files Touched

**Migrations:** create `core-go/migrations/000005_global_skills.up.sql` + `.down.sql`; test
`core-go/internal/repositories/migration_000005_test.go`.

**Go domain:** modify `internal/domain/warning.go` (+2 scopes), `internal/domain/operation.go`
(+`scan_global_skills`); create `internal/domain/global.go` (status enum + view structs).

**Go repositories:** create `internal/repositories/global_location_repo.go` (+`_test.go`) and
`internal/repositories/global_scan_commit.go` (+`_test.go`).

**Go providers:** create `internal/providers/global_adapter.go` (interface + `GlobalDetectResult`);
modify `internal/providers/generic_agents.go` (+`DetectGlobal`); test
`internal/providers/generic_agents_global_test.go`.

**Go filesystem:** modify `internal/filesystem/gateway.go` (+`HomeDir()`); modify
`internal/filesystem/paths.go` if a helper fits; test alongside.

**Go services:** create `internal/services/global_skills_service.go` (+`_test.go`); modify
`internal/services/interfaces.go` (+global repo/fs methods, provider registry `Get`, and existing
host/skill lister dependencies).

**Go RPC:** create `internal/rpc/handlers/global_scan.go` + `global_list.go`; modify
`internal/rpc/handlers/contract_test.go`.

**Go wiring:** construct repositories/services in `cmd/skillbox-core/main.go`; modify
`internal/app/wire.go` + `internal/app/wire_test.go` to accept/register the already-built service;
update `cmd/skillbox-core/main.go` capabilities.

**Contracts:** create `shared/api-contracts/methods/global.scan.json` + `global.list.json`; modify
`shared/api-contracts/index.json`; regen `shared/generated/`.

**Electron/renderer:** modify `electron/main/core-process/method-allowlist.ts`,
`renderer/src/lib/core-client/methods.ts`, `renderer/src/lib/query-keys.ts`,
`renderer/src/components/sidebar.tsx`, `renderer/src/app/router.tsx`; create
`renderer/src/features/global-skills/use-global-list.ts`, `.../use-scan-global.ts`,
`renderer/src/screens/global-skills-screen.tsx`; tests under
`renderer/src/features/global-skills/__tests__/` and `renderer/src/screens/__tests__/`.

---

# Checkpoint 1 — Data model, migration, domain, repositories

### Task 1 — Migration `000005_global_skills`
**Files:** create `core-go/migrations/000005_global_skills.up.sql`, `.down.sql`.
- [ ] `up.sql`: create `global_provider_locations` and `global_installs` exactly as spec §3.1
  (columns, `uq_global_loc_provider` on `provider_definition_id`,
  `uq_global_installs_loc_path` on `(global_provider_location_id, global_skill_path)`,
  `idx_global_installs_location`); then
  `UPDATE provider_definitions SET has_global_level=1 WHERE key='generic_agents';` and
  `UPDATE app_settings SET database_version=5 WHERE id=1;` (both with `updated_at` bump).
- [ ] `down.sql`: `DROP TABLE IF EXISTS global_installs;` then `global_provider_locations;`
  `UPDATE provider_definitions SET has_global_level=0 WHERE key='generic_agents';`
  `UPDATE app_settings SET database_version=4 WHERE id=1;`
- [ ] **Do not** touch `claude` (`has_global_level` stays 1).
- [ ] Stop for PM checkpoint.

### Task 2 — Migration test
**Files:** create `core-go/internal/repositories/migration_000005_test.go` (mirror `migration_000004_test.go`).
- [ ] **Test first:** after migrate-up, assert both tables exist (e.g. `INSERT` a location row succeeds),
  `provider_definitions.has_global_level=1 WHERE key='generic_agents'`,
  `claude.has_global_level=1` (unchanged), and `app_settings.database_version=5`.
- [ ] Run `cd core-go && go test ./internal/repositories/ -run TestMigration000005 -v` → PASS.
- [ ] Stop for PM checkpoint.

### Task 3 — Domain enums + structs
**Files:** modify `core-go/internal/domain/warning.go`, `core-go/internal/domain/operation.go`;
create `core-go/internal/domain/global.go`.
- [ ] In `warning.go` add to the `WarningScopeType` const block:
  `WarningScopeGlobalProviderLocation WarningScopeType = "global_provider_location"` and
  `WarningScopeGlobalInstall WarningScopeType = "global_install"`.
- [ ] In `operation.go` add `OperationTypeScanGlobalSkills OperationType = "scan_global_skills"`.
- [ ] Create `global.go`:

```go
package domain

type GlobalLocationStatus string

const (
	GlobalLocationStatusActive           GlobalLocationStatus = "active"
	GlobalLocationStatusNotConfigured    GlobalLocationStatus = "not_configured"
	GlobalLocationStatusMissing          GlobalLocationStatus = "missing"
	GlobalLocationStatusUnreadable       GlobalLocationStatus = "unreadable"
	GlobalLocationStatusInvalidStructure GlobalLocationStatus = "invalid_structure"
	GlobalLocationStatusEmpty            GlobalLocationStatus = "empty"
	GlobalLocationStatusDisabled         GlobalLocationStatus = "disabled"
)

// GlobalLocationView and GlobalInstallView back the global.list query (read model).
type GlobalInstallView struct {
	GlobalInstallID   int64
	SkillID           *int64
	SkillName         string
	Mode              InstallMode
	Status            InstallStatus
	GlobalSkillPath   string
	SourceSkillPath   *string
	SymlinkTargetPath *string
}

type GlobalLocationView struct {
	GlobalProviderLocationID int64
	ProviderKey              string
	ProviderDisplayName      string
	ProviderStatus           string
	Path                     *string
	SkillsPath               *string
	Status                   GlobalLocationStatus
	LastScannedAt            *string
	Entries                  []GlobalInstallView
	Warnings                 []Warning
}
```
- [ ] Run `cd core-go && go build ./internal/domain/...` → OK. Stop for PM checkpoint.

### Task 4 — Repo: `GlobalScanRepo.CommitGlobalScan`
**Files:** create `core-go/internal/repositories/global_scan_commit.go` (+ `global_scan_commit_test.go`).
Model on `project_scan_commit.go` (`upsertInstall`/`insertWarningTx`/`ptrToSQL`/`nullableStr` are
reusable package-level helpers in `repositories`).
- [ ] Define carrier structs:

```go
type GlobalInstallScanResult struct {
	SkillID                   *int64
	SkillName                 string
	InstallMode               domain.InstallMode
	InstallStatus             domain.InstallStatus
	GlobalSkillPath           string
	SourceSkillPath           *string
	SymlinkTargetPath         *string
	InstalledFromHostFolderID *int64
	Warning                   *domain.Warning // ScopeType=WarningScopeGlobalInstall; scope_id filled by commit
}

type GlobalScanRepo struct{ db *sql.DB }
func NewGlobalScanRepo(db *sql.DB) *GlobalScanRepo { return &GlobalScanRepo{db: db} }

// CommitGlobalScan persists one provider's global scan atomically:
//  1. Upsert global_provider_locations by provider_definition_id; capture location id.
//  2. Clear active warnings scoped to this location and its global_installs.
//  3. Upsert present installs; DELETE installs no longer on disk for this location.
//  4. Insert location-scoped warnings (scope_id=locationID) and install-scoped warnings.
//  5. Update location.last_scanned_at + status.
func (r *GlobalScanRepo) CommitGlobalScan(
	ctx context.Context,
	providerDefID int64,
	path, skillsPath *string,
	status domain.GlobalLocationStatus,
	installs []GlobalInstallScanResult,
	locationWarnings []domain.Warning,
	now time.Time,
) error
```
- [ ] Implementation notes: upsert location with `ON CONFLICT(provider_definition_id) DO UPDATE`
  (set path, skills_path, status, last_scanned_at); `SELECT id` back. Clear step:
  `UPDATE warnings SET is_resolved=1,... WHERE is_resolved=0 AND ((scope_type='global_provider_location'
  AND scope_id=?) OR (scope_type='global_install' AND scope_id IN (SELECT id FROM global_installs WHERE
  global_provider_location_id=?)))`. Upsert installs with
  `ON CONFLICT(global_provider_location_id, global_skill_path) DO UPDATE`; capture install id; attach
  install warning `scope_id`. **DELETE** absent installs:
  `DELETE FROM global_installs WHERE global_provider_location_id=? AND global_skill_path NOT IN (...)`
  (when none present, delete all for the location). Location warnings get `scope_id=locationID`.
- [ ] **Test first** (`global_scan_commit_test.go`, use the in-memory DB + migrate helper from
  `db_test.go`): seed `generic_agents` def id; commit one `active` location with two installs (one
  symlink/current with `skill_id`, one direct/current); assert one `global_provider_locations` row and
  two `global_installs` rows; re-commit with only the first install + a `broken_symlink` warning on it
  → assert the second install row is **deleted**, the prior warning cleared, the new
  `global_install`-scoped warning present with the right `scope_id`, and a
  `global_provider_location` warning (e.g. on a `missing` recommit) clears/regenerates.
- [ ] Run `cd core-go && go test ./internal/repositories/ -run TestGlobalScanRepo -v` → PASS.
  Stop for PM checkpoint.

### Task 5 — Repo: `GlobalLocationRepo` (provider lookup + list read)
**Files:** create `core-go/internal/repositories/global_location_repo.go` (+ `global_location_repo_test.go`).
- [ ] Methods:

```go
type GlobalLocationRepo struct{ db *sql.DB }
func NewGlobalLocationRepo(db *sql.DB) *GlobalLocationRepo { return &GlobalLocationRepo{db: db} }

// ProviderDef returns id, display_name, status for a provider key (for the service to scan it).
func (r *GlobalLocationRepo) ProviderDefByKey(ctx context.Context, key string) (id int64, displayName, status string, err error)

// ListForView returns persisted locations (any status) joined to provider_definitions,
// each with its global_installs (ordered by skill_name) and active warnings
// (global_provider_location scope + global_install scope).
func (r *GlobalLocationRepo) ListForView(ctx context.Context) ([]domain.GlobalLocationView, error)
```
- [ ] `ListForView` query: `global_provider_locations gl JOIN provider_definitions pd
  ON pd.id=gl.provider_definition_id` for the headers; per location fetch
  `global_installs WHERE global_provider_location_id=? ORDER BY skill_name`; and active warnings via
  the scope OR-clause from Task 4 step (filter `is_resolved=0`). Map warning rows into
  `domain.Warning` (ScopeType/ScopeID/Severity/Code/Message/ActionKey).
- [ ] **Test first:** after a `CommitGlobalScan` (reuse Task 4 seed), `ListForView` returns one location
  with correct `ProviderKey="generic_agents"`, `ProviderDisplayName`, `ProviderStatus`, status, the two
  installs in `skill_name` order, and any active warnings. `ProviderDefByKey("generic_agents")` returns
  a non-zero id.
- [ ] Run `cd core-go && go test ./internal/repositories/ -run TestGlobalLocationRepo -v` → PASS.
  Stop for PM checkpoint.

---

# Checkpoint 2 — Provider detection, service, operation, RPC, contracts

### Task 6 — Provider: `GlobalProviderAdapter` + `GenericAgentsAdapter.DetectGlobal`
**Files:** create `core-go/internal/providers/global_adapter.go`; modify
`core-go/internal/providers/generic_agents.go`; create
`core-go/internal/providers/generic_agents_global_test.go`.
- [ ] `global_adapter.go`:

```go
package providers

import "github.com/astraler/skillbox/core-go/internal/domain"

type GlobalDetectResult struct {
	Present          bool
	GlobalPath       string
	GlobalSkillsPath string
	Status           domain.GlobalLocationStatus
	Entries          []AdapterEntry
	Warnings         []AdapterWarning
}

// GlobalProviderAdapter is implemented only by adapters with a global level.
type GlobalProviderAdapter interface {
	ProviderAdapter
	DetectGlobal(homeDir string, fs FsReader) (GlobalDetectResult, error)
}
```
- [ ] Add `GenericAgentsGlobalDetectPath = ".agents"` / `GenericAgentsGlobalSkillsPath = ".agents/skills"`
  consts (or reuse the existing relative consts joined to `homeDir`).
- [ ] Implement `DetectGlobal` on `*GenericAgentsAdapter` mirroring `Detect` but rooted at
  `filepath.Join(homeDir, ".agents")`, returning `GlobalDetectResult` and mapping states to
  `domain.GlobalLocationStatus` per spec §4.2:
  - `~/.agents` missing → `Present=false`, `Status=missing`, warning
    `global_provider_location_missing` (scope `WarningScopeGlobalProviderLocation`, severity warning,
    action `rescan`).
  - `~/.agents` not a readable dir / is a file → `Status=invalid_structure` + warning.
  - `~/.agents/skills` missing → `Status=missing`, no entries (**no folder creation**).
  - `~/.agents/skills` readable, 0 entries → `Status=empty`.
  - `~/.agents/skills` readable, ≥1 entry → `Status=active`, entries via
    `fs.ListSkillEntries` + `entryFromProjectEntry` (same as `Detect`).
  - `~/.agents/skills` unreadable → `Status=unreadable` + warning.
- [ ] **Test first** (`generic_agents_global_test.go`, mirror `generic_agents_test.go`’s fake `FsReader`):
  one case per rule above; assert no write occurs (fake fs is read-only). Assert symlink/dir entries are
  returned as raw `AdapterEntry` (classification is the service’s job, not the adapter’s).
- [ ] Run `cd core-go && go test ./internal/providers/ -run TestGenericAgentsAdapter_DetectGlobal -v` → PASS.
- [ ] `ClaudeAdapter` is **not** modified (does not implement `GlobalProviderAdapter`). Stop for PM checkpoint.

### Task 7 — Filesystem: `HomeDir()` and narrow global FS interface
**Files:** modify `core-go/internal/filesystem/gateway.go`; modify `core-go/internal/services/interfaces.go`.
- [ ] Add to the gateway: `func (g *Gateway) HomeDir() (string, error) { return os.UserHomeDir() }`
  (read-only; no writes).
- [ ] **Do not broaden** the existing `Filesystem` interface (it is host-scan specific and many host
  tests mock it). Instead add a dedicated interface in `services/interfaces.go`:

```go
// GlobalFilesystem provides the read-only filesystem operations needed by GlobalSkillsService.
// filesystem.Gateway satisfies this interface.
type GlobalFilesystem interface {
	HomeDir() (string, error)
	PathInfo(path string) (filesystem.PathInfo, error)
	ListSkillEntries(skillsPath string) ([]filesystem.ProjectEntry, error)
}
```

- [ ] Run `cd core-go && go build ./...` → OK (only new global-service fakes need `HomeDir`).
  Stop for PM checkpoint.

### Task 8 — Service: `GlobalSkillsService.ScanGlobal`
**Files:** create `core-go/internal/services/global_skills_service.go` (+ `global_skills_service_test.go`);
modify `internal/services/interfaces.go`.
- [ ] Add lock-target constants (exported from the service file):
  `const GlobalScanTargetType = "global_scan"` and `const GlobalScanTargetID int64 = 0`.
- [ ] Add/extend service interfaces in `interfaces.go`:
  - `ProviderRegistry` gains `Get(key string) (providers.ProviderAdapter, bool)` because the concrete
    registry already has it and this slice must fetch only `generic_agents`.
  - Add `GlobalRepo`/`GlobalScanWriter` interface pairs matching Tasks 4–5 method sets.
  - Reuse existing `SkillHostLister` (`ListAll(ctx)`) and `SkillsByHostLister` (`ListByHost(ctx, hostID)`)
    for host-summary assembly. **Do not** pretend `HostRepo` can list all hosts.
- [ ] Constructor:

```go
func NewGlobalSkillsService(
	globalRepo GlobalRepo,
	scanRepo GlobalScanWriter,
	settingsRepo AppSettingsRepo,
	hostLister SkillHostLister,
	skillsByHost SkillsByHostLister,
	registry ProviderRegistry,
	fs GlobalFilesystem,
	runner OperationRunner,
) *GlobalSkillsService
```
- [ ] `ScanGlobal(ctx)` queues the op:

```go
target := operations.Target{Type: GlobalScanTargetType, ID: GlobalScanTargetID}
opID, err := s.runner.Start(ctx, target, domain.OperationTypeScanGlobalSkills,
	func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
		return s.scanGlobalInternal(opCtx, progress)
	})
```
  Map a non-`*domain.AppError` to `NewDatabaseError`; propagate `conflict_error` unchanged.
- [ ] `scanGlobalInternal`:
  1. `homeDir, err := s.fs.HomeDir()` → map err to `NewFilesystemError`.
  2. `defID, _, _, err := s.globalRepo.ProviderDefByKey(ctx, providers.GenericAgentsKey)` (Alt-A gate:
     only `generic_agents` this slice; Claude is not enumerated).
  3. `adapter, ok := s.registry.Get(providers.GenericAgentsKey)`; type-assert
     `ga, ok := adapter.(providers.GlobalProviderAdapter)`; if not ok, return summary with 0 entries.
  4. `res, err := ga.DetectGlobal(homeDir, s.fs)`.
  5. Build `[]HostSummary` exactly as the project scan full service does: active host id from
     `settingsRepo.Get(ctx)`, all hosts from `hostLister.ListAll(ctx)`, skills per host from
     `skillsByHost.ListByHost(ctx, hostID)`, with the active host first. Do not call non-existent
     `HostRepo.ListAll`.
  6. For each `res.Entries`, `c := ClassifyAdapterEntry(entry, hosts)`; build
     `repositories.GlobalInstallScanResult{ SkillID:c.SkillID, SkillName:entry.Name, InstallMode:c.Mode,
     InstallStatus:c.Status, GlobalSkillPath:entry.Path, SourceSkillPath:c.SourceSkillPath,
     SymlinkTargetPath:c.SymlinkTargetPath, InstalledFromHostFolderID:c.InstalledFromHostFolderID }`.
     If status is `broken_symlink`/`external_symlink`/`old_host`, attach a `global_install`-scoped
     `domain.Warning` (code `broken_symlink`/`external_symlink`/`old_host_symlink`, action `rescan`).
  7. Convert `res.Warnings` (adapter, location-scoped) to `[]domain.Warning` with
     `ScopeType=WarningScopeGlobalProviderLocation`.
  8. `pathPtr/skillsPtr` from `res.GlobalPath`/`res.GlobalSkillsPath` (nil when empty);
     `s.scanRepo.CommitGlobalScan(ctx, defID, pathPtr, skillsPtr, res.Status, installs, locWarnings, now)`.
  9. Return `globalScanSummary{ EntriesFound: len(installs), WarningsCreated: ... }`.
- [ ] `progress("reading_global_location",...)`, `progress("classifying_entries",...)`, `progress("done",...)`
  like `skill_host_service.go`.
- [ ] **Test first** (`global_skills_service_test.go`): set `HOME` fixture (temp dir) with
  `.agents/skills` containing a real symlink into an active host skills dir (→ `current`, `skill_id` set)
  and a plain dir (→ `direct`/`current`); run `ScanGlobal`, drive the op to completion, then assert via
  `GlobalLocationRepo.ListForView` that the location is `active` with two installs. Add cases:
  `.agents/skills` absent → location `missing`, `global_provider_location_missing` warning, **folder not
  created**; a broken symlink → `broken_symlink` install + `global_install` warning; second scan after
  removing an entry → that install row deleted and its warning cleared (regeneration). Add a
  concurrency test asserting a second `ScanGlobal` while one is in-flight returns `conflict_error`
  (mirror `runner_test.go` style).
- [ ] Run `cd core-go && go test ./internal/services/ -run TestGlobalSkillsService -v` → PASS.
  Stop for PM checkpoint.

### Task 9 — Service: `ListGlobal`
**Files:** modify `core-go/internal/services/global_skills_service.go` (+ test).
- [ ] Add `func (s *GlobalSkillsService) ListGlobal(ctx) ([]domain.GlobalLocationView, error)` delegating
  to `globalRepo.ListForView`; map DB error → `NewDatabaseError`. Read-only.
- [ ] **Test first:** after a scan, `ListGlobal` returns the location view with entries + warnings.
- [ ] Run `cd core-go && go test ./internal/services/ -run TestGlobalSkillsService_ListGlobal -v` → PASS.
  Stop for PM checkpoint.

### Task 10 — Contracts: `global.scan.json` + `global.list.json` + index + regen
**Files:** create `shared/api-contracts/methods/global.scan.json`, `global.list.json`; modify
`shared/api-contracts/index.json`.
- [ ] `global.scan.json`: request `{}` (no params); response `GlobalScanResponse`
  `{ "operationId": { "type": "integer" } }`, required `["operationId"]`, `additionalProperties:false`.
- [ ] `global.list.json`: request `{}`; response `GlobalListResponse` =
  `{ "locations": { "type":"array", "items": <GlobalListLocation> } }`, required `["locations"]`.
  `GlobalListLocation` props (all `additionalProperties:false`):
  `globalProviderLocationId:integer`, `providerKey:string`, `providerDisplayName:string`,
  `providerStatus:string`, `path:{type:["string","null"]}`, `skillsPath:{type:["string","null"]}`,
  `status:{enum:["active","not_configured","missing","unreadable","invalid_structure","empty","disabled"]}`,
  `lastScannedAt:{type:["string","null"]}`, `entries:array<GlobalListEntry>`,
  `warnings:array<GlobalListWarning>`. Required: all except nullable scalars per existing method-schema
  conventions (match `project.get.json` nullability style).
  `GlobalListEntry`: `globalInstallId:integer`, `skillName:string`, `skillId:{type:["integer","null"]}`,
  `mode:{enum:["symlink","rsync_copy","direct"]}`,
  `status:{enum:["current","outdated","missing","broken_symlink","old_host","external_symlink","conflict","needs_sync","error"]}`,
  `globalSkillPath:string`, `sourceSkillPath:{type:["string","null"]}`,
  `symlinkTargetPath:{type:["string","null"]}`.
  `GlobalListWarning`: `code:string`, `severity:{enum:["info","warning","error","blocking"]}`,
  `scopeType:{enum:["global_provider_location","global_install"]}`, `scopeId:{type:["integer","null"]}`,
  `actionKey:{type:["string","null"]}`, `message:string`.
- [ ] In `index.json` `schemas` array append:
  `{ "input": "methods/global.scan.json", "output": "methods/global-scan.ts" }` and
  `{ "input": "methods/global.list.json", "output": "methods/global-list.ts" }`.
- [ ] Run `cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift` → clean
  (commit `shared/generated/` is the PM’s job). Stop for PM checkpoint (include `shared/generated` in review).

### Task 11 — Handlers: `global.scan` + `global.list`
**Files:** create `core-go/internal/rpc/handlers/global_scan.go`, `global_list.go`; modify
`core-go/internal/rpc/handlers/contract_test.go`.
- [ ] `global_scan.go`: `globalScanService interface { ScanGlobal(ctx) (int64, error) }`; handler returns
  `{ operationId int json:"operationId" }`. Mirror `host_scan.go` error mapping (pass through
  `*domain.AppError`).
- [ ] `global_list.go`: `globalListService interface { ListGlobal(ctx) ([]domain.GlobalLocationView, error) }`;
  map view → response structs mirroring the contract; default `entries`/`warnings`/`locations` to `[]`
  (never null); map `*int64`/`*string` to JSON null.
- [ ] **Contract test first** in `contract_test.go` (mirror existing `skill.list`/`project.get` helpers):
  build a sample `GlobalListResponse` and validate it against `global.list.json` `GlobalListResponse`,
  and `{operationId:1}` against `global.scan.json` `GlobalScanResponse`.
- [ ] Run `cd core-go && go test ./internal/rpc/handlers/ -v` → PASS. Stop for PM checkpoint.

### Task 12 — Wiring + capabilities
**Files:** modify `core-go/internal/app/wire.go`, `core-go/internal/app/wire_test.go`,
`core-go/cmd/skillbox-core/main.go`.
- [ ] **Test first:** add `"global.scan"` and `"global.list"` to the expected method set in
  `wire_test.go` `TestAllMethodsRegistered` → FAIL.
- [ ] In `cmd/skillbox-core/main.go`: construct `globalScanRepo := repositories.NewGlobalScanRepo(db)`,
  `globalLocationRepo := repositories.NewGlobalLocationRepo(db)`, then
  `globalSvc := services.NewGlobalSkillsService(globalLocationRepo, globalScanRepo, appSettingsRepo,
  hostRepo, skillRepo, providerRegistry, fs, runner)`. Pass `globalSvc` into `app.New(...)`.
- [ ] In `app/wire.go`: update `app.New(...)` signature to accept the already-built
  `*services.GlobalSkillsService`; register `"global.scan": rpchandlers.NewGlobalScanHandler(globalSvc)`
  and `"global.list": rpchandlers.NewGlobalListHandler(globalSvc)`. Do **not** construct repositories
  or services in `wire.go`; it only registers handlers.
- [ ] In `wire_test.go`: update `app.New` test helpers for the new parameter and expected method set.
- [ ] In `main.go`: add `"global.scan"`, `"global.list"` to the `capabilities` slice.
- [ ] Run `cd core-go && go test ./internal/app/ ./cmd/... -v` → PASS. Stop for PM checkpoint.

---

# Checkpoint 3 — Renderer UI, hooks, routing, sidebar

### Task 13 — Electron allowlist + core-client methods + query key
**Files:** modify `apps/desktop/electron/main/core-process/method-allowlist.ts`,
`apps/desktop/renderer/src/lib/core-client/methods.ts`,
`apps/desktop/renderer/src/lib/query-keys.ts`.
- [ ] Add `"global.scan"` and `"global.list"` to the `ALLOWLIST` set.
- [ ] In `methods.ts` import `GlobalScanResponse`, `GlobalListResponse` from `@contracts/index.js`; add
  `scanGlobal: () => invoke<GlobalScanResponse>("global.scan", {})` and
  `listGlobal: () => invoke<GlobalListResponse>("global.list", {})`.
- [ ] In `query-keys.ts` add a `global: { list: () => ["global", "list"] as const }` group.
- [ ] Run `cd apps/desktop && pnpm typecheck` → PASS. Stop for PM checkpoint.

### Task 14 — Hooks: `use-global-list` + `use-scan-global`
**Files:** create `apps/desktop/renderer/src/features/global-skills/use-global-list.ts`,
`.../use-scan-global.ts`; tests in `.../__tests__/`.
- [ ] `use-global-list.ts`: `useQuery({ queryKey: queryKeys.global.list(), queryFn: () => methods.listGlobal() })`.
- [ ] `use-scan-global.ts`: copy `useScanHost` structurally (subscribe-all-then-RPC buffer + progress
  subscription + sonner toasts), but `mutationFn` calls `methods.scanGlobal()` (no args) and on terminal
  success invalidates `queryKeys.global.list()`. Toasts: "Scanning global skills…" / "Global skills scanned".
- [ ] **Test first** (`__tests__/use-scan-global.test.tsx`, mirror `use-scan-project.test.tsx`): mock
  `methods.scanGlobal` + progress; assert it invalidates `["global","list"]` on terminal success; and
  `use-global-list.test.tsx`: fetches via `methods.listGlobal`.
- [ ] Run `cd apps/desktop && pnpm test -- global-skills` → PASS. Stop for PM checkpoint.

### Task 15 — Sidebar + route + `GlobalSkillsScreen`
**Files:** modify `apps/desktop/renderer/src/components/sidebar.tsx`,
`apps/desktop/renderer/src/app/router.tsx`; create
`apps/desktop/renderer/src/screens/global-skills-screen.tsx`; tests in `screens/__tests__/` and
`components/__tests__/sidebar.test.tsx`.
- [ ] `sidebar.tsx`: import `Globe` from `lucide-react`; insert
  `{ to: "/global", label: "Global Skills", icon: Globe }` into `NAV_ITEMS` **between** the `/skills` and
  `/projects` entries.
- [ ] `router.tsx`: import `GlobalSkillsScreen`; add
  `const globalRoute = createRoute({ getParentRoute: () => shellRoute, path: "/global", component: GlobalSkillsScreen })`;
  add `globalRoute` to `shellRoute.addChildren([...])`.
- [ ] `global-skills-screen.tsx` (read-only): use `useGlobalList()` + `useScanGlobal()`. Header buttons
  `[Scan Global]` (calls `scanGlobal.mutate()`, disabled while `operationId != null || isPending`) and
  `[Open Folder]` (calls `methods.openPath(location.skillsPath)` when present). Render a **Global Locations**
  table (Provider / Path / Status / entry count) and a **Global Entries** table grouped by provider
  (Provider / Skill / Mode / Status / `[Open]` → `methods.openPath(entry.globalSkillPath)`); render
  `warnings` rows beneath. Empty state when `locations` is empty or has no entries:
  **"No global skills found."** with only `[Scan Global]` (no Configure button). Show
  `providerDisplayName` (already "Shared Agent Skills (.agents)"). **No Relink/Remove/Install controls.**
- [ ] **Test first** (`screens/__tests__/global-skills-screen.test.tsx`, mock both hooks + `methods.openPath`,
  like `dashboard-screen.test.tsx`): renders a location + entries with statuses and a warning row; empty
  state shows only Scan Global; Open Folder calls `methods.openPath` with `skillsPath`; asserts **no**
  write-action controls (no "Relink"/"Remove"/"Install" text). Add `sidebar.test.tsx` assertion that
  "Global Skills" renders between Skills and Projects; add a router test that `/global` resolves.
- [ ] Run `cd apps/desktop && pnpm test -- global-skills-screen sidebar router` → PASS.
  Stop for PM checkpoint.

---

# Checkpoint 4 — Full validation and manual smoke

### Task 16 — Full validation
- [ ] `cd core-go && go test ./...` → all PASS.
- [ ] `cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...` → PASS.
- [ ] `cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test` → all clean/PASS.

### Task 17 — Manual smoke (`pnpm dev`)
- [ ] With a real `~/.agents/skills` containing one symlink into the active host + one plain dir:
  open **Global Skills** → **Scan Global** → location shows `active`, both entries appear with correct
  mode/status; the symlinked entry shows `current`.
- [ ] `[Open Folder]` opens `~/.agents/skills`; per-row `[Open]` opens the entry path.
- [ ] Introduce a broken symlink, rescan → entry shows `broken symlink` with a warning row.
- [ ] Temporarily remove `~/.agents/skills`, rescan → location shows `missing` and the folder is **not**
  created (verify on disk).
- [ ] Confirm no Relink/Remove/Install controls exist anywhere on the screen.
- [ ] Report results to PM. (PM commits after lead review — **tech does not commit**.)

---

## Validation Commands

```bash
cd core-go && go test ./...
cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...
cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test
```

---

## Acceptance Mapping (plan → spec §9)

- **AC1** (migration creates tables, flips `generic_agents.has_global_level`, bump v5, reverting down,
  Claude untouched) → Tasks 1–2.
- **AC2** (read-only `global.scan`, no folder creation, stable `global_scan/0` lock,
  `scan_global_skills`, returns `operationId`, `conflict_error` on concurrency) → Tasks 3, 6, 8, 11.
- **AC3** (`global.list` full identifier/path field set + warnings scope/action fields) → Tasks 5, 9, 10, 11.
- **AC4** (project-install symlink classification via `ClassifyAdapterEntry`; never merged with project
  installs) → Task 8 (classification) + Task 15 (separate screen/tables).
- **AC5** (warning scopes `global_provider_location` + `global_install`, regenerated each scan) →
  Tasks 3 (enums), 4 (clear+insert), 8 (emit).
- **AC6** (sidebar item, `/global`, read-only screen, only Scan Global / Open Folder, empty state) →
  Task 15.
- **AC7** (deferred items absent) → enforced throughout; verified Task 15 test (no write controls) + Task 17.
- **AC8** (contract plumbing end to end) → Tasks 10–14.
- **AC9** (`generic_agents` shown as "Shared Agent Skills (.agents)") → Task 5 (display name from DB,
  migration 000004) + Task 15 render + Task 17 smoke.
- **AC10** (all gates pass) → Task 16.

---

## Draft `/goal` (NOT run during planning)

> Implement Slice 3K per
> `docs/superpowers/plans/2026-05-26-skillbox-slice-3k-global-skills-readonly-implementation-plan.md` in
> the current main workspace (no worktree, no subagents). Done when: (1) migration `000005` creates
> `global_provider_locations` + `global_installs`, sets `generic_agents.has_global_level=1` and
> `database_version=5`, with a reverting down migration, leaving Claude untouched; (2) `global.scan` runs
> a read-only scan of `~/.agents/skills` only (no folder creation) under the stable `global_scan/0` lock
> with op type `scan_global_skills`, returns `operationId`, and returns `conflict_error` on concurrent
> runs; (3) entries are classified by reusing `ClassifyAdapterEntry` against known hosts; (4) warnings
> use scopes `global_provider_location` + `global_install` and are regenerated each scan; (5)
> `global.list` returns the full location/entry/warning field set from spec §5.2; (6) a Global Skills
> sidebar item, `/global` route, and read-only screen ship with only Scan Global / Open Folder; (7)
> deferred items (overlap, install-to-global, remove/relink, Settings configure, Dashboard count) are
> absent; (8) full contract plumbing is wired (global.scan.json, global.list.json, index.json, generated
> TS, Electron allowlist, core-client methods, query key, hooks); (9) `generic_agents` shows as "Shared
> Agent Skills (.agents)"; (10) `go test ./...`, `pnpm check:contracts-drift`, `pnpm typecheck`,
> `pnpm test` all pass. **Tech does not commit** — PM commits after lead review. Implement on Sonnet
> after approval. **Draft only — not executed during planning.**

---

## Self-Review

- **Spec coverage:** every spec section maps to a task — §3.1 migration → T1/T2; §3.3 enums → T3;
  §4.1/§4.2 adapter → T6; §4.3 classification reuse → T8 (verified `ClassifyAdapterEntry` returns
  `direct`/`current` for plain dirs, matching the spec’s §3.2 status subset and §4.3 wording); §5.1
  service + stable `global_scan/0` lock + read-only → T7/T8; §5.2 RPC shapes → T10/T11; §5.3 plumbing →
  T10–T14; §6 UI → T15; §7 warnings + regeneration → T3/T4/T8; §8 tests → every task is test-first +
  T16/T17.
- **Placeholder scan:** none. "Mirror X" notes point at real files that exist
  (`migration_000004_test.go`, `generic_agents_test.go`, `host_scan.go`, `use-scan-host.ts`,
  `use-scan-project.test.tsx`, `project_scan_commit.go`, `dashboard-screen.test.tsx`).
- **Type/name consistency:** lock constants `GlobalScanTargetType`/`GlobalScanTargetID` (T8) match
  spec §5.1; `domain.GlobalLocationStatus` values (T3) match the `global.list` status enum (T10) and the
  adapter result (T6); `GlobalInstallScanResult` fields (T4) align with `ClassifyAdapterEntry` outputs
  (`SourceSkillPath`/`SymlinkTargetPath`/`InstalledFromHostFolderID`, T8) and the upsert columns;
  `GlobalLocationView`/`GlobalInstallView` (T3) feed `ListForView` (T5) → handler (T11) → contract (T10)
  with matching field names; renderer `methods.scanGlobal`/`listGlobal` (T13), `queryKeys.global.list`
  (T13), `useScanGlobal`/`useGlobalList` (T14), and `/global` route param-free screen (T15) are
  consistent.
- **Open item flagged for PM/lead (not a blocker):** the exact repo method names for listing a host’s
  skills when assembling `[]HostSummary` (T8 step 5) must reuse whatever `project_scan_full_service`
  already calls — the implementer should follow that file rather than introduce a new method name.
