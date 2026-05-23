# Technical Architecture Brainstorm

## Reviewer

- Agent: Agent Tech (Claude Sonnet 4.6)
- Date: 2026-05-23
- Context: README, docs/06, docs/08, docs/09, docs/10

---

## Status of docs/10

docs/10 is well-structured and correctly hedges most decisions. The open
questions at the bottom are the right ones. The doc does not over-assert on IPC
transport, SQLite library, or keychain — these are explicitly left open.

Two places where docs/10 is slightly over-asserting:

1. **Module/folder structure** is spelled out in detail before the IPC transport
   decision. The shape of `electron/core-process/` depends on the transport
   choice. Don't lock the folder structure in code until transport is resolved.

2. **"Long-running command must return `operation_id` to let UI subscribe/poll
   progress."** The word "poll" implies a pull model. If we use JSON-RPC
   notifications (server-push), polling is unnecessary. The doc should not
   imply polling as the primary progress mechanism.

Everything else in docs/10 is conceptually correct and safe to keep.

---

## Decision 1: Electron ↔ Go Transport

### Options

| Option | Description | Complexity |
|---|---|---|
| stdio JSON-RPC 2.0 | Go reads stdin, writes stdout. Electron main sends/receives via child process streams. | Low |
| Unix socket + JSON-RPC | Go creates socket at temp path, Electron connects via `net.Socket`. | Medium |
| Local HTTP REST | Go listens on random localhost port. Electron uses fetch/axios. | Medium |
| gRPC | Protobuf over HTTP/2. Requires codegen. | High |

### Recommendation

**stdio JSON-RPC 2.0** for Phase 1.

- No port management. No port conflicts. No address discovery mechanism.
- Electron main spawns `core-go` as child process, pipes stdin/stdout.
- JSON-RPC 2.0 supports both request/response and server notifications
  (notification = message without `id` field). Go uses notifications for
  progress events.
- This is the same pattern used by all LSP servers (gopls, typescript-language-server),
  so it's battle-tested in Electron contexts.
- Easy to test Go side in isolation: pipe JSON lines to stdin, read stdout.

Example Go notification for operation progress:

```json
{"jsonrpc":"2.0","method":"operation.progress","params":{"operation_id":"abc123","percent":45,"message":"Scanning .agents/skills"}}
```

### Tradeoffs

- Stdout is reserved for JSON-RPC. Go binary cannot use `fmt.Println` for debug;
  use stderr or a log file instead. This is a discipline requirement for the team.
- If multiple UI windows ever need independent connections, stdio becomes awkward
  (Phase 2 problem, not Phase 1).

### Risk

Low for Phase 1. If Phase 2 needs multi-window or CLI reuse over the same
running core, migrate to unix socket. The JSON-RPC protocol stays the same;
only the transport changes.

### Questions to resolve before coding

- Does the team prefer an existing Go JSON-RPC server library
  (`sourcegraph/jsonrpc2`, `creachadair/jrpc2`) or a minimal custom handler?
- Who writes the Electron-side JSON-RPC client? This is ~100 lines of code
  around `child_process.spawn`. Is it hand-written or using a library?

---

## Decision 2: Go Core Lifecycle — Sidecar vs Daemon

### Options

| Option | Description |
|---|---|
| Sidecar | Electron main spawns Go at app launch, kills it when app exits. |
| Persistent daemon | Go runs as OS service (launchd/Windows Service), Electron connects on open. |
| User-space daemon | Go runs as background process in user session, not a system service. |

### Recommendation

**Sidecar** for Phase 1.

- Match Go lifecycle to app window lifecycle. Simple.
- No OS service registration. No launchd plist. No Windows Service installer.
- When Electron quits, kill the child process. On crash recovery, Electron can
  restart the child automatically.
- Phase 1 has no background operations (user triggers everything). Sidecar is
  sufficient.

Electron main should:
1. Spawn `core-go` with `spawn()`, not `exec()` (streams, not buffer).
2. Attach `process.on('exit')` to kill the child.
3. Monitor child stdout for JSON-RPC messages.
4. Monitor child stderr for log lines (write to log file, show in dev mode).
5. Restart child if it exits unexpectedly during app session (max 3 attempts,
   then show blocking error).

