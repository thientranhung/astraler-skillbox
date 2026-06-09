# Implementation Patterns

This document confirms the patterns to use when implementing Astraler Skillbox.
The goal is to translate the architecture in `10-technical-architecture.md` and
the tech stack in `11-tech-stack-and-scaffold-decisions.md` into concrete code
rules.

## Pattern Principles

- Patterns serve boundaries, not code complexity.
- UI does not own business rules, filesystem writes, SQLite, or provider logic.
- Electron main contains no business logic; it holds lifecycle, the preload
  bridge, native dialogs, and the allowlist.
- Go core is where command/query handlers, services, repositories, provider
  adapters, the filesystem gateway, and the operation runner live.
- Every operation with side effects must have validation, audit/log, and a clear
  error path.

## 1. Process Coordinator

Where it applies:

```text
apps/desktop/electron/main/
apps/desktop/electron/core-process/
```

Responsibility:

- Spawn Go sidecar with `spawn()`, not `exec()`.
- Parse stdout as a JSON-RPC NDJSON protocol stream.
- Forward responses/notifications safely to the renderer through the preload
  bridge.
- Read stderr as a log stream.
- Wait for `server.ready` for up to 10 seconds.
- If Go exits before `server.ready` or times out, show a blocking startup error.
- On app quit, send SIGTERM, wait 3 seconds, then SIGKILL if needed.
- Mid-session crash may restart up to 3 times, then show a blocking error.

Do not:

- Contain business logic.
- Read/write SQLite.
- Directly operate on the Skill Host Folder or project files.
- Expose raw Go transport details to the renderer.

## 2. Narrow Preload Bridge

Where it applies:

```text
apps/desktop/electron/preload/
apps/desktop/renderer/src/lib/core-client/
```

Responsibility:

- Expose a narrow API such as `invoke(method, params)` and
  `onEvent(event, callback)`.
- Renderer does not import `ipcRenderer` directly.
- Renderer does not know the Go binary path, stdin/stdout, or process lifecycle.
- Electron main validates the method allowlist before forwarding to Go.

Security defaults:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

## 3. JSON-RPC Command/Query Boundary

Where it applies:

```text
core-go/internal/rpc/
shared/api-contracts/
shared/generated/
```

Decision:

```text
transport = stdio
protocol = JSON-RPC 2.0
framing = NDJSON
go_library = creachadair/jrpc2
ready_notification = server.ready
progress_notification = operation.progress
```

Rules:

- Stdout is for JSON-RPC protocol messages only.
- Logs go to stderr or a log file.
- Requests have `id`; notifications do not have `id`.
- App error codes do not use the JSON-RPC reserved range `-32768` to `-32000`.
- Long-running commands return `operation_id`.
- Progress goes via server-push notifications; do not use polling as the primary
  model.
- Contract schemas live in `shared/api-contracts`.
- Generated TypeScript types are committed.
- Go structs are hand-written in Phase 1; contract tests validate JSON Schema.

## 4. CQRS For UI-Facing API

Where it applies:

```text
core-go/internal/services/
core-go/internal/rpc/
apps/desktop/renderer/src/
```

Queries:

- No side effects.
- Return view models prepared for the UI.
- May join multiple tables through the repository layer.
- React renders data; it does not self-join or reason about complex business
  logic.

Examples:

```text
getDashboard()
listSkills()
getProjectDetail(projectId)
getGlobalSkills()
getUpdateOverview()
```

Commands:

- Validate input.
- May write SQLite or the filesystem.
- For long-running tasks, create an operation and return `operation_id`.
- Do not run write operations directly in the renderer.

Examples:

```text
chooseSkillHostFolder(path)
scanSkillHostFolder(hostId)
addProject(path)
scanProject(projectId)
scanGlobalSkills()
installSkillToProject(input)
syncInstall(installId)
updateSkill(skillId)
operation.cancel(operationId)
```

## 5. Application Service Layer

Where it applies:

```text
core-go/internal/services/
```

Responsibility:

- Orchestrate use cases.
- Call repositories, filesystem gateway, provider adapters, operation runner.
- Make product policy decisions.
- Do not write SQL directly.
- Do not bypass the filesystem gateway.

Examples:

```text
SkillHostService
SkillLibraryService
ProjectService
GlobalSkillsService
InstallService
UpdateService
OperationService
SettingsService
```

