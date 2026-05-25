# Slice 2F: Add Skill To Project (Symlink MVP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Install one or more active Skill Host skills into a project provider (`generic_agents`/`claude`) as symlinks via `install.skill`, with conflict validation and a post-install rescan.

**Architecture:** New filesystem-gateway write methods + install methods on `ProjectService` that reuse the existing operation runner and `scanProjectInternal` for reconciliation. Install records are derived by the inline rescan, not hand-written. UI is a minimal Add Skill wizard.

**Tech Stack:** Go (modernc sqlite, creachadair/jrpc2), Electron + React + TanStack Query + Vitest.

**Spec:** `docs/superpowers/specs/2026-05-25-skillbox-slice-2f-add-skill-to-project-symlink-design.md`

**Metadata rule (lead note):** `failed = requested - created`, computed after stop-on-first-error, so it counts the errored skill plus all unattempted remaining skills.

---

## Conventions

- Backend tests: from repo root `cd core-go && go test ./internal/...`; race-sensitive paths `go test -race ./internal/operations/... ./internal/filesystem/... ./internal/services/...`.
- Frontend: `cd apps/desktop && pnpm test`, `pnpm typecheck`, `pnpm generate:contracts`, `pnpm check:contracts-drift`.
- TDD per task: write failing test → run (confirm fail) → minimal impl → run (confirm pass) → commit. Commit messages below are the checkpoint per task.

---

## Phase 1 — Backend foundations

### Task 1: Operation type constant
- Modify: `core-go/internal/domain/operation.go`
- [ ] Add `OperationTypeInstallSkill OperationType = "install_skill"` to the const block.
- [ ] Run `cd core-go && go build ./...`; commit: `feat(domain): add install_skill operation type`.

### Task 2: Gateway write methods
- Create: `core-go/internal/filesystem/write.go` — package funcs `LstatExists(path) (bool,error)` (os.Lstat; treat `fs.ErrNotExist` as false; a broken/external symlink returns true), `EnsureDir(path) error` (os.MkdirAll 0o755), `CreateSymlink(source, linkPath) error` (os.Symlink).
- Modify: `core-go/internal/filesystem/gateway.go` — add three delegating methods.
- Test: `core-go/internal/filesystem/write_test.go` using `t.TempDir()`: LstatExists for missing / real dir / regular file / broken symlink; EnsureDir idempotent (call twice); CreateSymlink success then existing-target returns error.
- [ ] Failing test → impl → `go test ./internal/filesystem/...` → commit: `feat(filesystem): add LstatExists, EnsureDir, CreateSymlink`.

### Task 3: Runner persists metadata on failure
- Modify: `core-go/internal/operations/runner.go` — in `run()`, marshal returned `meta` once and pass it to `UpdateStatus` on BOTH the error and success paths (currently nil on error). Keep existing behavior when `meta == nil`.
- Test: `core-go/internal/operations/runner_test.go` — a WorkFn returning `(someStruct, err)` results in a FAILED op whose stored `MetadataJSON` is non-nil and contains the struct fields. Add/extend a fake OperationRepo capturing the metadata arg.
- [ ] Failing test → impl → `go test -race ./internal/operations/...` → commit: `fix(operations): persist work metadata on failed operations`.

---

## Phase 2 — Install service (methods on ProjectService)

### Task 4: Skill-name segment validation
- Create: `core-go/internal/services/install_validation.go` — `validateSkillSegment(name string) error` returning `validation_error` when name is empty, `.`/`..`, `filepath.IsAbs`, contains `/`/`filepath.Separator`/NUL, or `filepath.Clean(name) != name`. Add `isWithin(root, path string) bool`.
- Test: `core-go/internal/services/install_validation_test.go` — table test: accept `documentation-writer`; reject `""`, `.`, `..`, `/abs`, `a/b`, `a/../b`, `./a`, `a/`, "a\x00b". isWithin true for root child, false for `/etc`.
- [ ] Failing test → impl → `go test ./internal/services/...` → commit: `feat(services): add install skill-name validation helpers`.

