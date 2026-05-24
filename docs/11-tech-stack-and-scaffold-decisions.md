# Tech Stack And Scaffold Decisions

Tài liệu này gom các quyết định tech stack và scaffold trước khi bắt đầu tạo
codebase thật. Mục tiêu là tiết kiệm thời gian bằng cách chọn framework/library
phù hợp, nhưng vẫn giữ architecture boundary đã chốt trong
`10-technical-architecture.md`.

Decision status:

```text
decided       = đã chốt, dùng làm constraint khi scaffold
recommended   = đề xuất mặc định, cần review/chốt trước khi code
open          = còn cần thảo luận hoặc spike nhỏ
defer         = chưa cần ở Phase 1
```

## Stack Summary

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

Go core lifecycle:
  status = decided for Phase 1
  choice = sidecar process managed by Electron main
```

## Scaffold Principles

- Không clone nguyên boilerplate nếu boilerplate làm mờ boundary của Skillbox.
- Dùng boilerplate để học packaging, Electron security, Vite config, test setup,
  và folder conventions.
- Scaffold phải phản ánh architecture boundary:
  - React renderer không đụng filesystem/database.
  - Electron main chỉ quản lý lifecycle/bridge/native dialogs.
  - Golang core giữ business logic, SQLite, provider adapters, filesystem
    gateway, source integrations, operation runner.
- Ưu tiên structure dễ review bởi AI và người: folder rõ, file nhỏ, contract rõ.
- Không đưa library lớn nếu chưa có screen/use case cần nó.

## Candidate Project Structure

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

- `apps/desktop` gom Electron + React app.
- `core-go` là Go module riêng, có thể build/test độc lập.
- `shared/api-contracts` là nơi giữ JSON Schema hoặc protocol contract.
- `fixtures` phục vụ provider/filesystem scan tests.
- `scripts` chứa build/dev/release helpers.

Open:

- Có cần `apps/desktop/renderer` hay giữ `ui/` ở root cho ngắn hơn.

Decided:

- Không dùng `go.work` ở Phase 1 vì chỉ có một Go module.
- Không dùng pnpm workspace ở Phase 1 nếu chỉ có một JS package.
- Generated TypeScript types được commit vào repo và CI kiểm tra drift.

## Boilerplate Research Direction

Không chọn boilerplate cuối cùng trong tài liệu này, nhưng khi khảo sát nên dùng
criteria sau:

```text
Electron security:
  contextIsolation = true
  nodeIntegration = false
  preload bridge narrow
  no renderer filesystem access

Build/dev:
  Vite for renderer
  fast HMR
  Electron main/preload build supported
  external binary packaging supported

Packaging:
  electron-builder support
  extraResources support for Go binary
  macOS signing/notarization path clear

Testing:
  Vitest or equivalent for UI/core TS
  Playwright for desktop/e2e possible
  Go test independent

Maintainability:
  simple folder structure
  no overgrown SaaS/dashboard template assumptions