### Tradeoffs

If the product adds background fetch (auto-check for updates on a schedule),
the sidecar architecture means the background task only runs while the app is
open. A daemon would enable true background operation.

### Risk

Medium for Phase 2. If background auto-fetch becomes a product requirement,
migrating from sidecar to daemon requires OS service integration work. Make the
decision explicit in Phase 2 planning rather than retrofitting.

### Questions to resolve

- Is background auto-fetch (without app open) a Phase 2 requirement?
- What is the restart policy if Go crashes? (Max restarts, show error, or
  keep trying silently?)

---

## Decision 3: Operation Progress, Cancel, Retry

### Current state in docs/10

docs/10 says operations emit progress to UI but does not specify the mechanism.
The text "subscribe/poll" implies ambiguity.

### Recommendation

**JSON-RPC notifications for progress (server push, no polling).**

Go sends `operation.progress` notifications to Electron main via stdout.
Electron main forwards to renderer via `contextBridge` event emitter.

Event shape:

```json
{
  "jsonrpc": "2.0",
  "method": "operation.progress",
  "params": {
    "operation_id": "abc123",
    "status": "running",
    "percent": 60,
    "message": "Classifying install mode: adr-helper"
  }
}
```

UI subscribes to `operation.progress` events scoped by `operation_id`.
When status becomes `success` or `failed`, UI re-fetches the affected view model.

**Cancel:**
UI sends `operation.cancel` command with `operation_id`. Go uses
`context.WithCancel`. Operations check `ctx.Done()` at natural checkpoints
(after each entry processed, before each filesystem write). This is idiomatic Go.

**Retry:**
Retry is a new command from UI, not automatic inside Go. After a failed
operation, UI shows "Retry" button which sends the original command again. Go
does not auto-retry. This keeps retry logic visible to the user.

**Locking:**
docs/10 already specifies one-operation-per-target locking. Implementation:
Go maintains an in-memory map of `target_type+target_id → running operation_id`.
Before starting, check map. If conflict, return `conflict_error` with the
running operation_id so UI can surface "already scanning" state.

### Risk

Medium. Progress delivery via JSON-RPC notifications requires the Electron main
process to correctly parse and forward every notification message. Any bug in
that relay silently drops progress events. Write integration test that verifies
progress events reach renderer for a scan operation.

### Questions to resolve

- Should the UI optimistically show progress (update view immediately on
  notification) or wait for operation completion and re-fetch?
  Optimistic is better UX but more complex state management.
- Minimum progress granularity: per-entry, per-phase, or percent-only?

---

## Decision 4: API Contract / Codegen Between TS and Go

### Current state

docs/10 specifies `shared/api-contracts/` folder but doesn't define what lives
there or how Go and TypeScript types stay in sync.

### Options

| Option | Source of truth | Tooling |
|---|---|---|
| Manual types both sides | None | Discipline |
| JSON Schema files | `shared/api-contracts/*.json` | `quicktype` for TS, hand-matched Go structs |
| OpenAPI spec | `shared/api-contracts/openapi.yaml` | `oapi-codegen` for Go, `openapi-typescript` for TS |
| Protobuf / gRPC | `.proto` files | `protoc` codegen for both sides |
| Go structs → TS via `tygo` or similar | Go | Go struct tags generate TS |

### Recommendation

**JSON Schema in `shared/api-contracts/` with `quicktype` for TypeScript codegen.**

- Write each command/query request and response shape as a JSON Schema file.
- Generate TypeScript types: `quicktype --lang ts --src *.json`
- Go structs are hand-written to match the schema (or use `go-jsonschema` to
  generate Go structs from JSON Schema).
- One source of truth: the JSON Schema. Both Go and TS derive from it.
- Low toolchain complexity compared to protobuf/gRPC.

Why not protobuf: overkill for a local stdio JSON-RPC API. Adds protobuf
compiler dependency. JSON Schema is readable and auditable without tooling.

Why not manual types: as the API grows (currently ~20 commands/queries), drift
between Go and TS types will cause hard-to-debug serialization mismatches.

