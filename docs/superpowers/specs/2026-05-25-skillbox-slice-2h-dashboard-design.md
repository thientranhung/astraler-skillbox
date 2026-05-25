# Slice 2H: Dashboard — Design

Date: 2026-05-25
Status: approved (pre-spec decisions from lead applied)
Scope: add a read-only Dashboard overview surface backed by a single aggregate
JSON-RPC query, and make it the post-setup landing screen.

## Purpose / User Value

Skillbox is positioned as a local operational control center, and `docs/09`
treats the Dashboard as the primary landing surface. Today the app has no
Dashboard: after setup (or on boot with an active host) the index route
redirects straight to `/skills`, so the user never gets a system overview.

This slice fills that gap. The Dashboard answers "what is the overall state of my
Skillbox right now?" in one screen: active Skill Host Folder status, headline
counts (skills, projects, warnings), an install-mode breakdown across all
managed installs, and a list of active warnings. It is entirely read-only — it
aggregates data the app already persists from prior slices (host scan, project
scan, install, remove). No filesystem writes, no new mutating endpoints.

Secondary value: building the Dashboard forces a cross-table aggregate query over
`skills`, `projects`, `installs`, and `warnings`, which validates the
consistency of metadata accumulated by the install/remove slices without
introducing any new write risk.

## Applied Pre-Spec Decisions

1. **Dedicated Go RPC method `dashboard.get`.** The Dashboard is an aggregate
   view model. Cross-table business aggregation lives in a Go service, not in the
   renderer composing multiple endpoints. The renderer calls one query and
   renders the result.
2. **Global skills count and updates-available count are excluded.** Global
   scan, fetch, and version tracking are not implemented (no
   `global_provider_locations`, `global_installs`, or `fetch_results` tables yet).
   The `dashboard.get` response intentionally omits these fields. The UI renders
   them as muted, explicitly-labeled "Not in this slice" placeholders — never as
   fake `0` values that would imply real data.
3. **Install-mode aggregate is included now.** Derived from
   `installs.install_mode` across non-removed projects/providers. It will read
   `symlink: N, rsyncCopy: 0, direct: M` today, but it establishes the forward
   Dashboard contract so the rsync/copy slice changes data, not shape.
4. **Routing change.** Add `/dashboard`, add a sidebar item, and change the
   post-setup / boot index redirect from `/skills` to `/dashboard` whenever an
   active host exists.

## In Scope

- New query-only RPC method `dashboard.get` (no params) returning a
  `DashboardView`.
- New Go `DashboardService` that composes existing repositories into the view
  model.
- New read-only repository aggregate queries (counts and an active-warnings
  list); no schema migration.
- JSON Schema contract `methods/dashboard.get.json`, manifest entry, regenerated
  TypeScript types.
- Electron main allowlist entry `dashboard.get`.
- `core-client` wrapper `getDashboard()`.
- New renderer route `/dashboard`, `dashboard-screen.tsx`, a `use-dashboard`
  query hook, a sidebar nav item, and the index-redirect change.
- Loading / error / empty / zero-data states for the screen.
- Go and frontend tests, plus a full-stack smoke test.

## Out Of Scope / Non-Goals

- **Global skills count / Global Skills surface** — deferred; tables do not
  exist. Shown as a muted placeholder only.
- **Updates-available count / fetch** — deferred; no `fetch_results`. Muted
  placeholder only.
- **Warning quick-fix actions** (Relink, Remove, Update Path, Sync, Retry from
  the Dashboard) — out of scope. Dashboard warnings are display-only; a
  project-scoped warning row may navigate to its Project Detail, but performs no
  remediation.
- **Recent Operations panel** (shown in the `docs/09` wireframe) — deferred for
  this slice. The `operations` table stores no human-friendly target label, so a
  useful panel needs denormalization/join work that would expand scope. Note as a
  follow-up.
- **rsync/copy install support** — not implemented here. Only the aggregate
  contract field is added (reads `0`).
- **Any filesystem write or new mutating RPC.** `dashboard.get` is a pure query.
- **Schema migration.** This slice adds only `SELECT`-style repo methods.
- **Dashboard "Add Project" / "Fetch All" / "Scan Global" global action bar** —
  the screen may link to existing screens (e.g. a CTA to `/projects`), but adds
  no new global action commands.

