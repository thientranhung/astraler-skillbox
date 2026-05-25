# Slice 3A: Packaging Spike (Unsigned macOS DMG) — Design

Date: 2026-05-26
Status: approved
Scope: prove the app can be packaged as a self-contained, unsigned macOS `.dmg` that runs from `/Applications` with a bundled Go sidecar, no dev toolchain required.

## 1. Purpose and User Value

Today the app only runs in dev: Electron main spawns the sidecar with `go run ./cmd/skillbox-core` from a repo-relative path, so it requires a Go toolchain, the repo checkout, and a dev PATH. This spike establishes the first distributable artifact — an unsigned `.dmg` a user can mount, drag to `/Applications`, and launch — validating that the Electron + Go sidecar architecture survives packaging before we invest in signing/notarization and release automation.

This is a spike: the goal is a working, reproducible packaged build and a documented smoke flow, not a shippable release. Signing comes in 3B.

## 2. In Scope

- Add `electron-builder` packaging config and build scripts (mac target, `dmg`).
- Build `skillbox-core` as a bundled `darwin/arm64` binary as part of the packaging pipeline.
- Bundle the sidecar binary outside the ASAR archive via `extraResources` (or equivalent), so it is a real on-disk executable.
- Packaged-vs-dev core path resolution in Electron main: `go run` in dev, bundled binary in packaged mode.
- Confirm app-data DB path resolves under `~/Library/Application Support/Astraler Skillbox/` in packaged mode (already cwd-independent in `resolveDBPath`).
- Produce an unsigned `.dmg` artifact.
- Document and exercise a packaged smoke-test flow.

## 3. Non-Goals

- Code signing, notarization, stapling (deferred to 3B).
- Auto-update (no `electron-updater`, no feed).
- Universal binary (`darwin/arm64` only; no `amd64` / `lipo`).
- Windows / Linux targets.
- CI release automation (build is run locally for this spike).
- Any product feature changes, schema changes, or RPC contract changes.

## 4. Architecture Decisions

### 4.1 Where the Go binary is built
The sidecar is compiled with `go build` (not `go run`) targeting `GOOS=darwin GOARCH=arm64`, producing a single `skillbox-core` executable. The build is driven by a script invoked as an electron-builder `beforeBuild`/`beforePack` hook (or a `pnpm` prepackage step that runs before `electron-builder`), so packaging always bundles a freshly built binary rather than a stale checked-in artifact. The binary is not committed.

### 4.2 Where it is placed in packaged app resources
The binary is bundled via `extraResources`, landing under the app bundle's `Contents/Resources/` (e.g. `Resources/core/skillbox-core`), reachable at runtime via `process.resourcesPath`. It must live **outside** `app.asar` because ASAR-packed files are not directly executable; `extraResources` keeps it as a normal file with its mode bits intact.

### 4.3 How Electron main resolves dev vs packaged path
`core-go-path.ts` (or the manager) branches on `app.isPackaged`:
- **Dev**: keep current behavior — spawn `go run ./cmd/skillbox-core` with cwd resolved from `__dirname` via `resolveCoreGoPath`.
- **Packaged**: resolve the bundled binary at `path.join(process.resourcesPath, "core", "skillbox-core")` and spawn it directly (no `go`, no cwd dependence on the repo).

The spawn call in `manager.ts` is parameterized over `{ command, args, cwd }` so the rest of the lifecycle (ready timeout, restart policy, SIGTERM→SIGKILL shutdown) is unchanged between modes.

### 4.4 How DB / app-data path works in packaged mode
No change required. `resolveDBPath` in `core-go/cmd/skillbox-core/main.go` already derives the path from `os.UserHomeDir()` → `~/Library/Application Support/Astraler Skillbox/skillbox.db` and `MkdirAll`s the directory, independent of cwd or PATH. The spike only verifies this holds when launched from `/Applications`. `SKILLBOX_DB_PATH` remains the dev/test override.

### 4.5 How the executable bit is preserved / verified
`extraResources` copies files preserving mode, but to be safe the build hook explicitly `chmod 0755` the binary before packaging, and the smoke flow verifies the bit on the installed bundle (`ls -l`/`test -x` on `Contents/Resources/core/skillbox-core`). If a future build path strips the bit, Electron main may also defensively `chmod` on first spawn — decided during implementation, not mandated here.

### 4.6 Build entrypoint (scripts, hook order, output path)

Expected `pnpm` scripts in `apps/desktop/package.json` (names are the contract implementation must honor):

- `build:core` — compile the sidecar: `GOOS=darwin GOARCH=arm64 go build -o <staging>/skillbox-core ./cmd/skillbox-core` (run against `core-go`) and `chmod 0755` the output. `<staging>` is a path referenced by electron-builder `extraResources` (e.g. `apps/desktop/resources/core/skillbox-core`).
- `package:mac:unsigned` — the spike entrypoint: runs `electron-vite build`, then `electron-builder --mac dmg` with signing disabled. Reserve `package:mac` for the signed 3B flow so the unsigned spike command is unambiguous.

Hook / prepackage order (strict): the Go binary MUST exist before electron-builder packs the app. Two acceptable wirings, decided at implementation:
1. Chain in the script: `package:mac:unsigned` = `pnpm build:core && pnpm build && electron-builder --mac dmg`.
2. electron-builder `beforePack`/`beforeBuild` hook that invokes `build:core`.

Either way the invariant is: **renderer/main bundle build + freshly built `skillbox-core` staged → then electron-builder packs**. electron-builder must not run before `build:core` completes.