### Tradeoffs

JSON Schema codegen is imperfect for complex Go types (interfaces, embedded
structs). Keep API types simple (flat structs, no interface fields) to keep
codegen clean.

### Risk

Low for Phase 1 with ~20 commands. The JSON Schema approach scales to ~50
endpoints easily. If Phase 2 adds CLI reuse or a public API, migrate to OpenAPI.

### Questions to resolve

- Who owns the JSON Schema files — Go team or TypeScript team or shared?
- Is codegen a build step (run `quicktype` in CI) or a commit-generated file?
  (Commit generated files: simpler CI. Generate in CI: no stale diffs in repo.)

---

## Decision 5: SQLite Library and Migrations in Go

### Library options

| Library | CGO | Notes |
|---|---|---|
| `mattn/go-sqlite3` | Yes | Most mature, best perf, requires C compiler |
| `modernc.org/sqlite` | No | Pure Go SQLite port, 10-20% slower, no C deps |
| `zombiezen.com/go/sqlite` | Yes | Low-level API, good for advanced use |

### Recommendation

**`modernc.org/sqlite`** for Phase 1.

- No CGO means no C compiler in CI pipeline. Simpler cross-compilation.
  Especially important when building Go binary for macOS arm64 + amd64 fat
  binary inside Electron app.
- Performance difference is negligible for Skillbox's query patterns
  (dozens of rows per query, not millions).

Revisit if: query volume grows significantly or WAL performance becomes an issue.

### Migration tool

**`golang-migrate/migrate`** with embedded SQL files.

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

- SQL-based migration files: readable and auditable without Go knowledge.
- Supports up/down migrations, version tracking in `schema_migrations` table.
- Migrations run at app startup before UI opens.
- On migration failure: show blocking error, do not open app main window.

Seeding provider definitions:
**Embed as a seed migration file** (e.g., `000002_seed_providers.up.sql`).
This keeps seed data version-controlled alongside schema. Avoids the question
of "code seed vs JSON seed vs migration seed."

### Risk

Low. Both library choices are well-maintained. Migration approach is standard.

### Questions to resolve

- WAL mode vs default journal mode? WAL allows concurrent reads during writes,
  useful if Go background tasks read while UI queries. Recommend enabling WAL.
- What is the SQLite file path? App support directory (`~/Library/Application Support/Astraler Skillbox/`) vs next to binary? Use OS-standard app data path.

---

## Decision 6: OS Keychain Integration

### Options

| Library | Platform coverage | Notes |
|---|---|---|
| `zalando/go-keyring` | macOS, Windows, Linux | Pure Go, uses OS native APIs |
| `99designs/keyring` | macOS, Windows, Linux | More backends, larger API surface |
| Electron `safeStorage` | All | Encrypt in Electron process, store ciphertext in SQLite |
| None (env vars only) | All | Users set `GITHUB_TOKEN` env var |

### Recommendation

**`zalando/go-keyring`** for GitHub/Vercel tokens.

- Integrates directly in Go core, not Electron. Keeps credential management in
  the same process that uses them (source adapters in Go).
- On macOS: Keychain Access, on Windows: Credential Manager.
- `api_credentials.credential_ref` stores the keychain service/account key.
  Actual secret never touches SQLite.
- Simpler than Electron `safeStorage` because the credential is already where
  it's needed: Go source adapters.

Linux note: requires `libsecret` (Secret Service API). If Linux support is
planned, document as a system dependency. Alternative: fall back to encrypted
SQLite value on Linux if libsecret is unavailable.

### Tradeoffs

Keychain access on macOS requires the app to be code-signed. This is expected
for a shipping desktop app, but it means dev builds will prompt for keychain
access unless added to the allowed apps list.

### Questions to resolve

- Is there a Phase 1 fallback if keychain is unavailable (e.g., on CI, test
  machines, or Linux without libsecret)? Suggest: env var fallback
  (`SKILLBOX_GITHUB_TOKEN`), never store plaintext in SQLite.
- Should GitHub/Vercel token validation happen in Go at startup (check
  credential is still valid) or only on demand (first fetch)?

