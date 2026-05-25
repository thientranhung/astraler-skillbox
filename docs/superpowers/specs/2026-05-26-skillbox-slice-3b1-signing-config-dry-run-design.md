# Slice 3B1: macOS Signing Config + Entitlements + Dry-Run Readiness — Design

Date: 2026-05-26
Status: draft (for lead review)
Depends on: Slice 3A (unsigned macOS arm64 DMG, lead-approved)
Blocks: Slice 3B2 (real Developer ID signing + notarization + stapling)

## 0. PM Decision Recorded

3B is split into two slices:

- **3B1 (this spec):** signing configuration, entitlements, scripts, docs, and dry-run validation that require **no Apple credentials**. Fully unblocked.
- **3B2 (deferred):** real Developer ID signing, notarization, and stapling. **Blocked** until the user provides: Apple Developer Program membership, a Developer ID Application certificate + private key, the Team ID, and notarization credentials.

## 1. Goal and Non-Goals

### Goal
Land everything needed for a signed + notarized macOS customer release **except the parts that require Apple credentials**: the electron-builder signing configuration, hardened-runtime entitlements files, a signed packaging script, documentation, and a dry-run verification flow that proves the bundle is structurally signable and that the nested Go sidecar is reached by the signing step. After 3B1, 3B2 should be "plug in credentials and run."

### Non-Goals
- Real Developer ID signing, Apple notarization, or stapling (3B2).
- Any Apple-account-dependent step (cert install, notarytool submission, Gatekeeper-clean download test).
- Auto-update / `electron-updater`, CI release automation, Windows/Linux, universal (`amd64`/`lipo`) binaries, `.pkg`/Mac App Store distribution — each is a later slice.
- Product, schema, or JSON-RPC contract changes.
- Removing or regressing the 3A unsigned path.

## 2. Why 3B1 Is Valuable Without Apple Credentials

- **Isolates the real blocker.** Only notarization and Developer ID signing need an Apple account. Everything else (config shape, entitlements, sidecar-signing wiring, scripts, docs) can be authored and verified now, so 3B2 becomes a short, low-risk slice.
- **De-risks the #1 technical hazard early.** A nested binary bundled via `extraResources` is the most likely cause of a notarization rejection (any unsigned Mach-O fails notarization). 3B1 proves — via electron-builder's **own ad-hoc signing path** (`-c.mac.identity=-`) — that `mac.binaries` actually reaches `Contents/Resources/core/skillbox-core` in the packed bundle, without needing a Developer ID cert.
- **Keeps the unsigned spike alive.** 3B1 makes signing the default config while preserving a working `package:mac:unsigned`, so the team retains a buildable artifact throughout.
- **Reviewable in isolation.** Config + entitlements + dry-run is a small, self-contained diff a lead can fully assess without an Apple account on the review machine.

## 3. Proposed electron-builder Config Changes (in principle)

Single config file, env-gated — do **not** fork into two YAMLs. `electron-builder.yml` becomes the **signed default**; the 3A command opts out explicitly.

### 3.1 Signed default (`electron-builder.yml`)
Add, in principle (exact YAML fixed at implementation):
- `mac.hardenedRuntime: true` — required for notarization.
- `mac.gatekeeperAssess: false` — don't run a local Gatekeeper assessment mid-build (it would fail before notarization exists).
- `mac.entitlements: build/entitlements.mac.plist`
- `mac.entitlementsInherit: build/entitlements.mac.inherit.plist`
- `mac.binaries: [Contents/Resources/core/skillbox-core]` — force-sign the nested sidecar (see §5). **Path note:** in electron-builder 26.8.1, relative `mac.binaries` entries resolve from the built `.app` path (existing paths are treated as external artifacts). The entry must therefore point at the **installed bundle** sidecar (`Contents/Resources/core/skillbox-core`), **not** the staging source `apps/desktop/resources/core/skillbox-core` and **not** a `.app/resources/...` variant. The exact form (`Contents/Resources/...` vs an implementation-verified equivalent) is confirmed against the actual packed output during implementation.
- `mac.notarize: true` — Electron Builder's built-in notarization (notarytool). Inert in 3B1 because no credentials are present in the environment; it only engages in 3B2. Custom `afterSign` notarization is **not** used — built-in `mac.notarize` is the supported path.
- `mac.identity` left to keychain auto-discovery (a Developer ID Application identity). No `identity` key in the committed file.
- Keep `appId: com.astraler.skillbox`, `category`, `target: dmg / arch: [arm64]` from 3A.

