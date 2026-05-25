# Slice 2A: Read-Only Projects And Generic Agents Scan — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read-only Projects (nav/list/detail, Add Project, scan) with `generic_agents` detection and filesystem entry classification, with zero write-path into any project folder.

**Architecture:** Extend the Slice 1 layers (do not refactor). M1 lands schema + domain; M2 locks JSON Schema contracts; M3 builds the Go core (read-only FS, provider adapter, repos, services, handlers) with TDD; M4 wires the React UI; M5 verifies and documents.

**Tech Stack:** Electron + React + TanStack, Go, modernc.org/sqlite, golang-migrate, creachadair/jrpc2, JSON Schema, Vitest, `go test`.

---

## Source Of Truth

- Design spec: `docs/superpowers/specs/2026-05-25-skillbox-slice-2a-read-only-projects-design.md`
- Data model / schema: `docs/06-data-model.md`, `docs/07-schema-dictionary.md`
- Provider model: `docs/08-provider-model.md`
- Patterns + boundaries: `docs/12-implementation-patterns.md`, `CLAUDE.md`
- Slice 1 plan (reference for style): `docs/superpowers/plans/2026-05-25-skillbox-scaffold-slice-1-plan.md`

## Commit Boundaries

- M1: `Add slice 2A projects migration and domain`
- M2: `Add slice 2A project API contracts`
- M3: `Add slice 2A read-only project scan core`
- M4: `Add slice 2A projects UI`
- M5: `Add slice 2A smoke docs`

Do not stage unrelated untracked files (`AGENTS.md`, `CLAUDE.md`). All `pnpm` commands run from `apps/desktop`; all `go` commands from `core-go`.

## Cross-Milestone Conventions (keep names consistent)

- Migration files: `core-go/migrations/000002_projects.up.sql` + `.down.sql` (embedded via existing `*.sql` glob). Seed `generic_agents` definition + `.agents` (detect) and `.agents/skills` (skills) candidates **in the migration SQL** via `INSERT OR IGNORE`. Do NOT create `scan_results`.
- New domain enums/structs: `ProjectStatus`, `Project`; `ProviderStatus`, `DetectionStatus`, `ProviderDefinition`, `ProjectProvider`; `InstallMode`, `InstallStatus`, `Install`. Add `WarningScopeProject/ProjectProvider/Install` to `warning.go` and `NewProviderError` to `errors.go` (code `provider_error=1003` already in the map).
- RPC methods: `project.add`, `project.list`, `project.get`, `project.scan`; Electron-handled `dialog.openProjectFolder`. Reuse `operation.progress` (phase is free-text — no contract change). 2A phases: `reading_project`, `detecting_providers`, `classifying_entries`, `done`.
- Read-only invariant: the filesystem gateway must expose **no** project-write method (no symlink/copy/remove). This is the slice's primary guard.

---

## M1: Schema + Domain

**Files:** Create `core-go/migrations/000002_projects.up.sql`, `core-go/migrations/000002_projects.down.sql`, `core-go/internal/repositories/migration_000002_test.go`, `core-go/internal/domain/project.go`, `core-go/internal/domain/provider.go`, `core-go/internal/domain/install.go`, `core-go/internal/domain/project_test.go`. Modify `core-go/internal/domain/warning.go`, `core-go/internal/domain/errors.go`.

- [ ] **Step 1 (test-first):** In `migration_000002_test.go` reuse `NewTestDB(t)` (from `db_test.go`) to assert tables `projects, provider_definitions, provider_path_candidates, project_providers, installs` exist, `scan_results` does NOT, one `generic_agents` definition row exists, and exactly 2 candidate rows (`detect`/`.agents`, `skills`/`.agents/skills`). Run `go test ./internal/repositories -run Migration000002` → FAIL.
- [ ] **Step 2:** Write up/down SQL bound to `docs/07` field-for-field. Tables get `UNIQUE` on `projects.path`, `provider_definitions.key`, `(project_id, provider_definition_id)`, `(project_provider_id, project_skill_path)`. FKs: `project_providers→projects/provider_definitions`, `installs→project_providers/skills/skill_host_folders`. Re-run → PASS.
- [ ] **Step 3:** Add domain types + the warning scopes + `NewProviderError`. Add a focused `project_test.go` asserting new enum string values and `NewProviderError().RPCCode()==1003`. Run `go test ./internal/domain` → PASS.
- [ ] **Step 4 (commit):** `go test ./...`, then commit migration + domain files only.