### Task 5: InstallFilesystem dep + InstallSkills entrypoint
- Modify: `core-go/internal/services/interfaces.go` — add `InstallFilesystem` interface (`LstatExists`, `EnsureDir`, `CreateSymlink`).
- Modify: `core-go/internal/services/project_service.go` — add field `installFS InstallFilesystem` + builder `WithInstallDeps(installFS) *ProjectService`.
- Create: `core-go/internal/services/project_install_service.go` — `InstallSkills(ctx, projectID int64, providerKey string, skillIDs []int64) (int64, error)`: synchronous shape validation (non-empty + unique positive `skillIDs`; `providers.InstallTargetByProviderKey` known); load project, must exist + `status=active`; then `runner.Start` with `Target{Type:"project", ID:projectID}`, `OperationTypeInstallSkill`, closure → `installSkillsInternal`. Pass through `*domain.AppError` (e.g. runner conflict) unchanged.
- Test: extend `core-go/internal/services/project_service_test.go` (reuse `mockRunner`, `newMockProjectRepo`): empty skillIDs → validation_error; duplicate ids → validation_error; unknown providerKey → validation_error; missing/removed project → validation_error; happy shape → returns runner opID; runner conflict surfaces as `conflict_error`.
- [ ] Failing test → impl → `go test ./internal/services/...` → commit: `feat(services): add InstallSkills entrypoint and InstallFilesystem dep`.

### Task 6: installSkillsInternal worker (full implementation, happy path)
- Modify: `core-go/internal/services/project_install_service.go` — implement `installSkillsInternal(ctx, project *domain.Project, providerKey string, skillIDs []int64, progress operations.ProgressFn) (any, error)` per spec flow:
  1. `progress("validating",…)`; resolve `InstallTarget`; load provider def via `providerDefRepo.GetByKey` (nil → validation_error; status not supported/experimental → provider_error).
  2. `ppRepo.ListByProject`; find summary with `ProviderKey==providerKey` (absent → validation_error; `DetectionStatus` not detected/configured → validation_error).
  3. `skillsPath = fs.NormalizeAbs(filepath.Join(project.Path, target.RelativeSkillsPath))`; `root = fs.NormalizeAbs(project.Path)`; `isWithin(root, skillsPath)` else validation_error.
  4. Active host: `hostLister.ListAll` → first `Status==active` (none → validation_error); `skillsByHostLister.ListByHost`; map by id; for each requested id resolve skill (missing → validation_error; `Status!=available` → validation_error). Preserve request order.
  5. For each skill: `validateSkillSegment(name)`; `linkPath=filepath.Join(skillsPath,name)`; require `filepath.Dir(linkPath)==skillsPath` else validation_error.
  6. Conflict (fail-fast, pre-write): `installFS.LstatExists(linkPath)` (err → filesystem_error); collect existing names; any → `conflict_error` listing names.
  7. Ensure dir: `fs.PathInfo(skillsPath)`; if absent and `!pd.CanCreateStructure` → provider_error; else `installFS.EnsureDir`.
  8. `progress("creating_symlinks",0,requested,"")`; loop `installFS.CreateSymlink(skill.AbsolutePath, linkPath)`; on first error break, keep `createErr`, `created` counts successes.
  9. Always `scanProjectInternal(ctx, project, progress)` (keep `rescanErr`).
  10. `failed := requested - created`; build `installMetadata{requested,created,failed,providerKey}`; return `(meta, createErr)` if createErr; else `(meta, rescanErr)`; else `(meta, nil)`.
- Test: `core-go/internal/services/project_install_service_test.go` happy path — real `filesystem.NewGateway()` for both `fs` and `installFS`, real `providers.NewGenericAgentsAdapter()` in registry, temp project dir with existing `.agents/skills`, temp host dir with skill folders, `mockHostLister` active host (real SkillsPath), `mockSkillsByHostLister` (skills with real AbsolutePath), `mockProviderDefRepo` (`generic_agents`: supported, CanCreateStructure true), `mockProjectProviderRepo` (detected summary), `mockProjectScanCommitter` capturing installs. Assert: symlink created on disk; captured install `InstallMode=symlink`, `InstallStatus=current`; returned meta `{created:1,failed:0}`.
- [ ] Failing test → impl → `go test -race ./internal/services/...` → commit: `feat(services): implement install symlink worker with inline rescan`.

