# Tech Stack And Scaffold Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-24
- Context used: README.md, docs/index.md, docs/06-data-model.md, docs/08-provider-model.md, docs/09-ui-wireframes.md, docs/10-technical-architecture.md, docs/11-tech-stack-and-scaffold-decisions.md, docs/archive/review-results/technical-architecture-brainstorm.md, docs/archive/review-results/transport-decision-brainstorm.md, tech-reviewer.md
- Browsing used: No

---

## Decision

**Approved With Changes**

The overall stack direction is sound and the architecture boundaries are well-thought-out. There are five issues that must be resolved before scaffold begins, plus several decisions still listed as "open" that will cause pain if deferred past day one of coding. The UI and Go dependency choices are appropriate. Nothing here requires a structural rethink — just closing the remaining open decisions and adding two missing SQLite PRAGMAs.

---

## Critical Issues

Issues that must be fixed before scaffold.

---

**Issue 1**

- Severity: Critical
- File/section: docs/11 → Go SQLite Stack; docs/10 → Data Access Layer
- Problem: `PRAGMA foreign_keys=ON` is missing from the SQLite startup sequence. SQLite disables foreign key enforcement by default. The data model has 15+ tables with multiple FKs (e.g., `installs.project_provider_id`, `skills.source_id`, `global_installs.global_provider_location_id`). Without this PRAGMA, orphan rows can be inserted silently.
- Why it matters: Install records can reference deleted project_providers with no error at write time. Scan reconcile logic could accumulate orphaned rows that corrupt view model queries. This is data integrity — once bad rows are in production SQLite files, migration to fix them is painful.
- Recommended fix: Add `PRAGMA foreign_keys=ON` to Go startup, immediately after opening the SQLite connection, before migrations run. It must be set on every connection, including test connections.

---

**Issue 2**

- File/section: docs/11 → JSON-RPC Protocol; docs/11 → Decisions Before Scaffold
- Severity: Critical
- Problem: JSON-RPC framing (NDJSON vs LSP-style `Content-Length`) and Go JSON-RPC library are listed as "open" but are scaffold-blocking. The folder structure of `core-go/internal/rpc/` and the Electron main reader depend on which framing is chosen. Starting scaffold with a placeholder approach and changing later means rewriting both the Go reader and the Electron main stream parser.
- Why it matters: Every other module in Go core (services, domain, repositories) depends on the RPC handler signature. If the library or framing changes mid-scaffold, handler registration code changes across the codebase.
- Recommended fix: Close both decisions before first commit.
  - **Library**: Use `creachadair/jrpc2`. It is actively maintained (unlike `sourcegraph/jsonrpc2` which is semi-abandoned since Sourcegraph migrated internally), has a clean bidirectional API, and natively supports the handler pattern the architecture needs. Custom handler is acceptable if the team wants zero dependencies, but adds test burden for framing edge cases.
  - **Framing**: NDJSON (one JSON object per line) is acceptable and simpler to implement. Go's `encoding/json` correctly escapes all literal newlines in string values, so line splitting on `\n` is unambiguous. `creachadair/jrpc2` supports both; pick NDJSON for Phase 1, `Content-Length` if the chosen library defaults to it.

---

**Issue 3**

- File/section: docs/11 → JSON-RPC Protocol → `server.ready` handshake; docs/10 → Transport Decision
- Severity: Critical
- Problem: The `server.ready` handshake is specified ("Go core sends `server.ready` before Electron forwards renderer requests") but the timeout and failure path are not. If Go crashes before emitting `server.ready`, Electron main waits indefinitely. Nothing in the stack doc specifies the timeout duration, the error UI, or the restart policy in this specific scenario.
- Why it matters: macOS `codesign` denial, binary path misconfiguration, or a Go panic at init all produce "Go never sends server.ready." The app hangs with a blank window. First-run UX on unsigned dev builds will hit this exact case.
- Recommended fix: Before scaffold, define the startup contract explicitly:
  1. Electron main waits up to 10 seconds for `server.ready` after spawning Go.
  2. If timeout: kill child, show blocking error window ("Skillbox core failed to start"), log stderr contents.
  3. On unexpected child exit before `server.ready`: same error window, no retry.
  4. This is separate from the in-session restart policy (3 attempts on mid-session crash).
  Document this in the scaffold as the Go binary startup sequence in `electron/core-process/`.

