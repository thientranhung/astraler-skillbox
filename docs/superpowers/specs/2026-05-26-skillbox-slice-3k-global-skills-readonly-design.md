# Slice 3K — Global Skills Read-Only Visibility (Design Spec)

- **Date:** 2026-05-26
- **Status:** Draft (design only — not implemented)
- **Scope:** Read-only filesystem. New schema migration (`000005`). One supported provider only.
- **Lead review applied:** Alt A (Shared Agent Skills `.agents` only); no Claude global scan; stable
  singleton scan-lock target; explicit global identifiers in `global.list`; explicit global warning
  scopes + regeneration; overlap warning / install-to-global / remove / relink / Settings configure /
  Dashboard count all deferred.

## 1. Scope and Non-Goals

### 1.1 In Scope

1. New tables `global_provider_locations` and `global_installs` via migration `000005`, plus
   `generic_agents.has_global_level = 1`.
2. Read-only global scan of **Shared Agent Skills (.agents)** at `~/.agents/skills` only, surfaced
   through a new `global.scan` command (returns `operationId`).
3. Read-only `global.list` query returning the persisted global location(s) and their entries with
   stable identifiers and full path fields.
4. New sidebar item **Global Skills**, route `/global`, and a read-only `GlobalSkillsScreen`.
5. New warning scopes `global_provider_location` and `global_install`, regenerated on each scan.
6. Actions limited to **Scan Global** and **Open Folder** (native `dialog.openPath`).

### 1.2 Non-Goals (explicitly deferred)

- **No Claude global scan or persisted Claude global location.** Provider-model.md:368 states the
  Claude global convention is unverified. Claude may appear in UI copy as a deferred/experimental
  note, but this slice performs no Claude global scan and writes no `global_provider_locations` row
  for Claude.
- No install-to-global, no remove, no relink, no switch-mode — no global **write** actions of any kind.
- No `global_project_skill_overlap` warning (deferred).
- No Settings "Global Provider Locations" configure/change flow (no user-set global path this slice;
  path is auto-resolved from `~/.agents/skills`).
- No Dashboard "Global skills: N" count wiring.
- No folder creation: scan only **reads** `~/.agents/skills`; it never runs the `EnsureAgentsSkills`
  scaffold used by host setup.
- No fetch/update/version/source integration.

## 2. User Experience

```text
Global Skills

[Scan Global] [Open Folder]

Global Locations
  Provider                       Path                 Status   Entries
  Shared Agent Skills (.agents)  ~/.agents/skills     active   4

Global Entries
  Provider                       Skill            Mode     Status            Actions
  Shared Agent Skills (.agents)  research-writer  direct   current           [Open]
  Shared Agent Skills (.agents)  adr-helper       symlink  current           [Open]
  Shared Agent Skills (.agents)  old-cmd          symlink  broken symlink    [Open]

[warning] Global skill old-cmd has a broken symlink.
```

- First visit (no scan yet): the location row is shown with the persisted status, or the empty state
  **"No global skills found."** with a single `[Scan Global]` button (no Configure button — deferred).
- `[Scan Global]` runs the read-only scan, shows a scanning/progress state, then refreshes the list.
- `[Open Folder]` opens the location `skillsPath` natively; `[Open]` per entry opens that entry's path.
- Global entries are **never merged** with project installs and are presented as a separate surface.
- `[Relink]`/`[Remove]` from the wireframe are **not rendered** this slice (write actions deferred).

## 3. Data Model and Migration Outline

### 3.1 Migration `000005_global_skills`

`000005_global_skills.up.sql` creates two tables (columns mirror data-model.md §10/§12) and flips the
provider flag:

```sql
CREATE TABLE IF NOT EXISTS global_provider_locations (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id  INTEGER NOT NULL REFERENCES provider_definitions(id),
    name                    TEXT,
    path                    TEXT,            -- global root (~/.agents); nullable when not_configured
    skills_path             TEXT,            -- ~/.agents/skills
    status                  TEXT NOT NULL DEFAULT 'not_configured',
    last_scanned_at         TEXT,
    created_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_global_loc_provider ON global_provider_locations(provider_definition_id);

CREATE TABLE IF NOT EXISTS global_installs (
    id                            INTEGER PRIMARY KEY AUTOINCREMENT,
    global_provider_location_id   INTEGER NOT NULL REFERENCES global_provider_locations(id),
    skill_id                      INTEGER REFERENCES skills(id),
    skill_name                    TEXT    NOT NULL,
    install_mode                  TEXT    NOT NULL,                 -- symlink | rsync_copy | direct
    install_status                TEXT    NOT NULL DEFAULT 'current',
    global_skill_path             TEXT    NOT NULL,
    source_skill_path             TEXT,
    symlink_target_path           TEXT,
    installed_from_host_folder_id INTEGER REFERENCES skill_host_folders(id),
    installed_version             TEXT,
    installed_commit              TEXT,
    installed_checksum            TEXT,
    last_synced_at                TEXT,
    last_scanned_at               TEXT,
    created_at                    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at                    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_global_installs_loc_path
    ON global_installs(global_provider_location_id, global_skill_path);
CREATE INDEX IF NOT EXISTS idx_global_installs_location
    ON global_installs(global_provider_location_id);

UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 5, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
```