Rule:

- Provider adapters return facts/capabilities.
- Services decide which actions are permitted.

## 6. Repository Pattern

Where it applies:

```text
core-go/internal/repositories/
```

Responsibility:

- The only place where SQL is written directly.
- Manages transactions.
- Provides query methods for services.
- Keeps SQLite details out of the service layer.

Rules:

- Multi-table writes use transactions.
- Scan/reconcile updates entity status and stale rows in a transaction.
- Hot queries need indexes; check with `EXPLAIN QUERY PLAN` when needed.
- Repository tests use a temporary SQLite database.

SQLite startup:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

## 7. Filesystem Gateway

Where it applies:

```text
core-go/internal/filesystem/
```

Responsibility:

- Normalize absolute paths.
- Resolve realpath when needed.
- Validate allowed roots before any write.
- Detect symlink, broken symlink, external symlink.
- Create/remove symlink.
- Copy folder for rsync/copy. _(Reserved — not implemented in current release.)_
- Remove managed install entries.
- Read directory entries for scan.

Hard rules:

- No service may call `os.WriteFile`, `os.Remove`, `os.Rename`, or
  copy/symlink helpers directly when operating on skills, projects, providers,
  or the host folder; all writes go through the gateway.
- Invalid writes outside the allowed root are hard-blocked — no "continue
  anyway".
- Direct/unmanaged entries require a confirmation policy at the service/UI level
  before the gateway is called.
- When removing a symlink, do not follow the target and delete it; only remove
  the link.
- Paths from the renderer are always treated as untrusted input.

## 8. Provider Adapter Pattern

Where it applies:

```text
core-go/internal/providers/
```

Responsibility:

- Detect provider within a project.
- Resolve provider project paths.
- Resolve provider global locations if the provider has a global level.
- Scan entries in the provider scope.
- Classify facts: detected paths, entries, capabilities, warnings.

Do not:

- Write to the DB.
- Write to the filesystem.
- Make install/update policy decisions.
- Render UI state.

Adapter output should be facts:

```text
provider key
detected path
skills path
entries
warnings
capabilities
global locations if applicable
```

Plugin config writers and removers also live in `providers/` because they are
provider-specific logic (JSON for Claude, TOML for Codex):

```text
pluginWriterFn  func(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error
pluginRemoverFn func(filePath, allowedDir, pluginName, marketplaceName string) error
```

- `WriteJSONPluginEnabled` / `RemoveJSONPlugin`: for JSON-based providers (Claude).
- `WriteTOMLPluginEnabled` / `RemoveTOMLPlugin`: for TOML-based providers (Codex).
- All enforce path confinement, symlink checks, file size cap (1 MiB).
- `ProviderPluginService.writerFor(providerKey)` / `removerFor(providerKey)`
  selects the implementation based on the provider config format.

## 9. Source Adapter Pattern

Where it applies:

```text
core-go/internal/sources/
```

Responsibility:

- GitHub/Vercel/local/manual source metadata.
- Fetch latest version metadata.
- Download/update host copy when user confirms.
- Map auth/network/not-fetchable errors to a common taxonomy.

Do not:

- Know about project providers.
- Decide affected projects.
- Sync rsync/copy installs.

UpdateService uses the DB to compute affected project installs and global
installs.

## 10. Operation Runner And State Machine

Where it applies:

```text
core-go/internal/operations/
```

State:

```text
queued
running
succeeded
failed
cancelled
```

Responsibility:

- Create operation record.
- Run long-running tasks in a goroutine with context.
- Emit `operation.progress` notifications.
- Support `operation.cancel`.
- Lock per target to avoid conflicting operations.
- Mark running operations as failed when the sidecar shutdown/crash path needs
  cleanup.

Phase 1 locking:

- Single active operation per target.
- If the target is busy, return a `conflict_error` fail-fast.
- No automatic queueing in Phase 1.

Examples:

```text
target = skill_host_folder:{id}
target = project:{id}
target = global_provider_location:{id}
target = install:{id}
```

Progress rules:

- Do not flood IPC.
- Progress should track phase/entry; do not fabricate a fake percent if not
  measurable.
- When the operation ends, the UI re-fetches the view model.

## 11. Manual Constructor DI

Where it applies:

```text
core-go/cmd/skillbox-core/main.go
core-go/internal/app/
```