**Acceptance:** Migration 000002 creates the five tables + seed (no `scan_results`); `go test ./internal/repositories ./internal/domain` green.

---

## M2: API Contracts

**Files:** Create `shared/api-contracts/methods/project.add.json`, `project.list.json`, `project.get.json`, `project.scan.json`, `shared/api-contracts/electron/dialog.openProjectFolder.json`. Modify `shared/api-contracts/index.json`. Regenerate `shared/generated/**`.

- [ ] **Step 1:** Author schemas mirroring existing files (draft-07, `oneOf` request/response, `definitions`, `additionalProperties:false` on **both** request and response to match Slice 1 files). Shapes per spec §8: `project.add{path}→{projectId,name,path,status}`; `project.list{}→{projects:[{id,name,path,status,providers:[{key,displayName,providerStatus,detectionStatus}],skillCount,warningCount,lastScannedAt}]}`; `project.get{projectId}→{project,providers:[…entryCount],entries:[{id,projectProviderId,providerKey,name,mode,status,projectSkillPath,symlinkTargetPath,skillId}],warnings:[{code,severity,message,scopeType,scopeRef,actionKey}]}`; `project.scan{projectId}→{operationId}`; `dialog.openProjectFolder{}→{path:string|null}`. Enums: project status `active|missing|unreadable`; entry mode `symlink|direct`; entry status `current|old_host|external_symlink|broken_symlink|missing|error`.
- [ ] **Step 2:** Add the five entries to `index.json`. Run `pnpm install` then `pnpm generate:contracts` then `pnpm check:contracts-drift` → no drift.
- [ ] **Step 3 (commit):** Commit `shared/api-contracts` + `shared/generated`.

**Acceptance:** `pnpm check:contracts-drift` passes; generated TS types exist for all new methods.

---

## M3: Go Core (read-only scan)

**Files:** Create `core-go/internal/filesystem/scan_project.go` (+`scan_project_test.go`); `core-go/internal/providers/adapter.go`, `registry.go`, `generic_agents.go` (+`generic_agents_test.go`); `core-go/internal/repositories/project_repo.go`, `provider_definition_repo.go`, `project_provider_repo.go`, `install_repo.go`, `project_scan_commit.go` (+ `*_test.go`); `core-go/internal/services/project_service.go` (+`project_service_test.go`, `project_mocks_test.go`); `core-go/internal/rpc/handlers/project_add.go`, `project_list.go`, `project_get.go`, `project_scan.go`. Modify `core-go/internal/filesystem/gateway.go`, `core-go/internal/repositories/skill_host_folder_repo.go` (+ its interface), `core-go/internal/services/interfaces.go`, `core-go/internal/rpc/handlers/contract_test.go`, `core-go/internal/rpc/handlers/handler_test.go`, `core-go/internal/app/wire.go`, `core-go/cmd/skillbox-core/main.go`, `apps/desktop/electron/main/core-process/method-allowlist.ts`.