Expected DMG output: under `apps/desktop/dist/` (electron-builder's default `output` dir for this package), named per electron-builder's default `${productName}-${version}-arm64.dmg` (e.g. `Astraler Skillbox-0.0.0-arm64.dmg`). The exact `productName`/`version`/`artifactName` is fixed during implementation; the spec only pins the directory (`apps/desktop/dist/`) and that it is a single arm64 `.dmg`.

## 5. Acceptance Criteria

- A documented build command produces an unsigned `.dmg`.
- The packaged app launches successfully after being installed to `/Applications`.
- The packaged app does **not** require `go` on PATH, the repo checkout, or a dev PATH (verify by launching outside the repo / in a clean shell environment).
- The bundled `skillbox-core` is executable, located **outside** `app.asar`, and resolved from `process.resourcesPath` (or equivalent).
- `server.ready` arrives from the bundled sidecar within the existing 10s timeout.
- The SQLite DB is created and migrated under `~/Library/Application Support/Astraler Skillbox/`.
- Smoke flow passes: host scan, skill list, project add + scan, symlink install + remove, and Dashboard render.
- Quitting the app leaves no orphaned `skillbox-core` process.
- Desktop typecheck, tests, and contract drift checks still pass.

## 6. Tests and Smoke Plan

Automated unit coverage:
- Extend `core-go-path` tests to cover the packaged branch: given a fake `process.resourcesPath`, the resolver returns `<resources>/core/skillbox-core`; the dev branch is unchanged.
- If the spawn command/args become a pure function of mode, add a small unit test asserting dev → `go run …` and packaged → bundled binary path.

Manual packaged smoke (run on the `.dmg` installed to `/Applications`):
1. Mount `.dmg`, drag app to `/Applications`, eject.
2. Launch from `/Applications` (double-click), ideally from a shell with a minimal PATH to prove no `go` dependency.
3. Confirm window opens and `server.ready` is received (no startup error banner). Confirm via logs that it came from the **packaged** sidecar (see Observable Startup Evidence below).
4. Confirm `~/Library/Application Support/Astraler Skillbox/skillbox.db` is created.
5. Run host scan; confirm skills appear in Skills Library (`skill.list`).
6. Add a project, run project scan.
7. Install a skill to the project via symlink, then remove it; confirm filesystem effect and DB state.
8. Open Dashboard; confirm it renders aggregated state.
9. Quit the app; confirm via `pgrep -fl skillbox-core` that no sidecar process remains.
10. Verify `test -x` on the bundled binary and that it sits outside `app.asar`.

### Observable Startup Evidence (bundled sidecar)

`server.ready` alone does not prove the sidecar was the **packaged** binary rather than a stray dev `go run`. To make the source observable, the existing `manager.ts` stderr lines are the evidence channel, and the spawn log line MUST include the resolved sidecar command/path so it is verifiable in packaged mode:

- On spawn, main logs `[manager] spawning Go core from <path>`. In packaged mode this MUST show the bundled binary path under `process.resourcesPath` (e.g. `…/Astraler Skillbox.app/Contents/Resources/core/skillbox-core`), **not** a `go run`/repo-relative path. (Implementation note: the current dev log prints the cwd; the packaged branch must log the actual resolved executable path so this check is meaningful.)
- On ready, main logs `[manager] Go core ready`.

How to capture these for a packaged `/Applications` launch (Electron main stderr is not visible in a normal double-click launch):

- **Launch from a terminal** to see stderr inline: `/Applications/Astraler\ Skillbox.app/Contents/MacOS/Astraler\ Skillbox` and observe the two `[manager]` lines.
- Or capture to a file: `… /Astraler\ Skillbox 2> /tmp/skillbox-packaged.log` and grep for `spawning Go core from` and `Go core ready`.
- Cross-check the running process: `pgrep -fl skillbox-core` should show the binary path inside the app bundle's `Contents/Resources/`, confirming the live sidecar is the packaged one.

Evidence is considered sufficient when the spawn log line points at the in-bundle `Resources/core/skillbox-core` path AND `Go core ready` follows AND the live process path matches the bundle. (A dedicated app log file under Application Support is out of scope for this spike; terminal/redirected stderr is the documented location.)

## 7. Risks and Mitigations

- **ASAR vs executable**: a binary packed into `app.asar` cannot be exec'd. Mitigation: `extraResources` keeps it outside ASAR; smoke step 10 verifies location and exec bit.
- **Executable bit stripped during copy**: Mitigation: explicit `chmod 0755` in the build hook; smoke verifies `test -x`; optional defensive `chmod` on first spawn.
- **Gatekeeper blocks an unsigned binary** (quarantine / "cannot be opened"): expected for unsigned builds. Mitigation: document the manual override (right-click → Open, or `xattr -dr com.apple.quarantine`) for the spike; the real fix is signing in 3B.
- **Architecture mismatch** (binary built for the wrong arch): Mitigation: pin `GOARCH=arm64` in the build hook and test on Apple Silicon; universal binary is explicitly a non-goal.
- **Stale bundled binary**: Mitigation: rebuild the sidecar in the prepackage hook every build; never commit the artifact.
- **Path drift between dev and packaged resolution**: Mitigation: single `app.isPackaged` branch with unit tests on both paths; keep lifecycle logic shared.
- **Orphaned sidecar on quit**: existing SIGTERM→3s→SIGKILL logic should hold, but a packaged binary (vs `go run`'s child go process) may behave differently. Mitigation: smoke step 9 explicitly checks for orphans.

## 8. Follow-up: Slice 3B Signing/Notarization Boundary

3A stops at an unsigned, locally produced `.dmg` that requires a manual Gatekeeper override. Slice 3B picks up from here and owns: Developer ID signing of the app and bundled sidecar, hardened runtime + entitlements, Apple notarization and stapling, and the Gatekeeper-clean launch experience. CI release automation, auto-update, universal binaries, and other platforms remain out of scope until their own slices.
