# Data Model

This document outlines the high-level data model for SQLite. The goal is to be
specific enough to support UI, user flows, edge cases, fetch/update/sync, and
provider adapters, without locking in implementation details such as migration
syntax or ORM.

The filesystem remains the source of truth for skill content. SQLite is the
source of truth for management metadata.

## Design Principles

- Skill content lives in the Skill Host Folder, not in the database.
- The database stores metadata so the UI knows about skills, sources, global
  provider locations, projects, providers, global installs, project installs,
  scans, fetches, updates, syncs, and warning state.
- The filesystem is the true state when scanning a project or Skill Host Folder.
- Scans have authority to reconcile the database with the filesystem.
- All paths stored in the database should be absolute paths for stable UI/scan.
- Tables should have `created_at` and `updated_at`.
- Enums should be stored as text for easier debugging.

## Core Entities

```text
app_settings
api_credentials
skill_host_folders
skills
skill_sources
projects
provider_definitions
provider_path_candidates
project_providers
global_provider_locations
installs
global_installs
fetch_results
scan_results
warnings
operations
```

## 1. app_settings

Stores global app configuration.

Suggested fields:

```text
id
active_skill_host_folder_id
default_install_mode
database_version
created_at
updated_at
```

Notes:

- `active_skill_host_folder_id` points to the current Skill Host Folder.
- `active_skill_host_folder_id` is nullable to support first-time setup before
  the user selects a Skill Host Folder.
- The current product supports one active host.
- `default_install_mode` can be `symlink` or `rsync_copy`. Currently only
  `symlink` has UI/RPC support; `rsync_copy` is reserved.

## 2. api_credentials

Stores metadata about credentials used for GitHub/Vercel fetch. Actual secret
values should preferably be in the OS keychain. If the implementation chooses to
store them in SQLite, they must be stored as encrypted values.

Suggested fields:

```text
id
provider_key
credential_type
storage_type
credential_ref
value_encrypted
status
last_validated_at
created_at
updated_at
```

Provider key:

```text
github
vercel
```

Credential type:

```text
token
oauth
ssh_key
```

Storage type:

```text
os_keychain
encrypted_sqlite
environment
```

Status:

```text
active
missing
invalid
expired
```

Notes:

- `credential_ref` points to the keychain item or environment variable name.
- `value_encrypted` is used only if `storage_type = encrypted_sqlite`.
- Do not store plaintext tokens in SQLite.

## 3. skill_host_folders

Stores folders previously selected by the user as a Skill Host Folder.

Suggested fields:

```text
id
name
path
skills_path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
missing
unreadable
unwritable
invalid_structure
empty
inactive
```

Notes:

- `path` is the folder the user selected.
- `skills_path` is typically `<skill-host-folder>/.agents/skills`.
- `status` helps Dashboard and Settings display warnings quickly.
- When switching the Skill Host Folder, the old host does not necessarily get
  deleted from the database.

## 4. skills

Represents a skill in the Skill Host Folder.

Suggested fields:

```text
id
skill_host_folder_id
name
display_name
relative_path
absolute_path
status
source_id
current_version
current_commit
current_checksum
last_scanned_at
created_at
updated_at
```

Status:

```text
available
missing
unreadable
local_modified
unknown
```

Notes:

- `name` is the folder name or canonical skill id.
- `relative_path` is typically `.agents/skills/<skill-name>`.
- `absolute_path` is the actual path in the Skill Host Folder.
- `source_id` is nullable to support local/manual skills.
- `current_version` or `current_commit` is used for Fetch/Update if available.
- `current_checksum` is used to detect local modifications with sources that
  have no clear git commit.

## 5. skill_sources

Stores upstream/source metadata for skills.

Suggested fields:

```text
id
source_type
url
github_owner
github_repo
github_path
github_ref
vercel_skill_id
local_source_path
resolved_version
resolved_commit
last_fetched_at
last_successful_fetch_at
last_fetch_status
last_fetch_error
created_at
updated_at
```

Source type:

```text
github
vercel_skills
local
manual
```

Fetch status:

```text
never_fetched
up_to_date
update_available
failed
auth_required
not_found
network_error
needs_review
not_fetchable
```

Notes:

- A GitHub source may be a repo root or a subfolder.
- `github_ref` may be a branch, tag, or commit.
- Vercel skills use `vercel_skill_id` or equivalent metadata.
- `last_fetched_at` is the most recent fetch attempt, including failed attempts.
- `last_successful_fetch_at` is the most recent successful fetch.
- Local/manual sources may use `not_fetchable`.

## 6. projects

Stores projects the user has added to Skillbox.

Suggested fields:

```text
id
name
path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
missing
unreadable
removed
```

Notes:

- `path` is the project root absolute path.
- Warning presence and `no_provider_detected` are derived state from the
  `warnings` table, not stored in `projects.status`.
- Projects removed from the database can use either hard delete or soft delete
  with `removed`, depending on implementation.

## 7. provider_definitions

Stores the list of providers/conventions that Skillbox knows about.

Suggested fields:

```text
id
key
display_name
provider_type
icon_key
status
can_create_structure
has_global_level
created_at
updated_at
```

Provider type:

```text
claude
codex
opencode
antigravity_cli
generic_agents
custom
unsupported
```

Status:

```text
supported
experimental
unsupported
disabled
```

Notes:

- Provider adapter implementations use this table as UI/config metadata.
- `can_create_structure` indicates whether core Skillbox logic can scaffold the
  provider folder for this provider, or may only scan/install into an existing
  structure.
- `has_global_level` indicates whether the provider has a global/user-level
  location that Skillbox can scan or configure.

## 8. provider_path_candidates

Stores the path candidates that a provider adapter uses to detect or install
skills. A provider may have multiple candidate paths.

Suggested fields:

```text
id
provider_definition_id
relative_path
purpose
priority
description
created_at
updated_at
```

Purpose:

```text
detect
skills
commands
config
```

Notes:

- `relative_path` is relative from the project root.
- `priority` helps the adapter choose the primary candidate when multiple valid
  paths exist.
- This table avoids locking a provider into a single
  `default_relative_skills_path`.
- For simple providers, only a single row with `purpose = skills` is needed.

## 9. project_providers

Stores providers detected or configured in each project.

Suggested fields:

```text
id
project_id
provider_definition_id
detected_path
skills_path
detection_status
last_scanned_at
created_at
updated_at
```

Detection status:

```text
detected
configured
missing
unsupported
invalid_structure
format_unknown
```

Notes:

- A project may have multiple providers.
- The Add Skill flow uses this table to select a provider target.
- `skills_path` is where skills are installed for that provider.
- When scanning a provider, `detected_path` should come from the `purpose =
  detect` candidate with the lowest priority that exists on disk.
- `skills_path` should come from the `purpose = skills` candidate resolved for
  that provider.

## 10. global_provider_locations

Stores provider global locations at the user/machine level.

Suggested fields:

```text
id
provider_definition_id
name
path
skills_path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
not_configured
missing
unreadable
invalid_structure
empty
disabled
no_global_skills
```

Notes:

- `path` is the absolute path to the provider global root or global convention
  path. Nullable when `status = not_configured` or `status = no_global_skills`.
- `skills_path` is the absolute path where the provider global level accepts
  skill/global entries if applicable.
- Global locations do not belong to any project.
- The provider adapter is responsible for resolving/configuring candidate global
  paths; core Skillbox logic is responsible for scanning/writing if permitted.

## 11. installs

Stores the installation of a skill into a project/provider.

Suggested fields:

```text
id
project_provider_id
skill_id
skill_name
install_mode
install_status
project_skill_path
source_skill_path
symlink_target_path
installed_from_host_folder_id
installed_version
installed_commit
installed_checksum
last_synced_at
last_scanned_at
created_at
updated_at
```

Install mode:

```text
symlink        — current stable path
rsync_copy     — reserved; not current UI or RPC support
direct         — unmanaged
```

Install status:

```text
current
outdated
missing
broken_symlink
old_host
external_symlink
conflict
needs_sync
error
```

Notes:

- `project_id` is not stored directly because `project_provider_id` already
  implies the project via `project_providers.project_id`.
- `skill_id` is nullable for `direct` or unknown skills.
- `skill_name` still needs to be stored for display when `skill_id` cannot be
  mapped.
- `skill_name` is written at the time of scan/install and does not automatically
  sync back from `skills.name`.