## Data Contract

Method: `dashboard.get`
Params: none (`{}`).
Response: `DashboardView`.

```jsonc
{
  // null when no active host is configured (defensive; index normally
  // redirects to /setup in that case).
  "activeHost": {
    "hostId": 1,
    "path": "/abs/path/to/host",
    "skillsPath": "/abs/path/to/host/.agents/skills",
    "status": "active",            // SkillHostStatus enum
    "lastScanAt": "2026-05-25T10:31:00Z" // ISO 8601 or null
  },
  "summary": {
    "skills": 42,      // skills for the active host (0 when no host)
    "projects": 12,    // non-removed projects
    "warnings": 2      // total active warnings, all scopes
  },
  "installsByMode": {  // installs whose project.status != 'removed'
    "symlink": 9,
    "rsyncCopy": 0,
    "direct": 3
  },
  "warningsBySeverity": {
    "info": 0,
    "warning": 2,
    "error": 0,
    "blocking": 0
  },
  "warnings": [        // active warnings, capped (limit 50), ordered by id desc
    {
      "code": "broken_symlink",
      "message": "Broken symlink: project-a / skill-x",
      "severity": "warning",
      "scopeType": "install",   // WarningScopeType enum
      "scopeId": 17,            // nullable
      "actionKey": "relink"     // nullable; informational only this slice
    }
  ]
}
```

Contract notes:

- The response **omits** `globalSkills` and `updatesAvailable` entirely
  (decision 2). They are not `null`, not `0` — they are absent. The UI supplies
  the muted placeholder text statically.
- `activeHost: null` is the only "no host" signal; `summary.skills` is `0` and
  the other aggregates still reflect projects/installs/warnings that exist
  independently of a host.
- `installsByMode` keys are fixed (`symlink`, `rsyncCopy`, `direct`); any mode
  not present in the DB reads `0`. `rsyncCopy` maps to the stored
  `install_mode = 'rsync_copy'`.
- `warnings[]` is capped (50) to keep the payload bounded; `summary.warnings`
  and `warningsBySeverity` are full counts, so the UI can show "showing 50 of N".
- All three of `summary.warnings`, `warningsBySeverity`, and `warnings[]`
  **exclude** warnings owned by a removed project (project/provider/install scope
  whose project is `status = 'removed'`). Non-project-scoped warnings are always
  included. See the repo predicate under Backend Shape.
- All enums reuse existing domain enum string values; no new enum is introduced.
- Errors: database failures map to `database_error`. No `validation_error` path
  exists because the method takes no params.

## Backend Shape

### Repository methods (read-only, no migration)

Add focused aggregate queries beside existing repos, following the established
`QueryRowContext` / `QueryContext` + scan-helper patterns. All counts must
exclude `projects.status = 'removed'` where a project is in the join path.

- `SkillRepo.CountByHost(ctx, hostID) (int, error)` — `COUNT(*)` from `skills`
  for the host. (Avoids loading full rows just to count.)
- `ProjectRepo.CountActive(ctx) (int, error)` — `COUNT(*)` from `projects WHERE
  status <> 'removed'`.
- `InstallRepo.CountByModeActive(ctx) (domain.InstallModeCounts, error)` —
  `SELECT install_mode, COUNT(*) FROM installs i JOIN project_providers pp ON
  pp.id = i.project_provider_id JOIN projects p ON p.id = pp.project_id WHERE
  p.status <> 'removed' GROUP BY install_mode`. Service maps rows into the fixed
  `{symlink, rsyncCopy, direct}` struct (unknown modes ignored defensively).
- `WarningRepo.CountActiveBySeverity(ctx) (domain.WarningSeverityCounts, error)`
  — `SELECT severity, COUNT(*) FROM warnings WHERE is_resolved = 0 AND <not
  owned by a removed project>` ... `GROUP BY severity`.
- `WarningRepo.ListActive(ctx, limit int) ([]domain.Warning, error)` — active
  warnings, same removed-project exclusion as above, `ORDER BY id DESC LIMIT ?`,
  reusing the existing `scanWarning` helper.

