# Skillbox Scaffold Slice 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable Astraler Skillbox vertical slice: Electron + React + Go sidecar scaffold, JSON-RPC handshake, Skill Host Folder setup, scan, and Skills Library list.

**Architecture:** Build outside-in. M1 proves Electron ⇄ Go integration with ping; M2 locks JSON Schema contracts; M3 builds Go core with TDD; M4 wires React UI; M5 verifies and documents the scaffold.

**Tech Stack:** Electron, electron-vite, React, TanStack Router/Query, Tailwind/shadcn/ui, Go, modernc.org/sqlite, golang-migrate, creachadair/jrpc2, JSON Schema, Vitest.

---

## Source Of Truth

- Design spec: `docs/superpowers/specs/2026-05-25-skillbox-scaffold-slice-1-design.md`
- Data model: `docs/06-data-model.md`
- Schema dictionary: `docs/07-schema-dictionary.md`
- Architecture: `docs/10-technical-architecture.md`
- Implementation patterns: `docs/12-implementation-patterns.md`

## Commit Boundaries

- M1 commit: `Add Electron Go walking skeleton`
- M2 commit: `Add slice 1 API contracts`
- M3 commit: `Add Go core skill host slice`
- M4 commit: `Add skill host and library UI`
- M5 commit: `Add scaffold smoke docs`

Do not include unrelated untracked files such as `AGENTS.md` or `CLAUDE.md` unless explicitly requested.

---

## M1: Walking Skeleton

**Objective:** `pnpm dev` opens Electron, spawns Go, receives `server.ready`, and supports React → Go `ping`.

**Files:**
- Create: `apps/desktop/package.json`
- Create: `apps/desktop/electron.vite.config.ts`
- Create: `apps/desktop/tsconfig.json`
- Create: `apps/desktop/tsconfig.node.json`
- Create: `apps/desktop/tsconfig.web.json`
- Create: `apps/desktop/electron/main/index.ts`
- Create: `apps/desktop/electron/main/core-process/manager.ts`
- Create: `apps/desktop/electron/main/core-process/method-allowlist.ts`
- Create: `apps/desktop/electron/main/core-process/ipc-bridge.ts`
- Create: `apps/desktop/electron/preload/index.ts`
- Create: `apps/desktop/renderer/index.html`
- Create: `apps/desktop/renderer/src/main.tsx`
- Create: `apps/desktop/renderer/src/App.tsx`
- Create: `apps/desktop/renderer/src/lib/core-client/client.ts`
- Create: `apps/desktop/renderer/src/lib/core-client/types.ts`
- Create: `core-go/go.mod`
- Create: `core-go/cmd/skillbox-core/main.go`
- Create: `core-go/internal/app/wire.go`
- Create: `core-go/internal/rpc/server.go`
- Create: `core-go/internal/rpc/handlers/ping.go`
- Create: `apps/desktop/electron/main/core-process/json-rpc-client.ts`
- Create: `apps/desktop/scripts/build-go.sh`
- Modify: `.gitignore`

Repo is non-workspace (single JS package at `apps/desktop`); all `pnpm` commands below are run from `apps/desktop`.

- [ ] **Step 1: Scaffold packages and ignore rules**

Create `apps/desktop/package.json` with scripts:

```json
{
  "name": "astraler-skillbox-desktop",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "electron-vite dev",
    "build": "electron-vite build",
    "test": "vitest run"
  },
  "dependencies": {
    "@vitejs/plugin-react": "latest",
    "electron": "latest",
    "electron-vite": "latest",
    "react": "latest",
    "react-dom": "latest"
  },
  "devDependencies": {
    "@types/node": "latest",
    "@types/react": "latest",
    "@types/react-dom": "latest",
    "typescript": "latest",
    "vite": "latest",
    "vitest": "latest"
  }
}
```

Append `.gitignore` entries:

```gitignore
node_modules/
dist/
out/
*.db
*.db-shm
*.db-wal
```

- [ ] **Step 2: Add Go ping tests first**

Create `core-go/internal/rpc/handlers/ping_test.go`:

```go
package handlers

import "testing"

func TestPingReturnsPong(t *testing.T) {
	got := Ping()
	if !got.Pong {
		t.Fatal("expected pong")
	}
	if got.TS == "" {
		t.Fatal("expected timestamp")
	}
}
```

Run: `cd core-go && go test ./...`
Expected: fail until `go.mod` and `Ping` exist.

- [ ] **Step 3: Implement minimal Go core**

Create `core-go/go.mod`:

```go
module github.com/astraler/skillbox/core-go

go 1.22
```

Implement `Ping()` in `core-go/internal/rpc/handlers/ping.go` and a stdin/stdout JSON-RPC server in `core-go/cmd/skillbox-core/main.go`. On startup, emit a `server.ready` notification containing `version`, `pid`, and `capabilities:["ping"]`. Keep all logs on stderr.

Run: `cd core-go && go test ./...`
Expected: pass.

- [ ] **Step 4: Add Electron main/preload bridge**

Implement `CoreProcessManager` in `manager.ts` using `spawn("go", ["run", "./cmd/skillbox-core"], { cwd: coreGoPath })`. Add 10s `server.ready` timeout, SIGTERM then SIGKILL on quit, and restart up to 3 times for mid-session exits.

Implement `JsonRpcStdioClient` in `json-rpc-client.ts` per the "JSON-RPC client (Electron main side)" section of the design spec. Required surface:

- `call<T>(method, params, opts?: { timeoutMs?: number }): Promise<T>` with monotonic request id, pending-promise map, default 30s timeout, opts to opt out.
- `on(method, handler): () => void` for server-push notifications (no id).
- `shutdown(reason)` rejects all pending requests with `core_unavailable` and is called on child `exit`/`error`.
- stdout parsed via `readline` line-by-line (NDJSON); stderr forwarded to `core.log`; non-parseable lines logged and skipped.
- Orphan responses (id with no pending) logged, never reject random pending.

`CoreProcessManager` owns the spawn lifecycle; `JsonRpcStdioClient` owns the wire correlation. They compose: manager constructs the client with the spawned child, returns `goClient` used by `ipc-bridge.ts`.

Expose in preload:

```ts
contextBridge.exposeInMainWorld("core", {
  invoke: (method: string, params: unknown) => ipcRenderer.invoke("core:invoke", method, params),
  onEvent: (event: string, cb: (params: unknown) => void) => {
    const handler = (_: unknown, method: string, params: unknown) => {
      if (method === event) cb(params);
    };
    ipcRenderer.on("core:event", handler);
    return () => ipcRenderer.off("core:event", handler);
  },
});
```

- [ ] **Step 5: Add React ping screen**

`App.tsx` renders a "Ping Go" button. On click call `window.core.invoke("ping", {})` and render the JSON response.

- [ ] **Step 6: Verify M1**

Run:

```sh
cd apps/desktop && pnpm install
cd ../core-go && go test ./...
cd ../apps/desktop && pnpm test
cd ../apps/desktop && pnpm dev
```

Acceptance:
- Electron opens within 10s.
- Ping button displays `{ "pong": true, "ts": "..." }`.
- Go stdout contains only JSON-RPC bytes.
- Quitting Electron leaves no `skillbox-core` process.

- [ ] **Step 7: Commit M1**

```sh
git add .gitignore apps/desktop core-go
git commit -m "Add Electron Go walking skeleton"
```

`apps/desktop/scripts/build-go.sh` is covered by `apps/desktop` path; no separate add line.

---

## M2: API Contracts

**Objective:** Add JSON Schema contracts and generated TypeScript types for slice 1.

**Files:**
- Create: `shared/api-contracts/index.json`
- Create: `shared/api-contracts/package.json`
- Create: `shared/api-contracts/README.md`
- Create: `shared/api-contracts/methods/host.choose.json`
- Create: `shared/api-contracts/methods/host.scan.json`
- Create: `shared/api-contracts/methods/skill.list.json`
- Create: `shared/api-contracts/methods/operation.cancel.json`
- Create: `shared/api-contracts/methods/settings.get.json`
- Create: `shared/api-contracts/notifications/server.ready.json`
- Create: `shared/api-contracts/notifications/operation.progress.json`
- Create: `shared/api-contracts/shared/operation.json`
- Create: `shared/api-contracts/shared/error.json`
- Create: `shared/api-contracts/shared/skill.json`
- Create: `shared/api-contracts/shared/warning.json`
- Create: `shared/api-contracts/electron/dialog.openHostFolder.json`
- Create: `apps/desktop/scripts/generate-contracts.mjs`
- Create: `shared/generated/**`

