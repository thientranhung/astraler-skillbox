# Tech Stack And Scaffold Decisions

This document collects decisions about the tech stack and scaffold structure
before creating the real codebase. The goal is to save time by choosing
appropriate frameworks/libraries while preserving the architecture boundaries
confirmed in `10-technical-architecture.md`.

Decision status:

```text
decided       = confirmed, used as constraint when scaffolding
recommended   = suggested default, needs review/confirmation before coding
open          = still needs discussion or a small spike
defer         = not needed in Phase 1
```

## Tech Stack Summary

```text
Desktop shell:
  status = decided
  choice = Electron

UI runtime:
  status = decided
  choice = React

Core runtime:
  status = decided
  choice = Golang

Database:
  status = decided
  choice = SQLite

Electron <-> Go transport:
  status = decided for Phase 1
  choice = stdio JSON-RPC 2.0

Go core lifecycle management:
  status = decided for Phase 1
  choice = sidecar process managed by Electron main
```

## Scaffold Principles

- Do not clone a boilerplate wholesale if it obscures Skillbox's boundaries.
- Use boilerplates to learn packaging, Electron security, Vite configuration,
  test setup, and folder naming conventions.
- Scaffold must accurately reflect the architecture boundaries:
  - React renderer does not touch the filesystem or database.
  - Electron main only manages lifecycle, the bridge, and native dialogs.
  - Golang core holds business logic, SQLite, provider adapters, the filesystem
    gateway, source integrations, and the operation runner.
- Favor structures that are easy to review by both AI and humans: clear
  directories, small files, clear contracts.
- Do not pull in a large library if no screen or use case needs it yet.

## Recommended Project Structure

Recommended:

```text
astraler-skillbox/
  README.md
  docs/

  apps/
    desktop/
      package.json
      pnpm-lock.yaml
      electron/
        main/
        preload/
        core-process/
      renderer/
        src/
          app/
          screens/
          components/
          features/
          lib/
          styles/

  core-go/
    go.mod
    cmd/
      skillbox-core/
    internal/
      app/
      domain/
      services/
      repositories/
      providers/
      filesystem/
      sources/
      operations/
      migrations/
      rpc/
    migrations/

  shared/
    api-contracts/
    generated/

  scripts/
  fixtures/
```

Rationale:

- `apps/desktop` groups the Electron + React app together.
- `core-go` is a separate Go module that can be built/tested independently.
- `shared/api-contracts` holds JSON Schema or protocol contracts.
- `fixtures` serves provider/filesystem scan tests.
- `scripts` holds helpers for build/dev/release.

Open:

- Whether to use `apps/desktop/renderer` or keep `ui/` at the root level for
  brevity.

Decided:

- No `go.work` in Phase 1 since there is only one Go module.
- No pnpm workspace in Phase 1 if there is only one JS package.
- Generated TypeScript types are committed to the repo; CI checks for drift.

## Boilerplate Research Direction

No boilerplate is selected as final in this document, but when evaluating, use
these criteria:

```text
Electron security:
  contextIsolation = true
  nodeIntegration = false
  narrow preload bridge
  renderer has no filesystem access

Build/dev:
  Vite for renderer
  Fast HMR
  Electron main/preload build support
  External binary packaging support

Packaging:
  electron-builder support
  extraResources for Go binary
  Clear macOS signing/notarization path

Testing:
  Vitest or equivalent for UI/core TS
  Playwright for desktop/e2e
  Go test runs independently

Maintainability:
  Simple folder structure
  Not a complex SaaS/dashboard template
```

Reference sources to evaluate:

- `electron-vite-react`
- `vite-electron-builder`
- `electron-react-boilerplate`
- Official Electron Forge Vite template

Recommendation:

- Use a Vite/Electron boilerplate as a reference, not as gospel.
- Self-scaffold if the template conflicts with the Go sidecar, JSON-RPC, or
  security boundaries.

## Frontend Build Tool

Status: recommended.

Choice: Vite.

Why:

- Fast React dev server and HMR.
- Common in modern Electron + React scaffolds.
- Good production bundling for the renderer.
- Works well with Tailwind, shadcn/ui, Vitest.

Risks:

- Electron main/preload builds need explicit configuration so Node/Electron APIs
  are not bundled incorrectly.
- Separate build targets needed for renderer, main, and preload.