**Removed-project warning exclusion (required).** Both dashboard warning queries
must exclude warnings that belong to a project whose `projects.status =
'removed'`. A soft-removed project's files are untouched on disk, but its stale
warnings must not inflate the Dashboard. Apply the exclusion per scope:

- `scope_type = 'project'`: exclude when `scope_id` is a removed project.
- `scope_type = 'project_provider'`: exclude when `scope_id`'s
  `project_providers.project_id` is a removed project.
- `scope_type = 'install'`: exclude when `scope_id`'s install →
  `project_providers.project_id` is a removed project.
- All other scope types (`app`, `skill_host_folder`, `skill`,
  `global_provider_location`, `global_install`, `source`, `database`) are not
  project-owned and are always included.

Concrete predicate (shared by both queries):

```sql
WHERE is_resolved = 0
  AND NOT (
        (scope_type = 'project'
           AND scope_id IN (SELECT id FROM projects WHERE status = 'removed'))
     OR (scope_type = 'project_provider'
           AND scope_id IN (
                SELECT pp.id FROM project_providers pp
                JOIN projects p ON p.id = pp.project_id
                WHERE p.status = 'removed'))
     OR (scope_type = 'install'
           AND scope_id IN (
                SELECT i.id FROM installs i
                JOIN project_providers pp ON pp.id = i.project_provider_id
                JOIN projects p ON p.id = pp.project_id
                WHERE p.status = 'removed'))
  )
```

A `NULL` `scope_id` cannot match any `IN (...)` subquery, so app/database-scoped
warnings with no `scope_id` pass through unaffected.

`summary.warnings` total = sum of `warningsBySeverity` (one count query is
enough; service sums it, no separate total query required). Because both queries
share the same predicate, the total and the list stay consistent — the list is
just the same population capped and ordered.

Defensive note: any warning row whose `severity` is not one of the recognized
values (`info`, `warning`, `error`, `blocking`) is ignored by
`CountActiveBySeverity` (not bucketed into a recognized severity, so it does not
distort counts) and is excluded from `ListActive` by a SQL `severity IN (...)`
filter — preserving the outbound `dashboard.get` contract which only allows
those four values.

### Service

`core-go/internal/services/dashboard_service.go`

- `DashboardView`, `DashboardSummary`, `InstallModeCounts`,
  `WarningSeverityCounts`, `DashboardWarningItem`, `DashboardActiveHost` view
  structs (mirroring the `SkillsLibraryView` / `SettingsView` style — exported
  view structs, no JSON tags here; JSON tags live on the handler response
  structs).
- `DashboardService` with repo interface dependencies: app-settings repo, host
  repo, skill repo, project repo, install repo, warning repo. New repo methods
  are added to the corresponding interfaces in `services/interfaces.go`.
- `Get(ctx) (*DashboardView, error)`:
  1. Read `app_settings`; resolve active host (nil-safe). If active host id is
     set but the row is missing, `activeHost = nil` (defensive, same pattern as
     `SettingsService.Get`).
  2. `skills` count: `0` if no active host, else `CountByHost`.
  3. `projects` count, `installsByMode`, `warningsBySeverity`, active warnings
     list — independent of host. The two warning queries already exclude
     removed-project-owned warnings at the repo layer, so the service does no
     extra filtering; it sums the severity buckets for `summary.warnings`.
  4. Any repo error → `domain.NewDatabaseError(...)`.

### Handler

`core-go/internal/rpc/handlers/dashboard_get.go`

- Narrow interface `dashboardService { Get(ctx context.Context)
  (*services.DashboardView, error) }`.
- Response structs with camelCase `json` tags matching the contract; reuse
  `formatTimePtr` for `lastScanAt`, `wrapError` for error mapping.
- `dashboard.get` ignores params (no request struct needed beyond an empty
  object), consistent with `settings.get`.

### Wiring

- `core-go/internal/app/wire.go`: add `"dashboard.get":
  rpchandlers.NewDashboardGetHandler(dashboardSvc)` to the handler map. `New(...)`
  gains a `dashboardSvc *services.DashboardService` parameter.