- `project_skill_path` is the entry in the provider folder.
- `source_skill_path` is the path in the Skill Host Folder if managed.
- `install_mode` only stores the management/install intent, not filesystem
  anomalies.
- When a scan finds a symlink on disk, `install_mode = symlink` regardless of
  whether the symlink was created by Skillbox or the user manually.
  `install_status` distinguishes managed/current, old host, broken, or external
  symlink.
- `symlink_target_path` helps distinguish valid symlink, old host,
  external_symlink, and broken_symlink in `install_status`.
- `installed_checksum` is reserved metadata; not current UI/RPC support.
- Phase 1 uses hard delete for installs when the user removes a skill via
  Skillbox.
- `missing` represents an install record still in the database but whose
  filesystem entry has been modified/deleted outside the app.
- `error` is the catch-all for filesystem entries that cannot be safely
  classified during a scan.

## 12. global_installs

Stores skills/global entries that exist in a provider global location.

Suggested fields:

```text
id
global_provider_location_id
skill_id
skill_name
install_mode
install_status
global_skill_path
source_skill_path
symlink_target_path
installed_from_host_folder_id
installed_version
installed_commit
installed_checksum
last_synced_at
last_scanned_at
created_at
updated_at
```

Install mode:

```text
symlink        — current stable path
rsync_copy     — reserved; not current UI or RPC support
direct         — unmanaged
```

Install status:

```text
current
outdated
missing
broken_symlink
old_host
external_symlink
conflict
needs_sync
error
```

Notes:

- Global installs use the same semantics as project installs, but are scoped to
  the provider global level.
- `skill_id` is nullable for direct/unmanaged global entries.
- Global installs need the UI to clearly distinguish them from project-level
  installs to avoid confusing global contamination with project-specific
  behavior.

## 13. fetch_results

Stores fetch results from upstream for a skill/source.

Suggested fields:

```text
id
source_id
status
host_version_at_fetch
upstream_version_at_fetch
host_commit_at_fetch
upstream_commit_at_fetch
fetched_at
error_message
raw_metadata_json
created_at
```

Status:

```text
up_to_date
update_available
failed
auth_required
not_found
network_error
needs_review
not_fetchable
```

Notes:

- This table allows future skill-source update surfaces to display the most
  recent fetch history.
- `source_id` is the primary FK. Skill context is inferred via `skills.source_id`.
- If the implementation needs fast queries by skill, a helper denormalized
  `skill_id` may be added, but it should not be treated as an independent FK.
- `raw_metadata_json` aids debugging without requiring every provider field to be
  schema-ized from the start.
- Phase 1 should limit retention, e.g. keep only the N most recent fetch results
  per `source_id`, to prevent unbounded table growth.

## 14. scan_results

Stores the most recent or lightweight scan history for a host/project/provider.

Suggested fields:

```text
id
target_type
target_id
status
started_at
finished_at
summary_json
error_message
created_at
```

Target type:

```text
skill_host_folder
project
project_provider
global_provider_location
```

Status:

```text
success
partial
failed
cancelled
```

Notes:

- The UI does not need to store every scan detail in this table if details have
  already been reconciled into `skills`, `project_providers`, and `installs`.
- `summary_json` may store counts such as skills found, providers found, warnings.
- If `operations` is already sufficient for audit trail, the implementation may
  merge scan results into `operations.metadata_json`. This entity is kept here to
  clarify what scan data is needed.

## 15. warnings

Stores warnings/recoverable errors for consistent display across Dashboard,
Projects, and Project Detail.

Suggested fields:

```text
id
scope_type
scope_id
severity
code
message
action_key
source_operation_id
is_resolved
created_at
updated_at
resolved_at
```

Scope type:

```text
app
skill_host_folder
skill
project
project_provider
install
global_provider_location
global_install
source
database
```

Severity:

```text
info
warning
error
blocking
```

Code examples:

```text
skill_host_missing
skill_host_unwritable
project_missing
no_provider_detected
unsupported_provider
broken_symlink
old_host_symlink
external_symlink
rsync_outdated
fetch_failed
database_corrupt
```

Action key examples:

```text
choose_folder
rescan
retry
relink
sync
remove
configure_source
open_folder
```

Notes:

- Warnings can be regenerated after a scan.
- `source_operation_id` is nullable, pointing to the operation/scan that created
  the warning if applicable.
