# Slice 3G: DMG Mount-and-Launch Smoke — Implementation Plan

> **For agentic workers:** Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. `/goal` is appropriate only after this plan is approved and implementation is delegated as a long-running verification loop.

## Goal

Add a credential-free `pnpm release:mac:dmg-smoke` command that proves the *actual distributable artifact* boots: it mounts the produced DMG read-only, finds the top-level `Astraler Skillbox.app`, copies it to a temp install dir (never `/Applications`), launches the copied app from a neutral cwd with a temp `--user-data-dir` and `SKILLBOX_DB_PATH`, waits for the bundled Go core to be ready, shuts down, asserts no orphaned `skillbox-core` from the copied bundle, detaches the DMG, and cleans temp dirs.

## Why this slice

Slice 3F (`release:mac:launch-smoke`) boots the *staged* app at `dist/mac-arm64/Astraler Skillbox.app`. That is the build output directory, not the artifact a customer mounts. 3G closes the gap by exercising the DMG → mount → copy-out → launch path, catching artifact-only failures (binary fails when read from a mounted volume, `.app` not copied into the DMG correctly, mount/volume permission differences) that the staging smoke cannot see. It is fully verifiable today against the ad-hoc DMG from `release:mac:dry-run`, with no Apple credentials.

## Hard Constraints

- **No Apple services, no keychain, no notarization, no stapling, no network, no upload.**
- **Does not run** `release:mac:check` or `release:mac:full`, and does not package, sign, or build. It consumes an already-produced DMG.
- **No credentials.** Credential env vars (`CSC_`, `APPLE_`, `NOTARYTOOL_`) are stripped from the launch environment (reuse `buildLaunchEnv`).
- **Read-only mount only.** Mount with `hdiutil attach -readonly -nobrowse -mountpoint`. Never write to the mounted volume.
- **Copy out, never install in place.** Copy the `.app` to a temp install dir under `os.tmpdir()`. Never touch `/Applications`. Launch the *copy*, not the app on the mounted volume.
- **Exact bundle required.** The top-level mounted app must be `Astraler Skillbox.app`; a single differently named `.app` is a failure.
- **Neutral cwd.** Launch the copied app with `cwd` set to the temp root (not the repo, not the mounted volume).
- **No customer data.** Set a temp `--user-data-dir` and `SKILLBOX_DB_PATH=<tmp>/skillbox.db`.
- **Fail if detach fails.** A failed `hdiutil detach` (even after `-force`) must surface as a non-zero exit after the report prints. Never leak a mounted volume silently. Mirror `release-mac-verify.mjs:44-66,187-191`.
- **GUI session required.** Electron creates a real `BrowserWindow`. On premature exit / timeout, print a clear display-session diagnostic and exit non-zero rather than passing silently.
- **Fail fast and clean.** Terminate the app, detach the DMG, and remove all temp dirs on every exit path (success, failure, startup error). Orphan check runs on every path.

## Reuse / Refactor Decisions

Reuse proven, already-exported helpers; do **not** modify the working release gate `release-mac-verify.mjs` in this slice.

Reuse directly (import, no changes):
- From `scripts/release-mac-launch-smoke.lib.mjs`: `isReadyLine`, `isFailureLine`, `extractFailureDiagnostic`, `buildLaunchEnv`, `detectOrphanedSidecar`, `isTimedOut`. `detectOrphanedSidecar(procs, appPath)` already matches a sidecar whose exe path is inside any given `appPath` — pass the **copied bundle path** so only copied-bundle orphans count.
- From `scripts/release-mac-verify.parse.mjs`: `discoverDmg(entries)` (single-DMG selection + explicit-path rules) and `pickTopLevelApp(entries)` (non-recursive top-level `.app` selection — ignores nested helper apps).