---

**Issue 4**

- File/section: docs/11 → Security Defaults; docs/11 → Decisions Before Scaffold
- Severity: Critical
- Problem: Electron security defaults (`contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`) are listed as "recommended" status, not "decided". A scaffold that initializes `BrowserWindow` with wrong defaults — which every boilerplate template gets subtly wrong — is a security regression that is easy to miss in code review.
- Why it matters: `nodeIntegration: true` in renderer gives any XSS or injected local HTML direct access to Node.js APIs and the filesystem. This is particularly dangerous for Skillbox which does filesystem writes. The brainstorm calls this "Medium risk" — I'd call it "High" for a tool with write access to user project directories.
- Recommended fix: Lock all three settings to decided status before scaffold. The scaffold first commit must have these in `BrowserWindow` options, not a TODO comment. Additionally, add to the security defaults:
  - Electron main holds a static allowlist of valid JSON-RPC method names. Renderer requests for unlisted methods are rejected before forwarding to Go core.
  - CSP header on renderer window: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`. No `unsafe-eval`.

---

**Issue 5**

- File/section: docs/11 → Go SQLite Stack → Open; docs/10 → Architecture Decisions To Confirm
- Severity: Critical
- Problem: SQLite file path is listed as "open." This cannot be decided after scaffold because the file path determines: (a) where migrations run at startup, (b) where tests create/clean temp databases, (c) what goes in the `Settings` view ("Database → Location"), and (d) what the packaging build must ensure is writable at runtime.
- Why it matters: If scaffold begins with a hardcoded `./skillbox.db` path and the correct answer is `~/Library/Application Support/Astraler Skillbox/skillbox.db`, every test, migration run, and dev script needs to change. The brainstorm already states the correct answer — it just wasn't promoted to decided in the stack doc.
- Recommended fix: Decide now:
  - macOS: `~/Library/Application Support/Astraler Skillbox/skillbox.db` via `os.UserConfigDir()`
  - Windows: `%APPDATA%\Astraler Skillbox\skillbox.db`
  - Linux: `~/.config/astraler-skillbox/skillbox.db`
  - Dev/test override: `SKILLBOX_DB_PATH` env var pointing to a temp path.
  Lock this in the Go startup code from day one.

---

## Non-Blocking Suggestions

Useful improvements that can be handled before or during scaffold.

---

**S1 — Add PRAGMA busy_timeout**

Missing from the SQLite startup sequence alongside WAL mode. Without a busy timeout, concurrent reads during a write return `SQLITE_BUSY` immediately. Go's SQLite driver surfaces this as an error that will confuse callers.

Recommended startup PRAGMAs (run after connection open, before migrations):
```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

`synchronous=NORMAL` with WAL is safe and reduces write latency versus the default `FULL`.

---

**S2 — Graceful Go sidecar shutdown sequence**

The brainstorm describes sidecar lifecycle but doesn't specify the shutdown sequence when Electron main quits or crashes. Go binary should:

1. Handle SIGTERM: flush in-progress operation state (write `status=failed` to `operations` table for any running ops), close SQLite, exit within 2 seconds.
2. Electron main sends SIGTERM on `app.on('before-quit')`, waits 3 seconds, then SIGKILL.
3. On unexpected Electron crash (no clean quit): Go detects EOF on stdin, treats it as shutdown signal, performs the same flush-and-exit.

Without this, a mid-write operation on app quit can leave SQLite in a state where the next launch finds an in-progress operation with no clear outcome.

---

**S3 — Linux keychain fallback must be in scaffold**

`zalando/go-keyring` on Linux requires `libsecret` (Secret Service API). The brainstorm notes this but the stack doc does not. If any team member develops on Linux or if CI runs on Linux, the first `go build` will reveal this dependency gap.