- [ ] **Step 1: Add schema files**

Implement schemas exactly matching the method shapes in the design spec. Use integer IDs, `additionalProperties:false` on responses, and keep `dialog.openHostFolder` under `electron/`. Include `settings.get` returning `{ activeSkillHostFolderId, defaultInstallMode, databaseVersion, activeHost | null }` — this query backs first-load routing and reopen persistence in M4.

- [ ] **Step 2: Add generator**

Install `json-schema-to-typescript` in `apps/desktop/package.json`. Create `apps/desktop/scripts/generate-contracts.mjs` (NOT at repo root — consistent with non-workspace decision) that reads `../../shared/api-contracts/index.json`, generates TS to `../../shared/generated/`, and supports `--check`.

`apps/desktop/package.json` scripts:

```json
"generate:contracts": "node scripts/generate-contracts.mjs",
"check:contracts-drift": "node scripts/generate-contracts.mjs --check"
```

- [ ] **Step 3: Generate and check drift**

Run:

```sh
cd apps/desktop && pnpm install
cd apps/desktop && pnpm generate:contracts
cd apps/desktop && pnpm check:contracts-drift
```

Expected: generated files are stable and drift check passes.

- [ ] **Step 4: Commit M2**

```sh
git add apps/desktop shared
git commit -m "Add slice 1 API contracts"
```

`apps/desktop/scripts/generate-contracts.mjs` and `pnpm-lock.yaml` (inside `apps/desktop/`) are covered by `apps/desktop` path.

---

## M3: Go Core Slice

**Objective:** Build domain, SQLite, filesystem gateway, repositories, operation runner, services, and JSON-RPC handlers with TDD.