- [ ] **Step 1 — Filesystem read methods (TDD):** Add `PathInfo{Exists,IsDir,Readable}`, `ProjectEntry{Name,Path,IsDir,IsSymlink,SymlinkTargetRaw,ResolvedTarget,Broken,ResolveError}`, package funcs `ValidateProjectPath` (exists+dir, **not** writable), `StatPathInfo` (follows symlinks; `ENOENT→Exists:false`), `ScanProjectSkills` (top-level only; `EvalSymlinks` → `ErrNotExist`=Broken else other err=ResolveError). Surface via `Gateway.{ValidateProjectPath,PathInfo,ListSkillEntries}`; reuse `NormalizeAbs`/`Realpath`. Tests use `t.TempDir()` + real symlinks. Run `go test ./internal/filesystem` → PASS. No write methods added.
- [ ] **Step 2 — Provider adapter (TDD):** `ProviderAdapter{Key();Detect(root,FsReader)}`, `FsReader{PathInfo;ListSkillEntries}` (gateway satisfies it), `DetectResult{Present,DetectedPath,SkillsPath,DetectionStatus,Entries,Warnings}`, `AdapterEntry`, `AdapterWarning`, `Registry`. `GenericAgentsAdapter` hardcodes `.agents`/`.agents/skills` and implements spec §6 rules (missing→`Present=false`+`no_provider_detected`; file/unreadable→`invalid_structure`; empty-skills→detected/0 entries; never writes). Adapter returns raw facts only — host comparison stays in the service. Tests use real temp dirs. Run `go test ./internal/providers` → PASS.
- [ ] **Step 3 — Repositories (TDD):** `ProjectRepo` (UpsertByPath idempotent on normalized path — no realpath; GetByID, List, UpdateStatus, UpdateLastScannedAt); `ProviderDefinitionRepo.GetByKey`; `ProjectProviderRepo.ListByProject` (joined `provider_definitions` view incl. per-provider `entryCount` subquery); `InstallRepo.ListByProject` (joined entry view); `SkillHostFolderRepo.ListAll` (active + inactive); `WarningRepo.CountActiveForProject` + `ListActiveForProject` (project + project_provider + install scopes via subqueries). `ProjectScanRepo.CommitProjectScan` and `CommitProjectTerminal` in `project_scan_commit.go`, mirroring `scan_commit.go`: a single tx clears project/provider/install-scoped active warnings, upserts `project_providers` (or marks `missing` when absent per §6 rule 7), upserts/reconciles `installs` by `(project_provider_id, project_skill_path)`, marks absent installs `missing` (never hard-delete), inserts install-scoped warnings using the new install id, updates `last_scanned_at`/`projects.status`. Terminal variant updates only `projects.status` + project-scoped warning and skips provider/install mutation. Run `go test -race ./internal/repositories` → PASS.
- [ ] **Step 4 — Service (TDD):** `ProjectService.{AddProject,ScanProject,scanProjectInternal,ListProjects,GetProject}`; extend `interfaces.go` (`ProjectRepo`, `ProviderDefinitionRepo`, `ProjectProviderRepo`, `InstallRepo`, `ProjectScanCommitter`, `ProviderRegistry`, `ProjectFilesystem`; reuse `HostRepo`+`ListAll`, `SkillRepo`, `OperationRunner`). `scanProjectInternal`: emit phases; on missing/unreadable root call terminal commit and return; else adapter.Detect → classify entries vs known hosts (active→`current`, inactive→`old_host`, outside→`external_symlink`, broken→`broken_symlink`, loop/IO→`error`, plain dir→`direct/current`, other→`direct/error`); `skill_id` strict match (host-relative `relative_path`, else unique `absolute_path` canonical, else null) per §7; commit in one tx; summary→`operations.metadata_json`. `ScanProject` uses `operations.Target{Type:"project",ID}` (lock → `conflict_error`). Tests use mock repos/fs + a fake adapter returning canned `DetectResult`. Run `go test -race ./internal/services` → PASS.
- [ ] **Step 5 — Handlers + wiring (TDD):** Add four handlers following existing pattern (`wrapError`, request/response structs, camelCase). Register in `wire.go` (extend `app.New` with `projectSvc`); construct deps + `Registry` in `main.go` and add the four methods to `server.ready` capabilities. Add the four methods + `dialog.openProjectFolder` to `method-allowlist.ts`. Add response-shape cases to `contract_test.go` (validate against new schemas) and success/error cases to `handler_test.go` (incl. `project.scan` → 1005 conflict, `project.get` → 1001 unknown id). Run `go test ./internal/rpc/...` then `go test -race ./...` → PASS.
- [ ] **Step 6 — Standalone smoke:** `SKILLBOX_DB_PATH=/tmp/sb-2a.db go run ./cmd/skillbox-core`; send NDJSON `project.add` (temp dir with `.agents/skills` fixtures: symlink→active host, plain dir, broken symlink, external symlink), `project.scan`, `project.get`; verify `sqlite3` rows + `operations.metadata_json`; confirm fixture inode/mtime unchanged (read-only proof).
- [ ] **Step 7 (commit):** Commit `core-go` + `method-allowlist.ts`.

**Acceptance:** classification matches §7 for all cases; `skill_id` only set on strict unique match; empty `.agents/skills`→detected/0; `.claude` ignored; missing root→`missing`+warning, provider scan skipped; reconcile marks absent installs `missing`; per-project lock → `conflict_error`; warnings carry only `rescan`/`open_folder`; `go test -race ./...` green; gateway exposes no project-write method.

---

## M4: React UI