Scaffold must include from day one:
- An env var fallback path: `SKILLBOX_GITHUB_TOKEN`, `SKILLBOX_VERCEL_TOKEN`.
- A comment in the keychain code documenting the `libsecret` requirement.
- This also unblocks dev and CI usage without OS keychain setup.

---

**S4 — Decide Zod schema ownership**

The stack doc notes that duplicating validation between Zod, JSON Schema, and Go structs is a risk. Before scaffold, decide:

Option A (recommended): Zod schemas are written independently for UI-specific form validation. The JSON Schema in `shared/api-contracts/` covers only the wire contract (command/query shapes). Zod validates form input before calling the bridge; Go validates params on the other side. Duplication is intentional: form UX constraints (max length, allowed characters) differ from wire contract constraints.

Option B: Generate Zod schemas from JSON Schema. More complex toolchain, questionable value given that form UX constraints rarely map 1:1 to wire types.

Pick A and document it so future devs don't build a full codegen pipeline expecting B.

---

**S5 — Commit generated TypeScript types**

The stack doc leaves open: "commit generated TypeScript files or generate in CI." 

Commit them. Reasons:
- PRs show the contract drift clearly (reviewer sees when a Go struct change broke a TS type).
- AI review and code search work without running codegen.
- CI stays simple: no need to run quicktype before type-checking.
- The only downside (stale diffs in PRs) is outweighed by the above.

Add a `pnpm generate:types` command and run it as part of the Go API change workflow. Gate with a CI step that fails if generated types differ from committed types.

---

**S6 — pnpm workspace scope**

The stack doc recommends pnpm workspace, but the candidate project structure shows only one JS application (`apps/desktop`). A pnpm workspace adds value when there are multiple JS packages sharing code. With one package, it's an extra layer of indirection.

Recommendation: start with a single `package.json` at `apps/desktop/` root. Add workspace when a second JS package is introduced (e.g., a separate `packages/api-contracts` when the contract codegen toolchain is added). Do not scaffold `pnpm-workspace.yaml` on day one if there's only one package.

---

**S7 — `go.work` is premature**

`go.work` is for multi-module Go workspaces. Phase 1 has one Go module (`core-go/`). Adding `go.work` now is unnecessary and can confuse Go tooling that checks `go.work` existence for module resolution.

Remove `go.work` from the scaffold structure. Add it when (if) a second Go module is introduced.

---

**S8 — TanStack Table — defer to first sortable table**

TanStack Table is headless and verbose. For Phase 1, the Skills Library table, Projects table, and Project Detail installs list can start as simple `<table>` elements with basic sort state in React. TanStack Table becomes justified when:
- Multiple tables need shared sort/filter behavior.
- Column resizing or virtualization is needed.
- Row selection with multi-action support is required.

Recommendation: do not include TanStack Table in the scaffold. Add it when the first table screen needs sort/filter. This removes one learning curve from day one without losing anything — it's easy to introduce later.

---

**S9 — Dev mock-core mode needs a discipline plan**

The brainstorm's three dev modes (Go-only, mock-core, full-stack) are not in the stack doc. The `SKILLBOX_MOCK_CORE=true` mode where Electron main returns fixture JSON is valuable for UI development but creates a maintenance risk: fixture files can become stale relative to the real Go API.

Before scaffold, decide how fixtures stay in sync:
- Option A: Fixtures are generated by running Go integration tests and capturing stdout. Run `make capture-fixtures` to update them.
- Option B: Fixtures are hand-written and validated against JSON Schema in CI.

Without a policy, mock fixtures will drift within the first month.

---

## Decision-by-Decision Assessment

### Project structure

The proposed structure is appropriate. `apps/desktop/{electron,renderer}`, `core-go/`, `shared/api-contracts/` maps cleanly to the architecture boundary. The `fixtures/` folder at root for provider/filesystem test fixtures is correct.

Remove `go.work` (see S7). Clarify whether `apps/desktop/renderer` and `apps/desktop/electron` share a `package.json` or are separate pnpm packages. Given single-workspace recommendation (S6), they likely share one `package.json` at `apps/desktop/`.

### Boilerplate direction

