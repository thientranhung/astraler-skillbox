# Technical Architecture

This document outlines the technical architecture of Astraler Skillbox at the
module and boundary level. The goal is to help the team build the app without
mixing UI, database, filesystem writes, provider conventions, and
operation/audit logic.

Confirmed stack:

- Desktop framework: Electron.
- UI framework: React.
- Core runtime language: Golang.

The boundaries below remain important because Electron/React should not directly
own the database, filesystem writes, provider adapters, or long-running jobs.

Implementation decisions below are of two kinds:

- Confirmed: foundational stack and responsibility boundaries.
- Recommended defaults: technical suggestions that need to be brainstormed and
  confirmed before scaffolding real code.

Current technical brainstorms are recorded in:

- `docs/archive/review-results/technical-architecture-brainstorm.md`
- `docs/archive/review-results/transport-decision-brainstorm.md`

## Architecture Goals

- GUI is the primary experience.
- Skill Host Folder is the source of truth for skill content.
- SQLite is the source of truth for management metadata.
- Filesystem scans always have authority to reconcile the database with the true
  state.
- Provider conventions live in adapters, not hardcoded throughout the UI.
- Filesystem writes must go through a service that has validation and audit.
- Long-running work such as scan, fetch, update, sync must create an operation
  record.
- UI does not directly operate on the filesystem.

## High-Level Shape

```text
Electron Desktop App
  -> React UI Layer
  -> Electron Bridge / IPC Client
  -> Golang Core Runtime
     -> Application Services
     -> Domain Services
     -> Data Access Layer
     -> SQLite
     -> Filesystem Gateway
     -> Provider Adapters
     -> External Sources
```

Meaning:

- React UI Layer only renders state, receives input, calls commands/queries.
- Electron Bridge / IPC Client is the boundary between UI and Golang core.
- Golang Core Runtime owns application services and all side-effecting
  operations.
- Application Services are the entry point for each use case.
- Domain Services hold shared business logic.
- Data Access Layer reads/writes SQLite.
- Filesystem Gateway consolidates all file/folder/symlink/copy operations.
- Provider Adapters understand conventions for Claude, Generic Agents, Codex,
  etc.
- External Sources handle GitHub, Vercel skills, local import.

## Runtime Processes

Skillbox should be thought of in three runtime parts:

```text
Electron main process
  -> app window lifecycle
  -> launch/manage Golang core runtime
  -> expose narrow IPC bridge to renderer

Electron renderer process / React UI
  -> render screens
  -> send commands/queries through the bridge
  -> receive progress/results/warnings

Golang core runtime
  -> SQLite
  -> filesystem access
  -> provider adapters
  -> fetch/update/sync jobs
  -> operation audit
```

Decision: Golang core runs as a sidecar process managed by the Electron main
process in Phase 1.

Decision: the transport between the Electron main process and Golang core is
stdio JSON-RPC 2.0 in Phase 1. Key reasons: no local port needed, no port
conflicts, no macOS firewall prompt, easy desktop app packaging, and support
for both request/response and server-push notifications.

Decision: JSON-RPC Phase 1 uses NDJSON framing and `creachadair/jrpc2`, unless
a spike implementation finds a specific blocker.

The current product has one desktop consumer. Keep the JSON-RPC protocol and
stdio transport unless a concrete accepted requirement changes that boundary.

Even with a stdio transport, the UI contract should be a command/query API rather
than direct implementation detail calls.

UI should not directly import database clients, filesystem APIs, or provider
adapter implementations.

## Transport Decision

Phase 1 uses stdio JSON-RPC 2.0:

```text
Electron main process
  -> spawn Go core binary
  -> write JSON-RPC requests to child stdin
  -> read JSON-RPC responses/notifications from child stdout
  -> forward safe events to React renderer through preload bridge

Go core runtime
  -> read JSON-RPC requests from stdin
  -> write only JSON-RPC protocol messages to stdout
  -> write logs/debug output to stderr or log file
```

Rules:

- Stdout is the protocol boundary. Do not use `fmt.Print*` or ordinary log
  output to stdout in Go core.
- Go core must send `server.ready` notification before Electron main forwards
  renderer requests.
- Electron main waits up to 10 seconds for `server.ready` after spawning Go
  core.