`000005_global_skills.down.sql` drops both tables (installs first), reverts
`generic_agents.has_global_level = 0`, and sets `database_version = 4`.

**Claude is untouched:** `claude.has_global_level` remains `1` from migration 000003, but the scan
service (§5) does not enumerate Claude this slice, so no Claude `global_provider_locations` row is
ever written.

### 3.2 `install_status` values used this slice

Because there are no global rsync/copy records yet (no install-to-global), scanned statuses are a
subset of the existing project-install enum: `current`, `old_host`, `external_symlink`,
`broken_symlink`, `error`. The column accepts the full enum for forward compatibility; the service
only emits this subset.

### 3.3 Domain enum additions

- `domain.WarningScopeType`: add `WarningScopeGlobalProviderLocation = "global_provider_location"` and
  `WarningScopeGlobalInstall = "global_install"` (warning.go).
- `domain.OperationType`: add `OperationTypeScanGlobalSkills = "scan_global_skills"` (operation.go).
- New `domain.GlobalLocationStatus` enum with values `active`, `not_configured`, `missing`,
  `unreadable`, `invalid_structure`, `empty`, `disabled`.

## 4. Provider / Filesystem Detection Behavior

### 4.1 Optional global adapter interface

Add an optional interface so only global-capable adapters implement it; adapters stay pure (no
`os.UserHomeDir`, no filesystem writes):

```go
type GlobalProviderAdapter interface {
    ProviderAdapter
    DetectGlobal(homeDir string, fs FsReader) (GlobalDetectResult, error)
}

type GlobalDetectResult struct {
    Present         bool
    GlobalPath      string                 // ~/.agents
    GlobalSkillsPath string                // ~/.agents/skills
    Status          domain.GlobalLocationStatus
    Entries         []AdapterEntry         // reuse existing AdapterEntry
    Warnings        []AdapterWarning
}
```

`GenericAgentsAdapter` implements `DetectGlobal`, resolving `homeDir/.agents` and
`homeDir/.agents/skills`. `ClaudeAdapter` does **not** implement it this slice (or implements it but is
never invoked, since the service filters Claude out — see §5). The service resolves `homeDir` once via
the filesystem gateway (`os.UserHomeDir`) and passes it in.

### 4.2 Detection rules (read-only, mirror `generic_agents_test.go` project rules)

Resolved against `~/.agents`:

1. `~/.agents` missing → `status = missing`, no entries; warning `global_provider_location_missing`.
2. `~/.agents` exists but is a file or unreadable → `status = invalid_structure`; warning.
3. `~/.agents` dir, `~/.agents/skills` missing → `status = missing` (skills root absent); no entries.
   (Scan **does not create** the folder.)
4. `~/.agents/skills` exists, readable, zero entries → `status = empty`.
5. `~/.agents/skills` exists, readable, ≥1 entry → `status = active`; emit entries.
6. `~/.agents/skills` exists but unreadable → `status = unreadable`; warning.

### 4.3 Entry classification

Global entries are classified with the **project-install** semantics (not host semantics), comparing
symlink targets to the active Skill Host Folder:

- Symlink target resolves inside the active host `skills_path` → `install_mode = symlink`,
  `install_status = current`; `source_skill_path` = resolved target; `symlink_target_path` = raw target.
- Symlink target resolves inside a known-but-inactive host → `old_host`.
- Symlink target resolves outside any known host → `external_symlink`.
- Symlink target does not exist → `broken_symlink`.
- Plain directory (no symlink) → `install_mode = direct`, `install_status = current`,
  `skill_id = NULL`. (Phase-1 has no global rsync/copy DB records, so plain dirs are direct-mode
  current entries.)
