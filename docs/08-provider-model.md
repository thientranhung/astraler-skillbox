# Provider Model

The Provider Model describes how Skillbox understands and works with different
agent providers. The goal is to avoid hardcoding paths/conventions scattered
throughout the app, and instead centralize provider logic into clear adapters.

## What Is a Provider

A provider is an agent or convention that a project uses to hold skills,
commands, config, or workflow files.

Examples:

- Claude
- Codex
- opencode
- Antigravity CLI
- Generic `.agents`
- Custom/unsupported provider

A project may have multiple providers at the same time. For example, a project
may use both the Claude convention and the shared `.agents` convention.

## Provider Adapter

A provider adapter is the layer that knows how to work with a specific provider.

Responsibilities:

- Detect the provider within a project.
- Detect the provider's global location if the provider has a global level.
- Resolve provider paths from the project root.
- Resolve provider global paths from user/machine conventions if applicable.
- Resolve the skill install path.
- Scan installed skills in the provider scope.
- Classify install state.
- Report whether the provider folder structure can be created.
- Report unsupported/invalid/missing state.
- Provide UI metadata such as display name, icon, and support status.

Adapters should not make product policy decisions such as update/sync strategy.
Those policies belong to core Skillbox logic.

## Provider Definitions

`provider_definitions` is a lookup table for providers that Skillbox knows about.

Key fields:

```text
key
display_name
provider_type
icon_key
status
can_create_structure
has_global_level
```

Status:

```text
supported
experimental
unsupported
disabled
```

Meaning:

- `supported`: adapter is stable enough to scan/install.
- `experimental`: adapter is usable but conventions may still change.
- `unsupported`: Skillbox recognizes the provider but does not yet know how to
  safely operate on it.
- `disabled`: provider is turned off in config or not yet enabled for the user.

`can_create_structure` indicates whether core Skillbox logic can scaffold the
folder/path needed, or may only use the provider when its structure already
exists.

`has_global_level` indicates whether the provider has a global/user-level
location that Skillbox can scan or configure. Global scan only loads providers
with `has_global_level = 1` or those that already have a configured global
location.

`key` is the stable identifier for storing config, seed data, and external
references. `provider_type` is the enum/category for dispatching adapter
implementations. Built-in provider definitions own these values.

## Provider Path Candidates

`provider_path_candidates` stores the paths that an adapter uses to detect or
operate on a provider.

Fields:

```text
provider_definition_id
relative_path
purpose
priority
description
```

Purpose:

```text
detect
skills
commands
config
```

Meaning:

- `detect`: path used to determine whether the provider exists in a project.
- `skills`: path where the provider receives skill installs.
- `commands`: path for command-style files if the provider has this convention.
- `config`: path to the provider's config file/folder.

Resolution rules:

- The adapter resolves candidate paths from the project root.
- Lower priority wins. The adapter checks `priority = 1` before `priority = 10`.
- `project_providers.detected_path` should come from the best matching
  `purpose = detect` candidate that exists on disk.
- `project_providers.skills_path` should come from the `purpose = skills`
  candidate resolved for that provider.
- If multiple candidates of the same purpose coexist, the adapter chooses the
  one with the lowest priority.
- If multiple candidates of the same purpose have the same priority, the adapter
  chooses alphabetically by path so Phase 1 does not need extra UI for path
  selection.
- `commands` and `config` are reserved for future phases. Phase 1 adapters only
  need `detect` and `skills`.

## Project Provider

`project_providers` is a provider detected or configured within a project.

A project may have multiple `project_providers` rows.

Key fields:

```text
project_id
provider_definition_id
detected_path
skills_path
detection_status
last_scanned_at
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

Meaning:

- `detected`: provider convention exists and the adapter understands it.
- `configured`: user or app has configured the provider target, even if the path
  was not self-detected.
- `missing`: provider previously existed/was configured but the path no longer
  exists.
- `unsupported`: project shows signs of a provider, but Skillbox has no adapter
  to safely operate on it.
- `invalid_structure`: path exists but structure does not match expectations.
- `format_unknown`: structure exists but the internal format cannot be read.

`configured` is a future/manual setup state. Phase 1 does not need a separate
flow for the user to manually configure a provider target; adapters should not
set `configured` arbitrarily.

## Global Provider Location

A global provider location is a provider scope at the user/machine level, not
belonging to any specific project.

Provider adapters may support global scanning if the provider has a global
convention. Scan results are written to `global_provider_locations` and
`global_installs`.

Global detection must be kept separate from project detection:

- Project provider scope is written to `project_providers`.
- Global provider scope is written to `global_provider_locations`.

The UI must display global entries separately in Global Skills so the user knows
which global skills may affect multiple projects.

## Detection Flow

Flow:

```text
Project scan starts
  -> Load provider_definitions with status other than disabled
  -> For each provider, load provider_path_candidates
  -> Resolve candidate paths from the project root
  -> Check detect candidate paths
  -> If matched, create/update project_providers
  -> Resolve skills_path
  -> Scan installed skills in skills_path if available
  -> Write warnings if missing/unsupported/invalid
```

If no provider is detected:

```text
projects.status remains active
warnings.scope_type = project
warnings.code = no_provider_detected
```

Provider absence is not a blocking error. The user may choose to set up a
provider if the adapter supports `can_create_structure`.

Detection is allowed to recognize `unsupported` providers so the UI can report
them clearly to the user. Install target resolution is where writes to
unsupported providers are blocked.

When a rescan finds that a previously known provider path is now missing,
`project_providers.detection_status` should change to `missing`, and installs
belonging to that provider should be marked `install_status = missing` until the
user relinks/rescans the new path. Missing provider facts must not keep stale
current paths: clear `detected_path` and `skills_path`, update
`last_scanned_at`, and treat existing install rows as historical rather than
current UI counts.

## Global Detection Flow

Flow:

```text
Global scan starts
  -> Load provider_definitions with has_global_level = 1 or configured global paths
  -> Resolve global provider locations
  -> Scan global skills_path if available
  -> Create/update global_provider_locations
  -> Create/update global_installs
  -> Write warnings if missing/unreadable/unmanaged/overlap
```

Global scan must keep its scope separate from project scan. A global entry must
not automatically be treated as a project install.

Global provider paths do not use `provider_path_candidates.relative_path`
because that field is project-root relative. Global paths are resolved by the
adapter from user/machine conventions or from `global_provider_locations.path`
configured by the user in Settings.

Global scan classification: if an entry is a symlink, mode is `symlink`; if an
entry is a regular folder with no Skillbox DB record, mode is `direct`. Detecting
`rsync_copy` based on a DB record is reserved — rsync/copy install has no
UI/RPC support in the current release.

## Install Target Resolution

When the user installs a skill into a project:

```text
User opens Project Detail
  -> Selects Add Skill
  -> Skillbox retrieves the list of project_providers
  -> If only one valid provider target, may auto-select
  -> If multiple providers, user must select a provider target
  -> Adapter resolves skills_path
  -> Core install logic creates a symlink (rsync/copy: deferred, not current UI support)
```

A provider target is valid when:

- `project_providers.detection_status` is `detected` or `configured`.
- `provider_definitions.status` is `supported` or `experimental`.
- `skills_path` is resolvable.
- `skills_path` lies within the project root after canonicalize/normalize.
- If skills_path does not yet exist, the adapter must have `can_create_structure
  = 1` to scaffold it.

If a provider is `unsupported`, Skillbox must not write files to that provider
path.

## Scan Installed Skills

The adapter provides the provider scope. Core scan logic classifies install state
based on the filesystem and Skillbox metadata.

Install mode:

```text
symlink
rsync_copy
direct
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

Rules:

- If entry is a symlink, `install_mode = symlink`.
- If symlink points into the active Skill Host Folder, status may be `current`.
- If symlink points to an old Skill Host Folder, status is `old_host`.
- If symlink points outside the Skill Host Folder, status is
  `external_symlink`.
- If symlink target does not exist, status is `broken_symlink`.
- If entry is a regular folder with an `installs` DB record for that path with
  `install_mode = rsync_copy`, mode is `rsync_copy`. _(Reserved — rsync/copy
  install has no UI/RPC support in the current release.)_
- If entry is a regular folder with no Skillbox metadata, mode is `direct`.
- If entry cannot be safely classified, status is `error`.