- If timeout or Go exits before `server.ready`, Electron main kills the child
  if still alive, shows a blocking error window, and surfaces the stderr/log
  path.
- Operation progress uses JSON-RPC notifications such as `operation.progress`.
- Request/response uses JSON-RPC `id` to support multiple in-flight requests.
- Operation locking lives at the service layer, not the transport layer.
- Production does not open a local HTTP server.
- App error codes do not use the JSON-RPC reserved range `-32768` to `-32000`.

Implementation details:

- JSON-RPC Go library: `creachadair/jrpc2`.
- Framing: NDJSON, one JSON object per line.
- A dev-only debug server may be added later via `SKILLBOX_DEBUG_PORT` but is
  not part of the production path.

## Module Boundaries

```text
app/
  ui/
  electron/
  core-go/
  shared/
```

Candidate module shape in Golang core:

```text
core-go/
  services/
  domain/
  repositories/
  providers/
  filesystem/
  sources/
  operations/
  migrations/
```

Candidate module shape in React/Electron side:

```text
ui/
  screens/
  components/
  view-models/
  client/

electron/
  main/
  preload/
  core-process/

shared/
  api-contracts/
```

Proposed boundary:

- `ui`: React screens, components, view models, client API.
- `electron/main`: window lifecycle, app menu, native dialogs, core process
  lifecycle.
- `electron/preload`: narrow bridge exposed to the renderer.
- `electron/core-process`: start/stop/monitor Golang core runtime. This folder
  may be renamed or split after the transport is confirmed.
- `shared/api-contracts`: command/query request and response shapes.
- `core-go/services`: use case orchestration.
- `core-go/domain`: business rules, enums, validation.
- `core-go/repositories`: SQLite queries and transactions.
- `core-go/providers`: provider definitions, adapters, detection contracts.
- `core-go/filesystem`: safe path, symlink, copy, remove, scan helpers.
- `core-go/sources`: GitHub/Vercel/local/manual source integrations.
- `core-go/operations`: job runner, progress, cancellation, audit.
- `core-go/migrations`: SQLite schema migrations and seed data.

Previous conceptual grouping:

```text
app/
  ui/
  core/
    services/
    domain/
    repositories/
    providers/
    filesystem/
    sources/
    operations/
    migrations/
```

This conceptual boundary still holds, but the implementation folder remains a
candidate shape until real code is scaffolded.

## Application Services

Application Services are the API that the UI calls. Each service should expose
clear commands/queries and not leak SQL or raw filesystem details to the UI.

Key services:

```text
SettingsService
SkillHostService
SkillLibraryService
ProjectService
ProviderService
GlobalSkillsService
InstallService
UpdateService
ProviderPluginService
OperationService
WarningService
```

Mapping:

- `SettingsService`: app settings, active Skill Host Folder, default install
  mode, global provider location settings.
- `SkillHostService`: select host folder, init `.agents/skills`, scan host.
- `SkillLibraryService`: list/import/fetch/update skills.
- `ProjectService`: add project, scan project, project detail queries.
- `ProviderService`: provider detection, provider definitions, icons/status.
- `GlobalSkillsService`: scan global locations, list global entries, remediation.
- `InstallService`: install and remove project symlink installs.
- `UpdateService`: fetch all, update host copy, impact preview.
- `ProviderPluginService`: scan, toggle, and remove plugin overrides across
  layers (user/project/local). Owns `pluginWriterFn` and `pluginRemoverFn`
  abstractions for JSON and TOML config files.
- `OperationService`: start/read/cancel operation records.
- `WarningService`: list/resolve/dismiss warning state if needed.

## Command And Query Pattern

The React UI should call Golang core through two types of API:

```text
Query:
  getDashboard()
  listSkills()
  getProjectDetail(projectId)
  getGlobalSkills()
  getUpdateOverview()

Command:
  chooseSkillHostFolder(path)
  scanSkillHostFolder(hostId)
  addProject(path)
  scanProject(projectId)
  scanGlobalSkills()
  installSkillToProject(input)
  syncInstall(installId)
  updateSkill(skillId)
  providerPlugin.setEnabled(input)
  providerPlugin.removeOverride(input)
  updateCheck.run()
  app.resetAll()        -- truncate user data tables + reset settings to defaults
  app.checkUpdate()     -- query GitHub Releases API for latest app version (always-on)
```