- `core-go/cmd/skillbox-core/main.go`: construct `DashboardService` with its
  repos and pass it into `app.New(...)`.
- `wire_test.go`: extend the expected registered-method set with `dashboard.get`.

## Contract Files

- Add `shared/api-contracts/methods/dashboard.get.json` — draft-07, `oneOf`
  Request/Response, matching `skill.list.json` structure. Request is an empty
  object (`additionalProperties: false`, no required props). Response defines
  `DashboardView` with the fields above; `globalSkills` / `updatesAvailable` are
  deliberately absent.
- Register it in `shared/api-contracts/index.json` (output
  `methods/dashboard-get.ts`).
- Run `pnpm generate:contracts`; commit regenerated `shared/generated/`.
- `pnpm check:contracts-drift` must pass.

## Renderer Shape

### core-client

`apps/desktop/renderer/src/lib/core-client/methods.ts`:

```ts
getDashboard: () => invoke<DashboardGetResponse>("dashboard.get", {}),
```

Import `DashboardGetResponse` from `@contracts/index.js`.

### Query hook

`apps/desktop/renderer/src/features/dashboard/use-dashboard.ts` — a TanStack
Query hook mirroring `use-app-settings`, keyed `["dashboard"]`, calling
`methods.getDashboard()`.

### Screen

`apps/desktop/renderer/src/screens/dashboard-screen.tsx` — operational layout
(no SaaS hero/cards-in-cards, per `CLAUDE.md` UI style):

- **Skill Host Folder** block: path + status badge + last scan time. (Display
  only this slice; no Change/Scan buttons — those belong to Settings.)
- **Summary** row: Skills, Projects, Warnings counts. Plus muted, clearly
  labeled placeholders for "Global skills — Not in this slice" and "Updates — Not
  in this slice" (decision 2).
- **Installs by mode**: symlink / rsync-copy / direct counts.
- **Warnings** list: severity, message, scope. Rows are display-only; a
  project-scoped row links to its Project Detail. No remediation actions.

### Routing & sidebar

- `router.tsx`: add `dashboardRoute` under `shellRoute` at path `/dashboard`;
  add it to `shellRoute.addChildren([...])`. Change `IndexRedirector` to
  `navigate({ to: "/dashboard" })` (from `/skills`) when `data?.activeHost != null`.
- `sidebar.tsx`: add `{ to: "/dashboard", label: "Dashboard", icon:
  LayoutDashboard }` as the first `NAV_ITEMS` entry (lucide-react icon).

## States

- **Loading**: query pending → centered spinner (same pattern as
  `IndexRedirector`).
- **Error**: query error → inline error panel with a Retry button
  (`refetch()`); does not crash the shell, navigation stays usable.
- **No host (defensive)**: `activeHost === null` → show a "No Skill Host Folder
  configured" notice with a link to `/setup`. Normally unreachable because the
  index route redirects to `/setup`, but the screen must not assume a host.
- **Zero data**: host configured but `projects === 0` / `skills === 0` → show
  `0` counts (these are real zeroes, unlike the deferred placeholders) with a
  CTA linking to `/projects` (Add Project) and `/skills`.
- **No warnings**: warnings list shows an empty "No active warnings" state.

## Tests

### Go

- `dashboard_service_test.go` (table-driven, mock repos in the existing
  `mocks_test.go` style):
  - no active host → `activeHost == nil`, `skills == 0`, other aggregates still
    populated.
  - active host present → skills counted via `CountByHost`.
  - install-mode aggregation maps `symlink` / `rsync_copy` / `direct` correctly;
    absent modes read `0`; unknown mode ignored.
  - severity counts and `summary.warnings` total agree; warnings list capped at
    limit.
  - removed-project warnings are excluded: a mock returning severity counts and
    a warnings list that already omit removed-project rows flows through
    unchanged, and `summary.warnings` equals the (already-filtered) severity sum
    (confirms the service adds no second filter and double-counts nothing).
  - repo error → `database_error`.
