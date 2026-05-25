# Slice 3F: Packaged App Launch Smoke — Implementation Plan

> **For agentic workers:** Use `superpowers:executing-plans` for implementation. Use `/goal` only after this plan is lead-approved.

## Goal

Fix the hardened-runtime launch failure found after Slice 3E and add a credential-free packaged app launch smoke command that proves the staged `.app` actually boots, starts the bundled Go core, and shuts down without leaving an orphaned `skillbox-core`.

Root cause evidence:
- `release:mac:dry-run` produced a DMG/app that passed `codesign --verify`, but direct launch of `dist/mac-arm64/Astraler Skillbox.app/Contents/MacOS/Astraler Skillbox` failed before Electron boot with `Electron Framework ... not valid for use in process ... different Team IDs`.
- Main app entitlements had `allow-jit` and `allow-unsigned-executable-memory`, but not `com.apple.security.cs.disable-library-validation`.
- Re-signing the ignored local `.app` with `disable-library-validation=true` made it launch; stderr showed `[manager] spawning Go core`, `[manager] Go core ready`, renderer/helper processes, and clean Go core shutdown.
- The verifier derives expected entitlements from `build/entitlements.mac.plist`, so adding the key keeps preflight/verify aligned.
- This is a deliberate hardened-runtime exception for the Electron app process. It proves and fixes the local ad-hoc hardened dry-run launch failure; the Developer ID/notarized path still must be validated once real credentials exist.

## Hard Constraints

- Preserve `mac.hardenedRuntime: true`, `mac.notarize: true`, and the signed-release path.
- The new launch-smoke command must not call Apple services, keychain, notarization, upload, `release:mac:check`, or `release:mac:full`.
- Keep smoke credential-free and local.
- Avoid customer data: launch smoke must set a temporary `--user-data-dir` and `SKILLBOX_DB_PATH`.
- Fail fast and clean up: terminate the app, remove temp dirs, and fail if `skillbox-core` remains after shutdown.

## Files

- Modify `apps/desktop/build/entitlements.mac.plist`
- Create `apps/desktop/scripts/release-mac-launch-smoke.lib.mjs`
- Create `apps/desktop/scripts/release-mac-launch-smoke.mjs`
- Create `apps/desktop/scripts/release-mac-launch-smoke.test.mjs`
- Modify `apps/desktop/package.json`
- Modify `SMOKE.md`, `SCAFFOLD.md`, and `RELEASE.md`

## Task 1: Entitlement Fix

- [x] Add `com.apple.security.cs.disable-library-validation` to `build/entitlements.mac.plist`.
- [x] Add `com.apple.security.cs.disable-library-validation` to `build/entitlements.mac.inherit.plist`.
  **Evidence from launch-smoke testing:** main-app-only was tested first. With the key absent from
  `entitlements.mac.inherit.plist`, `release:mac:launch-smoke` failed immediately — `Astraler Skillbox
  Helper.app` could not load `Electron Framework`: `not valid for use in process: mapping process and
  mapped file (non-platform) have different Team IDs`. Electron forks GPU, network-service, and renderer
  helper processes that each independently load `Electron Framework`; the main-process entitlement does
  not propagate to them. The inherit entitlement is required for all Electron helper processes.
  This is a deliberate hardened-runtime exception. Developer ID/notarization behaviour must be
  validated separately once real credentials are available.
- [x] Confirm `plutil -lint build/entitlements.mac.plist` passes.
- [x] Confirm `plutil -lint build/entitlements.mac.inherit.plist` passes.
- [x] Confirm `release:mac:verify --allow-adhoc <dmg>` reports the new entitlement because expected keys come from the plist.

## Task 2: Pure Smoke Helpers

- [ ] Add pure helpers for:
  - resolving the staged app executable from `dist/mac-arm64/Astraler Skillbox.app`
  - detecting readiness from stderr (`[manager] Go core ready`)
  - detecting startup failure from stderr (`Library not loaded`, `not valid for use in process`, `server.ready timeout`, `[manager] FATAL`)
  - building a temp launch environment with `SKILLBOX_DB_PATH`
  - checking whether a process list contains orphaned `skillbox-core`
- [ ] Unit tests cover success matching, failure matching, timeout decision, app path resolution, temp env construction, and orphan detection.

## Task 3: IO Shell

- [ ] Add script: `"release:mac:launch-smoke": "node scripts/release-mac-launch-smoke.mjs"`.
- [ ] The command expects an already-packaged staged app at `dist/mac-arm64/Astraler Skillbox.app`; it does not package or run dry-run itself.
- [ ] Launch the executable directly with `--user-data-dir=<tmpdir>`.
- [ ] Set `SKILLBOX_DB_PATH=<tmpdir>/skillbox.db` so the Go core does not touch the real Application Support database.
- [ ] Stream and capture stdout/stderr with prefixes.
- [ ] Wait up to a bounded timeout for `[manager] Go core ready`.
- [ ] Quit/terminate the app after readiness, wait briefly, then assert no `skillbox-core` whose executable path is inside the staged app remains.
- [ ] On startup failure or timeout, print a concise diagnostic and exit non-zero.
- [ ] Always clean temp dirs.

## Task 4: Documentation

- [ ] Add a Slice 3F section to `SMOKE.md` showing:
  - run `pnpm release:mac:dry-run`
  - run `pnpm release:mac:launch-smoke`
  - expected `exit=0`, `Go core ready`, and no orphaned sidecar
- [ ] Update `SCAFFOLD.md` packaging/release notes with the new entitlement and smoke command.
- [ ] Update `RELEASE.md` to recommend `release:mac:launch-smoke` after `release:mac:dry-run` and before credentialed release attempts.

## Task 5: Verification

- [ ] `cd apps/desktop && pnpm exec vitest run scripts/release-mac-launch-smoke.test.mjs scripts/release-mac-verify.test.mjs scripts/release-mac-check.test.mjs`
- [ ] `cd apps/desktop && pnpm test`
- [ ] `cd apps/desktop && pnpm typecheck`
- [ ] `cd apps/desktop && pnpm check:contracts-drift`
- [ ] `cd core-go && go test ./...`
- [ ] `cd apps/desktop && pnpm release:mac:dry-run`
- [ ] `cd apps/desktop && pnpm release:mac:launch-smoke`
- [ ] Confirm the launch-smoke log no longer fails with `Electron Framework ... different Team IDs`, using only its temp `--user-data-dir` and `SKILLBOX_DB_PATH`.
- [ ] Optional operator check outside the launch-smoke command: `release:mac:full` should still stop at preflight on this no-credential machine. Do not bake this into `release:mac:launch-smoke`.

## Notes

- This smoke does not prove Gatekeeper acceptance or notarization; it proves the packaged app can boot locally after ad-hoc signing.
- A display session may be required because Electron creates a real BrowserWindow. If unavailable, the script should fail clearly with a display/session diagnostic rather than silently passing.