Phase 1 uses a DB record as Skillbox metadata for rsync/copy detection. Rsync/copy
install mode is reserved and has no UI/RPC support in the current release.

## Initial Provider Assumptions

Current assumptions:

- Claude has its own convention and needs its own adapter.
- Generic `.agents` is a shared convention for multiple providers.
- Codex, opencode, Antigravity CLI may start with generic `.agents` if a
  distinct adapter is not yet needed.
- When a provider convention changes, the adapter layer is the place to update,
  not scattered UI/core logic.

## Suggested Initial Provider Definitions

### Generic Agents

```text
key = generic_agents
display_name = Generic Agents
provider_type = generic_agents
icon_key = agents
status = supported
can_create_structure = true
has_global_level = true
```

Path candidates:

```text
purpose = detect, relative_path = .agents, priority = 10
purpose = skills, relative_path = .agents/skills, priority = 10
```

### Claude

```text
key = claude
display_name = Claude
provider_type = claude
icon_key = claude
status = experimental
can_create_structure = false
has_global_level = true
```

Path candidates should be finalized after provider convention research. Do not
implement Claude scan/install until this convention is verified from documentation
or local provider behavior.

### Codex

```text
key = codex
display_name = Codex
provider_type = codex
icon_key = codex
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until Codex
requires a distinct adapter. Phase 1 should not seed separate `.agents` path
candidates for Codex if `generic_agents` already covers the same convention, to
avoid one `.agents` folder triggering multiple duplicate provider detections.

### opencode

```text
key = opencode
display_name = opencode
provider_type = opencode
icon_key = opencode
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until opencode
requires a distinct adapter. Phase 1 should not seed separate `.agents` path
candidates for opencode if `generic_agents` already covers the same convention,
to avoid duplicate provider badges.

### Antigravity CLI