Queries should have no side effects. Commands may create `operations` records,
write to the DB, and operate on the filesystem.

IPC/transport rules:

- Renderer only calls APIs exposed through the Electron preload bridge.
- Renderer must not call Node filesystem APIs directly.
- Electron main should not contain business logic; it handles only
  lifecycle/bridge/native integration.
- Golang core returns typed responses for all commands/queries.
- Long-running commands must return an `operation_id`.
- If using stdio JSON-RPC, progress should go via JSON-RPC server-push
  notifications like `operation.progress`; do not use polling as the primary
  model.

## Data Access Layer

The repository layer is the only place where SQL is written directly.

Repository groups:

```text
AppSettingsRepository
SkillHostRepository
SkillRepository
SkillSourceRepository
ProjectRepository
ProviderRepository
ProjectProviderRepository
GlobalProviderLocationRepository
InstallRepository
GlobalInstallRepository
FetchResultRepository
ScanResultRepository
WarningRepository
OperationRepository
```

Rules:

- Each large command should use a transaction when updating multiple tables.
- Scan commands should write `scan_results`, update entity status, and reconcile
  stale rows in a single transaction after filesystem reads complete.
- Filesystem writes should be validated first, the write performed, then the DB
  updated in a transaction immediately after.
- Do not store plaintext secrets in SQLite.
- Migrations must run before the app opens its main window.

SQLite startup sequence:

```text
Open SQLite connection
  -> Apply connection PRAGMAs
  -> Run migrations
  -> Seed provider definitions through migration
  -> Open app main window only after success
```

Required PRAGMAs for every connection, including tests:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

SQLite file path:

```text
macOS:   ~/Library/Application Support/Astraler Skillbox/skillbox.db
Windows: %APPDATA%\Astraler Skillbox\skillbox.db
Linux:   ~/.config/astraler-skillbox/skillbox.db
Tests:   SKILLBOX_DB_PATH override to temp database path
```

## Filesystem Gateway

The Filesystem Gateway is the mandatory boundary for all path operations.

Responsibilities:

- Normalize absolute paths.
- Resolve realpath when needed.
- Validate that a path lies within the allowed root before any write.
- Detect symlink, broken symlink, external symlink.
- Create/remove symlink.
- Remove managed install entry.
- Read directory entries for scan.
- Open folder via OS shell when UI requests it.

Write safety rules:

- Do not write to a project/provider path if the provider adapter has not
  resolved the target.
- Do not remove a folder/file if the entry is not recognized as a managed
  install, unless the user has clearly confirmed it is a direct/unmanaged entry.
- When removing a symlink, do not follow the target and delete the target; only
  remove the link.
- Do not overwrite a direct install without confirmation and an impact preview.
- Do not create paths outside the project root for a project install.
- Do not create paths outside the configured global provider location for global
  remediation.

## Provider Adapter Boundary

Provider adapters do not access the UI and do not write to the database directly.

Adapter input:

```text
project_root
provider_definition
path_candidates
configured_paths
skill_host_folder
```

Adapter output:

```text
detected project providers
resolved skills paths
installed entries
global provider locations
global entries
warnings
capabilities
```

Core Skillbox logic receives this output and then decides:

- Which tables to write.
- Which warnings to create.
- Which install targets are valid.
- Which actions to enable/disable in the UI.

Provider adapters only return facts and capabilities. Product policy lives in
core services.

## Operation Model

The following operations should run through the Operation runner:

- Scan Skill Host Folder.
- Scan project.
- Scan global skills.
- Fetch updates.
- Update Skill Host Folder copy.
- Remove managed install.
- Change Skill Host Folder.

Operation lifecycle:

```text
queued
running
succeeded
failed
cancelled
```

The operation runner should:

- Write an `operations` record before running.
- Emit progress to the UI.
- Write the result/error summary.
- Not allow two conflicting operations to run concurrently on the same target.
- Allow retry if the error is not a validation error.

Recommended progress model when using stdio JSON-RPC:

- Go core sends `operation.progress` notifications via stdout.
- Electron main parses notifications and forwards them through the preload
  bridge.
- React UI subscribes by `operation_id`.
- When the operation ends, the UI re-fetches the related view model to get the
  state reconciled from SQLite.
