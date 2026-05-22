# Data Model Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: README.md, docs/index.md, docs/01-product-brief.md,
  docs/02-product-notes.md, docs/03-information-architecture.md,
  docs/04-user-flows.md, docs/05-edge-cases-and-ux-states.md,
  docs/06-data-model.md

## Executive Summary

The data model is substantially complete and well-reasoned for Phase 1. The
entity set, status enums, and user flow mappings are solid. However, four issues
should be fixed before implementation begins: (1) `install_mode` and
`install_status` mix user intent with detected filesystem state, causing
ambiguity at the boundary; (2) `projects.status = has_warnings` is a derived
value masquerading as a first-class entity state; (3) `fetch_results` has a dual
FK design (both `skill_id` and `source_id`) that creates ambiguity when a
skill's source changes; and (4) `provider_definitions` has only a single
`default_relative_skills_path` field, which is insufficient for providers that
use multiple candidate detection paths. Everything else is acceptable for Phase 1
with the suggested improvements noted below.

## Critical Issues

1. **`install_mode` blurs intent vs detected state.**
   `installs.install_mode` includes `external_symlink` and `unknown`, which are
   detected filesystem states, not modes the user chose. `install_mode` should
   encode only the user's or Skillbox's management intent (`symlink`,
   `rsync_copy`, `direct`). Detected states like `external_symlink` and
   `unknown` should live exclusively in `install_status`. Currently,
   `install_mode = external_symlink` and `install_status = external` carry
   overlapping semantics. Fix: remove `external_symlink` and `unknown` from
   `install_mode`; express those states fully in `install_status`.

2. **`projects.status = has_warnings` should not be a first-class status.**
   `has_warnings` is a derived flag, not a project lifecycle state. A project
   can be `active` and have warnings simultaneously. Using `has_warnings` as a
   status creates mutually exclusive ambiguity (is a project `active` or
   `has_warnings`?). Fix: remove `has_warnings` from `projects.status`. The
   Dashboard and Projects view should compute warning presence from the
   `warnings` table directly, or add a separate `has_warnings` boolean column
   if a denormalized flag is needed for performance.

3. **`fetch_results` dual FK (`skill_id` + `source_id`) is ambiguous.**
   The relationship overview shows `fetch_results` linked to both `skills.id`
   and `skill_sources.id`. A skill has at most one source at any time. If a
   skill's source changes later, old `fetch_results` rows become orphaned or
   misleading. Fix: link `fetch_results` to `source_id` only. The skill context
   is recoverable via `skills.source_id`. If `skill_id` is kept for query
   convenience, document it as a denormalized helper, not an independent FK.

4. **`provider_definitions` cannot support multi-path provider detection.**
   `default_relative_skills_path` is a single text field. Claude, for example,
   may need to check `.claude/commands` and `.agents/skills` during project
   scan. A single field blocks provider adapters from expressing this. Fix:
   add a separate `provider_path_candidates` table
   (`provider_definition_id`, `relative_path`, `priority`, `description`) or
   store a JSON array in `provider_definitions`. Resolve this before writing
   provider adapter code.

## Suggested Improvements

1. **Add `current_checksum` to `skills` for `local_modified` detection.**
   `skills.status = local_modified` implies the host folder copy diverges from
   the upstream source. The model stores `current_version` and `current_commit`
   but no checksum of the on-disk content. Without a `current_checksum`, the app
   cannot detect local modifications for non-git sources (local/manual). Add
   `current_checksum TEXT` to `skills`.

2. **Remove redundant `project_id` from `installs`.**
   `installs.project_provider_id` already implies a project through
   `project_providers.project_id`. Storing `project_id` again in `installs`
   creates a consistency risk if the two are ever out of sync. Either remove
   `installs.project_id` and join through `project_providers`, or enforce the
   consistency with a CHECK or application-level validation. Pick one approach
   before implementation.

3. **Add a FK (or metadata link) from `warnings` to the operation/scan that
   generated them.**
   Currently `warnings` has no link back to which scan or operation created
   them. After a scan regenerates warnings, stale warnings may linger. A
   `source_operation_id` field (nullable FK to `operations`) makes it possible
   to identify and prune warnings from superseded scans.

4. **Auth credentials need a home.**
   `app_settings` has no fields for GitHub or Vercel auth tokens. `Settings`
   in `docs/03-information-architecture.md` mentions "GitHub/Vercel settings if
   needed." Storing tokens in SQLite is acceptable for Phase 1 but they should
   be in a separate `api_credentials` table (`provider_key`, `credential_type`,
   `value_encrypted`), not in `app_settings`, to keep concerns separated and to
   allow token rotation without touching app config.