- `is_resolved` lets the UI hide old warnings while retaining history if needed.
- Phase 1 should prioritize regenerating active warnings after each scan rather
  than maintaining a long warning history.

## 16. operations

Stores long-running or important operations such as scan, fetch, update, install,
and remove.

Suggested fields:

```text
id
operation_type
target_type
target_id
status
started_at
finished_at
error_message
metadata_json
created_at
updated_at
```

Operation type:

```text
scan
fetch
update_host_skill
sync_install            — reserved (rsync_copy mode); not current UI/RPC support
install_skill
remove_install
switch_install_mode     — reserved (rsync_copy mode); not current UI/RPC support
change_skill_host_folder
scan_global_skills
```

Status:

```text
queued
running
success
failed
cancelled
partial
```

Notes:

- Used for loading state, progress, lightweight audit trail, and debugging.
- A full job system does not need to be built immediately; this table is still
  useful for UI.

## 17. provider_user_settings

Stores user-level preferences for each provider, e.g. whether the provider is
enabled.

Suggested fields:

```text
id
provider_definition_id
enabled
created_at
updated_at
```

Notes:

- A single row per `provider_definition_id` (UNIQUE).
- `enabled` is a boolean `0/1`. Phase 1 uses this only to let the user
  enable/disable a provider from the global scan and install target list.
- Distinct from `provider_definitions.status`: `status` is the support state
  decided by the app; `enabled` here is a preference decided by the user.
- When a provider definition is deleted, the corresponding row is cascade-deleted.

## 18. provider_path_overrides

Stores user overrides for a provider's built-in path candidates. One row per
`(provider_definition_id, scope, purpose)` combination. `paths_json` is a JSON
array of path strings that replaces (does not merge with) built-in candidates.

Suggested fields:

```text
id
provider_definition_id
scope
purpose
paths_json
created_at
updated_at
```

Scope:

```text
project
global
```

Purpose:

```text
detect
skills
config
commands
```

Notes:

- When an override exists, the adapter uses `paths_json` instead of
  `provider_path_candidates` for that `(scope, purpose)` slot. Built-in
  candidates are not silently merged.
- `paths_json` must parse as a valid JSON array of strings (CHECK constraint).
- Used when users need to point to a non-standard layout (e.g. Claude settings
  at a custom path) without requiring a new Skillbox release.
- UNIQUE `(provider_definition_id, scope, purpose)` ensures only one active
  override per slot.

## 19. Provider Plugin Layer System

Some providers (initially Claude, Codex, Antigravity CLI) support the **plugin**
concept through their own settings files (`~/.claude/settings.json`,
`~/.codex/config.toml`, …). A plugin may be declared at multiple layers
(user/project/local) and the effective state is merged by precedence
`local > project > user`.

Skillbox scans those settings files to display the enabled/disabled view per
project and per provider. Three tables work together:

```text
provider_plugin_layer_scans
provider_plugin_entries
provider_plugin_marketplaces
```

A `provider_plugin_layer_scan` row represents **one read** of a settings file at
a specific layer (`user`, `project`, or `local`) for one provider. Each scan
produces (if the file is readable) `provider_plugin_entries` (declarations of
enabled/disabled for each plugin) and `provider_plugin_marketplaces` (additional
marketplaces from which plugins are fetched).

Layer rules:

- `settings_layer = 'user'` → `project_id IS NULL`. This is the global layer at
  the user/machine level.
- `settings_layer IN ('project', 'local')` → `project_id IS NOT NULL`.
- Unique by `(provider_definition_id, settings_layer)` for the user layer
  (partial index where `project_id IS NULL`), and
  `(provider_definition_id, project_id, settings_layer)` for the project/local
  layer.

### 19.1 provider_plugin_layer_scans

Suggested fields:

```text
id
provider_definition_id
project_id
settings_layer
scan_status
settings_file_path
last_scanned_at
source_operation_id
scan_warnings
```

Settings layer:

```text
user
project
local
```

Scan status:

```text
ok
missing
unreadable
malformed
too_large
symlink
path_escape
```

Notes:

- `scan_status = ok` is the only condition under which entries/marketplaces
  generated by this scan are considered valid.
- `missing` is a valid state meaning the settings file does not exist at that
  layer (not an error).
