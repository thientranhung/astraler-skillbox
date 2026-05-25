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

## Packaging

### Unsigned (Slice 3A)
- `pnpm package:mac:unsigned` — `build:core` → `electron-vite build` → `electron-builder --mac dmg`
  with signing and hardened runtime disabled (`identity=null`, `hardenedRuntime=false`, `notarize=false`).
- Output: `apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg` (unsigned). Launching requires a
  Gatekeeper override (`xattr -dr com.apple.quarantine …`).

### Signed config + dry-run (Slice 3B1)
- The committed `electron-builder.yml` is the **signed default**: hardened runtime,
  entitlements (`build/entitlements.mac.*.plist`), `mac.binaries` (signs the nested sidecar),
  and `notarize: true`.
- Dry-run (no Apple cert): `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`
  signs with an ad-hoc identity so you can verify `mac.binaries` reaches
  `Contents/Resources/core/skillbox-core`. See SMOKE.md → "Signed Packaging Dry-Run (Slice 3B1)".

### Signed + notarized (Slice 3B2 — deferred, needs credentials)
- `pnpm package:mac` — full signed build. Requires in the environment:
  - Apple Developer Program membership.
  - Developer ID **Application** certificate + private key (login keychain, or `.p12` via `CSC_LINK`/`CSC_KEY_PASSWORD`).
  - **Team ID**.
  - Notarization credentials: App Store Connect API key (`APPLE_API_KEY` `.p8` + `APPLE_API_KEY_ID` + `APPLE_API_ISSUER`) **or** `APPLE_ID` + `APPLE_APP_SPECIFIC_PASSWORD` + `APPLE_TEAM_ID`.
- `.dmg`-only distribution → no Developer ID **Installer** cert / `.pkg` needed.
- Running `pnpm package:mac` without these credentials is expected to fail; that is not a 3B1 gate.

### Release preflight (Slice 3B2A)
- `pnpm release:mac:check` — read-only, offline gate. Reports signing-credential readiness
  (keychain Developer ID Application identity OR `CSC_LINK` + `CSC_KEY_PASSWORD`), notarization
  credential groups (API key, or Apple ID + app password + Team ID), electron-builder config
  invariants (hardened runtime, notarize, entitlements, `mac.binaries`, dmg/arm64), staged-sidecar
  sanity, and artifact/secret hygiene. Exits non-zero when a hard blocker is present.
- Run it BEFORE `pnpm package:mac` to surface credential/config gaps in <1s instead of minutes
  into a build. It never signs, notarizes, builds, calls Apple, mutates the keychain, or prints
  any secret value or path. See SMOKE.md → "Release Preflight (Slice 3B2A)".

### Release artifact verification (Slice 3B2B)
- `pnpm release:mac:verify [path]` — read-only **post-build** gate (the bookend to `release:mac:check`).
  Verifies a built `.app`/`.dmg` is customer-ready: Developer ID signature on the app **and** the
  nested sidecar, a single shared Team ID, hardened runtime, the expected entitlements, Gatekeeper
  acceptance of the app (`spctl -t exec`) and the DMG (`spctl -t open`), and a stapled ticket on both.
- Input: an explicit `.app`, an explicit `.dmg`, or (no arg) the single `apps/desktop/dist/*.dmg`
  (multiple → pass an explicit path). A `.dmg` is mounted **read-only**; the single top-level `.app`
  is verified (nested Electron helper apps are ignored), then unmounted.
- `--allow-adhoc` verifies the 3B1 ad-hoc dry-run bundle (signature/runtime/entitlements PASS;
  notarization/stapling/Team-ID reported INFO). `SKILLBOX_EXPECTED_TEAM_ID` optionally pins the team.
- It never builds, signs, notarizes, staples, calls Apple, or mutates the keychain. Run it AFTER
  `pnpm package:mac`. See SMOKE.md → "Release Artifact Verification (Slice 3B2B)".

### Release orchestrator — canonical customer-release command (Slice 3B2C)

> **Customer release runbook**: see [`RELEASE.md`](RELEASE.md) for credential setup, preflight, artifact verification, and troubleshooting.
- `pnpm release:mac:full` — composes `release:mac:check` → `package:mac` → `release:mac:verify <dmg>`
  → `release:mac:manifest <dmg>` in the only safe order. Fails fast at the first failed stage.
- DMG selection: detects exactly one `.dmg` created or modified in `dist/` between before/after snapshots
  using path+size+mtime metadata. Handles same-name overwrites. Errors on zero or multiple changed DMGs.
- Missing `dist/` is treated as an empty snapshot (clean checkout can still package).
- The manifest stage runs **only after** a successful verify, using the same selected DMG path.
  On manifest failure: `STOPPED: release:mac:manifest failed`; `failedStage: "manifest"`.
  On full success: reports `manifest` and `sums` paths.
- Never passes `--allow-adhoc`. Never calls `package:mac:unsigned`. Never reads or prints secret values.
- On a machine without Apple credentials: exits non-zero at preflight; `package:mac` is never invoked.
- See SMOKE.md → "Release Orchestrator (Slice 3B2C)".

### Release manifest + checksums (Slice 3C)
- `pnpm release:mac:manifest <path-to-dmg>` — credential-free artifact integrity generator.
  Given the **exact** path to a built `.dmg`, computes its SHA-256 and emits:
  - `dist/<artifact>.manifest.json` — structured integrity manifest with exactly eight fields:
    `appId`, `productName`, `version`, `artifact`, `arch`, `byteSize`, `sha256`, `buildTimestamp`.
  - `dist/SHA256SUMS` — coreutils-compatible checksum file customers verify with
    `shasum -a 256 -c` / `sha256sum -c`. Uses basename-only entries; deterministic upsert
    (replace existing line for the same artifact in place; append for new artifacts; never duplicates).
- Both outputs are written **atomically** (temp file in `dist/` then rename). A failed write
  leaves the previous `SHA256SUMS` intact; no truncated output is ever left as the visible file.
- Input: required explicit path — no `dist/` auto-discovery, no glob, no "latest DMG" heuristic.
  Missing/non-existent/non-`.dmg` path → non-zero exit with a clear error message.
- Metadata sources: `appId` and `productName` from `electron-builder.yml`; `version` from
  `package.json`; `arch` from `electron-builder.yml` `mac.target.arch` (filename parsing only
  as a fallback when config is ambiguous); `artifact` is the basename of the supplied path;
  `byteSize` and `sha256` from the file itself; `buildTimestamp` from `new Date().toISOString()`.
- Never reads credentials, calls Apple services, builds, signs, notarizes, or makes network requests.
- Works on any DMG regardless of signing status (unsigned/ad-hoc until real credentials land).
- See SMOKE.md → "Release Manifest + Checksums (Slice 3C)".

---

## Release Tag

The tag `slice-1-skills-library` is **deferred**. Do not create or push it until the owner explicitly approves the release checkpoint. Implementation agents must not self-publish release artifacts.

```sh
# Correct procedure (owner runs this after explicit approval):
git tag slice-1-skills-library
git push origin slice-1-skills-library
```