The guidance to use boilerplate as reference, not source of truth, is correct. Electron + React + Go sidecar is not a common boilerplate case. Any template that doesn't have Go binary lifecycle management in Electron main will require significant customization anyway.

Recommendation: study `electron-vite` for Vite config patterns, then write the actual scaffold from scratch with the documented folder structure. Do not `npx create-electron-vite` and commit the result.

### Vite/build config

Vite is correct. The main risk — separate build targets for renderer, main, and preload — is already noted. The decision to confirm ("electron-vite style vs custom configs") should resolve to:

Use `electron-vite` (the framework, not just the template) as it handles the three-target Vite config correctly out of the box, including the `externals` and `platform: 'node'` config for main/preload that developers routinely get wrong with manual Vite configs.

### Package manager

pnpm is fine. But see S6: workspace mode should be deferred until needed. Single-package pnpm install at `apps/desktop/` is sufficient for Phase 1.

### Electron packaging

electron-builder is the right choice. The macOS signing risk is correctly flagged. One addition: this should be treated as a **day-two task** (not post-launch) — set up signing with a real Apple Developer certificate before the first internal beta distribution. Signing on a "done" binary is harder than building it into the CI pipeline from the start.

`electron-updater` can be deferred until the first release. It's not a scaffold requirement.

### UI component stack

shadcn/ui + Radix + Tailwind + lucide-react is appropriate for an operational desktop tool. The direction to avoid SaaS dashboard templates and use Skillbox-specific components is correct.

The wireframes show dense information: tables with status badges, sidebar navigation, warning banners with inline actions. Radix provides the accessible primitives (Dialog, Select, Tooltip, Switch, Popover) that Skillbox needs without a full UI library opinion. Tailwind's utility classes fit dense app UIs better than component library styles.

Confirmed appropriate for Phase 1.

### Router

TanStack Router is acceptable. The type-safe search params will be useful for Skills Library and Projects filters (the wireframes show Source, Status, Provider, Search filters that benefit from URL-state persistence).

That said, React Router v7 (data router API) would also work and has a shallower learning curve. This is the lowest-stakes open decision in the stack. Pick one and don't revisit it.

If choosing TanStack Router: do not use `createHashHistory` — use `createMemoryHistory` for Electron's `file://` protocol context. TanStack Router's default browser history doesn't work with `file://` URLs.

### Query/state

TanStack Query is the right default even for local IPC. Loading/error/stale states, operation-completion invalidation, and cache keyed by entity ID are exactly what Skillbox screens need. The risk of over-caching is real — address it by keeping query stale time short (0-30 seconds) and invalidating aggressively after command execution.

React state for dialog/form/selection state, Zustand only for long-lived cross-screen state: correct hierarchy. Zustand should not be in the scaffold on day one — add it when the first cross-screen state sharing need arises.

### Forms/validation

react-hook-form + Zod: appropriate. The forms in Skillbox are modest (path input, install mode selection, credential entry). react-hook-form avoids re-render overhead on every keystroke. Zod validation provides type-safe form error handling.

See S4 for schema ownership decision.

### Tables

See S8. TanStack Table is justified eventually but not on day one of scaffold.

### JSON-RPC details

Three open items must close before scaffold. See Critical Issue 2 for library and framing decisions.

The remaining item — dev debug HTTP server — is valuable but should be an explicit design decision, not an afterthought. Recommendation: yes, implement a secondary HTTP debug server in Go behind `SKILLBOX_DEBUG_PORT` env var. It mirrors the JSON-RPC method dispatch over HTTP GET/POST. This accelerates Go solo development without touching the production code path. Build it in the first week of Go scaffold.

### API contracts

JSON Schema in `shared/api-contracts/` with quicktype for TypeScript: appropriate for Phase 1 with ~20 commands/queries.

See S5 for commit-vs-generate decision (commit).

Go structs hand-written to match JSON Schema is correct for Phase 1. Add a CI check that runs `quicktype` and compares output to committed types to catch drift without full Go struct generation.

### SQLite/migrations

modernc.org/sqlite + golang-migrate + embedded SQL: all three choices are correct.

Missing PRAGMAs: see Critical Issue 1 (foreign_keys) and S1 (busy_timeout, synchronous=NORMAL). WAL mode must be explicit — not assumed.