- Cancel uses the command `operation.cancel` with `operation_id`; Go uses
  `context.WithCancel` and checks for cancellation at natural checkpoints.
- Retry is a new command from the UI, not silent auto-retry inside Go.

Startup and shutdown lifecycle:

- Electron main spawns Go with `spawn()`, not `exec()`.
- Electron main waits up to 10 seconds for `server.ready`.
- If Go exits or times out before `server.ready`, show a blocking startup error;
  do not silently retry.
- During an app session, if Go exits unexpectedly, Electron main may restart up
  to 3 times before showing a blocking error.
- On `before-quit`, Electron main sends SIGTERM, waits 3 seconds, then SIGKILL.
- Go handles SIGTERM and stdin EOF by marking running operations as failed,
  closing SQLite, and exiting.

Suggested locking:

- A Skill Host Folder should have at most one active scan/update operation.
- A project should have at most one active scan/install/sync/remove operation.
- A global provider location should have at most one active
  scan/remediation operation.

## Scan And Reconcile

Scanning is the mechanism for bringing the database close to the true filesystem
state.

Project scan:

```text
read project path
detect providers
scan provider skills paths
classify entries
compare with installs table
mark missing/stale records
upsert project_providers and installs
write warnings
write scan_results
```

Global scan:

```text
load providers with has_global_level or configured global paths
resolve global locations
scan global skills paths
classify entries
compare with global_installs table
upsert global_provider_locations and global_installs
write warnings
write scan_results
```

Skill Host scan:

```text
read active Skill Host Folder
ensure or validate .agents/skills
scan skill folders
read source metadata when available
upsert skills and skill_sources
mark missing/unreadable/local_modified
write warnings
write scan_results
```

Reconcile rule:

- Filesystem state wins for existence/status.
- SQLite metadata wins for management intent, source mapping, and operation
  history.
- If filesystem and database disagree, the UI shows an explicit status instead
  of silently hiding the mismatch.

## Install, Sync, Remove

Install to project:

```text
validate skill exists in active host
validate project exists and provider target is supported
resolve target path via provider adapter
show impact preview if target exists
write symlink through filesystem gateway
upsert installs
write operation result
refresh project detail
```

Remove project install:

```text
validate target install
if managed symlink, remove target entry
if direct/unmanaged, require stronger confirmation
mark or delete install metadata based on product policy
write operation result
```

Phase 1 does not include Install Skill To Global Location. Global remediation
can support safe actions such as open folder, update configured path, or relink
managed broken symlinks if those entries were previously created/adopted by
Skillbox.

## Fetch And Update Sources

Source integrations should be separate from provider adapters.

```text
GitHubSourceAdapter
VercelSkillSourceAdapter
LocalSourceAdapter
ManualSourceAdapter
```

Responsibilities:

- Fetch latest version metadata.
- Compare current version/commit/checksum.
- Report auth/network/not-fetchable states.
- Download/update Skill Host Folder copy when user confirms.

GitHub/Vercel source logic should not know project providers. After a host
update, UpdateService computes affected project installs and global installs
from the DB.

## UI State Composition

Screens should be backed by view models assembled from queries, not by the UI
joining raw tables manually.

View models:

```text
DashboardView
SkillsLibraryView
SkillDetailView
ProjectsView
ProjectDetailView
GlobalSkillsView
UpdatesView
SettingsView
```

Each view model should include:

- Primary entities.
- Counts.
- Action availability.
- Warning summaries.
- Loading/operation state.
- Empty state reason.
- Next recommended action.

Action availability should come from core rules, not UI-only checks.

## Error Handling

Use typed application errors:

```text
validation_error
filesystem_error
provider_error
database_error
auth_error
network_error
conflict_error
operation_cancelled
unknown_error
```

Every command result should return:

```text
status
operation_id
changed_entities
warnings_created
user_message
technical_message
```

UI shows `user_message`. Logs/debug tools can show `technical_message`.

## Security And Privacy

- Do not store plaintext tokens in SQLite.
- Prefer OS keychain for GitHub/Vercel credentials.
- Treat project paths and skill content as local private data.
- Do not send local file content to an external service unless the user
  explicitly triggers a source/fetch feature that requires it.
- Log paths and operation metadata, but avoid logging secret values.
- Any future telemetry must be opt-in.

