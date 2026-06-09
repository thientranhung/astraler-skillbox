# Schema Dictionary

This document describes the tables and fields expected for SQLite in detail. It
is a field-level reference for AI, reviewers, and developers to understand the
business meaning of each column.

`06-data-model.md` is the conceptual model. This file is the schema reference.

## Conventions

- `integer` for primary keys, foreign keys, boolean as `0/1`, or priority.
- `text` for enums, paths, external system ids, messages, versions, commits,
  hashes.
- `datetime` stores ISO-8601 strings or SQLite-compatible timestamps.
- `json` is text containing valid JSON.
- Paths in the database should be absolute paths except for fields named
  `relative_path`.
- Enums are stored as text for easier debugging and AI readability.

## app_settings

Purpose: stores global app configuration.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. Typically only one active row for app settings. |
| `active_skill_host_folder_id` | integer | yes | FK to `skill_host_folders.id`. Nullable during first-time setup before the user selects a Skill Host Folder. |
| `default_install_mode` | text | no | Default install mode when installing a skill. Allowed: `symlink`, `rsync_copy`. Current UI/RPC support: `symlink` only; `rsync_copy` is reserved. |
| `database_version` | integer | no | Current schema version, used for migration. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## api_credentials

Purpose: stores metadata about credentials for GitHub/Vercel fetch. Does not
store plaintext tokens in SQLite.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_key` | text | no | Provider using the credential. Allowed: `github`, `vercel`. |
| `credential_type` | text | no | Credential type. Allowed: `token`, `oauth`, `ssh_key`. |
| `storage_type` | text | no | Where the real secret is stored. Allowed: `os_keychain`, `encrypted_sqlite`, `environment`. |
| `credential_ref` | text | yes | Reference to the keychain item or environment variable name. |
| `value_encrypted` | text | yes | Encrypted secret if `storage_type = encrypted_sqlite`. No plaintext. |
| `status` | text | no | Credential status. Allowed: `active`, `missing`, `invalid`, `expired`. |
| `last_validated_at` | datetime | yes | Most recent time the app verified the credential is still valid. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## skill_host_folders

Purpose: stores folders previously selected by the user as a Skill Host Folder.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `name` | text | yes | Display name set by the user or derived by the app from the folder name. |
| `path` | text | no | Absolute path to the folder the user selected as the Skill Host Folder. |
| `skills_path` | text | no | Absolute path to the skill storage location, typically `<skill-host-folder>/.agents/skills`. |
| `status` | text | no | Host state. Allowed: `active`, `missing`, `unreadable`, `unwritable`, `invalid_structure`, `empty`, `inactive`. |
| `last_scanned_at` | datetime | yes | Most recent time the app scanned the Skill Host Folder. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## skills

Purpose: represents a skill in the Skill Host Folder. Skill content lives on the
filesystem; the database stores only metadata.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `skill_host_folder_id` | integer | no | FK to `skill_host_folders.id`. |
| `name` | text | no | Canonical skill name, typically the folder name under `.agents/skills`. |
| `display_name` | text | yes | More user-friendly display name if the app can read metadata. |
| `relative_path` | text | no | Relative path from the Skill Host Folder, typically `.agents/skills/<skill-name>`. |
| `absolute_path` | text | no | Absolute path to the skill folder in the Skill Host Folder. |
| `status` | text | no | Skill state. Allowed: `available`, `missing`, `unreadable`, `local_modified`, `unknown`. |
| `source_id` | integer | yes | FK to `skill_sources.id`. Nullable for local/manual skills with no source metadata. |
| `current_version` | text | yes | Current version of the skill in the Skill Host Folder if the source has versioning. |
| `current_commit` | text | yes | Current commit of the skill in the Skill Host Folder if the source is git/GitHub. |
| `current_checksum` | text | yes | Hash/checksum of current content. Reserved for rsync/copy drift detection; not current UI/RPC support. |
| `last_scanned_at` | datetime | yes | Most recent time the app scanned this skill in the Skill Host Folder. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## skill_sources

Purpose: stores source/upstream metadata for skills.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `source_type` | text | no | Source type. Allowed: `github`, `vercel_skills`, `local`, `manual`. |
| `url` | text | yes | Original source URL if available. |
| `github_owner` | text | yes | GitHub owner/org if `source_type = github`. |
| `github_repo` | text | yes | GitHub repo if `source_type = github`. |
| `github_path` | text | yes | Subfolder within the repo if the skill is not at the repo root. |
| `github_ref` | text | yes | Branch, tag, or commit ref being tracked. |
| `vercel_skill_id` | text | yes | Identifier in the Vercel skills ecosystem if applicable. |
| `local_source_path` | text | yes | Absolute path to local source if `source_type = local`. |
| `resolved_version` | text | yes | Currently resolved version from the source. |
| `resolved_commit` | text | yes | Currently resolved commit from the source. |
| `last_fetched_at` | datetime | yes | Most recent fetch attempt, including failed attempts. |
| `last_successful_fetch_at` | datetime | yes | Most recent successful fetch. |
| `last_fetch_status` | text | no | Latest fetch summary. Allowed: `never_fetched`, `up_to_date`, `update_available`, `failed`, `auth_required`, `not_found`, `network_error`, `needs_review`, `not_fetchable`. |
| `last_fetch_error` | text | yes | Error message from the most recent fetch attempt if it failed. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## projects

Purpose: stores projects the user has added to Skillbox.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `name` | text | no | Project display name shown in the UI, typically derived from the folder name. |
| `path` | text | no | Absolute path to the project root. |
| `status` | text | no | Project lifecycle/filesystem state. Allowed: `active`, `missing`, `unreadable`, `removed`. |
| `last_scanned_at` | datetime | yes | Most recent time the app scanned the project. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

Notes:

- `has_warnings` and `no_provider_detected` are not in `projects.status`. They
  are derived state from the `warnings` table.

## provider_definitions

Purpose: stores the list of providers/conventions that Skillbox knows about.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `key` | text | no | Stable provider key used in code/config, e.g. `claude`, `generic_agents`. |
| `display_name` | text | no | Display name in the UI. |
| `provider_type` | text | no | Provider category. Allowed: `claude`, `codex`, `opencode`, `antigravity_cli`, `generic_agents`, `custom`, `unsupported`. |
| `icon_key` | text | yes | Key for the UI to pick the appropriate icon. |
| `status` | text | no | Adapter support state. Allowed: `supported`, `experimental`, `unsupported`, `disabled`. |
| `can_create_structure` | integer | no | Boolean `0/1`. Whether core Skillbox logic can scaffold the provider folder structure for this provider. |
| `has_global_level` | integer | no | Boolean `0/1`. Whether the provider has a global/user-level location that Skillbox can scan or configure. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

Notes:

- `generic_agents` uses `can_create_structure = 1` so project installs may
  create the selected project's `.agents/skills` folder. Provider-specific
  definitions such as `claude` must remain `0` until their conventions are
  verified and documented.

## provider_path_candidates

Purpose: stores candidate paths that a provider adapter uses to detect/config/install.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. |
| `relative_path` | text | no | Path relative to the project root. |
| `purpose` | text | no | Candidate purpose. Allowed: `detect`, `skills`, `commands`, `config`. |
| `priority` | integer | no | Lower value wins. Priority `1` is checked before priority `10`. Used to resolve the primary candidate when multiple paths exist. |
| `description` | text | yes | Explains why this path exists or what the provider uses it for. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## project_providers

Purpose: stores providers detected or configured in each project.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `project_id` | integer | no | FK to `projects.id`. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. |
| `detected_path` | text | yes | Absolute path resolved from the `purpose = detect` candidate. |
| `skills_path` | text | yes | Absolute path where this provider receives skill installs. Resolved from the `purpose = skills` candidate. |
| `detection_status` | text | no | Detection state within the project. Allowed: `detected`, `configured`, `missing`, `unsupported`, `invalid_structure`, `format_unknown`. |
| `last_scanned_at` | datetime | yes | Most recent time the app scanned this provider scope. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## installs

Purpose: stores the installation of a skill into a project/provider.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `project_provider_id` | integer | no | FK to `project_providers.id`. The project is inferred via this provider scope. |
| `skill_id` | integer | yes | FK to `skills.id`. Nullable for direct/manual/unknown installs that cannot be mapped to a skill in the host. |
| `skill_name` | text | no | Skill name written at the time of scan/install. Does not automatically sync back from `skills.name`. |
| `install_mode` | text | no | Install mechanism/intent only. Allowed: `symlink`, `rsync_copy`, `direct`. Current UI/RPC support: `symlink` and `direct`; `rsync_copy` is reserved. Does not store filesystem anomalies. |
| `install_status` | text | no | Detected current state. Allowed: `current`, `outdated`, `missing`, `broken_symlink`, `old_host`, `external_symlink`, `conflict`, `needs_sync`, `error`. |
| `project_skill_path` | text | no | Absolute path to the skill entry in the project's provider folder. |
| `source_skill_path` | text | yes | Absolute path to the skill in the Skill Host Folder if managed. |
| `symlink_target_path` | text | yes | Symlink target if `project_skill_path` is a symlink. Used to detect broken/old/external symlinks. |
| `installed_from_host_folder_id` | integer | yes | FK to `skill_host_folders.id` at the time of install. Used for old host detection. |
| `installed_version` | text | yes | Version installed/synced into the project if known. |
| `installed_commit` | text | yes | Commit installed/synced into the project if known. |
| `installed_checksum` | text | yes | Hash/checksum snapshot in the project. Reserved for rsync/copy drift detection; not current UI/RPC support. |
| `last_synced_at` | datetime | yes | Reserved for rsync/copy sync tracking; not current UI/RPC support. |
| `last_scanned_at` | datetime | yes | Most recent time this install was scanned from the filesystem. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

Notes:

- When a scan finds a symlink on disk, `install_mode = symlink` regardless of
  whether Skillbox or the user created it. `install_status` distinguishes the
  real state.
- Phase 1 uses hard delete when the user removes an install via Skillbox.

## global_provider_locations

Purpose: stores provider global locations at the user/machine level.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. |
| `name` | text | yes | Display name for the global location, e.g. Claude Global or Shared Agents Global. |
| `path` | text | yes | Absolute path to the provider global root/location. Nullable when the global location is not yet configured. |
| `skills_path` | text | yes | Absolute path where the provider global level accepts skill/global entries if applicable. |
| `status` | text | no | Global location state. Allowed: `active`, `not_configured`, `missing`, `unreadable`, `invalid_structure`, `empty`, `disabled`, `no_global_skills`. |
| `last_scanned_at` | datetime | yes | Most recent time the app scanned this global location. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## global_installs

Purpose: stores skills/global entries that exist at the provider global level.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `global_provider_location_id` | integer | no | FK to `global_provider_locations.id`. |
| `skill_id` | integer | yes | FK to `skills.id`. Nullable for direct/manual global entries that cannot be mapped to a skill in the host. |
| `skill_name` | text | no | Skill/global entry name written at the time of scan/install. |
| `install_mode` | text | no | Install mechanism/intent only. Allowed: `symlink`, `rsync_copy`, `direct`. |
| `install_status` | text | no | Detected current state. Allowed: `current`, `outdated`, `missing`, `broken_symlink`, `old_host`, `external_symlink`, `conflict`, `needs_sync`, `error`. |
| `global_skill_path` | text | no | Absolute path to the global skill/entry in the provider global location. |
| `source_skill_path` | text | yes | Absolute path to the skill in the Skill Host Folder if managed. |
| `symlink_target_path` | text | yes | Symlink target if `global_skill_path` is a symlink. |
| `installed_from_host_folder_id` | integer | yes | FK to `skill_host_folders.id` at the time of install. |
| `installed_version` | text | yes | Version installed/synced into the global location if known. |
| `installed_commit` | text | yes | Commit installed/synced into the global location if known. |
| `installed_checksum` | text | yes | Hash/checksum snapshot in the global location. Reserved for rsync/copy drift detection; not current UI/RPC support. |
| `last_synced_at` | datetime | yes | Reserved for rsync/copy sync tracking; not current UI/RPC support. |
| `last_scanned_at` | datetime | yes | Most recent time this global install was scanned from the filesystem. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

Notes:

- Global installs must be clearly distinguished from project-level installs in
  the UI.

## fetch_results

Purpose: stores the fetch results from upstream for a source.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `source_id` | integer | no | Primary FK to `skill_sources.id`. Skill context is inferred via `skills.source_id`. |
| `status` | text | no | Fetch result. Allowed: `up_to_date`, `update_available`, `failed`, `auth_required`, `not_found`, `network_error`, `needs_review`, `not_fetchable`. |
| `host_version_at_fetch` | text | yes | Version in the Skill Host Folder at the time of fetch. |
| `upstream_version_at_fetch` | text | yes | Upstream version discovered at the time of fetch. |
| `host_commit_at_fetch` | text | yes | Commit in the Skill Host Folder at the time of fetch. |
| `upstream_commit_at_fetch` | text | yes | Upstream commit discovered at the time of fetch. |
| `fetched_at` | datetime | no | Timestamp of the fetch attempt. |
| `error_message` | text | yes | Error message if the fetch failed. |
| `raw_metadata_json` | json | yes | Raw metadata from GitHub/Vercel/source adapter for debugging. |
| `created_at` | datetime | no | Timestamp when the row was created. |

Notes:

- Phase 1 should limit retention by `source_id`, e.g. keep only N most recent
  rows.

## scan_results

Purpose: stores the most recent or lightweight scan history for a host/project/provider.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `target_type` | text | no | Scan target. Allowed: `skill_host_folder`, `project`, `project_provider`, `global_provider_location`. |
| `target_id` | integer | no | ID of the corresponding target. Polymorphic FK; validated at the app layer. |
| `status` | text | no | Scan result. Allowed: `success`, `partial`, `failed`, `cancelled`. |
| `started_at` | datetime | no | Timestamp when the scan started. |
| `finished_at` | datetime | yes | Timestamp when the scan ended. Nullable while running. |
| `summary_json` | json | yes | Counts and summary such as skills found, providers found, warnings. |
| `error_message` | text | yes | Error message if the scan failed or was partial. |
| `created_at` | datetime | no | Timestamp when the row was created. |

## warnings

Purpose: stores warnings/recoverable/blocking states for consistent UI display.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `scope_type` | text | no | Warning scope. Allowed: `app`, `skill_host_folder`, `skill`, `project`, `project_provider`, `install`, `global_provider_location`, `global_install`, `source`, `database`. |
| `scope_id` | integer | yes | ID of the scoped object. Nullable for app-level or database-level warnings. Polymorphic FK; validated at the app layer. |
| `severity` | text | no | Severity. Allowed: `info`, `warning`, `error`, `blocking`. |
| `code` | text | no | Stable warning code, e.g. `broken_symlink`, `fetch_failed`, `project_missing`. |
| `message` | text | no | Display message or debug-friendly text. |
| `action_key` | text | yes | Suggested action key for the UI, e.g. `rescan`, `retry`, `relink`, `sync`, `choose_folder`. |
| `source_operation_id` | integer | yes | FK to `operations.id` if the warning was created by an operation/scan. |
| `is_resolved` | integer | no | Boolean `0/1`. Whether the warning has been resolved or superseded. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |
| `resolved_at` | datetime | yes | Timestamp when the warning was resolved, if applicable. |

## operations

Purpose: stores long-running or important operations for UI loading/progress/debug state.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `operation_type` | text | no | Operation kind. Allowed: `scan`, `fetch`, `update_host_skill`, `sync_install`, `install_skill`, `remove_install`, `switch_install_mode`, `change_skill_host_folder`, `scan_global_skills`. Note: `sync_install` and `switch_install_mode` are reserved for rsync/copy mode; not current UI/RPC support. |
| `target_type` | text | no | Target object type of the operation, e.g. `project`, `skill`, `install`, `skill_host_folder`. |
| `target_id` | integer | yes | ID of the target. Polymorphic FK; validated at the app layer. |
| `status` | text | no | Operation status. Allowed: `queued`, `running`, `success`, `failed`, `cancelled`, `partial`. |
| `started_at` | datetime | yes | Timestamp when the operation started. |
| `finished_at` | datetime | yes | Timestamp when the operation ended. |
| `error_message` | text | yes | Error message if failed or partial. |
| `metadata_json` | json | yes | Operation-specific metadata such as counts, affected projects, changed paths. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## provider_user_settings

Purpose: stores user-level preferences for each provider (Phase 1: enable/disable only).

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. UNIQUE — one row per provider. ON DELETE CASCADE. |
| `enabled` | integer | no | Boolean `0/1`. User preference for the provider. CHECK `enabled IN (0, 1)`. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

## provider_path_overrides

Purpose: stores user overrides for a provider's path candidates. One row per
`(provider_definition_id, scope, purpose)`. When an override exists, the adapter
uses `paths_json` instead of `provider_path_candidates` for that slot.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. |
| `scope` | text | no | Override scope. Allowed: `project`, `global`. |
| `purpose` | text | no | Overridden slot. Allowed: `detect`, `skills`, `config`, `commands`. |
| `paths_json` | json | no | JSON array of path strings replacing built-in candidates. Default `'[]'`. CHECK `json_valid` AND `json_type = 'array'`. |
| `created_at` | datetime | no | Timestamp when the row was created. |
| `updated_at` | datetime | no | Timestamp of the most recent row update. |

Notes:

- UNIQUE `(provider_definition_id, scope, purpose)` ensures only one active
  override per slot.
- Paths in `paths_json` may be absolute or start with `~` (user home); the
  adapter resolves them before use.

## provider_plugin_layer_scans

Purpose: stores the result of scanning one settings file at one provider plugin
layer. One row per `(provider_definition_id, project_id, settings_layer)`.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK to `provider_definitions.id`. ON DELETE CASCADE. |
| `project_id` | integer | yes | FK to `projects.id`. NULL when `settings_layer = user`. Required non-null for `project`/`local`. ON DELETE CASCADE. |
| `settings_layer` | text | no | Layer precedence. Allowed: `user`, `project`, `local`. |
| `scan_status` | text | no | Result of reading the settings file. Allowed: `ok`, `missing`, `unreadable`, `malformed`, `too_large`, `symlink`, `path_escape`. |
| `settings_file_path` | text | no | Absolute path to the settings file the scanner attempted to read. |
| `last_scanned_at` | datetime | no | Timestamp of the most recent scan. Default `now()`. |
| `source_operation_id` | integer | yes | FK to `operations.id` for the scan that created this row. ON DELETE SET NULL. |
| `scan_warnings` | json | no | JSON array string of parse-time warnings. Default `'[]'`. Does not store raw file content. |

Notes:

- Partial unique indexes:
  - `(provider_definition_id, settings_layer)` WHERE `project_id IS NULL` (user
    layer).
  - `(provider_definition_id, project_id, settings_layer)` WHERE `project_id IS
    NOT NULL` (project/local layer).
- Table CHECK constraint: user layer must have null `project_id`; project/local
  layer must have non-null `project_id`.
- `scan_status = ok` is the only condition under which entries/marketplaces
  from this scan are valid.

## provider_plugin_entries

Purpose: stores plugin declarations (enabled/disabled) in one settings file scan.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `layer_scan_id` | integer | no | FK to `provider_plugin_layer_scans.id`. ON DELETE CASCADE. |
| `plugin_name` | text | no | Plugin name as declared in the settings file. |
| `marketplace_name` | text | no | Marketplace name from which the plugin is resolved. |
| `declaration` | text | no | Declaration in the file. Allowed: `enabled`, `disabled`. |
| `version` | text | yes | *(migration 000021)* Installed version from `installed_plugins.json`. `NULL` when no record exists (non-Claude providers, plugin not installed). `"unknown"` is a valid literal when Claude cannot determine the version. |

Notes:

- UNIQUE `(layer_scan_id, plugin_name, marketplace_name)`.
- Effective status (`enabled`/`disabled`/`absent`/`unknown`) is resolved at the
  application layer by merging entries by precedence `local > project > user`;
  not stored directly in this table.
- Absence of an entry in a layer scan = `absent` at that layer.
- `version` is only populated for the Claude provider (user layer): read from
  `~/.claude/plugins/installed_plugins.json` at scan time.

## provider_plugin_marketplaces

Purpose: stores marketplace declarations in one settings file scan.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `layer_scan_id` | integer | no | FK to `provider_plugin_layer_scans.id`. ON DELETE CASCADE. |
| `marketplace_name` | text | no | Marketplace (named source) name from the settings file. |
| `source_type` | text | no | Source type. Validated at the application layer. Common values: `github`, `git`, `directory`, `url`, `settings`, `hostPattern`. |
| `source_summary` | text | no | Source description (owner/repo, URL, path). No raw credentials stored. |

Notes:

- `source_type` has no CHECK constraint in the migration; enum values are
  validated at the application layer per each provider's settings file format.
- A marketplace may appear in multiple layer scans (user/project/local); the
  effective marketplace list is resolved at the application layer.

## plugin_update_check_cache

*(migration 000022)* Purpose: caches the `git ls-remote` result for each
installed plugin, default TTL 6 hours. Upserted by UNIQUE key each time
`updateCheck.run` succeeds or fails.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_key` | text | no | Plugin provider. Phase 1: always `"claude"`. |
| `plugin_name` | text | no | Plugin name (the part before `@` in the plugin key). |
| `marketplace_name` | text | no | Marketplace name (the part after `@` in the plugin key). |
| `source_url` | text | no | HTTPS URL from `marketplace.json`. Always HTTPS (non-HTTPS is rejected before subprocess). |
| `source_ref` | text | yes | Tag or branch (`"v1.5.5"`, `"main"`). `NULL` when the source declares no ref. |
| `installed_sha` | text | yes | `gitCommitSha` from `installed_plugins.json`. `NULL` when not available. |
| `installed_version` | text | yes | Version string from `installed_plugins.json`. Reserved — Phase 1 does not use it for comparison. |
| `remote_sha` | text | yes | SHA returned by `git ls-remote`. `NULL` when the check fails or the ref is not found. |
| `remote_latest_tag` | text | yes | Reserved Phase 2 (semver tag scan). Always `NULL` in Phase 1. |
| `update_available` | integer | yes | `0`=up-to-date, `1`=update available, `NULL`=unknown (missing SHA or check error). |
| `checked_at` | text | no | ISO-8601 UTC timestamp of the most recent check. |
| `error` | text | yes | Error code if the check failed. Examples: `non_https_scheme_rejected`, `timeout`, `git_not_found`, `ref_not_found`, `host_backoff`. `NULL` on success. |