- Unclassifiable entry → `install_status = error`.
- `skill_id` is set only when the entry name matches a host skill; otherwise `NULL`.

Reuse the existing project classification helper rather than duplicating it.

## 5. Service / RPC / Contracts

### 5.1 Service

`GlobalSkillsService.ScanGlobal(ctx)`:

1. Resolve `homeDir` via filesystem gateway.
2. Load `provider_definitions WHERE has_global_level = 1 AND status != 'disabled'`, **then filter to
   `key = 'generic_agents'` this slice** (Claude excluded explicitly; documented as the Alt-A gate).
3. For each remaining provider, type-assert its adapter to `GlobalProviderAdapter`; skip if not
   implemented. Call `DetectGlobal(homeDir, fs)`.
4. Upsert one `global_provider_locations` row per scanned provider (stable by
   `provider_definition_id`), setting `path`, `skills_path`, `status`, `last_scanned_at`.
5. Classify entries (§4.3) and commit `global_installs` atomically via a `GlobalScanCommitter`:
   upsert present entries, delete `global_installs` no longer on disk for that location, regenerate
   warnings (§7).
6. Return a summary `{ entriesFound, warningsCreated }` as the operation result.

**Scan lock (stable before first scan):** the operation uses a fixed app-level singleton target,
`operations.Target{Type: "global_scan", ID: 0}` (constants `GlobalScanTargetType = "global_scan"`,
`GlobalScanTargetID int64 = 0`), with `OperationType = scan_global_skills`. This is required because no
`global_provider_locations.id` exists before the first scan. Phase-1 fail-fast: a second concurrent
`global.scan` returns `conflict_error`.

The service performs **no filesystem writes**: read-only `PathInfo`/`ListSkillEntries`/scan only.

### 5.2 RPC methods

**`global.scan`** (command) → `{ "operationId": integer }`. Progress via `operation.progress`
server-push notifications, consistent with `host.scan`.

**`global.list`** (query) → locations with full identifiers and paths:

```jsonc
{
  "locations": [
    {
      "globalProviderLocationId": 1,
      "providerKey": "generic_agents",
      "providerDisplayName": "Shared Agent Skills (.agents)",
      "providerStatus": "supported",          // provider_definitions.status
      "path": "/Users/x/.agents",             // nullable
      "skillsPath": "/Users/x/.agents/skills",// nullable
      "status": "active",                     // active|not_configured|missing|unreadable|invalid_structure|empty|disabled
      "lastScannedAt": "2026-05-26T...",      // nullable
      "entries": [
        {
          "globalInstallId": 10,
          "skillName": "adr-helper",
          "skillId": 7,                       // nullable
          "mode": "symlink",                  // symlink|rsync_copy|direct
          "status": "current",               // install_status subset (§3.2)
          "globalSkillPath": "/Users/x/.agents/skills/adr-helper",
          "sourceSkillPath": "/host/.agents/skills/adr-helper", // nullable
          "symlinkTargetPath": "/host/.agents/skills/adr-helper" // nullable
        }
      ],
      "warnings": [
        {
          "code": "broken_symlink",
          "severity": "warning",              // info|warning|error|blocking
          "scopeType": "global_install",      // global_provider_location|global_install
          "scopeId": 10,                      // nullable
          "actionKey": "rescan",              // nullable
          "message": "Global skill old-cmd has a broken symlink"
        }
      ]
    }
  ]
}
```

`global.list` returns locations even when `status` is `missing`/`empty`/`not_configured` so the UI can
render the row and guide the user. Entries are ordered by `skillName` for stable output.

### 5.3 Contract plumbing (end to end)

- New schemas `shared/api-contracts/methods/global.scan.json` and `global.list.json`;
  `additionalProperties: false`; add both to `shared/api-contracts/index.json`.
- `pnpm generate:contracts` then commit `shared/generated/`.
- Go handlers `global_scan.go` + `global_list.go` in `core-go/internal/rpc/handlers/`, registered and
  wired in the composition root (manual DI). Add both method names to the `server.ready`
  capability/method list if one is advertised.
- **Electron main method allowlist:** add `global.scan` and `global.list` so main forwards them.
- Renderer core-client: typed `globalScan()` and `globalList()`; TanStack Query keys
  `["global", "list"]` and a `useGlobalList()` / scan mutation hook under `features/global-skills/`.

## 6. UI Shape

