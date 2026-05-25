# Astraler Skillbox — Scaffold Guide

Developer setup, dev modes, database management, and troubleshooting for slice 1.

---

## Prerequisites

| Tool | Version | Check |
|------|---------|-------|
| Node.js | 20+ | `node -v` |
| pnpm | 9+ | `pnpm -v` |
| Go | 1.22+ | `go version` |
| Platform | macOS 13+ (primary); Linux (secondary); Windows deferred | |

> **No root `package.json`.** This repo has a single JS package at `apps/desktop/`. There is **no pnpm workspace** and no root-level `pnpm install`. Every `pnpm` command must be run from `apps/desktop/`.

---

## Install

```sh
# From repo root:
(cd apps/desktop && pnpm install)   # JS deps
(cd core-go && go mod download)     # Go module cache
```

---

## Dev Modes

### Full-Stack (default)

Electron launches Go core as a child process via `go run`. Both run together.

```sh
(cd apps/desktop && pnpm dev)
```

- Electron window opens within ~10 seconds.
- Terminal shows `[manager] Go core ready` when the JSON-RPC handshake completes.
- Hot-reload is active for the renderer (Vite HMR); main process and Go core restart on file change requires a manual re-run.

### Go-Only (TDD / backend iteration)

Run Go tests and optionally start Go core in isolation without Electron.

```sh
# Run all tests with race detector:
(cd core-go && go test -race ./...)

# Run a specific package:
(cd core-go && go test -race ./internal/services/...)

# Start Go core standalone (prints JSON-RPC over stdout, slog over stderr):
(cd core-go && SKILLBOX_DB_PATH=/tmp/dev.db go run ./cmd/skillbox-core)
```

Pipe a JSON-RPC request manually:

```sh
echo '{"jsonrpc":"2.0","id":1,"method":"settings.get","params":{}}' | \
  (cd core-go && SKILLBOX_DB_PATH=/tmp/dev.db go run ./cmd/skillbox-core)
```

### UI-Only (mock core)

**Not implemented in slice 1.**

The environment variable `SKILLBOX_USE_MOCK_CORE` is referenced in the spec but is not wired in the current codebase. `apps/desktop/electron/main/core-process/manager.ts` always spawns the real Go core. To work on the renderer without a running Go core, use the **full-stack mode** with a real Go binary, or stub responses by running Go core standalone (see above).

This mode is planned for a future slice when fixture responses are added to the manager.

---

## Database

| Setting | Value |
|---------|-------|
| Default path (macOS) | `~/Library/Application Support/Astraler Skillbox/skillbox.db` |
| Override | `SKILLBOX_DB_PATH=/tmp/test.db` |
| Engine | SQLite (WAL mode, FK enforcement, busy_timeout=5000ms) |

**Override in full-stack dev:**

```sh
# The env var is inherited by the Go child process:
(cd apps/desktop && SKILLBOX_DB_PATH=/tmp/dev.db pnpm dev)
```

**Override in Go-only mode:**

```sh
(cd core-go && SKILLBOX_DB_PATH=/tmp/dev.db go run ./cmd/skillbox-core)
```

**Inspect:**

```sh
DB=~/Library/Application\ Support/Astraler\ Skillbox/skillbox.db
sqlite3 "$DB" ".tables"
sqlite3 "$DB" "SELECT * FROM skill_host_folders;"
sqlite3 "$DB" "SELECT * FROM app_settings;"
sqlite3 "$DB" "SELECT name, status FROM skills ORDER BY name;"
sqlite3 "$DB" "SELECT operation_type, status, metadata_json FROM operations ORDER BY id DESC LIMIT 5;"
```

**Reset (delete all state):**

```sh
rm -rf ~/Library/Application\ Support/Astraler\ Skillbox/
```

Migrations run automatically on next startup via `golang-migrate`.

---

## Logs

In **full-stack dev mode** (`pnpm dev`):

- Electron main process output (`[manager] ...`) prints to the terminal that launched `pnpm dev`.
- Go core's stderr (slog output) is forwarded to the same terminal with a `[core]` prefix by `json-rpc-client.ts`. Example: `[core] level=INFO msg="skillbox-core started" pid=12345`.
- When the JSON-RPC `server.ready` handshake succeeds, the manager prints `[manager] Go core ready`.

To capture Go logs to a file in full-stack mode, redirect the terminal output:

```sh
(cd apps/desktop && pnpm dev) 2>&1 | tee /tmp/skillbox-dev.log
```

To run Go core standalone and inspect its JSON-RPC stdout:

```sh
(cd core-go && SKILLBOX_DB_PATH=/tmp/dev.db go run ./cmd/skillbox-core)
# stdout: JSON-RPC NDJSON (server.ready notification on first line)
# stderr: slog output (not prefixed with [core] in standalone mode)
```

