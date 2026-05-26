# Slice 3J — Skills Library Visibility (Design Spec)

- **Date:** 2026-05-26
- **Status:** Draft (design only — not implemented)
- **Scope:** Read-only. No new filesystem writes. No schema migration.

## 1. Purpose and Why This Slice Is Next

The Skills Library today is a static table (`Name / Status / Path / Source`) driven by
`skill.list`. It cannot tell the user *which projects consume a skill*, and there is no
Skill Detail surface at all. This is the largest remaining gap that can be closed with
**read-only data already in SQLite** (`installs` rows link projects to host skills via
`skill_id`), without touching the install engine, source integrations, or the filesystem.

Slice 3J adds the "Projects using" count, a read-only Skill Detail screen, and
search/status filtering. It is the safest high-value step: no writes, no operation
runner, no locking, no new dependencies — only one reverse aggregate query, one new
read-only RPC, and renderer composition.

## 2. In Scope

1. `skill.list` response gains `projectsUsingCount` per skill.
2. New read-only RPC `skill.get` returning skill metadata plus the list of project
   installs that reference it.
3. New renderer route/screen Skill Detail at `/skills/$skillId` (read-only).
4. Skills Library additions: a **Projects** column, a client-side **Search** box
   (filter by name), a **Status** filter, an **Open Skill Host Folder** button, and
   row-click navigation into Skill Detail.

## 3. Out of Scope

- Fetch / update / version / source integration (GitHub, Vercel) and any
  fetch/update status.
- Add / Import skill into the host folder.
- Global Skills screen and the "Global Usage" section of the wireframe Skill Detail.
- rsync/copy install mode and Switch Mode.
- Adding `Updates` or `Global Skills` items to the sidebar.
- Any filesystem write or schema migration.

## 4. Data / Contract Design

### 4.1 `projectsUsingCount` (added to `skill.list`)

Definition (authoritative):

```sql
SELECT COUNT(DISTINCT projects.id)
FROM installs
JOIN project_providers ON project_providers.id = installs.project_provider_id
JOIN projects          ON projects.id = project_providers.project_id
WHERE installs.skill_id = skills.id
  AND projects.status != 'removed';
```

Rules:

- Count is **`COUNT(DISTINCT projects.id)`**, not a count of install rows. The same
  project may install the same skill under multiple providers; it must count once.
- Match on **`installs.skill_id = skills.id`** only. Never match by `skill_name`.
  `skill_id` is the authoritative link and is **nullable** — direct/unknown installs
  with `skill_id IS NULL` are intentionally excluded from any skill's count.
- Soft-removed projects (`projects.status = 'removed'`) are excluded.

Contract change to `shared/api-contracts/methods/skill.list.json`, in
`SkillListSkill.properties`:

```json
"projectsUsingCount": { "type": "integer" }
```

Add `"projectsUsingCount"` to `SkillListSkill.required`. `additionalProperties` stays
`false`.

### 4.2 `skill.get` (new method)

New file `shared/api-contracts/methods/skill.get.json` with request and response.

Request:

| Field    | Type    | Notes                          |
|----------|---------|--------------------------------|
| `skillId`| integer | `skills.id` to fetch           |

Response:

`skill` object (metadata, all from the `skills` row + host path):

| Field          | Type             |
|----------------|------------------|
| `id`           | integer          |
| `name`         | string           |
| `relativePath` | string           |
| `absolutePath` | string           |
| `status`       | string enum (same as `skill.list`: `available`, `missing`, `unreadable`, `local_modified`, `unknown`) |
| `sourceLabel`  | string \| null   |
| `hostPath`     | string           |
| `lastScannedAt`| string \| null   |

`projects` array — **one row per project/provider install** (not deduplicated; a
project appearing under two providers yields two rows):

| Field                 | Type           | Source                                            |
|-----------------------|----------------|---------------------------------------------------|
| `projectId`           | integer        | `projects.id`                                     |
| `projectName`         | string         | `projects.name`                                   |
| `projectProviderId`   | integer        | `project_providers.id`                            |
| `providerKey`         | string         | `provider_definitions.key`                        |
| `providerDisplayName` | string         | `provider_definitions.display_name` (see §7 label rule) |
| `mode`                | string enum (`symlink`, `copy`, `direct`) | `installs.install_mode` |
| `status`              | string enum (`current`, `missing`, `broken_symlink`, `needs_sync`, `unknown`) | `installs.install_status` |
| `projectSkillPath`    | string         | `installs.project_skill_path`                     |

