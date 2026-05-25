# Astraler Skillbox — Manual Smoke Checklist

Slice 1 (Skills Library) end-to-end verification. Run this after every significant change to the scaffold or after building a release candidate.

All commands run from the **repo root** unless a different directory is specified.

---

## Pre-Conditions

- [ ] Fresh clone **or** clean working tree (`git status` is clean)
- [ ] Clean database — delete the data directory so migrations run from scratch:

  ```sh
  rm -rf ~/Library/Application\ Support/Astraler\ Skillbox/
  ```

- [ ] macOS 13+ (primary test platform for slice 1; Linux is a secondary target — substitute the path above with `~/.local/share/Astraler Skillbox/`)
- [ ] Node 20+, pnpm 9+, Go 1.22+ installed (verify: `node -v`, `pnpm -v`, `go version`)

---

## 1. Setup Smoke

- [ ] Install JS dependencies:

  ```sh
  (cd apps/desktop && pnpm install)
  ```

  Completes in under 2 minutes with no errors.

- [ ] Download Go modules:

  ```sh
  (cd core-go && go mod download)
  ```

  Completes in under 1 minute.

- [ ] Start the app in full-stack dev mode:

  ```sh
  (cd apps/desktop && pnpm dev)
  ```

  An Electron window opens in under 10 seconds. The terminal shows `[manager] Go core ready`.

- [ ] Confirm no red errors in the Electron DevTools console (open via `Cmd+Option+I`).

---

## 2. Handshake Smoke

- [ ] App renders the **Setup** screen (`/setup`) on first launch — no active host configured.
- [ ] Terminal shows `[manager] Go core ready` — confirms the JSON-RPC `server.ready` handshake succeeded.
- [ ] Terminal shows Go slog startup lines prefixed with `[core]`, e.g. `[core] level=INFO msg="skillbox-core started" pid=…`.
- [ ] The Electron window title is "Astraler Skillbox".
- [ ] Go core stdout contains only NDJSON lines (no stray text). Verify by running Go standalone in a separate terminal:

  ```sh
  (cd core-go && SKILLBOX_DB_PATH=/tmp/smoke-handshake.db go run ./cmd/skillbox-core)
  ```

  First line printed to **stdout** must be a valid JSON-RPC notification for `server.ready` with `version`, `pid`, and `capabilities` fields. (In full-stack mode this JSON is consumed internally by the manager and not printed to the terminal.)

---

## 3. Choose Host Smoke

- [ ] Create an empty test host directory:

  ```sh
  mkdir -p /tmp/skillbox-test-host
  ```

- [ ] In the app, click **"Choose Skill Host Folder…"**.
- [ ] The native macOS folder picker opens.
- [ ] Select `/tmp/skillbox-test-host` and confirm.
- [ ] App navigates to `/skills` immediately (one click — no second confirmation step).
- [ ] The skills directory was created automatically:

  ```sh
  ls /tmp/skillbox-test-host/.agents/skills/
  # Expected: empty directory (no error)
  ```

- [ ] Verify the database recorded the host:

  ```sh
  DB=~/Library/Application\ Support/Astraler\ Skillbox/skillbox.db
  sqlite3 "$DB" "SELECT id, path, status FROM skill_host_folders;"
  # Expected: 1 row, status = 'active'
  sqlite3 "$DB" "SELECT active_skill_host_folder_id FROM app_settings;"
  # Expected: non-null integer matching the host id above
  ```

---

## 4. Scan Smoke

- [ ] Create three skill directories:

  ```sh
  mkdir -p /tmp/skillbox-test-host/.agents/skills/{foo,bar,baz}
  ```

- [ ] In the app on `/skills`, click **"Scan"**.
- [ ] A toast appears showing scan progress phases (e.g., "Scanning skills…").
- [ ] After the scan completes, the toast shows "Skills scanned" (success).
- [ ] The skills table shows 3 rows: `foo`, `bar`, `baz`, all with status **Available**.
- [ ] Verify in the database:

  ```sh
  sqlite3 "$DB" "SELECT name, status FROM skills ORDER BY name;"
  # Expected: bar|available, baz|available, foo|available
  sqlite3 "$DB" \
    "SELECT operation_type, status, metadata_json
     FROM operations ORDER BY id DESC LIMIT 1;"
  # Expected: scan | success | JSON with skillsFound >= 3
  ```