5. **`scan_results` vs `operations` overlap.**
   Both tables can represent a scan operation. `operations` has `operation_type
   = scan` and `scan_results` also records a scan with `target_type`. Consider
   merging `scan_results` into `operations` via `metadata_json` for Phase 1, or
   clearly document which table is the authoritative scan audit trail.

6. **`app_settings` should allow `active_skill_host_folder_id = NULL`
   explicitly.**
   During first-time setup, no host folder exists yet. The FK to
   `skill_host_folders` should be nullable and the app must not crash if it is
   null. Document this constraint clearly in the schema.

7. **Add `removed` status to `installs` if soft delete is chosen.**
   The data model's open questions ask about soft delete vs hard delete for
   installs. If soft delete is chosen, add `removed` to `install_status` to
   distinguish an install that was actively removed from one that is `missing`
   due to external filesystem changes.

8. **`fetch_results` should record `previous_version` / `previous_commit`.**
   The Updates view shows "current version" vs "latest version." Currently,
   `fetch_results` stores `current_version` and `latest_version`. This is
   correct. However, after a skill is updated, the "current" version at fetch
   time becomes historical. The model handles this adequately via fetch history,
   but the naming `current_version` in `fetch_results` is ambiguous: it means
   "version at the time of this fetch," not "version right now." Rename to
   `host_version_at_fetch` and `upstream_version_at_fetch` to eliminate
   confusion.

## Missing Concepts Or Tables

- **`provider_path_candidates`**: Multi-path detection per provider (critical,
  noted above).
- **`api_credentials`**: Auth tokens for GitHub/Vercel sources.
- **`skill_format_metadata`** (Phase 2 prep): No table stores what format a
  skill is in (e.g., Claude markdown, opencode YAML). Phase 2 conversion will
  need this. Not a Phase 1 blocker, but flag the open question now.
- **`installs` soft-delete mechanism**: No `removed` status defined yet.
- **`current_checksum` in `skills`**: Missing for local modification detection.
- **`source_operation_id` in `warnings`**: Link back to generating scan/operation.

## Over-Modeled Or Risky Areas

- **`skill_sources` GitHub fields as top-level columns**: `github_owner`,
  `github_repo`, `github_path`, `github_ref`, and `vercel_skill_id` are all
  nullable and only apply to specific source types. This results in many nulls
  for `local` and `manual` sources. Acceptable for Phase 1 but consider
  migrating to a `source_config_json` column if more source types are added
  later.

- **`warnings` as a persisted table vs derived state**: If warnings are always
  regenerated by scan, persisting them long-term adds maintenance burden without
  benefit. A simpler approach: only persist active (unresolved) warnings and
  hard-delete them on the next successful scan of the same scope. The current
  design with `is_resolved` and `resolved_at` is more complex than needed for
  Phase 1.

- **`operations` table as full audit trail**: Phase 1 needs basic operation
  tracking for UI loading state. A full audit trail with history adds write
  volume and query complexity. Consider limiting `operations` retention (e.g.,
  last N operations per target) from the start.

- **`installs.skill_id` nullable + `skill_name` redundancy**: Dual-path for
  displaying the skill name is necessary but creates a semantic fork. Document
  the rule: `skill_name` is always written at install time and never updated
  from `skills.name` automatically. This prevents stale display names but means
  the fields can diverge.

## Enum And Status Review

### `installs.install_mode`

Current: `symlink`, `rsync_copy`, `direct`, `external_symlink`, `unknown`

Issue: `external_symlink` and `unknown` are detection outcomes, not install
modes. Remove them from `install_mode`. Keep `symlink`, `rsync_copy`, `direct`.

### `installs.install_status`

Current: `current`, `outdated`, `missing`, `broken_symlink`, `old_host`,
`external`, `conflict`, `needs_sync`, `unmanaged`, `error`

Suggestion: `unmanaged` overlaps with `install_mode = direct`. A `direct` mode
install with `unmanaged` status is redundant. One option: use `install_mode =
direct` and set `install_status = current` / `missing` etc. for those installs.
Remove `unmanaged` as a status and instead rely on `install_mode = direct` to
signal non-managed installs. Add `removed` if soft delete is chosen.

### `projects.status`

Current: `active`, `missing`, `unreadable`, `no_provider_detected`,
`has_warnings`, `removed`