- **Sidebar:** insert `{ to: "/global", label: "Global Skills", icon: Globe }` (lucide-react)
  between **Skills** and **Projects** in `NAV_ITEMS` (sidebar.tsx). (Updates remains out of scope.)
- **Router:** add `/global` under the app shell in `app/router.tsx`; new
  `screens/global-skills-screen.tsx`.
- **Screen:** header `[Scan Global] [Open Folder]`; a Global Locations table
  (Provider / Path / Status / Entries count); a Global Entries table grouped by provider
  (Provider / Skill / Mode / Status / `[Open]`); warning rows beneath; empty state
  **"No global skills found."** with only `[Scan Global]`.
- `[Open Folder]` and per-row `[Open]` reuse the existing native open-folder mechanism
  (`dialog.openPath`) — no new IPC write surface.
- The `generic_agents` provider is always shown as **"Shared Agent Skills (.agents)"** (use
  `providerDisplayName` from the DB; never surface the raw key).
- **No write controls** (no Relink/Remove/Install/Switch).

## 7. Error Handling and Warnings

### 7.1 Warning scopes and codes

- `global_provider_location_missing` — scope `global_provider_location`, `scopeId` = location id,
  severity `warning`, `actionKey = rescan`. Emitted when `~/.agents` or `~/.agents/skills` is absent.
- `invalid_structure` — scope `global_provider_location`, when `~/.agents` is a file/unreadable.
- `broken_symlink` — scope `global_install`, `scopeId` = `global_installs.id`, severity `warning`,
  `actionKey = rescan`.
- `external_symlink` — scope `global_install`, severity `warning`.
- `old_host_symlink` — scope `global_install`, severity `warning`.
- `global_project_skill_overlap` is **deferred** (not emitted this slice).

### 7.2 Regeneration semantics

On each successful scan, the commit step **clears all active warnings** scoped to the scanned
`global_provider_location` (by `scopeType = global_provider_location` + that location id) and to its
`global_install` rows, then inserts the freshly computed warnings. This mirrors host-scan
`CommitScanResults` (skill_host_service.go) so stale warnings never linger after the underlying entry
is fixed and rescanned.

### 7.3 Operation/transport errors

- Concurrent `global.scan` → `conflict_error` (app code, outside reserved JSON-RPC range).
- Home dir unresolvable → `filesystem_error`; the scan fails, prior DB state is left unchanged.
- Unreadable `~/.agents/skills` → location `status = unreadable` + scope warning; partial-success scan
  (not a hard failure).

## 8. Test / Verification Plan

**Backend (`go test ./...`, `-race` on filesystem/providers):**

- `GenericAgentsAdapter.DetectGlobal` rules §4.2: missing root, root-is-file, skills-missing,
  empty, active, unreadable. Assert **no folder is created** when `~/.agents/skills` is absent.
- Entry classification §4.3: symlink → `current` / `old_host` / `external_symlink` / `broken_symlink`;
  plain dir → `mode = direct`, `status = current`, with `skill_id NULL`; name-matched dir links
  `skill_id`.
- `GlobalSkillsService.ScanGlobal` against a temp `$HOME` fixture: upserts one location, commits
  installs, returns summary; Claude is **not** enumerated (no Claude location row written);
  `conflict_error` on concurrent scan; stable `global_scan/0` target before any location exists.
- Repo tests for `global_provider_locations` + `global_installs`: upsert-by-provider, delete-missing
  entries, idempotent re-scan.
- **Warning scope tests:** `global_provider_location_missing` created when skills root absent;
  `broken_symlink` scoped to the correct `global_install` id; **regeneration** — a warning present on
  scan 1 is cleared on scan 2 after the entry is fixed.
- `migration_000005_test.go`: up creates both tables and sets `generic_agents.has_global_level = 1`
  and `database_version = 5`; down drops tables, reverts the flag, and sets version `4`; re-running up
  is idempotent. Claude's `has_global_level` stays `1` (unchanged).
- `contract_drift_test` covers `global.scan` / `global.list`.

**Renderer (`pnpm test`, `pnpm typecheck`):**

- `GlobalSkillsScreen` renders locations, grouped entries, status badges, warning rows, and the empty
  state; asserts **no** write-action controls; Open Folder/Open wired to the native path action.
- Sidebar test: "Global Skills" item present between Skills and Projects, active-state styling.
- Router test: `/global` resolves to the screen.

**Contracts / types:** `pnpm generate:contracts` then `pnpm check:contracts-drift` clean;
`pnpm typecheck` green with new generated types.