**Files:** Create `apps/desktop/renderer/src/features/projects/{use-projects-list,use-project-detail,use-add-project,use-scan-project}.ts`, `{provider-badge,entry-status-badge}.tsx`, `apps/desktop/renderer/src/screens/{projects-screen,project-detail-screen}.tsx`, tests under `apps/desktop/renderer/src/features/projects/__tests__/`. Modify `apps/desktop/renderer/src/lib/core-client/methods.ts`, `lib/query-keys.ts`, `app/router.tsx`, `components/sidebar.tsx`, `apps/desktop/electron/main/core-process/ipc-bridge.ts`.

- [ ] **Step 1:** Add client wrappers (`openProjectFolder`, `addProject`, `listProjects`, `getProject`, `scanProject`) importing generated types; add `projects.list` + `projects.detail(id)` query keys. Add `dialog.openProjectFolder` handler in `ipc-bridge.ts` (Electron-native, mirrors `dialog.openHostFolder`, not forwarded to Go). Test client wrappers first (`pnpm test -- core-client`).
- [ ] **Step 2:** Add routes `/projects` and `/projects/$projectId` under the shell route; add `Projects` sidebar item (`FolderGit2`).
- [ ] **Step 3:** Build hooks — `use-add-project` (mutation → invalidate `projects.list`, navigate detail), `use-scan-project` (mirror `use-scan-host`: subscribe-all-before-call, terminal handling, invalidate `projects.detail`+`projects.list`), list/detail queries. Build screens with **read-only** actions only (Add Project, Scan, Scan All, Open Folder, Rescan) — no Remove/Relink/Set-Up-Provider; warnings render as non-blocking banners with read-only actions. Reuse `error-display`, `empty-state`, `operation-progress-toast`.
- [ ] **Step 4:** Hook tests with mocked `methods` (invalidation + terminal handling). Run `pnpm test`.
- [ ] **Step 5 — Manual acceptance:** `pnpm dev` → Add Project, Scan, verify provider + classified entries + warnings; old-host rescan; no-provider project; missing project rescan; Skills Library (Slice 1) still works.
- [ ] **Step 6 (commit):** Commit `apps/desktop`.

**Acceptance:** Projects nav + both routes render; Add Project idempotent; scan progress + invalidation work; no write/action controls present.

---

## M5: Smoke Docs + Verification

**Files:** Modify `SMOKE.md` (append a "Slice 2A — Read-Only Projects" section from spec §13); modify `README.md` only if a link is needed.

- [ ] **Step 1:** Add the read-only smoke checklist (add/scan/old-host/no-provider/missing/concurrency/cancel/regression, incl. the inode/mtime read-only proof).
- [ ] **Step 2 — Final verification:** `(cd core-go && go test -race ./...)`, `(cd apps/desktop && pnpm test)`, `(cd apps/desktop && pnpm check:contracts-drift)`, `git diff --check`; then run the manual `SMOKE.md` 2A checklist on macOS.
- [ ] **Step 3 (commit):** Commit `SMOKE.md` (+ `README.md` if changed). Do not tag/release without explicit owner approval.

**Acceptance:** all automated suites green; manual 2A smoke passes; no project filesystem mutation observed.

---

## Self-Check

- Spec coverage: §5 schema→M1; §8 contracts→M2; §6/§7/§9 core→M3; §10 UI→M4; §13 smoke→M5.
- `installs` used as observed entries (mode `symlink`/`direct` only); no `rsync_copy` written; absent entries → `missing` (no hard delete).
- Seed in migration SQL (open decision §14.4 resolved → migration); `scan_results` not created (summary in `operations.metadata_json`).
- `project.add` stores normalized abs path, idempotent by it, no realpath (symlinked roots preserved); realpath only for symlink/host comparison.
- `generic_agents` adapter hardcodes its own paths (it is the generic adapter); `provider_path_candidates` seeded for the data model; other providers/markers ignored in 2A.
- Read-only invariant enforced at gateway surface; only `rescan`/`open_folder` action keys emitted.
- Contract requests use `additionalProperties:false` to match existing Slice 1 files (runtime forward-compat via Go `UnmarshalParams`).

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-25-skillbox-slice-2a-read-only-projects-implementation-plan.md`. Two execution options:

1. **Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks.
2. **Inline Execution** — execute tasks in this session via executing-plans with checkpoints.