Issue: `has_warnings` is derived (see Critical Issues). `no_provider_detected`
is also arguably a derived state from `project_providers` being empty. Consider
keeping `projects.status` to filesystem-observable states only: `active`,
`missing`, `unreadable`, `removed`. Report `no_provider_detected` as a warning
record instead.

### `skill_sources.last_fetch_status` vs `fetch_results.status`

These two enums are nearly identical. The `skill_sources` version is a
denormalized summary of the latest `fetch_results` row. Document explicitly that
`last_fetch_status` is always updated atomically when a new `fetch_results` row
is written, and never updated independently. Add `not_fetchable` to
`skill_sources.last_fetch_status` to handle local/manual sources that were
never meant to be fetched.

### `provider_definitions.status` vs `project_providers.detection_status`

`provider_definitions.status = unsupported` means Skillbox has no adapter for
this provider. `project_providers.detection_status = unsupported` means a
project scan found signs of an unsupported provider. These are not circular: a
project can reference an `unsupported` provider_definition. However, the UI
must handle the case where `project_providers` references a
`provider_definitions` row with `status = unsupported` — this means the adapter
cannot write files into that provider's path.

## User Flow Coverage

1. **First-Time Setup**: Supported. Writes `skill_host_folders`,
   `app_settings`, `skills`, `scan_results`. Clean.

2. **Add Project**: Supported. Writes `projects`, `project_providers`,
   `installs` (discovered direct installs), `warnings`. Clean.

3. **Scan Project**: Supported. Reads and reconciles `project_providers`,
   `installs`, writes `scan_results`, `warnings`. Clean.

4. **Install Skill To Project**: Supported. Writes `installs`, `operations`,
   `warnings` on conflict or error. Clean.

5. **Fetch Skill Updates**: Supported. Writes `fetch_results`,
   `skill_sources.last_fetched_at`, `skill_sources.last_fetch_status`,
   `warnings`. Clean.

6. **Update Skill Host Folder**: Supported. Updates `skills.current_version`,
   `skill_sources.resolved_version/commit`, marks `installs.install_status =
   needs_sync` for rsync/copy installs, writes `operations`. Clean.

7. **Sync Rsync / Copy Project**: Supported. Updates `installs` fields
   (`installed_version`, `installed_commit`, `installed_checksum`,
   `last_synced_at`, `install_status`), writes `operations`. Clean.

8. **Switch Install Mode**: Supported. Updates `installs.install_mode` and
   `install_status`, writes `operations` with `switch_install_mode` type.
   The atomicity requirement (no partial metadata update on filesystem failure)
   relies on `operations` status tracking, which the model supports. Clean.

9. **Remove Skill From Project**: Partially supported. Hard delete from
   `installs` is straightforward. Soft delete needs `removed` in
   `install_status` (missing). Recommend deciding soft vs hard delete before
   implementation.

10. **Add Skill To Skill Host Folder**: Supported. Writes `skills`,
    `skill_sources`, triggers scan. Clean.

11. **Change Skill Host Folder**: Supported. Writes `skill_host_folders`,
    updates `app_settings.active_skill_host_folder_id`, rescans `skills`,
    generates `warnings` for old host symlinks. The relink flow (user wants to
    update symlinks to new host) requires updating `installs.symlink_target_path`,
    `source_skill_path`, and `installed_from_host_folder_id`. The model supports
    this. Clean.

12. **App Startup**: Supported. Reads `app_settings`, checks
    `skill_host_folders.status`, checks `projects.status`, reads `warnings`.
    Generates warnings for missing paths. Clean.

## Edge Case Coverage

### Skill Host Folder states

All states covered: `active`, `missing`, `unreadable`, `unwritable`,
`invalid_structure`, `empty`, `inactive`. Drive/synced folder availability is
handled by `missing` status + warnings. Permission errors have `unreadable`
and `unwritable` separately (good UX distinction). Change-folder flow uses
`warnings` for old host symlinks. Fully covered.

### Project states

`active`, `missing`, `unreadable`, `no_provider_detected`, `removed` cover the
main states. `has_warnings` issue noted in Critical Issues. The "project has
manual skills" case is handled by `direct` installs. Fully covered except for
the `has_warnings` derivation issue.

### Install states

All major install states are represented:
- Valid symlink: `install_mode=symlink`, `install_status=current`
- Broken symlink: `install_status=broken_symlink`
- Old host symlink: `install_status=old_host`
- External symlink: `install_mode=external_symlink` / `install_status=external`
  (see critical issue on overlap)