```text
key = antigravity_cli
display_name = Antigravity CLI
provider_type = antigravity_cli
icon_key = antigravity
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until Antigravity
CLI requires a distinct adapter. Phase 1 should not seed separate `.agents` path
candidates for Antigravity CLI if `generic_agents` already covers the same
convention, to avoid duplicate provider badges.

## UI Representation

Provider UI should display:

- Provider badge/icon from `icon_key`.
- Provider display name.
- Support state: supported, experimental, unsupported, disabled.
- Detection status within the project.
- Skill count by provider.
- Warning if provider is missing/unsupported/invalid.

In Project Detail, installed skills should be grouped or filtered by provider.
Skills with the same name in multiple providers should not be merged into one
ambiguous row. The Add Skill wizard implements this grouping as a tab strip,
where each installable provider is a separate tab.

`experimental` providers should display a light badge/tooltip so the user knows
the adapter may change. `disabled` providers should be hidden from the install
target list but may appear in Settings so the user can re-enable them if the app
supports provider toggles.

## Unsupported Provider Policy

If a scan detects signs of an unsupported provider:

```text
project_providers.detection_status = unsupported
provider_definitions.status = unsupported
warnings.code = unsupported_provider
```

UI should:

- Display the provider as unsupported.
- Not allow installing skills into that provider.
- Allow the user to view the related path.
- May allow the user to report/submit the provider convention later.

## Provider Adapter Boundary

Provider adapters should return structured results and not mutate the database
directly.

Adapters also do not perform filesystem writes themselves. Operations such as
`mkdir`, symlink creation, rsync/copy, delete, and relink are all performed by
core Skillbox logic after the adapter returns path and capability metadata. This
makes adapters easier to test and reduces the risk of writing to the wrong
location.

Example adapter output:

```text
provider_key
detected_path
skills_path
detection_status
warnings
entries
```

Minimum output contract:

```text
provider_key: text
detected_path: absolute path | null
skills_path: absolute path | null
detection_status: detected | configured | missing | unsupported | invalid_structure | format_unknown
warnings: list of {
  code: text
  severity: info | warning | error | blocking
  message: text
  action_key: text | null
}
entries: list of {
  name: text
  path: absolute path to the skill entry within the provider skills_path
  entry_type: symlink | directory | unknown
  symlink_target: path | null
}
```

Global adapter output contract:

```text
provider_key: text
global_path: absolute path | null
global_skills_path: absolute path | null
global_status: active | not_configured | missing | unreadable | invalid_structure | empty | disabled
warnings: list of {
  code: text
  severity: info | warning | error | blocking
  message: text
  action_key: text | null
}
entries: list of {
  name: text
  path: absolute path to the global skill entry within global_skills_path
  entry_type: symlink | directory | unknown
  symlink_target: path | null
}
```

Core Skillbox logic is responsible for:

- Writing `project_providers`.
- Writing `installs`.
- Writing `warnings`.
- Running install/sync/remove.
- Performing filesystem writes after validating adapter output.

This boundary makes adapters testable and prevents database logic from being
scattered.

## Provider Plugin Layer Model

Some providers (initially Claude, Codex, Antigravity CLI) support the **plugin**
concept through their own settings files (`~/.claude/settings.json`,
`~/.codex/config.toml`, `~/.gemini/antigravity-cli/settings.json`, …). Plugins
differ from skills: a plugin is a provider extension declared in the provider's
settings file, may come from an external marketplace, and can be enabled/disabled
without deleting it from disk.

Skillbox Phase 1 only reads/writes settings files to display state and allow
toggling enable/disable. Skillbox does not manage the downloading/installation of
marketplace content; the provider handles that.

### Layer Precedence

Plugin state is declared across three layers with clear precedence:

```text
local   (project-scoped, this machine, not committed)
project (project-scoped, shared in commit)
user    (global at user/machine level)
```

Effective rule: `local > project > user`. A higher-precedence layer overrides
declarations from lower-precedence layers. Absence of a declaration at a layer =
`absent` at that layer (falls through to the next layer).

Effective status after merge:

```text
enabled
disabled
absent   (not declared at any layer)
unknown  (declared but the layer containing the declaration has scan_status != ok)
```

### Toggle Semantics

The UI allows users to operate on plugin state at two scopes:

- **User layer (Global Plugins screen)**: 2-state toggle. Enable / Disable
  globally. Writes to `~/.claude/settings.json` (or equivalent) at user scope.
- **Project layer (Project Detail screen)**: 3-state cycle. Inherit (no
  declaration at project layer, falls through to user) → Enable (force enable at
  project) → Disable (force disable at project) → Inherit. "Inherit" is
  implemented by removing the entry from the project's `.claude/settings.json`.

The local layer (`settings.local.json`) is scan-only in Phase 1; Skillbox does
not write it. Users may still edit the local file manually for temporary
overrides.

### Plugin Version Display

Each provider uses a different source for version information (applies to both
user layer and project layer unless noted separately):

**Claude** — reads `~/.claude/plugins/installed_plugins.json` (same root
directory as settings.json). This file is written by Claude Code when installing
plugins, and contains `version` keyed by `pluginName@marketplaceName` with scope
`user`.

- Version `"unknown"` is a valid literal (Claude reports it when the version
  cannot be determined).
- JSON `null` or absent field → version = NULL → UI displays `—`.
- File missing or malformed → version for all entries = NULL; settings.json scan
  is unaffected.

Adapter: `ScanClaudeInstalledPluginsFile` — path confined under `~/.claude`,
lstat symlink reject, 1 MiB size cap, tolerant JSON.

**Codex** — reads cache dir
`~/.codex/plugins/cache/<marketplace>/<plugin>/<version-or-sha>/`. Codex has no
installed_plugins.json; version is resolved from the cache structure:

1. If `plugin.json` exists in the version dir → use the `"version"` field
   (authoritative; used for semver plugins, e.g. stitch-skills → `"1.0.0"`).
2. If no `plugin.json` → use the directory name verbatim (e.g. git-source plugin
   produces a dir named short SHA `"9b3c8689"`).
3. No version dir → NULL → UI displays `—`.

The cache dir is global (no per-project cache in Codex), so the same version map
is used for both user layer and project layer in `scanProjectInternal`.

Adapter: `ScanCodexCacheDir` — path confined under `~/.codex`, lstat symlink
reject on every level (marketplace/plugin/version dir), 64 KiB size cap for
plugin.json.

**Antigravity CLI** — no equivalent version source → version is always NULL.

Version is persisted in `provider_plugin_entries.version` (nullable column,
migration 000021).

### Scan Flow

A plugin scan operation:

```text
Trigger (manual or auto after opening project / Global Plugins screen)
  -> For each provider with plugin support:
       -> Resolve settings file path for the layer being scanned
            (user: from ~/.<provider>/settings.json,
             project: from <project>/.<provider>/settings.json,
             local: from <project>/.<provider>/settings.local.json)
       -> Defensive checks:
            * file must lie within user home / project root (path_escape)
            * do not follow symlinks (symlink)
            * size must be below threshold (too_large)
       -> Read + parse file (JSON/TOML depending on provider)
       -> Create/update provider_plugin_layer_scans with appropriate scan_status
       -> DELETE all provider_plugin_entries for this layer_scan_id
       -> DELETE all provider_plugin_marketplaces for this layer_scan_id
       -> If scan_status = ok:
            -> Reinsert provider_plugin_entries from parsed content
            -> Reinsert provider_plugin_marketplaces from parsed content
       -> Write parse-time warnings into scan_warnings (JSON array; bounded)