### 3.2 Preserving the 3A unsigned path (hard requirement)
The unsigned command must keep producing a runnable DMG. Because hardened runtime on an **unsigned** build is invalid, the unsigned command disables signing **and** hardened runtime together (per electron-builder mac guidance: `identity: null` skips signing, and hardened runtime must be off when signing is disabled):

- `package:mac:unsigned` → run electron-builder with `CSC_IDENTITY_AUTO_DISCOVERY=false` and CLI overrides `-c.mac.identity=null -c.mac.hardenedRuntime=false -c.mac.notarize=false`.
- `package:mac` (new, signed) → plain `electron-builder --mac dmg`; signs + notarizes only when credentials are present in the environment (3B2).

This keeps one source-of-truth config while guaranteeing the 3A artifact stays valid.

### 3.3 Versioning
Bump `version` off `0.0.0` (suggest `0.1.0`) so signed/unsigned artifacts get a meaningful name. Cosmetic but worth doing in 3B1.

## 4. Entitlements Files and Minimal Rationale

Two plist files under `apps/desktop/build/` (electron-builder's conventional `build/` resources dir):

### 4.1 `build/entitlements.mac.plist` (main app)
Minimal set Electron's renderer (V8 JIT) needs under hardened runtime:
- `com.apple.security.cs.allow-jit` — V8 JIT.
- `com.apple.security.cs.allow-unsigned-executable-memory` — V8 codegen.

### 4.2 `build/entitlements.mac.inherit.plist` (nested executables)
Inherited entitlements for nested binaries/helpers:
- `com.apple.security.cs.allow-jit` (+ inherit), per the standard Electron inherit template.

### 4.3 Rationale and what we deliberately omit
- The sidecar is a **pure-static Go binary** (`CGO_ENABLED=0`, `modernc.org/sqlite`): no dylib loads, no JIT. It needs hardened-runtime signing but **no special entitlements**.
- It is **spawned as a separate process** by `manager.ts`, not loaded into Electron. A plain `exec` of another signed binary is allowed under hardened runtime — no extra parent entitlement required.
- **`com.apple.security.cs.disable-library-validation` is intentionally NOT included.** Library validation only triggers when loading code into a process; with a separate static sidecar signed under the same Team ID there is no trigger. Add it **only if a verified failure** (e.g. a notarization log or a runtime crash in 3B2) proves it necessary. Omitting it is the safer, more-restrictive default.
- Likewise we omit `allow-dyld-environment-variables`, `disable-executable-page-protection`, and Keychain/network entitlements unless a concrete need is verified.

All plist files must pass `plutil -lint`.

## 5. Sidecar Signing Risk and Coverage

**Risk:** binaries placed via `extraResources` can be missed by the bundle signing walk; an unsigned nested Mach-O makes notarization fail in 3B2. The sidecar lives at `Contents/Resources/core/skillbox-core` — inside the bundle but not in a standard auto-signed location.

**Coverage in 3B1 (no Apple cert needed):**
- Declare `mac.binaries: [Contents/Resources/core/skillbox-core]` (path resolution per §3.1) so electron-builder explicitly signs it (inside-out: sidecar first, then the `.app`).
- Do **not** sign the sidecar inside `build:core` (Go already ad-hoc-signs arm64 output; electron-builder re-signs it). The build script stays as-is from 3A.
- **Dry-run proof — must exercise electron-builder's own signing, not a manual rescue.** Run the packaging with an **ad-hoc identity** (`electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`), letting electron-builder's signing path (`@electron/osx-sign`) sign the sidecar via `mac.binaries`. Then, **before any manual `codesign`**, inspect the sidecar with `codesign -dvvv` and confirm it carries an ad-hoc signature, and that `codesign --verify --strict` passes. Only this proves `mac.binaries` reached the sidecar.
- **Why not a manual deep sign:** a post-hoc `codesign -s - --deep --force "$APP"` would re-sign the sidecar after packaging and could *rescue* a `mac.binaries` misconfiguration, masking the exact failure 3B1 is meant to catch. The dry-run therefore inspects the binary as electron-builder left it; manual signing, if used at all, happens only after the verification check has been recorded.

**Coverage in 3B2:** the same path, but with a Developer ID identity, so `codesign -dvvv` then shows `Authority=Developer ID Application: … (TEAMID)` and `flags=…(runtime)` on both the app and the sidecar.

## 6. Dry-Run Acceptance Criteria (must pass on this machine, no Apple credentials)

- [ ] **Unsigned path intact:** `package:mac:unsigned` still produces a launchable `apps/desktop/dist/Astraler Skillbox-<version>-arm64.dmg`; the 3A functional smoke (host scan, skill list, project add/scan, symlink install/remove, Dashboard, clean quit, no orphan sidecar) still passes.
- [ ] **Entitlements valid:** `plutil -lint build/entitlements.mac.plist` and the inherit plist both report `OK`.
- [ ] **Config parses:** electron-builder accepts the signed config (build reaches packing) with `CSC_IDENTITY_AUTO_DISCOVERY=false`, i.e. no signing attempted, no config error.
- [ ] **Sidecar signed by electron-builder (ad-hoc identity):** after packaging with `-c.mac.identity=- -c.mac.notarize=false`, and **before any manual `codesign`**, `codesign -dvvv` shows a signature on `Contents/Resources/core/skillbox-core` and `codesign --verify --deep --strict --verbose=2` on the `.app` passes. (Proves `mac.binaries` reached the sidecar via electron-builder's own signing — not a post-hoc rescue — without a Developer ID cert.)
- [ ] **No notarization attempted:** with no credentials in env, the build does not call notarytool and does not fail for missing credentials.
- [ ] **All existing gates green:** `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] **No out-of-scope drift:** no JSON-RPC contract, schema, or product changes.

## 7. 3B2 Prerequisites and Handoff Criteria

3B2 is **not** implemented here. It may begin once the user provides:

1. Apple Developer Program membership (active).
2. Developer ID **Application** certificate + private key in the login keychain (or `.p12` + `CSC_KEY_PASSWORD`).
3. **Team ID** (10-char).
4. Notarization credentials — **either** an App Store Connect **API key** (`.p8` + Key ID + Issuer ID), **or** Apple ID + app-specific password + Team ID. API key preferred.
5. Confirmation of `.dmg`-only distribution (so no Developer ID **Installer** cert / `.pkg`).
6. A chosen release version string.

**Handoff criteria from 3B1 → 3B2:** all §6 dry-run criteria pass and the spec/config is lead-approved. At that point 3B2 only adds: provide credentials via env, run `package:mac`, and execute the 3B2 smoke (§8.2).

## 8. Smoke / Verification Commands

### 8.1 3B1 dry-run (runnable now)
```sh
cd apps/desktop

# Entitlements lint
plutil -lint build/entitlements.mac.plist
plutil -lint build/entitlements.mac.inherit.plist

# Unsigned 3A path still works (regression guard)
pnpm package:mac:unsigned
ls "dist/Astraler Skillbox-0.1.0-arm64.dmg"

# Prove mac.binaries reaches the sidecar via electron-builder's OWN signing,
# using an ad-hoc identity (no Developer ID cert needed). Do NOT manually sign first.
pnpm build:core && pnpm build && \
  electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false

APP="dist/mac-arm64/Astraler Skillbox.app"      # adjust to electron-builder's unpacked output
SIDE="$APP/Contents/Resources/core/skillbox-core"
# Inspect BEFORE any manual codesign — this is the real mac.binaries proof:
codesign -dvvv "$SIDE"                          # expect: ad-hoc signature present (Signature=adhoc)
codesign --verify --deep --strict --verbose=2 "$APP"   # expect: valid on disk
# (No manual `codesign -s - --deep` rescue — that would mask a mac.binaries misconfig.)

# Gates
( cd ../../core-go && go test ./... )
pnpm typecheck && pnpm test --run && pnpm check:contracts-drift && pnpm build
```

### 8.2 3B2 (documented now, run only with Apple credentials)
```sh
APP="/Applications/Astraler Skillbox.app"
SIDE="$APP/Contents/Resources/core/skillbox-core"
DMG="apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg"

# Developer ID signature + hardened runtime (expect Authority=Developer ID Application + flags=...(runtime))
codesign -dvvv "$APP"
codesign -dvvv "$SIDE"
codesign --verify --deep --strict --verbose=2 "$APP"
codesign -d --entitlements - "$APP"

# Gatekeeper + notarization (notarytool is the only accepted path; altool retired 2023-11-01)
spctl -a -vvv -t open "$APP"                # expect: accepted, source=Notarized Developer ID
xcrun stapler validate "$APP"
xcrun stapler validate "$DMG"

# Simulate a downloaded copy WITHOUT mutating /Applications: copy to a temp dir,
# mark it quarantined, launch, then clean up. Must open with NO right-click / xattr override.
TMP="$(mktemp -d)"
cp -R "$APP" "$TMP/"
TMP_APP="$TMP/Astraler Skillbox.app"
xattr -w com.apple.quarantine "0081;0;Safari;" "$TMP_APP"   # mimic a Safari download
xattr -p com.apple.quarantine "$TMP_APP"                    # confirm attr is set
open "$TMP_APP"                                             # expect: launches, NO Gatekeeper prompt
# verify it ran, then quit, then clean up the temp copy
pgrep -fl skillbox-core                                     # sidecar from the temp bundle is live
osascript -e 'quit app "Astraler Skillbox"'
rm -rf "$TMP"

# Process identity / no orphan (from 3A)
pgrep -fl skillbox-core                                     # expect: nothing after quit
```

## 9. Risks and Mitigations (3B1 scope)

- **Ad-hoc dry-run is not a notarization proof.** It only proves signability + sidecar reach. True Gatekeeper behavior is a 3B2 concern. Mitigation: §6 explicitly scopes the dry-run to structural signability; §8.2 carries the real checks to 3B2.
- **electron-builder unpacked output path drift.** The `.app` path used for the ad-hoc check (`dist/mac-arm64/…`) may differ by version/config. Mitigation: resolve the actual path from the build output at implementation; the criterion is "the packed `.app`", not a literal path.
- **Hardened runtime on unsigned build.** Invalid combination. Mitigation: the unsigned command forces both `identity=null` and `hardenedRuntime=false` (§3.2).
- **`mac.notarize: true` firing unexpectedly.** Could try to notarize during a dry-run. Mitigation: no credentials in env → no submission; the dry-run additionally passes `-c.mac.notarize=false` to be explicit.

## 10. Files Expected to Change in 3B1 (for the implementation plan, not this pass)

- `apps/desktop/electron-builder.yml` — add hardened runtime, entitlements, `mac.binaries`, `mac.notarize` (signed default).
- `apps/desktop/build/entitlements.mac.plist` — new (main app entitlements).
- `apps/desktop/build/entitlements.mac.inherit.plist` — new (inherited entitlements).
- `apps/desktop/package.json` — add `package:mac` (signed); update `package:mac:unsigned` to disable hardened runtime; bump `version`.
- `SMOKE.md` — add 3B1 dry-run section; document the 3B2 smoke (gated "requires Apple Developer ID").
- `SCAFFOLD.md` — document signed vs unsigned packaging commands and the 3B2 credential requirements.

No code in `electron/main`, `renderer`, or `core-go` is expected to change in 3B1.