---

## Decision 7: Packaging and Updater

### Go binary in Electron package

The Go binary must be:
1. Built for target platform (macOS arm64, macOS x86_64, Windows x64).
2. Placed in `extraResources` in the Electron build.
3. Launched from `process.resourcesPath` in Electron main.

On macOS: Go binary must be **code-signed** (Hardened Runtime) and
**notarized** as part of the app package. This is not optional for macOS 10.15+.

**electron-builder** configuration:

```json
{
  "extraResources": [
    { "from": "dist/core-go", "to": "core-go" }
  ],
  "mac": {
    "hardenedRuntime": true,
    "entitlements": "entitlements.mac.plist"
  }
}
```

### Recommendation

**`electron-builder` + `electron-updater`** for packaging and auto-update.

- `electron-builder` is the most mature Electron packager with the best
  support for bundled binaries, code signing, and notarization.
- `electron-updater` (part of the same ecosystem) handles differential app
  updates. When the app updates, both the Electron app and the bundled Go
  binary update together.
- Auto-update server: GitHub Releases for Phase 1. Simple, free, no server
  needed.

CI pipeline shape:

```text
1. go build -o dist/core-go -ldflags="-s -w" ./cmd/skillbox-core
2. npm run build (React → dist/ui)
3. electron-builder --mac --publish never (for local testing)
4. electron-builder --mac --publish always (for release CI)
   -> signs + notarizes go binary and app bundle
   -> uploads to GitHub Releases
```

### Risk

**Code signing and notarization are the highest-risk packaging item.** Apple
notarization requires an Apple Developer account, a macOS build agent (can't
notarize on Linux), and specific entitlements. The Go binary needs Hardened
Runtime entitlements for keychain access.

Plan this early. It's not a Phase 2 problem — unsigned apps won't launch on
recent macOS.

### Questions to resolve

- Apple Developer account: personal or organization?
- Will CI run on a macOS agent? (GitHub Actions macOS runners are available
  but slower/more expensive.)
- Auto-update: opt-in or opt-out by default? Recommend opt-out for first
  release, opt-in prompt after.

---

## Decision 8: Security Boundaries (Electron)

### Current state

docs/10 mentions "do not send local file content to external service" and
"prefer OS keychain" but doesn't specify Electron-level security configuration.

### Recommendation

**Strict Electron security configuration:**

```javascript
// electron/main: BrowserWindow options
webPreferences: {
  contextIsolation: true,      // required
  nodeIntegration: false,       // required
  sandbox: true,                // recommended
  preload: path.join(__dirname, 'preload.js')
}
```

**Preload script exposes only a narrow bridge:**

```typescript
contextBridge.exposeInMainWorld('skillbox', {
  invoke: (method: string, params: unknown) =>
    ipcRenderer.invoke('skillbox:invoke', method, params),
  onEvent: (eventName: string, callback: (data: unknown) => void) => {
    ipcRenderer.on(`skillbox:event:${eventName}`, (_e, data) => callback(data));
    return () => ipcRenderer.removeAllListeners(`skillbox:event:${eventName}`);
  }
});
```

Renderer can only call `window.skillbox.invoke(method, params)` and subscribe
to events. No direct access to Node.js, filesystem, or ipcRenderer.

**Content Security Policy on renderer window:**

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

No `unsafe-eval`. No external script sources.

**Electron main is the sole trust boundary:**
- Validates method names before forwarding to Go core (allowlist check).
- Sanitizes responses before sending to renderer.
- Does not log full request/response payloads (may contain paths with user data).

### Risk

Medium. Forgetting `contextIsolation: true` or accidentally enabling
`nodeIntegration` in renderer opens the app to XSS-to-RCE via local HTML
injection. Enforce this in code review checklist.

### Questions to resolve

- Should the method allowlist be static (hardcoded in main) or dynamic (fetched
  from Go core on startup)?
- Should the renderer window load from `file://` protocol or a local
  `http://localhost` dev server in production? (Production: always `file://`.)

---

## Decision 9: Dev Ergonomics and Test Strategy