- `symlink` and `path_escape` are defensive: the scanner refuses to read symlinks
  and paths that escape the user/project root to prevent leaks.
- `too_large` blocks unusually large files to limit parse cost.
- `scan_warnings` is a JSON array string containing parse-time warnings (not
  raw file content), with bounded size.
- `source_operation_id` is nullable, FK to `operations.id` representing the scan
  that created this row.

### 19.2 provider_plugin_entries

Suggested fields:

```text
id
layer_scan_id
plugin_name
marketplace_name
declaration
version          -- TEXT nullable (migration 000021); NULL = not available
```

Declaration:

```text
enabled
disabled
```

Notes:

- An entry is a single plugin declaration in the settings file for the
  corresponding layer. Absence of declaration = `absent` (no entry row).
- UNIQUE `(layer_scan_id, plugin_name, marketplace_name)` ensures the same
  plugin in the same marketplace cannot be declared twice in the same settings
  file.
- Effective status (`enabled` / `disabled` / `absent` / `unknown`) is resolved
  at the application layer by merging entry rows by layer precedence; it is not
  stored directly in this table.
- `version`: read from `~/.claude/plugins/installed_plugins.json` when scanning
  the user layer for the Claude provider. NULL for Codex/Antigravity (no
  equivalent file) and when the plugin has no install record. The literal
  `"unknown"` is valid (Claude reports it when the version cannot be determined).

### 19.3 provider_plugin_marketplaces

Suggested fields:

```text
id
layer_scan_id
marketplace_name
source_type
source_summary
```

Source type:

```text
github
git
directory
url
settings
hostPattern
```

Notes:

- Each row is a marketplace declared in the same settings file scan represented
  by `layer_scan_id`. A marketplace is the named source from which a plugin is
  resolved.
- `source_summary` is a string describing the source (e.g. `owner/repo`, git URL,
  path). Raw credentials are not stored.
- `source_type` has no CHECK constraint in the migration; enum values are
  validated at the application layer based on the provider's settings file format.

## 20. Plugin Update-Check Cache & Network Settings

*(migration 000022 — 2026-05-29)*

Two tables support the **G3c plugin update-check** feature. This feature is
**always-on, manual-trigger-only** (ADR-0002, supersedes ADR-0001): there is no
longer an opt-in gate; network is only called when the user clicks "Check
Updates".

> **Migration 000023 (2026-05-31):** column `network_settings.update_check_enabled`
> has been **dropped** (the old gate is obsolete). The `network_settings` table
> is kept for `cache_ttl_hours`.

```text
plugin_update_check_cache
network_settings
```

### 20.1 plugin_update_check_cache

Stores the result of each `updateCheck.run` call that checks the upstream SHA
for a specific plugin. Cache has a TTL of 6 hours (configurable via
`network_settings.cache_ttl_hours`); each "Check Updates" click upserts the row
by UNIQUE key.

Fields:

```text
id                 -- INTEGER PRIMARY KEY
provider_key       -- TEXT NOT NULL; "claude" (Phase 1 supports Claude only)
plugin_name        -- TEXT NOT NULL
marketplace_name   -- TEXT NOT NULL
source_url         -- TEXT NOT NULL; HTTPS URL from marketplace.json (allowlist from disk)
source_ref         -- TEXT nullable; tag/branch (e.g. "v1.5.5", "main")
installed_sha      -- TEXT nullable; gitCommitSha from installed_plugins.json
installed_version  -- TEXT nullable; version string from installed_plugins.json
remote_sha         -- TEXT nullable; SHA returned by git ls-remote
remote_latest_tag  -- TEXT nullable; reserved Phase 2 (semver tag scan)
update_available   -- INTEGER nullable; 0=false / 1=true / NULL=unknown
checked_at         -- TEXT NOT NULL; ISO-8601 UTC timestamp of the check
error              -- TEXT nullable; error code if check failed
UNIQUE(provider_key, plugin_name, marketplace_name)
```

Update-available logic:

```text
installed_sha IS NOT NULL AND remote_sha IS NOT NULL
  → update_available = (installed_sha != remote_sha)
otherwise
  → update_available = NULL (unknown)
```

Notes:

- `source_url` must be HTTPS — `GitLsRemoteClient` rejects non-HTTPS before
  spawning a subprocess.
- `error` contains the error code string, e.g.: `non_https_scheme_rejected`,
  `timeout`, `git_not_found`, `ref_not_found`, `host_backoff`.
- The row has no FK to any plugin table: the cache is an independent snapshot —
  deleting a plugin does not cascade-delete the cache.

### 20.2 network_settings

Singleton table (always exactly 1 row with `id = 1`, inserted by migration
000022). Stores cache settings for update-check.

Fields (after migration 000023):

```text
id                    -- INTEGER PRIMARY KEY CHECK (id = 1); always = 1
cache_ttl_hours       -- INTEGER NOT NULL DEFAULT 6; update-check cache TTL (hours)
created_at            -- TEXT NOT NULL; ISO-8601 UTC
updated_at            -- TEXT NOT NULL; ISO-8601 UTC; updated when set_ttl is called
```

Notes:

- Column `update_check_enabled` was dropped at migration 000023 (ADR-0002):
  update-check is always-on, the gate is gone. `UpdateCheckService.RunUpdateCheck`
  no longer reads any setting — it runs whenever the user triggers it.
- `cache_ttl_hours` is read-only from the UI (Phase 1); Phase 2 may expose a
  slider.

## Provider Plugin Relationships

```text
provider_definitions.id
  -> provider_user_settings.provider_definition_id (UNIQUE)

provider_definitions.id
  -> provider_path_overrides.provider_definition_id

projects.id
  -> provider_plugin_layer_scans.project_id (nullable; user layer = null)

provider_definitions.id
  -> provider_plugin_layer_scans.provider_definition_id

provider_plugin_layer_scans.id
  -> provider_plugin_entries.layer_scan_id (CASCADE DELETE)

provider_plugin_layer_scans.id
  -> provider_plugin_marketplaces.layer_scan_id (CASCADE DELETE)

operations.id
  -> provider_plugin_layer_scans.source_operation_id
```

## Relationship Overview

```text
app_settings.active_skill_host_folder_id
  -> skill_host_folders.id

skill_host_folders.id
  -> skills.skill_host_folder_id

skill_host_folders.id
  -> installs.installed_from_host_folder_id

skill_sources.id
  -> skills.source_id

projects.id
  -> project_providers.project_id

provider_definitions.id
  -> project_providers.provider_definition_id

provider_definitions.id
  -> provider_path_candidates.provider_definition_id

provider_definitions.id
  -> global_provider_locations.provider_definition_id

project_providers.id
  -> installs.project_provider_id

skills.id
  -> installs.skill_id

global_provider_locations.id
  -> global_installs.global_provider_location_id

skills.id
  -> global_installs.skill_id

skill_host_folders.id
  -> global_installs.installed_from_host_folder_id

skill_sources.id
  -> fetch_results.source_id

operations.id
  -> warnings.source_operation_id
```

## Data Needed By Main Views

### Dashboard

Needs:

- Active Skill Host Folder status.
- Count skills.
- Count global installs.
- Count projects.
- Count installs by mode.
- Count warnings by severity.
- Count update_available fetch results.

Tables:

```text
app_settings
skill_host_folders
skills
projects
installs
global_provider_locations
global_installs
fetch_results
warnings
```

### Host Skills

Needs:

- Skill list from active Skill Host Folder.
- Source type and fetch status.
- Project count per skill.
- Last fetched/update status.

Tables:

```text
skills
skill_sources
fetch_results
installs
```

### Projects

Needs:

- Project list.
- Provider badges.
- Skill/install counts.
- Warning status.

Tables:

```text
projects
project_providers
provider_definitions
installs
warnings
```

### Global Skills

Needs:

- Global provider locations.
- Global entries grouped by provider.
- Mode/status/source path per global install.
- Warning status.

Tables:

```text
global_provider_locations
global_installs
provider_definitions
skills
warnings
```

### Project Detail

Needs:

- Project path/status.
- Providers detected.
- Installed skills grouped by provider.
- Mode/status/source path per install.
- Warnings and available actions.

Tables:

```text
projects
project_providers
provider_definitions
installs
skills
warnings
```

### Deferred Skill Source Updates

Needs:

- Future source-update surfaces with skills that have updates available.
- Host/upstream version or commit from latest fetch result.
- Affected projects and install modes.
- Affected global installs and install modes.
- Rsync/copy installs needing sync.

Tables:

```text
skills
skill_sources
fetch_results
installs
projects
project_providers
global_installs
global_provider_locations
```

### Settings

Needs:

- Active Skill Host Folder.
- Database version/location.
- Default install mode.
- Provider definitions/config.
- GitHub/Vercel credential metadata if configured.

Tables:

```text
app_settings
api_credentials
skill_host_folders
provider_definitions
provider_path_candidates
global_provider_locations
```

## Mapping From User Flows

### First-Time Setup

Writes:

- `skill_host_folders`
- `app_settings.active_skill_host_folder_id`
- `skills` after initial scan
- `scan_results`

### Add Project

Writes:

- `projects`
- `project_providers`
- `installs` discovered during scan
- `warnings` if provider/path issues exist

### Scan Global Skills

Writes:

- `global_provider_locations`
- `global_installs` discovered during scan
- `warnings` if global path/provider issues exist
- `scan_results`

### Install Skill To Project

Writes:

- `installs`
- `operations`
- `warnings` if conflict or filesystem error occurs

### Deferred: Fetch Skill Updates

Writes:

- `fetch_results`
- `skill_sources.last_fetched_at`
- `skill_sources.last_fetch_status`
- `warnings` for fetch failures

### Deferred: Update Skill Host Folder

Writes:

- `skills.current_version/current_commit`
- `skills.current_checksum`
- `skill_sources.resolved_version/resolved_commit`
- `operations`
- No sync rows are written in the current product. Projects using symlink
  installs receive host updates through the filesystem link.

### Change Skill Host Folder

Writes:

- `skill_host_folders`
- `app_settings.active_skill_host_folder_id`
- `skills` after host scan
- `warnings` for old host symlinks

## Mapping From Edge Cases

### Missing Skill Host Folder

Represented by:

```text
skill_host_folders.status = missing
warnings.code = skill_host_missing
```

### Missing Project

Represented by:

```text
projects.status = missing
warnings.code = project_missing
```

### Missing Global Provider Location

Represented by:

```text
global_provider_locations.status = missing
warnings.code = global_provider_location_missing
warnings.scope_type = global_provider_location
```

### No Provider Detected

Represented by:

```text
warnings.code = no_provider_detected
warnings.scope_type = project
```

### Broken Symlink

Represented by:

```text
installs.install_mode = symlink
installs.install_status = broken_symlink
warnings.code = broken_symlink
```

### Global Direct Install

Represented by:

```text
global_installs.install_mode = direct
global_installs.install_status = current
global_installs.skill_id = null
```

### Global Skill Overlap

Represented by:

```text
global_installs.skill_name
installs.skill_name
warnings.code = global_project_skill_overlap
warnings.scope_type = global_install | project | install
```

### Old Host Symlink

Represented by:

```text
installs.install_status = old_host
warnings.code = old_host_symlink
```

### External Symlink

Represented by:

```text
installs.install_mode = symlink
installs.install_status = external_symlink
warnings.code = external_symlink
```

### Direct Install

Represented by:

```text
installs.install_mode = direct
installs.install_status = current
installs.skill_id = null
```

### Fetch Failure

Represented by:

```text
fetch_results.status = failed | auth_required | not_found | network_error
warnings.code = fetch_failed
warnings.scope_type = source
```

### Unsupported Provider

Represented by:

```text
provider_definitions.status = unsupported
project_providers.detection_status = unsupported
warnings.code = unsupported_provider
```

### Multi-Path Provider Detection

Represented by:

```text
provider_path_candidates.provider_definition_id
provider_path_candidates.relative_path
provider_path_candidates.priority
project_providers.detected_path
project_providers.skills_path
```

### Fetch Attempt Failed But Previous Fetch Was Successful

Represented by:

```text
skill_sources.last_fetched_at
skill_sources.last_successful_fetch_at
skill_sources.last_fetch_status = failed | network_error | auth_required
```

## Open Questions

- Do projects/skills need long-term soft delete? Phase 1 has chosen hard delete
  for user-initiated install removal.
- Should GitHub/Vercel auth credentials be stored in the OS keychain, an
  encrypted SQLite table, or the environment?