---

## 5. Reconcile Smoke

- [ ] Remove one skill from the filesystem:

  ```sh
  rm -rf /tmp/skillbox-test-host/.agents/skills/foo
  ```

- [ ] Click **"Scan"** again.
- [ ] The table shows `foo` with status **Missing**; `bar` and `baz` remain **Available**.
- [ ] Verify:

  ```sh
  sqlite3 "$DB" "SELECT name, status FROM skills WHERE name = 'foo';"
  # Expected: foo|missing
  ```

---

## 6. Broken Symlink Warning Smoke

- [ ] Create a broken symlink:

  ```sh
  ln -s /nonexistent /tmp/skillbox-test-host/.agents/skills/broken
  ```

- [ ] Click **"Scan"**.
- [ ] A warning banner appears on the `/skills` screen mentioning the broken symlink.
- [ ] Verify warnings in the database:

  ```sh
  sqlite3 "$DB" \
    "SELECT scope_type, code FROM warnings ORDER BY id DESC LIMIT 1;"
  # Expected: skill_host_folder | broken_symlink   (or similar scope)
  ```

---

## 7. Switch Host Smoke

- [ ] Create a second host with its own skill:

  ```sh
  mkdir -p /tmp/skillbox-test-host-2/.agents/skills/qux
  ```

- [ ] In the app, navigate to **Settings** (`/settings`) via the sidebar.
- [ ] Click **"Change"** next to Skill Host Folder.
- [ ] Select `/tmp/skillbox-test-host-2` in the folder picker.
- [ ] App navigates to `/skills` and shows only `qux` (from the new host).
- [ ] Verify host records:

  ```sh
  sqlite3 "$DB" "SELECT path, status FROM skill_host_folders ORDER BY id;"
  # Expected: 2 rows; first host inactive, second host active
  sqlite3 "$DB" "SELECT active_skill_host_folder_id FROM app_settings;"
  # Expected: id matching the second host
  ```

---

## 8. Lifecycle Smoke

### Graceful Quit

- [ ] Press `Cmd+Q` to quit the app.
- [ ] Verify Go core exited:

  ```sh
  ps aux | grep skillbox-core | grep -v grep
  # Expected: no output (process is gone)
  ```

### Reopen with Persistence

- [ ] Relaunch: `(cd apps/desktop && pnpm dev)`
- [ ] App navigates directly to `/skills` (not `/setup`) — active host persists from DB.
- [ ] The skill list from the second host is visible.

### Crash Restart

The restart counter only applies after Go has successfully reached the ready state (i.e., after `server.ready` was received). Pre-ready crashes trigger the 10-second startup timeout / fatal path instead.

- [ ] While the app is running and the terminal shows `[manager] Go core ready`, find the Go core PID from the `[core]` log line (e.g. `[core] level=INFO msg="skillbox-core started" pid=12345`) or via:

  ```sh
  pgrep -f skillbox-core
  ```

- [ ] Kill it after it has fully started:

  ```sh
  kill -9 <go_pid>
  ```

- [ ] Electron detects the exit and restarts Go (restart 1 of 3). Wait for `[manager] Go core ready` before proceeding.
- [ ] The terminal shows `[manager] Go core exited (code=…), restart 1/3`.

### Restart Limit

Each kill must happen after the restarted Go process has reached the ready state (wait for `[manager] Go core ready` between kills). Killing before ready is a startup timeout / fatal path, not the restart counter.

- [ ] Kill the Go process 3 more times after each restart reaches ready (4 total crashes after the initial ready state). After the 4th crash, the restart counter is exhausted.
- [ ] The terminal shows `[manager] FATAL: Go core crashed too many times; giving up` and a **blocking startup error** dialog appears in the Electron window.
- [ ] No further automatic restarts occur.
- [ ] Close and reopen the app to recover (a new `pnpm dev` session resets the counter).

---

## 9. Validation Smoke

### Invalid path via DevTools

