# Slice 3J — Skills Library Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add a read-only "Projects using" count to the Skills Library, a read-only Skill Detail screen, and client-side search/status filters.

**Architecture:** Pure read path. Two new SQLite read queries on `SkillRepo` feed `SkillLibraryService`; one new read-only RPC `skill.get` plus a `projectsUsingCount` field on `skill.list`. Renderer gains a Projects column, filters, an Open Skill Host Folder button (reusing `dialog.openPath`), and a `/skills/$skillId` screen. No filesystem writes, no migration.

**Tech Stack:** Go (`modernc.org/sqlite`, `creachadair/jrpc2`), JSON Schema → generated TS, Electron preload, React + TanStack Router/Query, `go test`, Vitest + RTL.

**Source spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3j-skills-library-visibility-design.md`

---

## Files Touched

**Contracts:** modify `shared/api-contracts/methods/skill.list.json` (+`projectsUsingCount`), create `shared/api-contracts/methods/skill.get.json`, register in `shared/api-contracts/index.json`; regen `shared/generated/`.
**Go:** `internal/domain/skill.go` (+`SkillProjectUsage`); `internal/repositories/skill_repo.go` (+3 queries) & test; `internal/services/interfaces.go` (extend `SkillRepo`); `internal/services/skill_library_service.go` (count in `List`, new `GetSkillDetail`) & test; `internal/rpc/handlers/skill_list.go` (+field), `skill_get.go` (new), `contract_test.go`; `internal/app/wire.go` & `wire_test.go`; `cmd/skillbox-core/main.go` (capabilities).
**Electron/renderer:** `electron/main/core-process/method-allowlist.ts`; `lib/core-client/methods.ts`; `lib/query-keys.ts`; `features/skills-library/use-skill-detail.ts` (new); `features/skills-library/skill-row.tsx`; `screens/skills-library-screen.tsx`; `screens/skill-detail-screen.tsx` (new); `app/router.tsx`; tests under `features/skills-library/__tests__/` and `screens/__tests__/`.

---

## Task Order

### Task 1 — Contract: `projectsUsingCount` on `skill.list`
- [ ] In `skill.list.json`, add `"projectsUsingCount": { "type": "integer" }` to `SkillListSkill.properties` and to `required`.
- [ ] Stop for PM checkpoint.

### Task 2 — Contract: new `skill.get`
- [ ] Create `skill.get.json`: request `{ skillId: integer }`; response `{ skill, projects[] }`. `skill` = id, name, relativePath, absolutePath, status (enum `available|missing|unreadable|local_modified|unknown`), sourceLabel (string|null), hostPath, lastScannedAt (string|null). `projects[]` = projectId, projectName, projectProviderId, providerKey, providerDisplayName, mode (enum `symlink|rsync_copy|direct`), status (enum `current|outdated|missing|broken_symlink|old_host|external_symlink|conflict|needs_sync|error`), projectSkillPath. All `additionalProperties:false`; doc errors: `validation_error` (1001) on unknown id and non-positive `skillId`.
- [ ] Register `{ "input": "methods/skill.get.json", "output": "methods/skill-get.ts" }` in `index.json`.
- [ ] Run `cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift` → clean. Stop for PM checkpoint (include `shared/generated` in review).

### Task 3 — Domain struct
- [ ] Add `SkillProjectUsage` to `internal/domain/skill.go`: fields `ProjectID, ProjectName int64/string`, `ProjectProviderID int64`, `ProviderKey, ProviderDisplayName, Mode, Status, ProjectSkillPath string`. `go build ./internal/domain/...`. Stop for PM checkpoint.

### Task 4 — Repo: `CountProjectsPerSkillByHost(ctx, hostID) (map[int64]int, error)`
- [ ] **Test first** in `skill_repo_test.go`: seed 1 host, 2 skills; project A active with TWO installs of skill1 (same skill, different paths) → count 1; project B removed with install of skill1 → excluded; skill2 → 0. Reuse seed helpers from `install_repo_test.go`/`project_repo_test.go`.
- [ ] Run `-run TestSkillRepo_CountProjectsPerSkillByHost` → FAIL.
- [ ] Implement: `LEFT JOIN installs ON skill_id=s.id`, `LEFT JOIN project_providers`, `LEFT JOIN projects ON ... AND status!='removed'`, `WHERE s.skill_host_folder_id=? GROUP BY s.id`, select `COUNT(DISTINCT p.id)`. → PASS. Stop for PM checkpoint.

### Task 5 — Repo: `GetByID(ctx, skillID) (*domain.Skill, error)`
- [ ] **Test first**: existing id returns skill; unknown id returns `(nil, nil)`.
- [ ] Implement using existing `scanSkill`, `WHERE id=?`. → PASS. Stop for PM checkpoint.

### Task 6 — Repo: `ProjectsUsingSkill(ctx, skillID) ([]domain.SkillProjectUsage, error)`
- [ ] **Test first**: skill1 used by active project A (1 row) and removed project B (excluded) → len 1, correct fields.
- [ ] Implement: `installs i JOIN project_providers pp JOIN projects p JOIN provider_definitions pd WHERE i.skill_id=? AND p.status!='removed' ORDER BY p.name, pd.display_name`. → PASS. Stop for PM checkpoint.

### Task 7 — Service interface
- [ ] Add the 3 methods to `SkillRepo` interface in `internal/services/interfaces.go`.
- [ ] `go build ./...`; if a service-layer fake implements `SkillRepo`, add the 3 methods to it. `go test ./internal/services/` → PASS. Stop for PM checkpoint.

### Task 8 — Service: count in `List`
- [ ] Add `ProjectsUsingCount int` to `SkillItem`.
- [ ] **Test first**: fake `CountProjectsPerSkillByHost` returns `{10:3, 11:0}`; assert items carry counts.
- [ ] Implement: call `CountProjectsPerSkillByHost` (map DB error → `NewDatabaseError`); set `ProjectsUsingCount: counts[sk.ID]`. → PASS. Stop for PM checkpoint.

### Task 9 — Service: `GetSkillDetail`
- [ ] Add `SkillDetailView{ Skill SkillDetailItem; Projects []domain.SkillProjectUsage }` and `SkillDetailItem` (id, name, relativePath, absolutePath, status, sourceLabel, hostPath, lastScannedAt). `SourceLabel` stays nil (source resolution out of scope).
- [ ] **Test first**: id 10 → metadata + 1 project (provider display "Shared Agent Skills (.agents)"); id 999 → `validation_error` (use the file's existing validation-error assertion style).
- [ ] Implement: `skillRepo.GetByID` (nil → `NewValidationError("Skill not found")`), `hostRepo.GetByID(skill.SkillHostFolderID)` for `hostPath`, `skillRepo.ProjectsUsingSkill`. → PASS. Stop for PM checkpoint.

### Task 10 — Handler: emit `projectsUsingCount`
- [ ] Add `ProjectsUsingCount int json:"projectsUsingCount"` to `skillListSkill`; set it in the map loop in `skill_list.go`. `go test ./internal/rpc/handlers/` → PASS. Stop for PM checkpoint.

### Task 11 — Handler: `skill.get`
- [ ] **Contract test first** in `contract_test.go`: validate a sample `skillGetResponse` against `skill.get.json` `SkillGetResponse` (mirror existing `skill.list`/`project.get` test helpers).
- [ ] Create `skill_get.go`: `skillGetService` interface `{ GetSkillDetail }`; request struct (reject `skillId<=0` with `validation_error`); response structs mirroring the contract; map view → response; default `Projects` to `[]`. → `go test ./internal/rpc/handlers/` PASS. Stop for PM checkpoint.

### Task 12 — Wire + capabilities
- [ ] Add `"skill.get"` to `wire_test.go` expected set (FAIL first).
- [ ] Register `"skill.get": rpchandlers.NewSkillGetHandler(libSvc)` in `wire.go`; add `"skill.get"` to `main.go` `capabilities`. `go test ./internal/app/ ./cmd/...` → PASS. Stop for PM checkpoint.

### Task 13 — Electron allowlist + client method
- [ ] Add `"skill.get"` to `ALLOWLIST` in `method-allowlist.ts`.
- [ ] In `methods.ts` import `SkillGetRequest`/`SkillGetResponse`; add `getSkill: (req) => invoke<SkillGetResponse>("skill.get", req)`. `pnpm typecheck` → PASS. Stop for PM checkpoint.

### Task 14 — Query key + `useSkillDetail` hook
- [ ] Add `detail: (skillId) => ["skills","detail",skillId]` to `query-keys.ts`.
- [ ] **Test first** (`__tests__/use-skill-detail.test.tsx`, mirror `use-skills-list.test.tsx`): no fetch when `null`; fetches `{skillId:10}` when provided.
- [ ] Implement `use-skill-detail.ts`: `useQuery({ queryKey: queryKeys.skills.detail(skillId ?? 0), queryFn: () => methods.getSkill({skillId: skillId!}), enabled: skillId != null })`. → PASS. Stop for PM checkpoint.

### Task 15 — Skills Library: column, search, status filter, Open Host Folder
- [ ] **Test first** (`screens/__tests__/skills-library-screen.test.tsx`, mock the hooks + `useNavigate` like `dashboard-screen.test.tsx`): renders `projectsUsingCount`; search narrows by name; status select filters by status.
- [ ] `skill-row.tsx`: add Projects cell (`skill.projectsUsingCount`); make `<tr>` navigate to `/skills/$skillId` (params `{ skillId: String(skill.id) }`) on click.
- [ ] `skills-library-screen.tsx`: add `search`/`statusFilter` state; Open Skill Host Folder button calling `methods.openPath(data.hostPath)`; search input + status `<select>` (all/available/missing/unreadable/local_modified/unknown); filter rows before mapping; add `Projects` column header between Status and Path. → PASS. Stop for PM checkpoint.

### Task 16 — Skill Detail screen + route
- [ ] **Test first** (`screens/__tests__/skill-detail-screen.test.tsx`, mock `useSkillDetail` + router): renders metadata + projects table (incl. "Shared Agent Skills (.agents)"); empty state "No projects use this skill."; non-numeric param → "Invalid skill ID".
- [ ] Create `skill-detail-screen.tsx`: validate `skillId` param (numeric > 0 else inline error); `useSkillDetail`; back link to `/skills`; metadata block; Open Folder via `methods.openPath(skill.hostPath)`; projects table (Project/Provider/Mode/Status/Path) with empty state. **No write controls.**
- [ ] `router.tsx`: import screen; add `skillDetailRoute` `path:"/skills/$skillId"`; add to `shellRoute.addChildren`. → PASS. Stop for PM checkpoint.

### Task 17 — Full validation
- [ ] `cd core-go && go test ./...` → all PASS.
- [ ] `cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test` → all clean/PASS.
- [ ] Manual `pnpm dev`: Projects counts show; search/status filter work; Open Host Folder opens folder; click a skill used by ≥1 project → detail count matches Project Detail, provider shows "Shared Agent Skills (.agents)"; zero-install skill shows empty state; no write controls on detail.

---

## Validation Commands

```bash
cd core-go && go test ./...
cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test
```

---

## Draft `/goal` (NOT run during planning)

> Implement Slice 3J per `docs/superpowers/plans/2026-05-26-skillbox-slice-3j-skills-library-visibility-plan.md`. Done when: (1) `skill.list` returns `projectsUsingCount` = `COUNT(DISTINCT projects.id)` via `installs -> project_providers -> projects` on `installs.skill_id = skills.id`, excluding `status='removed'`, never by name or install-row count; (2) read-only `skill.get` returns skill metadata + one row per project/provider install (projectId, projectName, projectProviderId, providerKey, providerDisplayName, mode, status, projectSkillPath), with mode enum `symlink|rsync_copy|direct`, status enum `current|outdated|missing|broken_symlink|old_host|external_symlink|conflict|needs_sync|error`, removed projects excluded, and unknown/non-positive id → `validation_error`; (3) Skills Library has a Projects column, working Search + Status filter (available/missing/unreadable/local_modified/unknown), an Open Skill Host Folder button using `dialog.openPath` with `hostPath`, and row-click navigation to a read-only Skill Detail at `/skills/$skillId`; (4) full plumbing wired (skill.get.json, index.json, generated TS, client + query key `["skills","detail",skillId]` + hook, Go handler + wire + capability, Electron allowlist); (5) no filesystem-write path and no migration added; (6) `go test ./...`, `pnpm check:contracts-drift`, `pnpm typecheck`, `pnpm test` all pass. Implement on Sonnet after approval. **Draft only — not executed during planning.**

---

## Self-Review

- **Spec coverage:** count definition → T1/T4/T8; `skill.get` → T2/T6/T9/T11; enum mode/status from `core-go/internal/domain/install.go` → T2; status filter = current skill statuses only → T15; Open Host Folder via `dialog.openPath`+`hostPath` → T15/T16; query key `["skills","detail",skillId]` → T14; read-only detail + empty/invalid states → T14/T16; provider label from `provider_definitions.display_name` (migration 000004) → T6 + T17 smoke; plumbing list → T2/T10–T14; no migration/no writes → enforced, verified T17.
- **Placeholders:** none; "mirror existing helper/mock" notes point at real codebase identifiers, not deferred work.
- **Type/name consistency:** `CountProjectsPerSkillByHost`/`GetByID`/`ProjectsUsingSkill` identical across repo (T4–6), interface (T7), service (T8–9); `domain.SkillProjectUsage` fields align with repo scan, handler mapping (T11), contract (T2); `useSkillDetail` + `queryKeys.skills.detail` + `methods.getSkill` consistent T13–16; route `/skills/$skillId` param `skillId` consistent T15–16.