**Files:**
- Create: `core-go/migrations/0001_init.sql`
- Create: `core-go/internal/domain/*.go`
- Create: `core-go/internal/filesystem/*.go`
- Create: `core-go/internal/repositories/*.go`
- Create: `core-go/internal/operations/*.go`
- Create: `core-go/internal/services/*.go`
- Create: `core-go/internal/rpc/handlers/{host_choose,host_scan,skill_list,operation_cancel,settings_get}.go`
- Create: matching `*_test.go` files for every package.
- Modify: `core-go/cmd/skillbox-core/main.go`
- Modify: `core-go/internal/app/wire.go`
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts`

- [ ] **Step 1: Domain TDD**

Write tests for `AppError`, enum validation, and JSON shape. Implement pure Go domain types using `int64` IDs and enums from docs/07.

Run: `cd core-go && go test ./internal/domain -v`

- [ ] **Step 2: SQLite TDD**

Write `db_test.go` verifying migrations, WAL, foreign keys, busy timeout, and singleton `app_settings`. Implement `OpenDatabase`. Migration `0001_init.sql` includes `app_settings`, `skill_host_folders`, `skills`, `skill_sources`, `operations`, `warnings` per docs/07. `skill_sources` is included (not deferred) so `skills.source_id` FK integrity holds even though slice 1 never writes to it. Defer all other tables.

Run: `cd core-go && go test ./internal/repositories -run TestOpenDatabase -v`

- [ ] **Step 3: Filesystem gateway TDD**

Write tests using `t.TempDir()` for path validation, `.agents/skills` init, normal folders, valid symlinks, broken symlinks, and external symlinks. Implement gateway functions.

Run: `cd core-go && go test ./internal/filesystem -v`

- [ ] **Step 4: Repository TDD**

Implement repos after tests. `SkillHostFolderRepository` exposes `GetByID`, `GetByPath`, `GetActive`, `UpdateStatus`, `UpdateLastScannedAt`, and one transactional unit-of-work method `UpsertAndActivate(ctx, name, path, skillsPath) → (hostId int64, isNew bool, err)`. `UpsertAndActivate` must, in a single transaction: upsert the host by path with `status='active'`, demote the prior active host to `status='inactive'` if different, and update `app_settings.active_skill_host_folder_id`. No separate `SetActive` method — ChooseHost calls only `UpsertAndActivate`. Other repos: skill `UpsertMany`/`MarkMissing`/`ListByHost`, operation `Insert`/`UpdateStatus`, warning `Insert`/`ListByScope`/`ClearByScope`.

Run: `cd core-go && go test ./internal/repositories -v`

- [ ] **Step 5: Operation runner TDD**

Test success, failure, cancellation, per-target lock conflict, panic recovery, and metadata persistence. Run with race detector:

```sh
cd core-go && go test -race ./internal/operations -v
```

- [ ] **Step 6: Services TDD**

Test `ChooseHost`, `ScanHost`, `SkillLibraryService.List`, and `SettingsService.Get` with mock filesystem/repos.

- `ChooseHost`: must be idempotent by path and switch active host without `conflict_error`. Service calls `fs.ValidateHostPath` → `fs.EnsureAgentsSkills` → `hostRepo.UpsertAndActivate` (single transactional call, no service-side transaction composition) → `hostRepo.GetByID` to return current status.
- `SettingsService.Get`: returns `activeHost=null` when no active id; populates `activeHost` from `hostRepo.GetByID(*activeId)`; tolerates orphan id (host row missing) by returning `activeHost=null` instead of erroring.

Run: `cd core-go && go test ./internal/services -v`

- [ ] **Step 7: RPC handlers and contract tests**

Add handlers for `host.choose`, `host.scan`, `skill.list`, `operation.cancel`, and `settings.get`. Register all in `cmd/skillbox-core/main.go`. Update `apps/desktop/electron/main/core-process/method-allowlist.ts` to `["ping", "host.choose", "host.scan", "skill.list", "operation.cancel", "settings.get"]`. Add contract tests validating handler responses against `shared/api-contracts`.

Run:

```sh
cd core-go && go test ./internal/rpc/... -v
cd core-go && go test -race ./...
```

- [ ] **Step 8: Standalone smoke**

Run:

```sh
cd core-go && SKILLBOX_DB_PATH=/tmp/skillbox-test.db go run ./cmd/skillbox-core
```

Send NDJSON requests for `ping`, `settings.get` (expect `activeHost=null`), `host.choose`, `settings.get` again (expect `activeHost` populated), `host.scan`, and `skill.list`. Verify DB rows with `sqlite3 /tmp/skillbox-test.db`; confirm `skill_sources` table exists (empty) alongside `skills`.

- [ ] **Step 9: Commit M3**

```sh
git add core-go apps/desktop/electron/main/core-process/method-allowlist.ts
git commit -m "Add Go core skill host slice"
```

---

## M4: React Slice

**Objective:** Add setup, Skills Library, Settings, operation progress, and typed client integration.

**Files:**
- Create: `apps/desktop/renderer/src/app/{router,query-client,providers}.tsx`
- Create: `apps/desktop/renderer/src/lib/core-client/{client,methods,progress}.ts`
- Create: `apps/desktop/renderer/src/lib/query-keys.ts`
- Create: `apps/desktop/renderer/src/features/skill-host/*.tsx`
- Create: `apps/desktop/renderer/src/features/skills-library/*.tsx`
- Create: `apps/desktop/renderer/src/screens/{setup-screen,skills-library-screen,settings-screen}.tsx`
- Create: `apps/desktop/renderer/src/components/*.tsx`
- Create: `apps/desktop/renderer/src/styles/globals.css`
- Modify: `apps/desktop/package.json`
- Modify: `apps/desktop/electron/main/core-process/ipc-bridge.ts`

- [ ] **Step 1: Install UI dependencies**

Add TanStack Router/Query, Tailwind, shadcn/ui dependencies, lucide-react, zod, react-hook-form, and sonner.

- [ ] **Step 2: Core client tests first**

Write Vitest tests for invoke, method wrappers (including `getSettings`), error mapping, missing `window.core`, and progress filtering by `operationId`.

Run: `cd apps/desktop && pnpm test -- core-client`

- [ ] **Step 3: Implement core client**

Import generated types from `shared/generated`. Implement `methods.openHostFolder`, `chooseHost`, `scanHost`, `listSkills`, `cancelOperation`, and `getSettings` (wraps `settings.get`, no params).

- [ ] **Step 4: Add router and providers**

Create memory routes: `/`, `/setup`, `/skills`, `/settings`. Add `useAppSettings` hook in `features/app-settings/use-app-settings.ts` that calls `methods.getSettings()` and is the single source of truth for active host. Root route `/` uses it to redirect: `data?.activeHost == null → /setup`, otherwise → `/skills`. Show a spinner while the query is pending.

- [ ] **Step 5: Add screens and hooks**

Implement setup flow, Skills Library list with Rescan, warning banner, operation progress toast, and settings Change Host Folder. Keep settings minimal. `useChooseHost` and `useScanHost` invalidate `settings.app` and `skills.list` on success so routing/UI converge.

- [ ] **Step 6: Hook tests**

Test `useAppSettings`, `useChooseHost`, `useScanHost`, and `useSkillsList` with mocked `methods`; verify query invalidation and error handling.

Run: `cd apps/desktop && pnpm test`

- [ ] **Step 7: Manual M4 acceptance**

Run `cd apps/desktop && pnpm dev`, choose `/tmp/test-host`, create skills, rescan, verify list/status/warnings, switch host, quit/reopen (reopen must land on `/skills` via `settings.get` persistence).

- [ ] **Step 8: Commit M4**

```sh
git add apps/desktop shared/generated
git commit -m "Add skill host and library UI"
```

---

## M5: Smoke Docs

**Objective:** Add complete manual smoke checklist and scaffold guide.

**Files:**
- Create: `SMOKE.md`
- Create: `SCAFFOLD.md`
- Modify: `README.md` if needed to link both files.

- [ ] **Step 1: Write `SMOKE.md`**

Include the full checklist from the design spec: setup, handshake, choose host, scan, reconcile, warning, switch host, lifecycle, and validation smoke.

- [ ] **Step 2: Write `SCAFFOLD.md`**

Document prerequisites, install, full-stack dev, Go-only dev, UI-only mock flag, DB path, logs, contracts, tests, and troubleshooting.

- [ ] **Step 3: Run final verification**

Run:

```sh
cd apps/desktop && pnpm test
cd ../core-go && go test -race ./...
cd ../apps/desktop && pnpm check:contracts-drift
git diff --check
```

Then run the manual `SMOKE.md` checklist on macOS.

- [ ] **Step 4: Commit M5**

```sh
git add SMOKE.md SCAFFOLD.md README.md
git commit -m "Add scaffold smoke docs"
```

Do not create or push `slice-1-skills-library` tag until the user explicitly approves the release checkpoint.

---

## Self-Check

- Spec coverage: M1-M5 all mapped to tasks.
- Native dialog decision: Electron opens dialog, Go receives `{ path }`.
- IDs: number in TS, `int64` in Go.
- `host.choose`: idempotent by path, switches active host inline via single `UpsertAndActivate` repo call, no conflict on switching.
- `settings.get`: contract, handler, allowlist, and UI client all aligned; backs `/` routing and reopen persistence.
- `skill_sources` table included in `0001_init.sql` (FK integrity); slice 1 has no write path.
- `scan_results`: deferred; scan summary in `operations.metadata_json`.
- `JsonRpcStdioClient`: custom TS client per spec (NDJSON, id correlation, pending map, notifications, timeout, shutdown).
- Scripts under `apps/desktop/scripts/` (non-workspace); all `pnpm` commands run from `apps/desktop`.
- Tag `slice-1-skills-library` not created or pushed — waits for owner approval.
- Related untracked files: `AGENTS.md` and `CLAUDE.md` must remain outside plan commits unless explicitly requested.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-25-skillbox-scaffold-slice-1-plan.md`.

Two execution options:

1. **Subagent-Driven (recommended)** - dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** - execute tasks in this session using executing-plans, batch execution with checkpoints.