Electron security decisions:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
preload exposes narrow typed bridge only
renderer never receives Go process path or transport details
Electron main validates JSON-RPC method allowlist before forwarding to Go
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
dev only: remote-debugging-port on 127.0.0.1 (default 49222, gated on ELECTRON_RENDERER_URL); packaged builds never open a debugging port
```

Dev exposes a Chrome DevTools Protocol port (default `49222`, override
`SKILLBOX_CDP_PORT`) so browser-automation agents can `connect` to the running
`pnpm dev` instance instead of launching a second app. It is gated on
`ELECTRON_RENDERER_URL` (set by electron-vite only in dev) and binds to loopback
only — packaged builds never open it. See `AGENTS.md` → "Agent Browser".

## Testing Strategy

Core test layers:

- Domain unit tests for install mode classification and impact preview.
- Provider adapter tests using fixture folders.
- Filesystem gateway tests in temp directories.
- Repository tests against temporary SQLite database.
- Service tests for scan/install/sync/update flows.
- UI tests for view states and disabled/enabled actions.

Critical fixtures:

- Empty Skill Host Folder.
- Missing Skill Host Folder.
- Project with `.agents/skills`.
- Project with multiple providers.
- Managed symlink install.
- Broken symlink install.
- External symlink install.
- Direct/unmanaged install.
- Global provider location missing.
- Global/project overlap.

## Phase Boundaries

Phase 1:

- GUI-first app.
- One active Skill Host Folder.
- SQLite metadata.
- Scan Skill Host Folder.
- Add project and scan project providers.
- Project install via symlink (current stable path).
- Global Skills scan/visibility/remediation surface.
- Fetch/update source metadata for GitHub/Vercel/local where supported.
- Updates impact preview.

## Architecture Decisions To Confirm

The decisions below are points that still need to be confirmed before scaffolding
real code. The Phase 1 transport has been confirmed as stdio JSON-RPC 2.0;
framing/library details remain open.

```text
IPC transport:
  phase_1_decision = stdio JSON-RPC 2.0
  migration_path = keep JSON-RPC protocol; change transport only for a concrete accepted requirement
  library = creachadair/jrpc2
  framing = NDJSON
  open = dev debug server

Go core lifecycle:
  phase_1_decision = sidecar process managed by Electron main
  alternative = persistent daemon if background work becomes product requirement

Operation progress:
  phase_1_decision = JSON-RPC server-push notifications
  avoid = polling as primary progress model

API contract:
  recommended = JSON Schema in shared/api-contracts, generate TypeScript types
  open = whether Go structs are generated or hand-matched

SQLite:
  recommended = modernc.org/sqlite for no-CGO Phase 1 builds
  migrations = embedded SQL migrations
  pragmas = WAL, foreign_keys=ON, busy_timeout=5000, synchronous=NORMAL
  path = OS app data directory, SKILLBOX_DB_PATH override for dev/test

Keychain:
  recommended = Go core owns credentials via zalando/go-keyring
  fallback = SKILLBOX_GITHUB_TOKEN, SKILLBOX_VERCEL_TOKEN for dev/CI

Packaging:
  recommended = electron-builder with bundled Go binary
  high-risk = macOS code signing and notarization for both app and Go binary

Provider seed data:
  recommended = seed via migration
  alternatives = bundled JSON or code seed

Outbound Network:
  scope = manual-trigger plugin update checks only (always-on, see ADR-0002 supersedes ADR-0001)
  trigger = user clicks "Check Updates" on Plugins screen; no background polling, no auto-check
  gate = none (the update_check_enabled opt-in column was dropped in migration 000023)
  mechanism = git ls-remote via system git (no new SDK); HTTPS URLs only
  security = HTTPS-only validation before subprocess; env-stripped (PATH + GIT_TERMINAL_PROMPT=0 only)
  timeout = 8s per-request, 60s batch deadline; max 4 concurrent subprocesses
  cache = plugin_update_check_cache table, 6h TTL default (network_settings.cache_ttl_hours)
  privacy = no telemetry, no Skillbox-operated server; app fully usable offline
  renderer_boundary = renderer never calls network; all outbound via Go core (UpdateCheckService)
  see = docs/decisions/0002-plugin-update-check-always-on.md
```