Migration run-before-UI rule (block app main window if migration fails with error) is correct. Document this as the startup sequence in the Go scaffold.

Seed provider definitions via `000002_seed_providers.up.sql` migration file: correct approach. Version-controlled, auditable, runs with schema migrations.

### Keychain

zalando/go-keyring in Go core: correct. Keep credential management in the process that uses it.

See S3 for Linux libsecret fallback — this must be in scaffold from day one.

Env var fallback (`SKILLBOX_GITHUB_TOKEN`) for dev/CI: must be in Phase 1, not deferred.

### Testing

Vitest + React Testing Library + go test + filesystem fixtures: all correct.

Playwright: defer to after first UI shell. Not a scaffold requirement.

Missing from the stack doc:
- `go test -race` should be mandated for all code touching the operation runner and provider scan. Add it to CI.
- Contract tests (serialize sample Go responses, validate against JSON Schema): these are important and should be in the scaffold from the first JSON-RPC method implementation. Not a post-scaffold add.
- The three dev modes (Go-only, mock-core, full-stack) must be documented in the scaffold README. See S9.

### Security

The security defaults are correct in direction but not locked in status.

Critical: lock contextIsolation=true, nodeIntegration=false, sandbox=true before scaffold. See Critical Issue 4.

Add to security defaults (not currently in docs/11):
- Method allowlist in Electron main before forwarding to Go core.
- CSP header on renderer window.
- `SKILLBOX_MOCK_CORE=true` env var must never be accessible to renderer code.
- Electron main should not log full JSON-RPC payloads (may contain paths with user data or credential metadata).

---

## Missing Decisions

Decisions that still need confirmation before scaffold:

1. **JSON-RPC library** (Critical Issue 2): `creachadair/jrpc2` vs custom minimal handler.
2. **JSON-RPC framing** (Critical Issue 2): NDJSON vs `Content-Length`.
3. **SQLite file path per platform** (Critical Issue 5): macOS / Windows / Linux paths, test override env var.
4. **server.ready timeout value and failure path** (Critical Issue 3): timeout duration, error window behavior, log surfacing.
5. **Electron security defaults moved from recommended to decided** (Critical Issue 4).
6. **Zod schema ownership** (S4): standalone form validation vs derived from JSON Schema.
7. **Generated TS type commit policy** (S5): commit (recommended) or CI-only.
8. **pnpm workspace mode** (S6): single package vs workspace from day one.
9. **Vite config strategy** (docs/11 open): `electron-vite` framework vs custom configs.
10. **Mock-core fixture sync policy** (S9): captured from Go integration tests vs hand-written with schema validation.
11. **PRAGMA startup set** (S1 + Critical Issue 1): foreign_keys, busy_timeout, synchronous, WAL.
12. **Go sidecar graceful shutdown sequence** (S2): SIGTERM timeout, in-flight operation flush policy.

---

## Overengineering Risks

- **pnpm workspace with one package**: adds indirection without current benefit. Defer until a second JS package exists.
- **go.work with one Go module**: unnecessary. Adds confusion for `go` toolchain invocations.
- **TanStack Table on day one**: the scaffold should not require TanStack Table before a table screen is actually built. Start with plain table elements; introduce TanStack Table when sort/filter lands.
- **TanStack Router if search params are not needed immediately**: acceptable overhead but if the first three screens are simple routes without search param state, React Router v7 is simpler to teach.

None of these are blockers. They're "watch for unnecessary complexity on day one" risks.

---

## Underengineering Risks

- **`PRAGMA foreign_keys=ON` missing**: data integrity gap. See Critical Issue 1. High impact.
- **`PRAGMA busy_timeout` missing**: will produce confusing `SQLITE_BUSY` errors in concurrent read/write scenarios. See S1.
- **No explicit graceful shutdown**: Go killed mid-write leaves operations table in an ambiguous state. See S2.
- **Method allowlist not in Electron main**: without it, the preload bridge is wider than intended. A renderer script error or injected script can call any Go method name. Low probability but architectural gap.
- **Linux keychain not handled**: will break first Linux dev setup. See S3.
- **Contract tests not in scaffold**: drift between Go responses and JSON Schema is silent until a UI bug surfaces. Should be in scaffold from first method implementation.
- **`go test -race` not mandated**: operation runner uses goroutines. Race conditions in scan/install are possible and hard to find without race detection.
- **server.ready timeout not specified**: startup failure silently hangs the app. See Critical Issue 3.