Decision:

- Use `electron-vite` to manage Vite targets for renderer, main, and preload.

## Package Manager

Status: recommended.

Choice: pnpm, single JS package at `apps/desktop`.

Why:

- Fast installs.
- Deterministic lockfile.
- Works well for Electron development without workspace mode.

Risks:

- Some Electron tooling docs default to npm/yarn; commands need to be clearly
  documented.

Decision:

- Do not scaffold `pnpm-workspace.yaml` on day one.
- Only add a pnpm workspace when a second JS package exists.

## Electron Packaging

Status: recommended.

Choice: electron-builder.

Why:

- Mature for packaging Electron apps.
- Supports `extraResources` for the bundled Go binary.
- Good macOS signing/notarization support.
- Integrates well with `electron-updater` if auto-update is added.

Risks:

- macOS signing and notarization are high-risk and should be tested early.
- The Go binary must be bundled, signed, and launched from the production
  resource path.

Decision:

- Use `electron-builder` instead of Electron Forge.
- Plan signing/notarization as a technical milestone, not a last-minute release
  task.
- Defer `electron-updater` until a release/update flow is needed.

## UI Component Stack

Status: recommended.

Choice:

```text
shadcn/ui
Radix UI primitives
Tailwind CSS
lucide-react
```

Why:

- Radix provides accessible primitives for dialog, menu, tab, popover, tooltip,
  select, switch, checkbox, toast, scroll area, and more.
- shadcn/ui provides pre-styled component source that lives in the repo and
  can be customized.
- Tailwind keeps styling local and fast for the app UI.
- lucide-react provides consistent icons that fit the design direction.

Risks:

- shadcn components are copied into the repo, so the team owns maintenance.
- Tailwind can become messy without documented layout rules.
- Discipline needed: do not blindly pull in large block/template collections.

Decisions to confirm:

- Use shadcn/ui as a component source, not a ready-made dashboard template.
- Build an app shell, sidebar, toolbar, table, warning, and Skillbox-specific
  status components rather than a generic SaaS template.

## App UI Style

Status: recommended.

Skillbox should feel like an operational desktop tool:

- Dense but readable.
- Sidebar navigation.
- Tables/lists for skills, projects, global locations, updates.
- Detail pane for selected entities.
- Clear status badges and warnings.
- Minimal marketing/hero styling.
- Functional dialogs and wizards.

Avoid:

- Landing-page layouts.
- Oversized hero sections.
- Decorative cards nested inside cards.
- Heavy gradients/illustrations.
- Generic SaaS dashboard templates that hide filesystem/provider details.

## Router

Status: decided for Phase 1.

Choice: TanStack Router.

Why:

- Type-safe route definitions.
- Suitable for app screens with nested detail routes.
- More capable than React Router when route params/search state matter.

Risks:

- Slightly more learning/setup than React Router.
- Route model must stay simple since this is a desktop app, not a public web app.

Alternative:

- React Router if the team wants a simpler, more familiar router.

Decision:

- Use TanStack Router with `createMemoryHistory` for the Electron/file URL
  context.

## Server State And View Models

Status: decided for Phase 1.

Choice: TanStack Query for local JSON-RPC queries.

Why:

- Even though data is local, screens still need loading/error/refetch/cache
  state.
- Completing an operation can invalidate related queries.
- Avoids the React UI manually managing every request lifecycle.

Rules:

- Queries call the Electron preload bridge client, not Go directly.
- Mutations call commands and return `operation_id` when needed.
- UI re-fetches view models after an operation completes.

Risks:

- Over-caching may show stale filesystem state after scan/update.
- Query keys must be designed with discipline.

Decision:

- Use TanStack Query from day one.
- Keep stale time short and invalidate aggressively after command/operation
  completes.

## Client-Side UI State

Status: recommended.

Choice: Use React state first; Zustand is deferred until cross-screen ephemeral
state is genuinely needed.

Use React state for:

- Dialog open/close.
- Current form values.
- Local selections.

Only use Zustand if needed for:

- App shell UI state.
- Selected project/skill context shared across panels.
- Long-lived operation panel state that does not belong to a single screen.

Avoid:

- Putting server/database state into Zustand.
- Duplicating TanStack Query cache in a global store.

## Forms And Validation

Status: decided for Phase 1.

Choice:

```text
react-hook-form
zod
```

Why:

- Settings and wizard flows need clear validation.
- Zod schemas can mirror API contract validation.
- React Hook Form avoids excessive re-renders for controlled inputs.

Rules:

- Zod schema is the validation for UI/forms.
- JSON Schema in `shared/api-contracts` is the validation for the wire contract.
- Go validates command/query params independently at the core.

Risks:

- Some validation is intentionally duplicated because form UI constraints and
  wire contract constraints do not always match exactly.

## Tables

Status: defer.

Choice: start with simple table components; add TanStack Table when the first
real table screen needs sorting/filtering.

Why:

- Skillbox has many table screens:
  - Skills Library
  - Global Skills
  - Projects
  - Project Detail installs
  - Updates affected projects/global installs
- Sort/filter/selection will be common.

Risks:

- TanStack Table is headless so code can be verbose.
- Shared table components are needed to avoid repeated setup.

Decision:

- Do not include TanStack Table in the initial scaffold.

## JSON-RPC Protocol

Status: partially decided.

Decided:

- Phase 1 transport is stdio JSON-RPC 2.0.
- JSON-RPC library for Go is `creachadair/jrpc2`.
- Framing is NDJSON.
- Go core sends `server.ready` before Electron forwards renderer requests.
- Electron main waits up to 10 seconds for `server.ready`.
- Operation progress uses JSON-RPC notifications.
- Production does not open a local HTTP server.

Open:

- Whether to enable a debug HTTP server in dev mode.

Startup failure flow:

- If Go exits before reporting `server.ready`, show a blocking startup error
  and surface the stderr/log path.
- If `server.ready` times out, kill the child process and show a blocking error.
- Mid-session crashes may restart up to 3 times, then show a blocking error.

Recommendation:

- A dev-only debug HTTP server may be added later via `SKILLBOX_DEBUG_PORT`
  after the first JSON-RPC method is working.

## API Contracts

Status: recommended.

Choice: JSON Schema in `shared/api-contracts`.

Why:

- Human-readable contract for commands/queries.
- Can generate TypeScript types.
- Fits JSON-RPC payloads.
- Lighter than protobuf/gRPC for local IPC.

Open:

- Generate Go structs from JSON Schema or hand-write matching Go structs.
- Naming/versioning policy for command/query schemas.

Decisions:

- Commit generated TypeScript types for easier AI/human review.
- Keep Go structs hand-written in Phase 1 unless drift becomes hard to manage.
- Add contract tests to serialize sample Go responses and validate against
  schema.
- Add CI check to ensure generated TypeScript types match committed types.

## Go SQLite Stack

Status: recommended.

Choice:

```text
driver = modernc.org/sqlite
migrations = embedded SQL migrations
```

Why:

- `modernc.org/sqlite` avoids CGO and simplifies cross-platform builds.
- Embedded SQL migrations are auditable and versioned.
- SQL keeps readability for both humans and AI.

Decisions:

- Use standard OS app data directory for SQLite.
- macOS: `~/Library/Application Support/Astraler Skillbox/skillbox.db`.
- Windows: `%APPDATA%\Astraler Skillbox\skillbox.db`.
- Linux: `~/.config/astraler-skillbox/skillbox.db`.
- Dev/test override: `SKILLBOX_DB_PATH` environment variable.
- Enable WAL.
- Enable foreign keys on every connection.
- Set `busy_timeout=5000`.
- Use `synchronous=NORMAL`.
- Use `golang-migrate` with embedded SQL migrations.

Startup PRAGMAs:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

## Keychain And Credentials

Status: recommended.

Choice: Go core owns credentials through the OS keychain.

Library choice: `zalando/go-keyring`.

Why:

- Source adapters live on the Go side.
- Secrets should live in the process that uses them.
- SQLite only stores credential metadata/references, not plaintext.

Decisions:

- Use `zalando/go-keyring` in Go.
- Allow environment variable fallback for dev/CI.
- Env vars: `SKILLBOX_GITHUB_TOKEN`, `SKILLBOX_VERCEL_TOKEN`.
- Document `libsecret` requirement on Linux if a keychain library needs the
  Secret Service API.
- Do not store plaintext tokens in SQLite.

## Go Module And Dependency Policy

Status: recommended.

Rules:

- Keep Go core dependencies minimal.
- Use the standard library where reasonable.
- Only add libraries for SQLite, migrations, keychain, and JSON-RPC after
  review.
- Keep provider adapters mostly as internal code.

Suggested initial Go packages:

```text
modernc.org/sqlite
golang-migrate/migrate
zalando/go-keyring
creachadair/jrpc2
```

## Testing Stack

Status: recommended.

Frontend/Electron:

```text
Vitest
React Testing Library
Playwright
```

Go:

```text
go test
temporary SQLite database
filesystem fixtures
contract tests against JSON Schema
```

Why:

- Vitest pairs with Vite.
- Playwright can test real Electron flows later.
- Go tests can validate provider scan/install/fs behavior without the UI.

Open:

- Whether to include Playwright immediately or after the first UI shell.
- How to run full-stack tests with Electron + Go sidecar on CI.

Required:

- `go test -race` for operation runner, provider scan, JSON-RPC, and filesystem
  gateway code.
- Contract tests from the first JSON-RPC method: serialize Go responses and
  validate against JSON Schema.
- Mock-core fixtures must be validated against JSON Schema.

## Dev Workflow

Status: recommended.

Desired commands:

```text
pnpm install
pnpm dev
pnpm test
pnpm lint
pnpm build
pnpm package
go test ./...
```

Dev modes:

```text
Go-only:
  run core tests and JSON-RPC harness without Electron

UI-only:
  React app using mock core client/view models

Full-stack:
  Electron main launches Go sidecar and renderer connects through preload
```

Open:

- Whether to use `air` or another Go watcher for hot reload.
- Whether mock core client is hand-written or generated from API contracts.

Decisions:

- Support three dev modes in the scaffold README:
  - Go-only: Go tests and JSON-RPC harness without Electron.
  - UI-only: Electron/React using mock core fixture responses.
  - Full-stack: Electron main launches a real Go sidecar.
- Mock-core fixtures are captured from Go integration tests or validated against
  JSON Schema on CI.

## Security Defaults

Status: decided for scaffold.

Electron:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
preload exposes narrow API only
renderer never receives Go process path or transport details
Electron main validates JSON-RPC methods against allowlist before forwarding
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

Go:

```text
stdout = JSON-RPC protocol only
stderr/log file = logs
validate all filesystem writes
never trust paths from the renderer without validation in core
```

Packaging:

```text
Go binary bundled via electron-builder extraResources
macOS signing/notarization tested early
```

## Build Size Notes

Status: informational.

- Electron accounts for most of the app size due to bundled Chromium and Node.
- Radix/shadcn/Tailwind are not the main size risk.
- shadcn/ui copies component source; bundle size depends on what is imported.
- lucide-react should import icons individually.
- Go binary should use release flags like `-ldflags="-s -w"` when packaging.

## Decisions Before Scaffolding

Need to confirm:

- Dev debug HTTP server: yes or no.
- Mock-core fixture generation policy.

May defer:

- Auto-update behavior.
- Persistent daemon.
- Multi-window.

## Recommended Phase 1 Scaffold Decision Set

```text
workspace:
  pnpm
  single package at apps/desktop
  no pnpm workspace until a second JS package exists

desktop:
  Electron
  electron-vite
  React
  electron-builder

ui:
  shadcn/ui
  Radix UI
  Tailwind CSS
  lucide-react
  TanStack Router
  TanStack Query
  React Hook Form
  Zod
  TanStack Table (deferred)
  Zustand (deferred)

core:
  Golang
  SQLite via modernc.org/sqlite
  golang-migrate with embedded SQL migrations
  zalando/go-keyring
  no go.work until a second Go module exists

transport:
  stdio JSON-RPC 2.0
  creachadair/jrpc2
  NDJSON framing
  operation progress via JSON-RPC notifications
  server.ready handshake with 10-second timeout

sqlite:
  WAL
  foreign_keys=ON
  busy_timeout=5000
  synchronous=NORMAL
  OS app data directory
  override via SKILLBOX_DB_PATH

testing:
  Vitest
  React Testing Library
  Playwright later or after shell
  go test
  go test -race for concurrent code
  filesystem fixtures
  contract tests

runtime CLI dependencies:
  git >= 2.20 — required for plugin update checks (updateCheck.run, ADR-0001)
  absent git → service returns status='git_not_found'; app remains fully usable offline
```