Notes:

- UNIQUE `(provider_key, plugin_name, marketplace_name)` — one row per plugin;
  upsert overwrites old results.
- No FK to any plugin table: cache is an independent snapshot; deleting a plugin
  does not cascade-delete the cache.
- `update_available` is computed as `installed_sha != remote_sha` when both are
  non-NULL; otherwise `NULL`.
- `source_url` is derived from `marketplace.json` on disk each time — the URL is
  not cached long-term (ADR-0001 allowlist-from-disk requirement).

## network_settings

*(migration 000022; column `update_check_enabled` dropped at 000023)* Purpose:
singleton table storing `cache_ttl_hours` for update-check. Always exactly 1 row
(`id = 1`) inserted by the migration.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. CHECK `(id = 1)` — enforces singleton. |
| `cache_ttl_hours` | integer | no | Update-check cache TTL in hours. Default `6`. |
| `created_at` | text | no | ISO-8601 UTC; set by migration. |
| `updated_at` | text | no | ISO-8601 UTC; updated when `SetCacheTTLHours` is called. |

Notes:

- Column `update_check_enabled` was dropped at migration 000023 (ADR-0002):
  update-check is always-on, the opt-in gate is gone.
  `UpdateCheckService.RunUpdateCheck` no longer reads this table — it runs
  whenever the user triggers it.
- Never delete this row; only UPDATE.

## Polymorphic References

SQLite cannot enforce polymorphic references such as:

- `warnings.scope_type` + `warnings.scope_id`
- `operations.target_type` + `operations.target_id`
- `scan_results.target_type` + `scan_results.target_id`

The app layer must validate these references.