- Repo tests (temp SQLite + fixtures, existing pattern): `CountByHost`,
  `CountActive` excludes `removed`, `CountByModeActive` excludes installs of
  removed projects, `CountActiveBySeverity` and `ListActive` ignore resolved
  warnings and order by id desc.
- Removed-project warning exclusion (required): seed warnings on a removed
  project across all three scopes (`project`, `project_provider`, `install`) plus
  a non-project-scoped warning (e.g. `skill_host_folder` or `app`). Assert
  `CountActiveBySeverity` and `ListActive` omit the three removed-project
  warnings and retain the non-project-scoped one. Include a control case where
  the same warnings on an `active` project are counted/listed. Cover a
  `NULL`-`scope_id` app/database warning to confirm it is never excluded.
- Defensive severity: seed a warning with an unrecognized `severity` value and
  assert `CountActiveBySeverity` does not bucket it into any recognized severity
  (totals unaffected) and `ListActive` does not return it (unrecognized severities
  are excluded by SQL filter to preserve the outbound contract).
- Handler contract test (`dashboard_get`) asserting the response JSON matches
  `methods/dashboard.get.json` and that `globalSkills` / `updatesAvailable` are
  absent.
- `wire_test.go` updated to expect `dashboard.get` registered.

### Frontend (Vitest + RTL)

- `use-dashboard` hook test with mocked `core-client` (success + error).
- `dashboard-screen` render tests: loading spinner, error + retry, loaded view
  with counts, muted deferred placeholders present, zero-data CTA, empty-warnings
  state.
- Router test: index redirects to `/dashboard` when `activeHost != null`, and to
  `/setup` when null.
- Sidebar test: Dashboard item present and active on `/dashboard`.
- `pnpm check:contracts-drift` passes (generated types committed).

## Smoke Test

Full-stack (`pnpm dev`, real Go sidecar):

1. With an active host plus a few projects and at least one symlink install,
   launch the app. It lands on `/dashboard` (not `/skills`). The sidebar shows
   Dashboard active.
2. Summary counts match what Skills Library and Projects screens show. Installs
   by mode shows `symlink = N`, `rsync-copy = 0`, `direct = M`.
3. Global skills and Updates render as muted "Not in this slice" placeholders,
   not as `0`.
4. If a warning exists (e.g. a broken symlink from a prior scan), it appears in
   the warnings list with correct severity; a project-scoped row navigates to
   that Project Detail.
5. Soft-remove a project that has at least one active warning (e.g. a broken
   symlink). Return to Dashboard (or refetch) → project count and any
   install-mode counts drop accordingly, AND the removed project's warning
   disappears from the warnings list, the `Warnings` summary count, and the
   severity breakdown. A warning on a still-active project remains visible.
6. Stop the Go sidecar mid-session → Dashboard shows the error state with Retry,
   shell remains navigable.

## Acceptance Criteria

- `dashboard.get` is registered, allowlisted, and returns the `DashboardView`
  contract; the response omits global/updates fields.
- Post-setup and boot-with-host both land on `/dashboard`; `/setup` still wins
  when no host.
- Counts exclude removed projects everywhere they appear, including all warning
  aggregates and the warnings list (project/provider/install-scoped warnings of a
  removed project are filtered out; non-project-scoped warnings are retained).
- Install-mode aggregate is present and correct (symlink populated, rsync-copy
  `0`, direct as applicable).
- Deferred metrics are shown as muted placeholders, never fake zeroes.
- No filesystem writes and no new mutating endpoints are introduced.
- Desktop typecheck, frontend tests, Go tests, build, and contract-drift checks
  all pass.

## Open Questions

- **Recent Operations panel**: deferred here for the denormalization reason
  above. Confirm it should be its own follow-up slice (likely needs a friendly
  target-label resolution step) rather than folded into a later Dashboard pass.
- **Warnings list cap**: spec assumes a 50-row cap with full counts shown
  separately. Confirm 50 is acceptable, or whether the first Dashboard pass
  should show only `warningsBySeverity` and defer the list entirely.
- **Deferred-placeholder presentation**: spec assumes muted inline labels
  ("Not in this slice"). Confirm this over simply hiding the two metrics until
  their slices land.