Decision:

- Phase 1 uses manual constructor dependency injection.
- Do not use `google/wire`, `uber-go/dig`, or a DI container.

Why:

- Easy to read.
- Easy to review for AI and humans.
- Less magic.
- Appropriate when the number of services remains manageable.

Recommended shape:

```go
db := repositories.OpenDatabase(dbPath)
fs := filesystem.NewGateway()
providers := providers.NewRegistry(...)
ops := operations.NewRunner(...)

projectService := services.NewProjectService(db.ProjectRepo, providers, fs, ops)
installService := services.NewInstallService(db.InstallRepo, providers, fs, ops)

rpcServer.Register("project.scan", projectService.ScanProject)
rpcServer.Register("install.skill", installService.InstallSkill)
```

If the composition root grows too large, create `internal/app` to group wiring
logic; do not introduce a DI framework prematurely.

## 12. View Model Composition

Where it applies:

```text
core-go/internal/services/
apps/desktop/renderer/src/screens/
```

Rules:

- Go query handlers return view models suitable for each screen.
- React does not self-join `skills`, `installs`, `projects`, `warnings`.
- View models include action availability, warnings, empty-state reason, and
  loading state as needed.
- TanStack Query caches view models and invalidates them after command/operation.

Examples:

```text
DashboardView
HostSkillsView (renderer code may still expose this as SkillsLibraryView)
SkillDetailView
ProjectsView
ProjectDetailView
GlobalSkillsView
GlobalPluginsView
FutureSkillSourceUpdateView
SettingsView
AboutView
```

## 13. UI Component Composition

Where it applies:

```text
apps/desktop/renderer/src/components/
apps/desktop/renderer/src/screens/
```

Stack:

```text
shadcn/ui
Radix UI
Tailwind CSS
lucide-react
```

Patterns:

- App shell with sidebar navigation.
- Screen-level layout components.
- Detail panes for selected entities.
- Status badges.
- Warning banners with actions.
- Dialogs/AlertDialog for destructive actions.
- Popovers/DropdownMenu for scoped actions.
- Tooltip for icon-only buttons.

Avoid:

- Generic SaaS dashboard template assumptions.
- Hero/marketing layout.
- Cards inside cards.
- Renderer-only business rules.

## 14. Form Validation Pattern

Where it applies:

```text
apps/desktop/renderer/src/
core-go/internal/services/
shared/api-contracts/
```

Decision:

- React Hook Form + Zod for UI/form validation.
- JSON Schema for wire/API contract.
- Go validates params again in command/query handlers.

Duplication is intentional:

- UI validation optimizes user experience.
- JSON Schema optimizes the contract.
- Go validation protects the core from untrusted renderer input.

## 15. Error Taxonomy Pattern

Where it applies:

```text
core-go/internal/domain/errors.go
core-go/internal/rpc/
apps/desktop/renderer/src/
```

Error categories:

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

Rules:

- JSON-RPC error responses map from the domain error taxonomy.
- `conflict_error` is used when the operation target is busy.
- UI displays the user message; logs hold the technical message.
- Do not log secrets or full payloads containing sensitive data.

## 16. Testing Pattern

Where it applies:

```text
core-go/
apps/desktop/
fixtures/
shared/api-contracts/
```

Required:

- Go unit tests for pure domain logic.
- Repository tests with temporary SQLite.
- Filesystem gateway tests in temp directories.
- Provider adapter tests with fixture folders.
- JSON-RPC contract tests against JSON Schema.
- `go test -race` for operation runner/provider scan/filesystem gateway code.
- React component tests for complex UI states.

Deferred:

- Playwright until the first UI shell exists.

## What We Keep From The External Pattern Report

Useful ideas:

- Process Coordinator for Go sidecar lifecycle.
- JSON-RPC bidirectional notifications.
- CQRS distinction between queries and commands.
- Adapter pattern for providers/sources.
- Filesystem Gateway as security boundary.
- Operation Runner as state machine.
- Repository pattern for SQLite.
- Manual constructor DI in Go.

Corrections applied:

- Use `server.ready`, not `system.ready`.
- Invalid filesystem writes are blocked, not confirmed-through.
- Phase 1 operation locking is fail-fast per target, not an open queueing
  decision.
- Avoid `os.Exit(0)` in normal shutdown prose; prefer normal return after
  cleanup.