New pure helpers (in a new `release-mac-dmg-smoke.lib.mjs`) so the IO shell stays thin and testable:
- `resolveCopiedApp(installDir, appBundleName)` → `{ appPath, execPath }` where `appPath = installDir/appBundleName` and `execPath = appPath/Contents/MacOS/Astraler Skillbox`.
- `buildAttachArgs(mountPoint, dmgPath)` → `["attach", "-readonly", "-nobrowse", "-mountpoint", mountPoint, dmgPath]`.
- `buildDetachArgs(mountPoint, force)` → `["detach", mountPoint]` or `["detach", "-force", mountPoint]`.
- `buildDittoArgs(srcAppPath, destAppPath)` → `[srcAppPath, destAppPath]` (copy a bundle with `/usr/bin/ditto`, which preserves bundle structure, symlinks, permissions, and extended attributes — `cp -R` is not safe for `.app` bundles).
- `execName(appBundleName)` → the Mach-O executable name derived from the bundle name (`"Astraler Skillbox.app"` → `"Astraler Skillbox"`).
- `assertExpectedAppBundle(appBundleName, expected = "Astraler Skillbox.app")` → returns the name when it matches; throws a clear error for any other bundle name.

Explicit non-goal: no dedupe of `release-mac-verify.mjs`'s inline mount/detach into a shared module in this slice. Duplicating the ~20-line proven detach pattern in the new IO shell is lower risk than refactoring a release gate. A future shared `release-mac-dmg-mount.mjs` extraction is noted as optional follow-up only.

## Files

- Create `apps/desktop/scripts/release-mac-dmg-smoke.lib.mjs` — new pure helpers listed above.
- Create `apps/desktop/scripts/release-mac-dmg-smoke.mjs` — IO shell: mount, copy via `ditto`, launch, wait, shutdown, orphan check, detach, cleanup.
- Create `apps/desktop/scripts/release-mac-dmg-smoke.test.mjs` — unit tests for the new pure helpers.
- Modify `apps/desktop/package.json` — add `"release:mac:dmg-smoke": "node scripts/release-mac-dmg-smoke.mjs"`.
- Modify `SMOKE.md` — add a Slice 3G section.
- Modify `RELEASE.md` — insert the DMG smoke into the recommended pre-credential sequence in §7.
- Modify `SCAFFOLD.md` — note the new command in the packaging/release inventory.

## Task 1: Pure Helpers + Tests

**Files:**
- Create: `apps/desktop/scripts/release-mac-dmg-smoke.lib.mjs`
- Test: `apps/desktop/scripts/release-mac-dmg-smoke.test.mjs`

- [ ] **Step 1: Write failing tests** in `release-mac-dmg-smoke.test.mjs` covering:
  - `resolveCopiedApp("/tmp/inst", "Astraler Skillbox.app")` returns `appPath = "/tmp/inst/Astraler Skillbox.app"` and `execPath` ending in `Contents/MacOS/Astraler Skillbox`.
  - `buildAttachArgs("/tmp/mp", "/tmp/x.dmg")` deep-equals `["attach","-readonly","-nobrowse","-mountpoint","/tmp/mp","/tmp/x.dmg"]`.
  - `buildDetachArgs("/tmp/mp", false)` → `["detach","/tmp/mp"]`; `buildDetachArgs("/tmp/mp", true)` → `["detach","-force","/tmp/mp"]`.
  - `buildDittoArgs("/vol/App.app", "/tmp/inst/App.app")` → `["/vol/App.app","/tmp/inst/App.app"]`.
  - `execName("Astraler Skillbox.app")` → `"Astraler Skillbox"`.
  - `assertExpectedAppBundle("Astraler Skillbox.app")` returns the name; `assertExpectedAppBundle("Other.app")` throws with the expected bundle name in the message.
  - Detach finalization helper behavior: when normal detach fails and forced detach also fails, the resulting status is non-zero, includes the manual-detach hint, and marks the mount point as preserved rather than removable.
  - A re-export sanity check: importing `detectOrphanedSidecar` / `buildLaunchEnv` from `release-mac-launch-smoke.lib.mjs` and `discoverDmg` / `pickTopLevelApp` from `release-mac-verify.parse.mjs` resolves (guards against later path drift).
- [ ] **Step 2: Run tests, verify they fail.** Run: `pnpm exec vitest run scripts/release-mac-dmg-smoke.test.mjs`. Expected: FAIL (module/exports not defined).
- [ ] **Step 3: Implement the pure helpers** in `release-mac-dmg-smoke.lib.mjs` with no I/O and no `child_process` (mirror the header contract of `release-mac-launch-smoke.lib.mjs`). Use `node:path` only.
- [ ] **Step 4: Run tests, verify they pass.** Run: `pnpm exec vitest run scripts/release-mac-dmg-smoke.test.mjs`. Expected: PASS.
- [ ] **Step 5: Commit.** `feat(3g): pure helpers for dmg mount-and-launch smoke`.

