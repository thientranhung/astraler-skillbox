# Slice 2B Project Utilities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add safe project removal and Finder open actions without writing to project folders.

**Architecture:** Reuse the existing Slice 2A project layers. The Go core owns project state and soft-remove semantics; Electron main owns Finder integration; React exposes narrow list/detail actions and invalidates existing queries.

**Tech Stack:** Go, SQLite, JSON-RPC contracts, Electron main bridge, React, TanStack Query/Router, Vitest.

---

## Source Of Truth

- Design spec: `docs/superpowers/specs/2026-05-25-skillbox-slice-2b-project-utilities-design.md`
- Existing project implementation: `core-go/internal/services/project_service.go`, `core-go/internal/repositories/project_repo.go`, `apps/desktop/renderer/src/features/projects/`, `apps/desktop/renderer/src/screens/`
- Safety boundary: no project filesystem writes.

## Commit Boundaries

- M1: `Add slice 2B project utilities contracts`
- M2: `Add slice 2B project removal core`
- M3: `Add slice 2B project utility UI`
- M4: `Verify slice 2B project utilities`

Do not stage unrelated untracked files such as `CLAUDE.md`.

## M1: Contracts

**Files:**
- Create: `shared/api-contracts/methods/project.remove.json`
- Create: `shared/api-contracts/electron/dialog.openPath.json`
- Modify: `shared/api-contracts/index.json`
- Regenerate: `shared/generated/**`

- [ ] Add `project.remove` schema: request `{ "projectId": integer >= 1 }`, response `{ "removed": true }`, with `additionalProperties:false`.
- [ ] Add Electron `dialog.openPath` schema: request `{ "path": string, minLength: 1 }`, response `{ "opened": true }`.
- [ ] Register both methods in `shared/api-contracts/index.json`.
- [ ] Run `(cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift)`.
- [ ] Commit only contract/generated changes.

## M2: Go Core Removal

**Files:**
- Modify: `core-go/internal/repositories/project_repo.go`
- Modify/test: `core-go/internal/repositories/project_repo_test.go`
- Modify: `core-go/internal/services/interfaces.go`
- Modify: `core-go/internal/services/project_service.go`
- Modify/test: `core-go/internal/services/project_service_test.go`
- Create: `core-go/internal/rpc/handlers/project_remove.go`
- Modify/test: `core-go/internal/rpc/handlers/project_handler_test.go`, `core-go/internal/rpc/handlers/contract_test.go`
- Modify: `core-go/internal/app/wire.go`, `core-go/cmd/skillbox-core/main.go`

- [ ] Write repository tests proving `List` hides removed rows, `GetByID` returns nil for removed rows, `UpsertByPath` revives a removed path as active, and `MarkRemoved` returns false for missing/removed IDs.
- [ ] Implement repository behavior with SQL filters and `UPDATE projects SET status='removed' ... WHERE id=? AND status <> 'removed'`.
- [ ] Extend the service interface with `MarkRemoved`.
- [ ] Write service tests for successful remove, missing ID validation error, removed ID validation error, and re-add revival through `AddProject`.
- [ ] Implement `ProjectRemoveResult` and `RemoveProject(ctx, projectID)`.
- [ ] Add `project.remove` handler and register it in app wiring and server capabilities.
- [ ] Run `(cd core-go && go test ./...)`.
- [ ] Commit core changes.

## M3: Electron And React UI

**Files:**
- Modify: `apps/desktop/electron/main/core-process/ipc-bridge.ts`
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts`
- Modify/test: `apps/desktop/renderer/src/lib/core-client/methods.ts`, `apps/desktop/renderer/src/lib/core-client/__tests__/methods.test.ts`
- Create: `apps/desktop/renderer/src/features/projects/use-remove-project.ts`
- Create: `apps/desktop/renderer/src/features/projects/use-open-project-folder.ts`
- Modify: `apps/desktop/renderer/src/features/projects/project-row.tsx`
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx`

- [ ] Import `shell` in Electron main and handle `dialog.openPath` with `shell.openPath(path)`. Throw if Electron returns a non-empty error string.
- [ ] Allowlist `dialog.openPath` and `project.remove`.
- [ ] Add core-client wrappers `openPath` and `removeProject`.
- [ ] Add `useRemoveProject` mutation that invalidates `projects.list` and optionally navigates to `/projects`.
- [ ] Add `useOpenProjectFolder` mutation that calls `openPath({ path })`.
- [ ] Add icon buttons with accessible titles: `Open folder` (`FolderOpen`) and `Remove project` (`Trash2`).
- [ ] Stop row navigation from action buttons by keeping actions in their own buttons and not nesting them under the row navigation button.
- [ ] Use confirmation copy: `Remove this project from Skillbox? Files on disk will not be deleted.`
- [ ] Run `(cd apps/desktop && pnpm typecheck && pnpm test && pnpm build)`.
- [ ] Commit UI changes.

## M4: Verification And Smoke

- [ ] Run `(cd core-go && go test ./...)`.
- [ ] Run `(cd apps/desktop && pnpm check:contracts-drift && pnpm typecheck && pnpm test && pnpm build)`.
- [ ] Run `git diff --check`.
- [ ] Manual smoke with host `<global-documents>/my-agent-skills`: add project, scan, open folder from list, open folder from detail, remove project from list, re-add same path, remove from detail, confirm files still exist on disk.
- [ ] Ask reviewer to review the Slice 2B commits and smoke results.

## Self-Check

- Spec coverage: remove API, soft-remove semantics, re-add revival, hidden list rows, detail/scan validation, Electron open folder, list/detail UI actions, and manual smoke are all mapped to tasks.
- No migration is required because `removed` already exists in `projects.status`.
- No task writes to project folders.