```

References to evaluate:

- `electron-vite-react`
- `vite-electron-builder`
- `electron-react-boilerplate`
- official Electron Forge Vite template

Recommendation:

- Use a Vite/Electron boilerplate as reference, not as unquestioned source of
  truth.
- Scaffold our own structure if the template conflicts with Go sidecar,
  JSON-RPC, or security boundaries.

## Frontend Build Tool

Status: recommended.

Choice: Vite.

Why:

- Fast React dev server and HMR.
- Common in modern Electron + React scaffolds.
- Good production bundling for renderer.
- Works well with Tailwind, shadcn/ui, Vitest.

Risks:

- Electron main/preload build needs clear config so Node/Electron APIs are not
  bundled incorrectly.
- Need separate configs or build targets for renderer, main, and preload.

Decision:

- Use `electron-vite` to manage renderer, main, and preload Vite targets.

## Package Manager

Status: recommended.

Choice: pnpm, single JS package at `apps/desktop`.

Why:

- Fast install.
- Deterministic lockfile.
- Works well for Electron development without requiring workspace mode.

Risk:

- Some Electron tooling docs default to npm/yarn, so commands must be documented
  clearly.

Decision:

- Do not scaffold `pnpm-workspace.yaml` on day one.
- Add pnpm workspace only when a second JS package exists.

## Electron Packaging

Status: recommended.

Choice: electron-builder.

Why:

- Mature packaging for Electron apps.
- Supports `extraResources` for bundled Go binary.
- Strong macOS signing/notarization support.
- Pairs with `electron-updater` if auto-update is added.

Risks:

- macOS signing and notarization are high-risk and should be tested early.
- Go binary must be included, signed, and launched from production resource path.

Decision:

- Use `electron-builder` rather than Electron Forge.
- Plan signing/notarization as a technical milestone, not a late release task.
- Defer `electron-updater` until release/update flow is needed.

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

- Radix gives accessible primitives for dialogs, menus, tabs, popovers, tooltips,
  select, switch, checkbox, toast, scroll area, and more.
- shadcn/ui provides styled component source that can live in the repo and be
  customized.
- Tailwind keeps styling local and fast for app UI.
- lucide-react provides consistent icons and matches the design direction.

Risks:

- shadcn components are copied into the repo, so ownership is ours.
- Tailwind can become messy if layout conventions are not documented.
- Need restraint: do not add large block/template collections blindly.

Decision to confirm:

- Use shadcn/ui as component source, not a fully prebuilt dashboard template.
- Create Skillbox-specific app shell, sidebar, toolbar, table, warning, and
  status components instead of using a generic SaaS template wholesale.

## UI App Style

Status: recommended.

Skillbox should feel like an operational desktop tool:

- Dense but readable.
- Sidebar navigation.
- Tables/lists for skills, projects, global locations, updates.
- Detail panes for selected entities.
- Clear status badges and warnings.
- Minimal marketing/hero styling.
- Functional dialogs and wizards.

Avoid:

- Landing-page layout.
- Oversized hero sections.
- Decorative cards inside cards.
- Heavy gradients/illustrations.
- Generic SaaS dashboard template that hides filesystem/provider details.

## Router

Status: decided for Phase 1.

Choice: TanStack Router.

Why:

- Type-safe route definitions.
- Good fit for app screens with nested detail routes.
- Stronger than React Router when route params/search state become important.

Risks:

- Slightly more learning/setup than React Router.
- Need to keep route model simple because this is a desktop app, not a public
  web app.

Alternative:

- React Router if team wants a simpler, widely-known router.

Decision:

- Use TanStack Router with `createMemoryHistory` for Electron/file URL context.

## Server State And View Models

Status: decided for Phase 1.

Choice: TanStack Query for local JSON-RPC queries.

Why:

- Even though data is local, screens still need loading/error/refetch/cache.
- Operation completion can invalidate relevant queries.
- Keeps React UI from manually managing every request lifecycle.

Rules:

- Queries call the Electron preload bridge client, not Go directly.
- Mutations call commands and return `operation_id` when relevant.
- UI re-fetches view models after operation completion.

Risks:

- Over-caching can show stale filesystem state after scan/update.
- Query keys must be disciplined.

Decision:

- Use TanStack Query from day one.
- Keep stale time short and invalidate aggressively after command/operation
  completion.

## Client UI State

Status: recommended.

Choice: React state first; Zustand deferred until cross-screen ephemeral state is
actually needed.

Use React state for:

- Dialog open/close.
- Current form values.
- Local selections.

Use Zustand only if needed for:

- App shell UI state.
- Selected project/skill context shared across multiple panels.
- Long-lived operation panel state not owned by one screen.

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
- React Hook Form avoids excessive controlled-input re-rendering.

Rules:

- Zod schemas are UI/form validation schemas.
- JSON Schema in `shared/api-contracts` is wire contract validation.
- Go validates command/query params independently on the core side.

Risk:

- Some validation is intentionally duplicated because user-facing form
  constraints and wire contract constraints are not always identical.

## Tables

Status: defer.

Choice: start with simple table components; add TanStack Table when the first
sortable/filterable table screen proves it needs it.

Why:

- Skillbox has many table-heavy screens:
  - Skills Library
  - Global Skills
  - Projects
  - Project Detail installs
  - Updates affected projects/global installs
- Sorting/filtering/selection will be common.

Risks:

- TanStack Table is headless and can be verbose.
- Need shared table components to avoid repeating setup.

Decision:

- Do not include TanStack Table in the initial scaffold.

## JSON-RPC Protocol

Status: partially decided.

Decided:

- Phase 1 transport is stdio JSON-RPC 2.0.
- JSON-RPC Go library is `creachadair/jrpc2`.
- Framing is NDJSON.
- Go core sends `server.ready` before Electron forwards renderer requests.
- Electron main waits up to 10 seconds for `server.ready`.
- Operation progress uses JSON-RPC notifications.
- Production does not open local HTTP server.

Open:

- Whether dev mode includes a debug HTTP server.

Startup failure path:

- If Go exits before `server.ready`, show blocking startup error and surface
  stderr/log path.
- If `server.ready` timeout fires, kill child and show blocking startup error.
- Mid-session crash can restart up to 3 times, then show blocking error.

Recommendation:

- Dev-only debug HTTP server can be added behind `SKILLBOX_DEBUG_PORT` after the
  first JSON-RPC method works.

## API Contracts

Status: recommended.

Choice: JSON Schema in `shared/api-contracts`.

Why:

- Human-readable contract for commands/queries.
- Can generate TypeScript types.
- Fits JSON-RPC payloads.
- Less heavy than protobuf/gRPC for internal local IPC.

Open:

- Generate Go structs from JSON Schema or hand-match Go structs.
- Naming/versioning convention for command/query schemas.

Decision:

- Commit generated TypeScript types for easier AI/code review.
- Keep Go structs hand-written in Phase 1 unless drift becomes painful.
- Add contract tests that serialize sample Go responses and validate against
  schemas.
- Add CI check that generated TypeScript types match committed types.

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
- SQL remains readable to humans and AI.

Decision:

- Use OS-standard app data directory for SQLite.
- macOS: `~/Library/Application Support/Astraler Skillbox/skillbox.db`.
- Windows: `%APPDATA%\Astraler Skillbox\skillbox.db`.
- Linux: `~/.config/astraler-skillbox/skillbox.db`.
- Dev/test override: `SKILLBOX_DB_PATH`.
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

Choice: Go core owns credentials via OS keychain.

Choice: `zalando/go-keyring`.

Why:

- Source adapters live in Go.
- Secret should stay in the process that uses it.
- SQLite stores credential metadata/ref only, not plaintext.

Decision:

- Use `zalando/go-keyring` in Go.
- Allow environment variable fallback for dev/CI.
- Env vars: `SKILLBOX_GITHUB_TOKEN`, `SKILLBOX_VERCEL_TOKEN`.
- Document Linux `libsecret` requirement if using a keychain library that needs
  Secret Service API.
- Do not store plaintext token in SQLite.

## Go Module And Dependency Policy

Status: recommended.

Rules:

- Keep Go core dependency-light.
- Use standard library where reasonable.
- Use libraries for SQLite, migrations, keychain, and JSON-RPC only after
  review.
- Keep provider adapters mostly internal code.

Recommended initial Go packages:

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

- Vitest fits Vite.
- Playwright can test real Electron flows later.
- Go tests can validate provider scan/install/fs behavior without UI.

Open:

- Whether Playwright is introduced immediately or after first UI shell.
- How to run full-stack tests with Electron + Go sidecar in CI.

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
  React app uses mock core client/view models

Full-stack:
  Electron main launches Go sidecar and renderer connects through preload
```