## Task 2: IO Shell

**Files:**
- Create: `apps/desktop/scripts/release-mac-dmg-smoke.mjs`
- Modify: `apps/desktop/package.json`

- [ ] **Step 1: Add the npm script** to `package.json`: `"release:mac:dmg-smoke": "node scripts/release-mac-dmg-smoke.mjs"`.
- [ ] **Step 2: Implement the IO shell** in `release-mac-dmg-smoke.mjs` with this exact flow:
  - Resolve DMG: optional positional path arg; if absent, run `discoverDmg(readdirSync(dist))` and fail with its error message (single-DMG / explicit-path rules) when not exactly one DMG is present. Resolve relative path args against `process.cwd()`. Fail clearly if the DMG path does not exist.
  - Create two temp dirs under `os.tmpdir()` via `mkdtemp`: a mount point (`skillbox-dmgsmoke-mnt-`) and an install root (`skillbox-dmgsmoke-app-`). Track them for cleanup.
  - Mount read-only: `spawnSync("/usr/bin/hdiutil", buildAttachArgs(mountPoint, dmgPath))`. On non-zero exit, remove the empty mount temp dir, do **not** set the "mounted" flag, and exit non-zero. Only mark mounted after a successful attach (mirror `release-mac-verify.mjs:128-141`).
  - Select app: `pickTopLevelApp(readdirSync(mountPoint))`; on error, exit non-zero (after detaching). Immediately pass the result through `assertExpectedAppBundle(appBundleName)` so a DMG containing one top-level `.app` with the wrong name cannot proceed.
  - Copy out: `ditto` the mounted `.app` to `installRoot/<appBundleName>` via `spawnSync("/usr/bin/ditto", buildDittoArgs(srcAppPath, destAppPath))`. Non-zero exit → fail (after detach + cleanup).
  - Resolve the copied executable with `resolveCopiedApp(installRoot, appBundleName)`; assert `existsSync(execPath)`.
  - Launch: `spawn(execPath, ["--user-data-dir=" + userDataDir], { cwd: <temp root, neutral>, env: buildLaunchEnv(process.env, userDataDir), stdio: ["ignore","pipe","pipe"], detached: false })`. `userDataDir` is a third temp dir (or a subdir of the install root that is not the bundle).
  - Stream stdout/stderr with prefixes; watch stderr with `isReadyLine` / `isFailureLine` / `extractFailureDiagnostic`. Resolve on ready, on failure line, on premature `close`, or on `error`.
  - Bounded readiness timeout (`READY_TIMEOUT_MS = 30_000`). On timeout/premature-exit, print the display-session note (Electron requires a display session).
  - Shutdown + orphan check (run on **every** exit path): `SIGTERM` → wait `SHUTDOWN_WAIT_MS = 3_000` → `SIGKILL` → brief reap wait → list `skillbox-core` processes (`ps -axo pid=,command=`, same parser as launch-smoke) → `detectOrphanedSidecar(procs, copiedAppPath)`. Orphan present → report pids and exit non-zero.
  - Detach: `hdiutil detach`; on failure retry `hdiutil detach -force`; if still failing, set `detachFailed`, print the manual-detach hint, and do **not** `rmSync` the mount point. Mirror `release-mac-verify.mjs:44-66`. Keep this logic behind an injectable/testable helper so tests prove detach failure cannot pass silently.
  - Cleanup: remove the install root and user-data temp dirs (best-effort). After the report prints, if `detachFailed`, exit non-zero / throw (mirror `release-mac-verify.mjs:187-191`).
  - Exit 0 only when: Go core ready, clean shutdown, no copied-bundle orphan, and detach succeeded.
- [ ] **Step 3: Lint-run the script offline** (no DMG present) to confirm the discovery failure path is clean. Run: `pnpm release:mac:dmg-smoke` from a clean `dist/`. Expected: non-zero exit with `discoverDmg`'s "no .dmg found …" message, no temp dirs left, no mount created.
- [ ] **Step 4: Commit.** `feat(3g): dmg mount-and-launch smoke IO shell + npm script`.

## Task 3: End-to-End Verification Against a Real DMG

