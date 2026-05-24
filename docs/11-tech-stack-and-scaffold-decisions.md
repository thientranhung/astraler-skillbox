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
  package.json
  pnpm-lock.yaml
  go.work
  README.md
  docs/

  apps/
    desktop/
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
- `go.work` có cần ngay không nếu Phase 1 chỉ có một Go module.
- `shared/generated` có commit generated TypeScript types hay generate trong CI.

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

Decision to confirm:

- Use `electron-vite` style integrated config or custom Vite configs.

## Package Manager

Status: recommended.

Choice: pnpm.

Why:

- Fast install.
- Good workspace support.
- Deterministic lockfile.
- Works well for Electron monorepo-ish structure.

Risk:

- Some Electron tooling docs default to npm/yarn, so commands must be documented
  clearly.

Decision to confirm:

- Use pnpm workspace from day one.

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

Decision to confirm:

- Use `electron-builder` rather than Electron Forge.
- Plan signing/notarization as a technical milestone, not a late release task.

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

Status: recommended.

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

Decision to confirm:

- TanStack Router vs React Router.

## Server State And View Models

Status: recommended.

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

Decision to confirm:

- Use TanStack Query from day one, or start with a thin custom query layer and
  add TanStack Query when screens grow.

## Client UI State

Status: recommended.

Choice: React state first; Zustand only for cross-screen ephemeral state.

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

Status: recommended.

Choice:

```text
react-hook-form
zod
```

Why:

- Settings and wizard flows need clear validation.
- Zod schemas can mirror API contract validation.
- React Hook Form avoids excessive controlled-input re-rendering.

Risks:

- Duplicating validation rules between Zod, JSON Schema, and Go structs.

Decision to confirm:

- Whether UI validation schemas are derived from shared API contracts or written
  separately for user-facing form constraints.

## Tables

Status: recommended.

Choice: TanStack Table.

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

Decision to confirm:

- Use TanStack Table from first table screen, or start with simple tables and
  introduce it when sorting/filtering lands.

## JSON-RPC Protocol

Status: partially decided.

Decided:

- Phase 1 transport is stdio JSON-RPC 2.0.
- Go core sends `server.ready` before Electron forwards renderer requests.
- Operation progress uses JSON-RPC notifications.
- Production does not open local HTTP server.

Open:

- Framing: NDJSON vs LSP-style `Content-Length`.
- Go JSON-RPC library: `sourcegraph/jsonrpc2`, `creachadair/jrpc2`, or custom
  minimal handler.
- Whether dev mode includes a debug HTTP server.

Recommendation:

- Prefer LSP-style `Content-Length` framing if using a mature JSON-RPC library
  that already supports it.
- Prefer custom NDJSON only if the team wants minimal implementation and accepts
  stdout discipline/testing.
- Do not build a production HTTP API in Phase 1.

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
- Commit generated TypeScript files or generate in CI.
- Naming/versioning convention for command/query schemas.

Recommendation:

- Commit generated TypeScript types for easier AI/code review.
- Keep Go structs hand-written in Phase 1 unless drift becomes painful.
- Add contract tests that serialize sample Go responses and validate against
  schemas.

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

Open:

- Migration library: `golang-migrate` vs lightweight custom runner.
- SQLite file location.
- WAL mode policy.
- Busy timeout and locking policy.

Recommendation:

- Use OS-standard app data directory for SQLite.
- Enable WAL unless packaging/platform tests show issues.
- Use embedded SQL migrations.

## Keychain And Credentials

Status: recommended.

Choice: Go core owns credentials via OS keychain.

Candidates:

- `zalando/go-keyring`
- `99designs/keyring`

Why:

- Source adapters live in Go.
- Secret should stay in the process that uses it.
- SQLite stores credential metadata/ref only, not plaintext.

Open:

- Exact Go keychain library.
- Fallback when keychain unavailable.
- Whether env var fallback is Phase 1.

Recommendation:

- Start with keychain library in Go.
- Allow environment variable fallback for dev/CI.
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
golang-migrate/migrate or custom embedded runner
zalando/go-keyring or 99designs/keyring
JSON-RPC library TBD
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

## Security Defaults

Status: recommended.

Electron:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
preload exposes narrow API only
renderer never receives Go process path or transport details
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

- Vite config strategy: electron-vite style vs custom configs.
- Package manager: pnpm yes/no.
- Packaging: electron-builder yes/no.
- UI stack: shadcn/ui + Radix + Tailwind + lucide-react yes/no.
- Router: TanStack Router vs React Router.
- Server state: TanStack Query now vs later.
- JSON-RPC framing: NDJSON vs `Content-Length`.
- JSON-RPC Go library: existing library vs custom.
- SQLite migration approach: golang-migrate vs custom embedded runner.
- Keychain library: `zalando/go-keyring` vs `99designs/keyring`.

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
  pnpm workspace

desktop:
  Electron
  Vite
  React
  electron-builder

ui:
  shadcn/ui
  Radix UI
  Tailwind CSS
  lucide-react
  TanStack Router
  TanStack Query
  TanStack Table
  React Hook Form
  Zod

core:
  Golang
  SQLite via modernc.org/sqlite
  embedded SQL migrations
  OS keychain via Go library

transport:
  stdio JSON-RPC 2.0
  operation progress via JSON-RPC notifications
  server.ready handshake

testing:
  Vitest
  React Testing Library
  Playwright later or after shell
  go test
  filesystem fixtures
  contract tests
```