---

## Recommended Scaffold Set

Final recommended Phase 1 scaffold stack with all open items resolved:

```text
workspace:
  pnpm — single package at apps/desktop/ (no workspace until needed)

desktop:
  Electron
  electron-vite (build framework)
  React
  TypeScript
  electron-builder

ui:
  shadcn/ui (component source in repo)
  Radix UI primitives
  Tailwind CSS
  lucide-react (individual icon imports)
  TanStack Router (with createMemoryHistory for file:// context)
  TanStack Query (stale time 0-30s, aggressive post-command invalidation)
  React Hook Form
  Zod (standalone form validation, not derived from JSON Schema)
  TanStack Table — DEFER to first sortable table screen

state:
  React state for local/dialog/form state
  Zustand — DEFER until first cross-screen state need

core:
  Golang (single module, no go.work)
  modernc.org/sqlite (no CGO)
  golang-migrate + embedded SQL migrations
  creachadair/jrpc2 (JSON-RPC library)
  zalando/go-keyring (+ env var fallback SKILLBOX_GITHUB_TOKEN)

sqlite startup PRAGMAs:
  PRAGMA journal_mode=WAL;
  PRAGMA foreign_keys=ON;
  PRAGMA busy_timeout=5000;
  PRAGMA synchronous=NORMAL;

transport:
  stdio JSON-RPC 2.0
  NDJSON framing (one JSON object per line)
  server.ready handshake with 10s timeout in Electron main
  operation.progress JSON-RPC notifications (server push, no polling)
  dev debug HTTP server behind SKILLBOX_DEBUG_PORT env var

electron security (all decided, not recommended):
  contextIsolation: true
  nodeIntegration: false
  sandbox: true
  preload: narrow typed bridge only
  method allowlist in Electron main before forwarding to Go
  CSP: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'

api contracts:
  JSON Schema in shared/api-contracts/
  quicktype for TypeScript codegen
  generated types committed to repo
  Go structs hand-written to match schema
  CI check: quicktype output must match committed types
  contract tests: serialize Go responses, validate against JSON Schema

sqlite file path:
  macOS: ~/Library/Application Support/Astraler Skillbox/skillbox.db
  Windows: %APPDATA%\Astraler Skillbox\skillbox.db
  Linux: ~/.config/astraler-skillbox/skillbox.db
  Dev/test override: SKILLBOX_DB_PATH env var

sidecar lifecycle:
  Electron main spawns Go with spawn() not exec()
  Go emits server.ready on stdout; Electron waits up to 10s
  In-session crash: restart up to 3 times, then blocking error
  Shutdown: Electron sends SIGTERM, waits 3s, SIGKILL
  Go handles SIGTERM: flush in-flight ops to status=failed, close SQLite, exit

testing:
  Vitest + React Testing Library (UI)
  go test + go test -race (all goroutine-touching code)
  Filesystem fixtures in fixtures/ at repo root
  Contract tests against JSON Schema (from first Go method)
  Playwright — DEFER to after first UI shell
  Three dev modes documented in scaffold README:
    - Go-only (go test + JSON pipe to stdin)
    - mock-core (SKILLBOX_MOCK_CORE=true, fixture JSON in Electron main)
    - full-stack (Electron spawns real Go binary)
  Mock-core fixtures: generated from Go integration test captures

deferred (not Phase 1):
  electron-updater / auto-update
  persistent daemon
  CLI
  multi-window
  TanStack Table (until first sortable table)
  Zustand (until cross-screen state)
  pnpm workspace (until second JS package)
  go.work (until second Go module)
  Playwright (until UI shell exists)
  Install Skill To Global Location
  Custom provider UI
  Phase 2: unix socket transport migration
```