- [ ] **Step 1:** Produce an ad-hoc DMG. Run: `pnpm release:mac:dry-run`. Expected: a single `Astraler Skillbox-<version>-arm64.dmg` in `dist/`, manifest + SHA256SUMS written, exit 0.
- [ ] **Step 2:** Run the new smoke in a GUI session. Run: `pnpm release:mac:dmg-smoke`. Expected stdout includes the mount point, the copied install dir, the temp `SKILLBOX_DB_PATH`, `[manager] Go core ready`, a clean shutdown, `no orphaned sidecar`, a successful detach, and exit 0.
- [ ] **Step 3:** Confirm no leaks after the run:
  - `pgrep -fl skillbox-core || echo "no orphaned core"` → "no orphaned core".
  - `hdiutil info | grep -i skillbox || echo "no leaked mount"` → "no leaked mount".
  - `ls "$TMPDIR" | grep skillbox-dmgsmoke || echo "no temp dirs left"` → "no temp dirs left".
- [ ] **Step 4 (negative check, optional):** Pass an explicit path while two DMGs exist to confirm the single-DMG guard: copy the DMG to a second name, run `pnpm release:mac:dmg-smoke` with no arg → expect the "multiple .dmg files" failure; then run with the explicit path → expect it to proceed. Remove the copy afterward.

## Task 4: Documentation

**Files:** `SMOKE.md`, `RELEASE.md`, `SCAFFOLD.md`

- [ ] **Step 1:** Add a Slice 3G section to `SMOKE.md` showing the sequence `pnpm release:mac:dry-run` → `pnpm release:mac:dmg-smoke`, the expected `exit=0` / `Go core ready` / `no orphaned sidecar` / successful detach output, and the no-leak checks from Task 3 Step 3.
- [ ] **Step 2:** Update `RELEASE.md` §7. Insert `release:mac:dmg-smoke` into the recommended pre-credential sequence, after `release:mac:launch-smoke`, with one line explaining it boots the app *from the mounted DMG* (the distributable artifact) rather than the staging dir. Reiterate it is credential-free, non-distributable, and does not prove notarization or Gatekeeper acceptance.
- [ ] **Step 3:** Add `release:mac:dmg-smoke` to the packaging/release command inventory in `SCAFFOLD.md`.
- [ ] **Step 4: Commit.** `docs(3g): document dmg mount-and-launch smoke`.

## Task 5: Final Verification Gate

- [ ] `cd apps/desktop && pnpm exec vitest run scripts/release-mac-dmg-smoke.test.mjs scripts/release-mac-launch-smoke.test.mjs scripts/release-mac-verify.test.mjs scripts/release-mac-check.test.mjs` → all PASS.
- [ ] `cd apps/desktop && pnpm test` → PASS.
- [ ] `cd apps/desktop && pnpm typecheck` → PASS.
- [ ] `cd apps/desktop && pnpm check:contracts-drift` → no drift.
- [ ] `cd core-go && go test ./...` → PASS (no Go changes expected; confirms nothing regressed).
- [ ] `cd apps/desktop && pnpm release:mac:dry-run && pnpm release:mac:dmg-smoke` → exit 0, Go core ready, no orphan, clean detach, no temp/mount leaks.
- [ ] Focused detach-failure test proves normal detach failure plus forced detach failure exits non-zero, prints the manual-detach hint, and preserves the mount point for manual cleanup.
- [ ] Operator-only sanity (do **not** bake into the smoke): on this no-credential machine `pnpm release:mac:full` still stops at `release:mac:check`. Confirms 3G did not weaken the credentialed gate.

## Notes

- This smoke does not prove Developer ID signing, notarization, stapling, or Gatekeeper acceptance for customer distribution. A locally built DMG carries no `com.apple.quarantine` attribute, so the ad-hoc copied app launches from the mount on the build machine; that is expected and is **not** evidence of distributability. Use `pnpm release:mac:full` (with credentials) for a customer-ready build.
- Requires a GUI/display session because Electron opens a real `BrowserWindow`. On a headless runner the smoke must fail with the display-session diagnostic, never silently pass.
- `ditto` is mandatory for copying the bundle; `cp -R` can corrupt `.app` symlinks/extended attributes and would invalidate the launch result.
- Future optional follow-up (out of scope here): extract a shared `release-mac-dmg-mount.mjs` consumed by both `release-mac-verify.mjs` and `release-mac-dmg-smoke.mjs`, gated on all verify tests staying green.