### Task 7: Auto-create skills dir (generic_agents)
- Test (same file): project dir WITHOUT `.agents/skills`; expect dir created, symlink made, classified `current`, meta `created:1`.
- [ ] Add test → run → passes (impl from Task 6) → commit: `test(services): cover install auto-create of shared agents skills dir`.

### Task 8: Claude no-scaffold block
- Test: `claude` provider def (experimental, CanCreateStructure false), `.claude/skills` absent → `provider_error`, no writes (assert dir still absent), scan committer not called with claude installs.
- [ ] Add test → run → passes → commit: `test(services): cover claude no-scaffold provider_error`.

### Task 9: Conflict abort (atomic)
- Test: two cases — pre-existing real dir at `linkPath`; pre-existing broken symlink at `linkPath`. Expect `conflict_error` naming the skill, and NO new symlink for the other selected skill (filesystem unchanged besides the planted conflict).
- [ ] Add test → run → passes → commit: `test(services): cover atomic conflict abort`.

### Task 10: Validation + within-root coverage
- Test: provider not in project → validation_error; provider def status unsupported → provider_error; detection_status missing → validation_error; skill id absent on active host → validation_error; skill status not available → validation_error; no active host → validation_error; crafted unsafe name (inject a skill whose `Name` is `../escape`) → validation_error and no write.
- [ ] Add tests → run → passes → commit: `test(services): cover install validation and within-root enforcement`.

### Task 11: Multi-skill partial filesystem failure
- Test: define a `flakyInstallFS` in the test file wrapping real os behavior — `CreateSymlink` succeeds on call 1 (actually creates), returns error on call 2; `LstatExists`/`EnsureDir` delegate to real os. Two skills selected. Assert: loop stops after first error; `scanProjectInternal` still ran (committer called once); the first symlink is classified `current`; returned error is `filesystem_error`; returned meta `{requested:2, created:1, failed:1}`.
- [ ] Failing test → (impl already supports; add flaky fake) → `go test -race ./internal/services/...` → commit: `test(services): cover multi-skill partial failure and rescan`.

---

## Phase 3 — RPC, contract, wiring

### Task 12: install.skill handler
- Create: `core-go/internal/rpc/handlers/install_skill.go` — `installSkillRequest{ProjectID int64, ProviderKey string, SkillIDs []int64}` (json `projectId`/`providerKey`/`skillIds`), `installSkillResponse{OperationID int64}`, `NewInstallSkillHandler(svc)` mirroring `project_scan.go` (`wrapError` on failure).
- Define interface `installSkillService interface { InstallSkills(ctx, int64, string, []int64) (int64, error) }`.
- Modify: `core-go/internal/app/wire.go` — register `"install.skill": rpchandlers.NewInstallSkillHandler(projectSvc)`.
- Test: `core-go/internal/rpc/handlers/project_handler_test.go` (or new `install_skill_handler_test.go`) — bad params → validation_error; service opID returned as `operationId`; service AppError surfaced.
- [ ] Failing test → impl → `go test ./internal/rpc/...` → commit: `feat(rpc): add install.skill handler`.

### Task 13: Contract schema + drift type
- Create: `shared/api-contracts/methods/install.skill.json` — draft-07 oneOf Request/Response, mirroring `project.scan.json`. Request: `projectId` integer, `providerKey` string enum `["generic_agents","claude"]`, `skillIds` array of integer `minItems:1` `uniqueItems:true`, all required, `additionalProperties:false`. Response: `operationId` integer required.
- Modify: `shared/api-contracts/index.json` — add `{ "input": "methods/install.skill.json", "output": "methods/install-skill.ts" }`.
- Test: add `TestContract_InstallSkill_Response` to `core-go/internal/rpc/handlers/project_contract_test.go` validating `installSkillResponse{OperationID:1}`.
- [ ] Failing test → impl → `go test ./internal/rpc/...` → commit: `feat(contracts): add install.skill schema`.