> **Slice 1 limitation:** Log-file routing for the packaged app (`~/Library/Logs/Astraler Skillbox/`) is not yet configured. It is planned for a future slice when `electron-log` or equivalent is added.

---

## Contracts

JSON Schema → TypeScript types. Run from `apps/desktop`:

```sh
# Regenerate generated/ from shared/api-contracts/:
(cd apps/desktop && pnpm generate:contracts)

# Check for drift (CI gate — fails if generated/ is stale):
(cd apps/desktop && pnpm check:contracts-drift)
```

Generated files live in `shared/generated/` and are committed. Do not edit them by hand.

---

## Tests

```sh
# Renderer + Electron unit tests (Vitest):
(cd apps/desktop && pnpm test)

# TypeScript type check:
(cd apps/desktop && pnpm typecheck)

# Go unit + integration tests:
(cd core-go && go test ./...)

# Go tests with race detector (run this before every commit):
(cd core-go && go test -race ./...)

# Race-sensitive packages only (faster CI pass):
(cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/repositories/...)

# Single Go test:
(cd core-go && go test -run TestScanHost_RunnerConflictError_PassedThrough ./internal/services/...)
```

---

## Troubleshooting

### `server.ready` timeout

**Symptom:** Electron shows a fatal error dialog; terminal shows `[manager] Go core did not send server.ready within 10s`.

**Causes and fixes:**

1. **Go build failure** — run `(cd core-go && go build ./cmd/skillbox-core)` to see compilation errors.
2. **Go not in PATH** — `which go` should return a path; add Go to PATH in your shell profile.
3. **Module cache missing** — run `(cd core-go && go mod download)`.
4. **Port or resource conflict** — Go core writes to stdout; ensure nothing else is consuming the pipe.

### `method_not_allowed` error in renderer

**Symptom:** Calling a method from the UI results in a `method_not_allowed` JSON-RPC error.

**Fix:** Add the method name to `apps/desktop/electron/main/core-process/method-allowlist.ts`. The allowlist is checked in the IPC bridge before forwarding to Go.

### SQLite "database is locked"

**Symptom:** Go core logs `database is locked` or `SQLITE_BUSY`.

**Causes and fixes:**

1. **Multiple Go processes sharing the same DB file** — check `ps aux | grep skillbox-core` and kill duplicates.
2. **`PRAGMA busy_timeout` not applied** — all connections must apply the 5000ms timeout. See `repositories.OpenDatabase()`.
3. **WAL checkpoint stuck** — delete the `-wal` and `-shm` files alongside the `.db` file (only when no process has the DB open).

### Renderer build fails with `Cannot find module '@contracts/...'`

**Symptom:** TypeScript or Vite errors about missing `@contracts` imports.

**Fix:** Regenerate contracts: `(cd apps/desktop && pnpm generate:contracts)`. The alias is configured in both `electron.vite.config.ts` and `tsconfig.web.json`.

### App opens but stays on `/setup` after setting a host

**Symptom:** After choosing a host, the app navigates to `/skills` but then bounces back to `/setup` on next launch.

**Fix:** Check that `app_settings.active_skill_host_folder_id` is non-null in the DB (`sqlite3 "$DB" "SELECT * FROM app_settings;"`). If null, the `settings.get` handler is not returning an `activeHost` — check that the host row exists with `status = 'active'`.

### `pnpm` command not found / wrong version

**Symptom:** `pnpm: command not found` or unexpected behavior from an old pnpm.

**Fix:** Install pnpm via `npm install -g pnpm@latest` or via [pnpm.io install instructions](https://pnpm.io/installation). Do not run `npm install` in this repo.

---

## Packaging (Slice 3A — unsigned macOS DMG)

- `pnpm build:core` — compiles `core-go` to `apps/desktop/resources/core/skillbox-core` (darwin/arm64, CGO off).
- `pnpm package:mac:unsigned` — runs `build:core`, then `electron-vite build`, then `electron-builder --mac dmg`.
- Output: `apps/desktop/dist/Astraler Skillbox-<version>-arm64.dmg` (unsigned).
- The sidecar is bundled via `extraResources` at `Contents/Resources/core/skillbox-core` (outside ASAR).
- Signing/notarization is deferred to Slice 3B.

---

## Release Tag

The tag `slice-1-skills-library` is **deferred**. Do not create or push it until the owner explicitly approves the release checkpoint. Implementation agents must not self-publish release artifacts.

```sh
# Correct procedure (owner runs this after explicit approval):
git tag slice-1-skills-library
git push origin slice-1-skills-library
```