- Rsync current/outdated: `install_status=current` / `outdated`
- Direct: `install_mode=direct`, `install_status=unmanaged`
- Conflict on install: `install_status=conflict`

The `conflict` state on install (target exists) is handled. Atomicity on switch
mode failure is handled via `operations`. Covered.

### Fetch and update states

`fetch_results.status` covers `up_to_date`, `update_available`, `failed`,
`auth_required`, `not_found`, `network_error`, `needs_review`, `not_fetchable`.
The "local skill modified vs upstream" case maps to `skills.status =
local_modified` + `fetch_results.status = needs_review`. Network offline maps
to `network_error`. Covered. One gap: no explicit field to store "last known
good fetch" timestamp separately from "last attempted fetch." This matters when
a fetch fails — the UI should still show the last successful fetch date. Add
`last_successful_fetch_at` to `skill_sources`.

### Provider states

`provider_definitions.status` (`supported`, `experimental`, `unsupported`,
`disabled`) and `project_providers.detection_status` (`detected`, `configured`,
`missing`, `unsupported`, `invalid_structure`, `format_unknown`) together cover:
- Recognized provider: `detected`
- Unknown/unsupported provider: `unsupported`
- Provider folder with unexpected structure: `invalid_structure` /
  `format_unknown`
- Claude and `.agents` coexisting: two `project_providers` rows
Covered.

### Database and app state

`app_settings.database_version` handles schema migration tracking. The corrupt
database case cannot be stored in SQLite itself (you can't write to a corrupt
DB). This is purely app startup logic — the model appropriately has nothing to
represent it since there's no DB to write to. The "DB lags filesystem" case is
handled by scan reconciliation. Covered.

### UI/UX states

The model supports empty states (count queries returning 0), loading states
(via `operations.status = running`), destructive action confirmation (UI logic
driven by impact preview data from joins), recoverable warnings vs blocking
errors (via `warnings.severity`), and quick actions (via `warnings.action_key`).
Covered.

## SQLite Assessment

SQLite is appropriate for this product. Rationale:

- Local-first, single-user, single-machine. No concurrent writer contention.
- Schema complexity is moderate (12 tables). SQLite handles this comfortably.
- File-based storage fits the "local control center" design philosophy.
- Migrations are manageable with any lightweight migration library (e.g.,
  `better-sqlite3` with versioned migration scripts tracked via
  `app_settings.database_version`).

Schema/migration concerns:

- Polymorphic FKs (`warnings.scope_id`, `scan_results.target_id`,
  `operations.target_id`) cannot be enforced by SQLite foreign key constraints.
  This requires app-level validation. Document this explicitly so future
  contributors don't assume DB-level safety.
- SQLite does not support `ALTER COLUMN` (rename/type change) — plan migrations
  carefully. Use additive migrations where possible in early phases.
- WAL mode should be enabled for better read concurrency during long scans.
- Consider `STRICT` tables (SQLite 3.37+) for stronger type enforcement.

## Provider Adapter Readiness

Partially ready. The `provider_definitions` table gives adapters a metadata row
to query. `project_providers.detected_path` and `skills_path` give
per-project install context. However:

- Single `default_relative_skills_path` is insufficient for providers with
  multiple detection candidates (critical issue).
- No field indicates whether the provider adapter can create the provider
  folder structure, or only install into existing structure.
- Phase 1 adapters will need app-level logic beyond what the schema provides.
  The schema is a starting point, not a complete adapter contract.

Recommended additions before adapter implementation:
- `provider_path_candidates` table with priority ordering.
- Boolean `can_create_structure` on `provider_definitions` to distinguish
  read-only detection from write-capable setup.

## Phase 2 Conversion Readiness

The current model does not prepare for Phase 2 skill format conversion but does
not block it. The open questions in `docs/06-data-model.md` explicitly raise
this.

For Phase 2, the model would need:

- A way to record what format a skill is currently stored in (e.g.,
  `skills.detected_format TEXT`).
- A `skill_variants` or `provider_skill_formats` table mapping
  `(skill_id, provider_definition_id) -> converted_path` for storing
  provider-specific converted copies.
- Conversion operation types added to `operations.operation_type`.

None of this needs to exist in Phase 1, but the schema should not make it
harder to add. The current design is additive-friendly; Phase 2 tables can be
added via migration without breaking Phase 1 tables.

## Recommended Data Model Changes

Changes that should be made before implementation begins:

- **`installs.install_mode`**: Remove `external_symlink` and `unknown`. Keep
  `symlink`, `rsync_copy`, `direct` only. Express `external_symlink` and
  unknown states via `install_status` only.

- **`projects.status`**: Remove `has_warnings`. Keep `active`, `missing`,
  `unreadable`, `no_provider_detected`, `removed`. (Consider moving
  `no_provider_detected` to warnings as well.)

- **`fetch_results`**: Remove `skill_id` FK or demote it to a denormalized
  helper column. Make `source_id` the primary FK. Add comment explaining the
  relationship.

- **`provider_definitions`**: Replace `default_relative_skills_path TEXT` with
  a new `provider_path_candidates` table: `(id, provider_definition_id,
  relative_path TEXT NOT NULL, priority INTEGER, description TEXT)`.

- **`skills`**: Add `current_checksum TEXT` for local modification detection.

- **`installs`**: Remove `project_id` or add a documented CHECK equivalent to
  enforce consistency with `project_providers.project_id`. Do not allow both to
  diverge silently.

- **`skill_sources`**: Add `last_successful_fetch_at DATETIME` (distinct from
  `last_fetched_at` which includes failed attempts). Add `not_fetchable` to
  `last_fetch_status` enum.

- **`fetch_results`**: Rename `current_version` → `host_version_at_fetch` and
  `latest_version` → `upstream_version_at_fetch` to clarify point-in-time
  semantics.

- **`installs`**: Decide and document soft delete vs hard delete. If soft
  delete: add `removed` to `install_status`.

- **`warnings`**: Add `source_operation_id INTEGER REFERENCES operations(id)`
  (nullable) to track which scan or operation generated the warning.

## Open Questions For The Product Owner

1. **Soft delete vs hard delete for `installs`, `skills`, `projects`?** The
   data model's open questions flag this. Hard delete simplifies queries but
   loses history. Soft delete complicates queries but allows "recently removed"
   UX.

2. **`checksum` for rsync/copy outdated detection: whole-folder hash or
   per-file manifest?** Whole-folder hash is simpler but sensitive to
   irrelevant file changes (e.g., `.DS_Store`). A manifest-based approach is
   more accurate but requires more implementation work.

3. **`warnings` regeneration policy**: Should warnings be hard-deleted and
   recreated on every scan, or accumulated with `is_resolved` toggling? The
   former is simpler; the latter supports warning history. Phase 1 recommendation
   is regenerate-on-scan with no long-term history.

4. **GitHub/Vercel auth credentials storage**: In SQLite, OS keychain, or
   environment? This affects whether an `api_credentials` table is needed.

5. **Phase 2 timeline**: If Phase 2 is within 6 months, adding
   `skills.detected_format` now costs nothing and saves a migration later. If
   Phase 2 is speculative, leave it out.

6. **Provider adapter write permissions**: Should `provider_definitions` record
   whether an adapter can scaffold the provider folder structure, or is that
   hardcoded in the adapter implementation?

7. **Multi-host Phase 2**: The current model keeps old `skill_host_folders`
   rows when the active host changes. Is this intentional for future multi-host
   support, or just incidental? Clarify policy so migrations don't assume it can
   be cleaned up.

## What Looks Solid

- **Relationship design** between `skills`, `skill_sources`, `installs`, and
  `project_providers` is clean and the user flow mappings in `docs/06-data-model.md`
  demonstrate solid coverage.

- **`warnings` table design** with `scope_type`, `severity`, `code`, and
  `action_key` is a strong pattern for driving UI warning components from a
  single query. The `action_key` concept is particularly well thought out.

- **`installs.symlink_target_path`** correctly addresses the "old host
  symlink" and "external symlink" detection problem without requiring a live
  filesystem check in every query.

- **`skill_sources` fetch status fields** (`last_fetched_at`,
  `last_fetch_status`, `last_fetch_error`) as denormalized summaries on the
  source row give the Skills Library view fast access without joining
  `fetch_results` on every render.

- **`operations` table** is the right pattern for tracking long-running UI
  state without building a full job queue. The operation types map cleanly to
  the user flows.

- **`provider_definitions` as a managed lookup table** rather than hardcoded
  enums is the right call for a product that expects new providers to be added
  over time.

- **`installed_from_host_folder_id`** in `installs` correctly tracks which
  host was the source at install time, enabling the "symlink points to old host"
  detection.

- **`app_settings.database_version`** for migration tracking is simple and
  correct for a local SQLite app.

- **Design principle of absolute paths** in all database records is correct and
  prevents relative-path resolution bugs across working directory changes.