```

Replace-by-scan strategy: DELETE happens **unconditionally** each scan, regardless
of scan_status. Reinsert only happens when `scan_status = ok`. Result: if a file
becomes `missing`, `malformed`, etc., the old entries + marketplaces for that
layer are wiped rather than preserved. No diff/migration logic needed in code.

### Settings File Paths

Provider settings file paths are seeded in `provider_path_candidates` with
`purpose = config`. Two layers:

- `scope = global`: user-level settings (e.g. `~/.claude/settings.json`).
- `scope = project`: project-level settings. Two path candidates in this scope do
  not compete — they fill **two separate layer slots** via `ORDER BY priority
  DESC`: `.claude/settings.json` (priority = 10) → index 0 → `project` layer
  slot; `.claude/settings.local.json` (priority = 9) → index 1 → `local` layer
  slot. Higher priority column is processed first (DESC sort), determining which
  slot is the project layer and which is the local layer — unrelated to layer
  merge precedence. When merging effective state, `local` still has **higher
  precedence** than `project` (the rule `local > project > user` is unchanged).

Users may override these paths via `provider_path_overrides` with the same
`(scope, purpose = config)`.

### Marketplace Concept

A marketplace is a **named source** from which a plugin is resolved. The concept
is defined by the provider; Skillbox only records metadata. Common source types:

```text
github      (owner/repo)
git         (git URL)
directory   (local path)
url         (HTTP URL)
settings    (marketplace metadata defined in the settings tree)
hostPattern (provider-specific routing)
```

`source_type` does not enforce a CHECK in the migration; each provider may have
its own source types. Marketplace metadata does not contain credentials.

### Provider Plugin Service Boundary

Provider plugin scanner/service responsibilities:

- Read settings files by layer.
- Validate defensive rules before parsing.
- Persist scan results into 3 tables (`provider_plugin_layer_scans`,
  `provider_plugin_entries`, `provider_plugin_marketplaces`).
- Resolve effective state per project / global view.
- Write enable/disable changes to the settings file of the appropriate layer when
  the user toggles (user/project layer only).

Provider plugin service does NOT:

- Download/install marketplace content (the provider handles that).
- Edit the local layer (`settings.local.json`).
- Modify managed settings (out of scope for Phase 1; `ManagedOutOfScope = true`
  is always returned so the UI can display it).

### Domain Object: provider_plugin

The domain layer exposes key structs (see
`core-go/internal/domain/provider_plugin.go`):

- `PluginLayerScan` — result of scanning one layer.
- `PluginEntry` — a single plugin declaration in one scan.
- `PluginMarketplace` — a marketplace declaration.
- `PluginEffectiveEntry` — plugin after resolving effective status, with
  per-layer provenance.
- `GlobalPluginView` — view for the Global Plugins screen (user layer per
  provider).
- `ProjectPluginView` — view for a project (merge local + project + user).
- `PluginCount` / `PluginProviderCount` — aggregates for Dashboard / Projects.

## Open Questions

- What exactly should the Claude convention be, and which paths should the
  adapter support?
- Do Codex/opencode/Antigravity CLI need their own adapters immediately, or
  should they use `generic_agents` first?
- Should provider icons use bundled assets, icon keys, or a package icon set?