> The native folder picker enforces directory-only selection, so a file path cannot be selected through the normal UI flow. Use DevTools to invoke `host.choose` directly with an invalid path.

- [ ] Open DevTools (`Cmd+Option+I` → Console).
- [ ] Call with a file path (not a directory):

  ```js
  await window.core.invoke("host.choose", { path: "/etc/hosts" })
  ```

- [ ] The call throws in the console with a structured error object: `code: "validation_error"` and a human-readable `userMessage`. **This error appears in the DevTools console only** — it does not surface in the UI's ErrorDisplay, because the call bypasses `chooseMutation`.

### Validation error in the UI

The Settings "Change" button and the Setup "Choose Skill Host Folder" button both open a directory-only native picker, so they cannot naturally produce a validation error in slice 1. To verify UI error display works for `chooseMutation`, test via the Setup screen after choosing a host folder that the Go core rejects (e.g., a path with no write permission):

- [ ] Create a read-only directory:

  ```sh
  mkdir -p /tmp/no-write-host && chmod 555 /tmp/no-write-host
  ```

- [ ] From the Setup screen, click **"Choose Skill Host Folder…"** and select `/tmp/no-write-host`.
- [ ] The app shows an error message via the ErrorDisplay component below the button (the chooseMutation error is surfaced in the UI with `userMessage`).
- [ ] Clean up: `chmod 755 /tmp/no-write-host`

---

## Packaged macOS DMG Smoke (Slice 3A)

Run from the repo root. Produces and verifies an unsigned arm64 `.dmg`.

### Build
- [ ] `(cd apps/desktop && pnpm package:mac:unsigned)`
- [ ] Confirm artifact exists: `ls "apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg"`

### Install
- [ ] Open the `.dmg`, drag **Astraler Skillbox** to `/Applications`, eject the volume.
- [ ] Clear quarantine (unsigned build): `xattr -dr com.apple.quarantine "/Applications/Astraler Skillbox.app"`

### Launch with observable evidence
- [ ] Launch from a **neutral, non-repo cwd** (e.g. `/tmp`) with stderr captured, to strengthen the "no repo dependency" claim (cwd is inherited by the child, so this proves the sidecar does not rely on being run from the checkout):
  ```sh
  (cd /tmp && "/Applications/Astraler Skillbox.app/Contents/MacOS/Astraler Skillbox" 2> /tmp/skillbox-packaged.log)
  ```
- [ ] `grep "spawning Go core" /tmp/skillbox-packaged.log` shows a path under
  `…/Astraler Skillbox.app/Contents/Resources/core/skillbox-core` (NOT `go run` / a repo path).
- [ ] `grep "Go core ready" /tmp/skillbox-packaged.log` is present (server.ready from the bundled sidecar).
- [ ] Sidecar location/exec bit: `test -x "/Applications/Astraler Skillbox.app/Contents/Resources/core/skillbox-core" && echo OK`
- [ ] **Installed** binary is arm64 (check the bundle, not just the staged artifact):
  ```sh
  file "/Applications/Astraler Skillbox.app/Contents/Resources/core/skillbox-core"
  ```
  Expected: `Mach-O 64-bit executable arm64`.
- [ ] Sidecar is outside ASAR: the path above is a real file, not inside `app.asar`.
- [ ] Live process is the bundled one: `pgrep -fl skillbox-core` shows the in-bundle Resources path.

### Functional smoke (packaged app)
- [ ] DB created under Application Support: `ls ~/Library/Application\ Support/Astraler\ Skillbox/skillbox.db`
- [ ] Host scan succeeds; Skills Library lists host skills.
- [ ] Add a project; project scan succeeds.
- [ ] Install a skill to the project via symlink; then remove it (filesystem + DB reflect both).
- [ ] Dashboard renders aggregated state.

### Shutdown
- [ ] Quit the app (Cmd+Q).
- [ ] No orphaned sidecar: `pgrep -fl skillbox-core` returns nothing.

---

## Signed Packaging Dry-Run (Slice 3B1)

No Apple credentials required. Proves the signing config is valid, entitlements
lint, the unsigned path still works, and `mac.binaries` reaches the sidecar via
electron-builder's **own** ad-hoc signing (not a manual rescue).