Removed projects (`projects.status = 'removed'`) are excluded from `projects`.

Errors: `validation_error` (1001) when `skillId` is not found. No app error code may
fall in the reserved JSON-RPC range.

### 4.3 Generated types

Regenerate `shared/generated/` so the new `skill.get` types and the
`projectsUsingCount` field appear in committed TypeScript. `shared/api-contracts/index.json`
must list `skill.get`.

## 5. Backend Design

- **Repository layer** (only place with SQL):
  - Extend the existing skills-list query to compute `projectsUsingCount` per skill,
    using the §4.1 definition (correlated subquery or a `LEFT JOIN ... GROUP BY` that
    preserves skills with zero installs as count `0`).
  - Add a reverse-lookup query for `skill.get`: fetch the `skills` row by id, then the
    per-install rows joined `installs -> project_providers -> projects` and
    `project_providers -> provider_definitions`, filtered to `projects.status != 'removed'`,
    ordered by `projectName, providerDisplayName` for stable output.
- **Service layer** (`skill_library_service.go`): add a `GetSkillDetail(skillId)` use
  case that returns the view model or a typed `not found` mapped to `validation_error`.
  Keep it read-only — no filesystem, no writes.
- **RPC handler**: register `skill.get` alongside `skill.list` in
  `core-go/internal/rpc/handlers/`.
- **App wiring**: wire the new handler in the composition root
  (`cmd/skillbox-core/main.go` or `internal/app/`). Manual constructor DI — no container.
- **Server allowlist / capabilities**: add `skill.get` wherever the Go side advertises
  or validates method names; if `server.ready` carries a capability/method list, include
  `skill.get` there.

## 6. Renderer Design

### 6.1 Contract plumbing (renderer side)

- Core-client (`lib/core-client/`): add a typed `skillGet(skillId)` method wrapping the
  preload bridge call to `skill.get`.
- **Electron main method allowlist**: add `skill.get` so main forwards it to Go.
  Without this the request is rejected before reaching the sidecar.
- TanStack Query: add a query key `["skills", "detail", skillId]` and a `useSkillDetail(skillId)`
  hook under `features/skills-library/`.

### 6.2 Skills Library screen

- Add a **Projects** column rendering `projectsUsingCount`.
- **Search** input: client-side filter over the already-loaded `skills` by `name`
  (case-insensitive substring). No new request.
- **Status** filter: dropdown over the current skill status values only —
  `available`, `missing`, `unreadable`, `local_modified`, `unknown` — plus an "all"
  option. This is `skill.list` status; **fetch/update status is explicitly future work**.
- **Open Skill Host Folder** button: call the existing `dialog.openPath` mechanism with
  `hostPath`. Do not call `dialog.openHostFolder`, which is the chooser flow. No new
  IPC surface is required.
- Row click navigates to `/skills/$skillId`.

### 6.3 Skill Detail screen (new, read-only)

- Route `/skills/$skillId` under the app shell (sibling of `/skills`), added in
  `app/router.tsx`. New `screens/skill-detail-screen.tsx`.
- Parse and validate `skillId` like `project-detail-screen.tsx` does for `projectId`
  (numeric, > 0; otherwise show an inline invalid-id error).
- Render: a back link to Skills Library; metadata block (name, host path, relative
  path, status badge, source label, last scanned); and a "Projects Using This Skill"
  table with columns Project / Provider / Mode / Status / Path from the `projects`
  array. Include an Open Folder action for `hostPath` (reuses open-folder; no writes).
- Empty state when `projects` is empty: "No projects use this skill."
- **No action that writes** — no install, remove, switch, fetch, or update controls.

## 7. Error Handling and Edge Cases

- `skill.get` with unknown `skillId` → `validation_error` (1001); Skill Detail shows
  the standard `ErrorDisplay`.
- Skill with zero installs → `projectsUsingCount = 0` and empty detail table (skill
  must still appear in the library and be openable).
- `installs.skill_id IS NULL` (direct/unknown entries) contribute to **no** skill's
  count and appear in **no** skill detail — correct and intentional.