### Gaps in docs/10 testing strategy

docs/10 lists test fixtures but doesn't address:
- How UI dev runs without a live Go binary.
- How Go tests run without Electron.
- How end-to-end tests work.

### Recommendation

**Three independent development modes:**

**Mode 1 — Go core standalone:**
`core-go` can be tested entirely with `go test`. No Electron needed. Fixture
directories checked into `testdata/`. The JSON-RPC server can be exercised by
piping JSON lines to stdin in a test harness.

**Mode 2 — UI with mock core:**
In dev mode, Electron main intercepts all `skillbox:invoke` IPC calls and
returns fixture JSON from `electron/mock-responses/*.json`. This lets React
development proceed without a running Go binary. Toggle via env var
(`SKILLBOX_MOCK_CORE=true`).

**Mode 3 — Full stack:**
Electron main spawns real Go binary. Used for integration testing and final
QA before releases.

**Test layer map:**

| Layer | Tool | What it tests |
|---|---|---|
| Go domain | `go test` | Install mode classification, path validation, warning rules |
| Go repositories | `go test` + temp SQLite | SQL queries, migrations, transactions |
| Go services | `go test` + temp dirs + temp SQLite | Scan, install, sync, reconcile flows |
| Go adapters | `go test` + fixture directories | Provider detection, entry classification |
| Electron IPC bridge | Vitest or Jest | Preload bridge serialization, method allowlist |
| React UI | Vitest + React Testing Library | View model rendering, action states, empty states |
| End-to-end | Playwright + Electron integration | Full stack smoke tests for critical paths |

**Hot reload in dev:**
- React: Vite dev server with HMR.
- Go: `air` watcher for automatic rebuild + restart on file change.
- Electron main: electron-reload or manually relaunch.

### Risk

Low, but the mock-core mode requires discipline to keep fixture responses in sync
with the actual Go API. Stale fixtures cause false confidence in UI tests.
Codegen or schema validation on fixtures can help.

### Questions to resolve

- Who maintains `electron/mock-responses/`? Should it be generated from
  Go integration tests (run against real Go core, capture responses) or
  hand-written?
- Is Playwright + Electron feasible on CI macOS agents? (Yes, but slower
  than unit tests — probably gate it to nightly or pre-release runs.)

---

## Summary: Decisions Needed Before Coding Starts

| # | Decision | Recommendation | Risk if deferred |
|---|---|---|---|
| 1 | IPC transport | stdio JSON-RPC 2.0 | High — shapes all other modules |
| 2 | Go core lifecycle | Sidecar process | Low — easy to commit |
| 3 | Operation progress model | JSON-RPC notifications (server push) | Medium — poll workaround works but pollutes API |
| 4 | API contract strategy | JSON Schema + quicktype codegen | Medium — drift causes silent bugs |
| 5 | SQLite library | `modernc.org/sqlite` | Low — swappable if needed |
| 5 | Migration tool | `golang-migrate` + embedded SQL | Low — easy to commit |
| 6 | Keychain library | `zalando/go-keyring` in Go | Low — needed before credential feature |
| 7 | Packaging | `electron-builder` + `electron-updater` | **Critical** — code signing blocks shipping |
| 8 | Electron security | `contextIsolation` + narrow preload bridge | High — security regression if wrong |
| 9 | Dev ergonomics | Three modes (Go-only, mock-core, full-stack) | Medium — blocks parallel dev |

---

## What docs/10 Should Update

1. **Replace "subscribe/poll progress" language** with "subscribe to JSON-RPC
   notifications" once Decision 3 is resolved.

2. **Add Go binary path resolution** in `electron/core-process/` — how Electron
   main finds the binary in `process.resourcesPath` in production vs
   local dev path.

3. **Add Electron security config** (contextIsolation, nodeIntegration, preload
   bridge shape) to the Security section.

4. **Add migration startup sequence** to the Architecture Goals or a new
   "Startup Sequence" section.

5. **Do not finalize the folder structure** (app/, ui/, electron/, core-go/,
   shared/) until Decision 1 (transport) is settled — the `electron/core-process/`
   shape depends on it.