### Entitlements lint
- [ ] `plutil -lint apps/desktop/build/entitlements.mac.plist` → `OK`
- [ ] `plutil -lint apps/desktop/build/entitlements.mac.inherit.plist` → `OK`

### Unsigned path regression (3A must still work)
- [ ] `(cd apps/desktop && pnpm package:mac:unsigned)`
- [ ] Artifact exists: `ls "apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg"`
- [ ] (Optional) re-run the "Packaged macOS DMG Smoke (Slice 3A)" functional steps against this DMG.

### mac.binaries proof via electron-builder ad-hoc signing
- [ ] Pack with an ad-hoc identity (electron-builder signs; no Developer ID cert needed):
  ```sh
  (cd apps/desktop && pnpm build:core && pnpm build && \
     electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false)
  ```
- [ ] Locate the packed app (adjust to electron-builder's actual output dir):
  ```sh
  APP="apps/desktop/dist/mac-arm64/Astraler Skillbox.app"
  SIDE="$APP/Contents/Resources/core/skillbox-core"
  ```
- [ ] **Before any manual `codesign`**, confirm electron-builder signed the sidecar:
  ```sh
  codesign -dvvv "$SIDE"          # expect: a signature present (ad-hoc / Signature=adhoc)
  codesign --verify --deep --strict --verbose=2 "$APP"   # expect: valid on disk
  ```
  Do **not** run `codesign -s - --deep --force` first — a post-hoc deep sign would
  rescue a `mac.binaries` misconfiguration and mask the exact failure this step exists to catch.
- [ ] If the sidecar shows **no** signature, `mac.binaries` did not reach it — fix the
  path in `electron-builder.yml` (§3.1 of the spec) and re-run before proceeding.

---

## Signed + Notarized Smoke (Slice 3B2 — requires Apple Developer ID)

DO NOT run in 3B1. Requires: Apple Developer Program, a Developer ID Application
certificate + private key, the Team ID, and notarization credentials
(App Store Connect API key preferred). notarytool is the only accepted upload
path (altool retired 2023-11-01).

```sh
APP="/Applications/Astraler Skillbox.app"
SIDE="$APP/Contents/Resources/core/skillbox-core"
DMG="apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg"

# Developer ID signature + hardened runtime (expect Authority=Developer ID Application + flags=...(runtime))
codesign -dvvv "$APP"
codesign -dvvv "$SIDE"
codesign --verify --deep --strict --verbose=2 "$APP"
codesign -d --entitlements - "$APP"

# Gatekeeper + notarization
spctl -a -vvv -t exec "$APP"                # app: expect accepted, source=Notarized Developer ID
spctl -a -vvv -t open "$DMG"                # dmg container: expect accepted, source=Notarized Developer ID
xcrun stapler validate "$APP"
xcrun stapler validate "$DMG"

# Simulate a downloaded copy WITHOUT mutating /Applications: temp copy, launch, clean up
TMP="$(mktemp -d)"; cp -R "$APP" "$TMP/"; TMP_APP="$TMP/Astraler Skillbox.app"
xattr -w com.apple.quarantine "0081;0;Safari;" "$TMP_APP"
xattr -p com.apple.quarantine "$TMP_APP"
open "$TMP_APP"                             # expect: launches, NO Gatekeeper prompt
pgrep -fl skillbox-core
osascript -e 'quit app "Astraler Skillbox"'
rm -rf "$TMP"
pgrep -fl skillbox-core                     # expect: nothing after quit
```

---

## Release Preflight (Slice 3B2A)

Read-only, offline credential/config doctor. No Apple credentials required; makes no
network call, signs/notarizes/builds nothing, and never mutates the keychain.

- [ ] Run the gate: `(cd apps/desktop && pnpm release:mac:check); echo "exit=$?"`
- [ ] On a machine WITHOUT credentials: `Signing credentials` and `Notarization credentials`
  are FAIL; the "Missing for a customer-ready notarized DMG" list contains exactly those
  two items; `exit=1`. Platform/tooling and electron-builder config invariants are PASS.
- [ ] No secret values/paths in output (targets real path/PEM indicators; variable names like `CSC_KEY_PASSWORD` are expected and fine):
  ```sh
  (cd apps/desktop && pnpm release:mac:check 2>&1 | grep -E '/Users/|-----BEGIN') || echo "clean"
  ```
  Expected: `clean`.
- [ ] When credentials ARE present (3B2), the same command exits `0` — run it before `pnpm package:mac`.

---

## Release Artifact Verification (Slice 3B2B)

Post-build, read-only verifier. No Apple credentials required for `--allow-adhoc`. The only
side effect is a read-only DMG mount/detach; it never builds, signs, notarizes, staples, calls
the network, or mutates the keychain.

### Dry-run against the 3B1 ad-hoc bundle
- [ ] Build the ad-hoc bundle (see "Signed Packaging Dry-Run (Slice 3B1)") so a `.dmg` exists under `apps/desktop/dist/`.
- [ ] `(cd apps/desktop && pnpm release:mac:verify --allow-adhoc); echo "exit=$?"`
  Expected: `exit=0`. App + sidecar signature-class (APP2/SID3, ad-hoc accepted) / `codesign --verify` / hardened-runtime / **entitlements (ENT1/ENT2)** are PASS; Gatekeeper/stapling/Team-ID lines are INFO.
- [ ] Release mode against the same artifact: `(cd apps/desktop && pnpm release:mac:verify); echo "exit=$?"`
  Expected: `exit=1`, FAILing on Developer ID signature, Team-ID equality, Gatekeeper (app + dmg), and stapling. The "Missing for a customer-ready release:" list is non-empty.
- [ ] No leftover mount after either run: `hdiutil info | grep -i skillbox-verify || echo "clean"` → `clean`.

### Release mode against a real notarized DMG (Slice 3B2 — needs credentials)
- [ ] After a real `pnpm package:mac`, run `(cd apps/desktop && pnpm release:mac:verify dist/"Astraler Skillbox-0.1.0-arm64.dmg"); echo "exit=$?"` → `exit=0`, all checks PASS.
- [ ] (Optional) pin the team: `SKILLBOX_EXPECTED_TEAM_ID=<TEAMID> pnpm release:mac:verify …`.

---

## Release Orchestrator (Slice 3B2C)

Canonical customer-release command. Composes `release:mac:check` → `package:mac` → `release:mac:verify <selected-dmg>` in the only safe order. Never uses `--allow-adhoc`. Never invokes `package:mac:unsigned`. Selects the DMG artifact by detecting exactly one created-or-modified `.dmg` in `dist/` using path+size+mtime metadata.

### Credential-less fail-fast check (no Apple credentials required)

On any machine without signing/notarization credentials, the orchestrator must exit non-zero at preflight without touching `dist/`.

- [ ] Snapshot dist before: `ls apps/desktop/dist/*.dmg 2>/dev/null || echo "(empty)"`
- [ ] Run: `(cd apps/desktop && pnpm release:mac:full); echo "exit=$?"`
  Expected: exits non-zero (`exit=1`). Output contains `[release:mac:check]` prefixed lines and `STOPPED: preflight (release:mac:check) failed`.
- [ ] Confirm `package:mac` was NOT invoked: no `[package:mac]` prefixed lines appear in the output.
- [ ] Confirm dist is unchanged: `ls apps/desktop/dist/*.dmg 2>/dev/null || echo "(empty)"` matches the before snapshot.
- [ ] No secret values in output:
  ```sh
  (cd apps/desktop && pnpm release:mac:full 2>&1 | grep -E -e '-----BEGIN' -e '/[^[:space:]]+\.(p12|p8)([[:space:]]|$)') || echo "clean"
  ```
  Expected: `clean` (no certificate/key paths or PEM blobs). Credential variable names in remediation text are okay.

### Signed/notarized release (needs Apple credentials)
- [ ] On a machine with credentials, run: `(cd apps/desktop && pnpm release:mac:full); echo "exit=$?"`
  Expected: `exit=0`. Output shows `[release:mac:check]` PASS, `[package:mac]` build output, `[release:mac:verify]` PASS, and `OK: all stages passed`.
- [ ] Confirm `dist/` contains exactly one new or overwritten `.dmg` matching what was verified.

---

## Release Manifest + Checksums (Slice 3C)

Credential-free. Works against any unsigned/ad-hoc DMG. Performs file reads and two atomic writes
into `dist/`; never builds, signs, notarizes, calls Apple, or makes a network request.

### Standalone manifest smoke (no Apple credentials required)

Use the existing unsigned/ad-hoc DMG built from Slice 3A/3B1.

- [ ] Run manifest against the existing DMG (adjust basename if needed):
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest "dist/Astraler Skillbox-0.1.0-arm64.dmg")
  ```
  Expected: exits `0`. Output shows `artifact`, `byteSize`, `sha256`, `manifest`, and `sums` lines.

- [ ] Inspect the manifest — must contain exactly eight fields in order:
  ```sh
  cat "apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg.manifest.json"
  ```
  Expected fields: `appId`, `productName`, `version`, `artifact`, `arch`, `byteSize`, `sha256`, `buildTimestamp`.
  - `appId` = `com.astraler.skillbox`
  - `productName` = `Astraler Skillbox`
  - `arch` = `arm64`
  - `byteSize` is an integer (no quotes)
  - `sha256` is 64 lowercase hex chars
  - `buildTimestamp` is UTC ISO-8601 (`…Z`)
  - `artifact` is the basename only (no path prefix)

- [ ] Verify the SHA-256 from the shell:
  ```sh
  cd apps/desktop/dist && shasum -a 256 -c SHA256SUMS && echo "shasum OK"
  ```
  Expected: `Astraler Skillbox-0.1.0-arm64.dmg: OK` and `shasum OK`.
  ```sh
  # Also run with sha256sum when available (e.g. via Homebrew coreutils):
  cd apps/desktop/dist && sha256sum -c SHA256SUMS && echo "sha256sum OK"
  ```

- [ ] Idempotency — re-run must not add a duplicate line:
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest "dist/Astraler Skillbox-0.1.0-arm64.dmg")
  wc -l apps/desktop/dist/SHA256SUMS
  ```
  Expected: same line count as before the second run (no stale duplicate for the same artifact).