**Manual (`pnpm dev`):** with a real `~/.agents/skills` containing one symlinked + one plain-dir
entry → Scan Global → verify rows/statuses; Open Folder opens `~/.agents/skills`; a broken symlink
shows a `global_install`-scoped warning; with `~/.agents/skills` absent, verify the location shows
`missing` and the folder is **not** created.

Full command set:

```bash
cd core-go && go test ./...
cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift && pnpm typecheck && pnpm test
```

## 9. Acceptance Criteria

1. Migration `000005` creates `global_provider_locations` and `global_installs`, sets
   `generic_agents.has_global_level = 1`, bumps `database_version` to 5; the down migration reverts all
   three. Claude's global flag and global data are untouched and never scanned.
2. `global.scan` runs a **read-only** scan of `~/.agents/skills` only, never creates folders, uses the
   stable singleton lock target `global_scan/0` with operation type `scan_global_skills`, returns
   `operationId`, and returns `conflict_error` on a concurrent scan.
3. `global.list` returns each location with `globalProviderLocationId`, `providerKey`,
   `providerDisplayName`, `providerStatus`, `path`, `skillsPath`, location `status`, `lastScannedAt`,
   and per entry `globalInstallId`, `skillName`, `skillId`, `mode`, `status`, `globalSkillPath`,
   `sourceSkillPath`, `symlinkTargetPath`; plus warnings carrying `code`, `severity`, `scopeType`,
   `scopeId`, `actionKey`, `message`.
4. Global entries are classified with project-install symlink semantics
   (`current`/`old_host`/`external_symlink`/`broken_symlink`/`error` statuses; plain dirs are
   `mode = direct`, `status = current`) and are never merged with project installs.
5. Warnings use scopes `global_provider_location` and `global_install` and are **regenerated** (old
   active warnings for the scanned location and its installs cleared, fresh ones inserted) on each scan.
6. New sidebar **Global Skills** item, `/global` route, and a read-only `GlobalSkillsScreen` with only
   **Scan Global** and **Open Folder** actions; no Relink/Remove/Install controls; empty state shows
   only `[Scan Global]`.
7. Deferred items are absent: no overlap warning, no install-to-global, no remove/relink, no Settings
   global-location configure flow, no Dashboard global count.
8. Contract plumbing complete end to end: `global.scan.json`, `global.list.json`, `index.json`,
   generated TS, renderer core-client + query/mutation hooks, Go handlers + app wiring, Electron main
   method allowlist (and `server.ready` capabilities if applicable).
9. `generic_agents` is surfaced to the user as **"Shared Agent Skills (.agents)"**.
10. `go test ./...`, `pnpm check:contracts-drift`, `pnpm typecheck`, and `pnpm test` all pass.

## 10. Draft `/goal` Condition (for implementation — NOT run during spec work)

> Implement Slice 3K (Global Skills read-only visibility) per
> `docs/superpowers/specs/2026-05-26-skillbox-slice-3k-global-skills-readonly-design.md`.
> Done when: (1) migration `000005` creates `global_provider_locations` + `global_installs`, sets
> `generic_agents.has_global_level = 1` and `database_version = 5`, with a reverting down migration,
> leaving Claude global untouched and unscanned; (2) `global.scan` performs a read-only scan of
> `~/.agents/skills` only (no folder creation), uses the stable singleton lock target `global_scan/0`
> with operation type `scan_global_skills`, returns `operationId`, and returns `conflict_error` on
> concurrent runs; (3) `global.list` returns locations + entries with the full identifier/path field
> set and warnings carrying scope/action fields (§5.2/§7); (4) entries are classified with
> project-install symlink semantics and never merged with project installs; (5) warnings use scopes
> `global_provider_location` + `global_install` and are regenerated each scan; (6) a new Global Skills
> sidebar item + `/global` route + read-only screen ship with only Scan Global / Open Folder actions;
> (7) deferred items (overlap, install-to-global, remove/relink, Settings configure, Dashboard count)
> are absent; (8) full contract plumbing is wired (global.scan.json, global.list.json, index.json,
> generated TS, renderer client/query, Go handlers + wiring, Electron allowlist); (9) `generic_agents`
> shows as "Shared Agent Skills (.agents)"; (10) `go test ./...`, `pnpm check:contracts-drift`,
> `pnpm typecheck`, and `pnpm test` all pass. Implement on Sonnet after this spec is approved.

*This `/goal` block is a draft only and is intentionally not executed during spec work.*