### Task 14: Composition root + Electron allowlist
- Modify: `core-go/cmd/skillbox-core/main.go` — append `.WithInstallDeps(fs)` to the `projectSvc` builder chain (same `fs` gateway instance).
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts` — add `"install.skill"`.
- [ ] `cd core-go && go build ./...`; `cd apps/desktop && pnpm typecheck` → commit: `feat: wire install.skill into core and electron allowlist`.

### Task 15: Generate + verify contracts
- [ ] `cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift` (commit regenerated `shared/generated/` if changed) → commit: `chore(contracts): regenerate types for install.skill`.

---

## Phase 4 — Renderer

### Task 16: core-client method
- Modify: `apps/desktop/renderer/src/lib/core-client/methods.ts` — add `installSkill: (req: InstallSkillRequest) => invoke<InstallSkillResponse>("install.skill", req)` with the generated types imported.
- Test: extend `apps/desktop/renderer/src/lib/core-client/__tests__/methods.test.ts` — asserts `invoke("install.skill", { projectId, providerKey, skillIds })`.
- [ ] Failing test → impl → `cd apps/desktop && pnpm test` → commit: `feat(renderer): add installSkill core-client method`.

### Task 17: useInstallSkill hook
- Create: `apps/desktop/renderer/src/features/projects/use-install-skill.ts` — mirror `use-scan-project.ts`: subscribe-all before RPC, handle buffered terminal, subscribe per-op; on terminal success toast "Installed N skill(s)", on failed toast `Install failed: ${message}` (message carries created/failed from worker error), on cancelled dismiss; invalidate `queryKeys.projects.detail(projectId)` + `queryKeys.projects.list()` ONLY on terminal event. Mutation input `{ projectId, providerKey, skillIds }`.
- Test: `apps/desktop/renderer/src/features/projects/__tests__/use-install-skill.test.tsx` — mirror scan hook test: sets operationId; invalidates detail+list on terminal success; error toast + invalidate on terminal failed; no invalidate on intermediate `running` event.
- [ ] Failing test → impl → `pnpm test` → commit: `feat(renderer): add useInstallSkill hook`.

### Task 18: Add Skill wizard component
- Create: `apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx` — prop-driven (`projectId`, `providers: ProjectGetProvider[]`, `skills: SkillItem[]`, `onClose`). Compute installable providers via coarse predicate: `providerStatus ∈ {supported,experimental} && detectionStatus ∈ {detected,configured}` (note: server is authoritative; renderer cannot see `can_create_structure`). Steps: select skills (multi), select provider (auto-select if one; hidden if zero), confirm (mode fixed "symlink"). If zero installable providers → `[Install]` disabled + inline reason text, never calls the hook. On confirm → `useInstallSkill().mutate({projectId, providerKey, skillIds})`.
- Test: `apps/desktop/renderer/src/features/projects/__tests__/add-skill-wizard.test.tsx` — (a) zero installable providers: Install disabled, reason shown, hook mutate not called; (b) one installable provider + selected skills: clicking Install calls mutate with correct args. Mock `use-install-skill`.
- [ ] Failing test → impl → `pnpm test` → commit: `feat(renderer): add minimal Add Skill wizard`.

### Task 19: Wire wizard into Project Detail
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — `[Add Skill]` button toggles wizard open; pass `providers` + project id from `useProjectDetail`, and active-host `skills` (fetch via existing `methods.getSettings` → active host id → `methods.listSkills`; reuse any existing skills hook if present). No new screen-level test required.
- [ ] `pnpm typecheck && pnpm test` → commit: `feat(renderer): open Add Skill wizard from project detail`.

---

## Final verification

- [ ] `cd core-go && go test ./... && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/services/...`
- [ ] `cd apps/desktop && pnpm typecheck && pnpm test && pnpm check:contracts-drift`
- [ ] Manual full-stack smoke (`pnpm dev`): add project with `.agents`, Add Skill → symlink appears as `current`; retry same skill → conflict surfaced; Claude project without `.claude/skills` → provider error.
- [ ] Confirm acceptance criteria in the spec are all met; commit any fixes.

## Self-review notes (coverage vs spec)

- Symlink-only; rsync/copy, remove, relink, switch-mode, global, replace-on-conflict NOT implemented (out of scope).
- `failed = requested - created` enforced in Task 6/11 metadata.
- Partial failure → FAILED op + persisted metadata (Tasks 3, 6, 11); rescan always runs.
- Skill-name + within-root validation (Tasks 4, 6, 10); conflict atomic pre-write (Task 9).
- Claude `can_create_structure=0` block (Task 8); static-capability read only.
- Renderer installable predicate is coarse (no `can_create_structure` on client); server authoritative — documented in Task 18.