Open:

- Use `air` or another Go watcher for hot reload.
- Whether mock core client is hand-written or generated from API contracts.

Decision:

- Support three dev modes in scaffold README:
  - Go-only: Go tests and JSON-RPC harness without Electron.
  - UI-only: Electron/React uses mock core fixture responses.
  - Full-stack: Electron main launches real Go sidecar.
- Mock-core fixtures are generated from Go integration test captures or
  validated against JSON Schema in CI.

## Security Defaults

Status: decided for scaffold.

Electron:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
preload exposes narrow API only
renderer never receives Go process path or transport details
Electron main validates JSON-RPC method allowlist before forwarding
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

Go:

```text
stdout = JSON-RPC protocol only
stderr/log file = logs
validate all filesystem writes
never trust renderer-provided paths without core validation
```

Packaging:

```text
Go binary bundled through electron-builder extraResources
macOS signing/notarization tested early
```

## Build Size Notes

Status: informational.

- Electron dominates app size because Chromium and Node are bundled.
- Radix/shadcn/Tailwind are not the main size risk.
- shadcn/ui copies component source; bundle size depends on what is imported.
- lucide-react should import icons individually.
- Go binary should use release flags such as `-ldflags="-s -w"` when packaging.

## Decisions Before Scaffold

Must decide:

- Dev debug HTTP server yes/no.
- Mock-core fixture generation policy.

Can defer:

- Auto-update behavior.
- Persistent daemon.
- CLI.
- Multi-window.
- Install Skill To Global Location.
- Custom provider UI.

## Recommended Phase 1 Scaffold Decision Set

```text
workspace:
  pnpm
  single package at apps/desktop
  no pnpm workspace until second JS package

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
  TanStack Table deferred
  Zustand deferred

core:
  Golang
  SQLite via modernc.org/sqlite
  golang-migrate with embedded SQL migrations
  zalando/go-keyring
  no go.work until second Go module

transport:
  stdio JSON-RPC 2.0
  creachadair/jrpc2
  NDJSON framing
  operation progress via JSON-RPC notifications
  server.ready handshake with 10 second timeout

sqlite:
  WAL
  foreign_keys=ON
  busy_timeout=5000
  synchronous=NORMAL
  OS app data directory
  SKILLBOX_DB_PATH override

testing:
  Vitest
  React Testing Library
  Playwright later or after shell
  go test
  go test -race for concurrent code
  filesystem fixtures
  contract tests
```