- Soft-removed projects never appear in counts or detail lists.
- A project consuming the skill under two providers: counts once in
  `projectsUsingCount`, appears as two rows in `skill.get.projects`.
- **User-facing provider label:** the `generic_agents` provider must display as
  **"Shared Agent Skills (.agents)"** wherever shown to the user (Skill Detail provider
  column). `providerDisplayName` from the DB is used, and the seeded display name /
  presentation must reflect this label; do not surface the raw `generic_agents` key.
- Invalid/non-numeric route param → inline invalid-id error, no request fired.

## 8. Testing and Validation

Backend (`go test ./...`):

- `projectsUsingCount` counts distinct projects, ignores extra provider rows for the
  same project, ignores `skill_id IS NULL` installs, and excludes removed projects.
- Skill with no installs → count 0.
- `skill.get` returns metadata + one row per install, excludes removed projects, orders
  deterministically, and maps not-found to `validation_error`.

Renderer (`pnpm test`):

- Skills Library renders the Projects column and filters by search and status.
- Skill Detail renders metadata + projects table, the empty state, and the invalid-id
  error; asserts no write-action controls are present.

Contract / types:

- `pnpm generate:contracts` then `pnpm check:contracts-drift` is clean.
- `pnpm typecheck` passes with the new generated types.

Manual (`pnpm dev`): open Skills Library, search/filter, open a skill that is installed
in ≥1 project, confirm the Skill Detail project list and the library count agree with
Project Detail.

Full command set:

```bash
cd core-go && go test ./...
cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test
```

## 9. Acceptance Criteria

1. `skill.list` returns `projectsUsingCount` defined as `COUNT(DISTINCT projects.id)`
   over `installs -> project_providers -> projects` where `installs.skill_id = skills.id`
   and `projects.status != 'removed'`; never counts install rows or matches by name.
2. New read-only `skill.get` returns skill metadata plus one row per project/provider
   install (`projectId, projectName, projectProviderId, providerKey, providerDisplayName,
   mode, status, projectSkillPath`), excluding removed projects; unknown id →
   `validation_error`.
3. Skills Library shows a Projects column, a working Search box and Status filter (over
   `available / missing / unreadable / local_modified / unknown` only), an Open Skill
   Host Folder button, and row-click navigation to `/skills/$skillId`.
4. Skill Detail screen renders read-only metadata and the projects-using table with no
   write controls.
5. Contract plumbing is complete end to end: `skill.get.json`, `index.json`, generated
   TS, renderer core-client method + query key/hook, Go handler + app wiring, Electron
   main method allowlist (and `server.ready` capabilities if applicable).
6. No new filesystem-write path and no schema migration are introduced.
7. `go test ./...`, `pnpm check:contracts-drift`, `pnpm typecheck`, and `pnpm test`
   all pass.
8. The `generic_agents` provider is shown to the user as
   "Shared Agent Skills (.agents)".

## 10. Draft `/goal` Condition (for implementation — NOT run during spec work)

> Implement Slice 3J (Skills Library visibility) per
> `docs/superpowers/specs/2026-05-26-skillbox-slice-3j-skills-library-visibility-design.md`.
> Done when: (1) `skill.list` returns `projectsUsingCount` as `COUNT(DISTINCT projects.id)`
> through `installs -> project_providers -> projects` on `installs.skill_id = skills.id`,
> excluding removed projects and never matching by name or counting install rows;
> (2) read-only `skill.get` returns skill metadata plus one row per project/provider
> install (projectId, projectName, projectProviderId, providerKey, providerDisplayName,
> mode, status, projectSkillPath), excluding removed projects, with unknown id mapped to
> `validation_error`; (3) Skills Library shows a Projects column, working Search + Status
> filter (current statuses only), an Open Skill Host Folder button, and row-click
> navigation to a new read-only Skill Detail screen at `/skills/$skillId`; (4) full
> contract plumbing is wired (skill.get.json, index.json, generated TS, renderer
> client/query, Go handler + wiring, Electron method allowlist); (5) no new
> filesystem-write path and no migration are added; (6) `go test ./...`,
> `pnpm check:contracts-drift`, `pnpm typecheck`, and `pnpm test` all pass; (7)
> `generic_agents` is surfaced as "Shared Agent Skills (.agents)". Implement on Sonnet
> after this spec is approved.

*This `/goal` block is a draft only and is intentionally not executed during spec work.*