- [ ] Confirm no secrets in output:
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest "dist/Astraler Skillbox-0.1.0-arm64.dmg" 2>&1 | grep -E '/[^[:space:]]+\.(p8|p12|pem)|-----BEGIN') || echo "clean"
  ```
  Expected: `clean`.

### Error handling checks

- [ ] Missing argument → non-zero exit:
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest); echo "exit=$?"
  ```
  Expected: `exit=1` and a usage message.

- [ ] Non-existent path → non-zero exit:
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest "dist/nonexistent.dmg"); echo "exit=$?"
  ```
  Expected: `exit=1` and a clear error message.

- [ ] Non-`.dmg` extension → non-zero exit:
  ```sh
  (cd apps/desktop && pnpm release:mac:manifest "dist/SHA256SUMS"); echo "exit=$?"
  ```
  Expected: `exit=1` and a clear error message.

### Wiring in release:mac:full (unit-test coverage; live run still stops at preflight)

- [ ] The manifest stage fires only after a successful verify:
  See `scripts/release-mac-full.test.mjs` — injected `runStage` tests verify that manifest
  is invoked with the selected DMG path after verify succeeds, is skipped when verify fails,
  and fails the orchestrator when manifest itself fails.

- [ ] Live `release:mac:full` still exits at preflight on this no-credential machine:
  ```sh
  (cd apps/desktop && pnpm release:mac:full); echo "exit=$?"
  ```
  Expected: `exit=1`, output contains `STOPPED: preflight`, no `[release:mac:manifest]` lines.

---

## Notes

Manual smoke **cannot be fully automated** in a headless environment because it requires:
- A display for the Electron window
- macOS native folder picker interaction
- Visual inspection of the UI state

Run this checklist on a developer machine with a display before tagging a release candidate. The automated test suites (`pnpm test`, `go test -race ./...`) cover unit and integration logic; this checklist covers the end-to-end UI and process wiring.
