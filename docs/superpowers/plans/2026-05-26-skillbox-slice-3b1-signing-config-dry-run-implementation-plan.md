# Slice 3B1 Implementation Plan: macOS Signing Config + Entitlements + Dry-Run

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the credential-free portion of the signed/notarized macOS release path — electron-builder signing config (signed default), hardened-runtime entitlements files, a signed `package:mac` script, version bump, and docs — while preserving a working `package:mac:unsigned`. Prove via electron-builder's **own ad-hoc signing** (`-c.mac.identity=-`) that `mac.binaries` reaches the nested sidecar, without any Apple credentials.

**Spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3b1-signing-config-dry-run-design.md` (lead-approved, commit `3c68ed9`).

**Architecture:** One env-gated `electron-builder.yml` becomes the signed default (hardened runtime + entitlements + `mac.binaries` + `notarize: true`). The 3A unsigned command opts out via CLI/env overrides (`identity=null`, `hardenedRuntime=false`, `notarize=false`). No application code, schema, or contracts change.

**Starting state (verified):** `electron-builder.yml` is still the 3A unsigned config (`mac.identity: null`, no hardened runtime/entitlements/binaries/notarize). `package.json` `version` is `0.0.0`; scripts have `package:mac:unsigned` (relies on the YAML's `identity: null`) but no signed `package:mac`. No `apps/desktop/build/` dir exists.

---

## File Structure

- `apps/desktop/build/entitlements.mac.plist` — **Create.** Main-app hardened-runtime entitlements (JIT + unsigned executable memory).
- `apps/desktop/build/entitlements.mac.inherit.plist` — **Create.** Inherited entitlements for nested executables.
- `apps/desktop/electron-builder.yml` — **Modify.** Convert from 3A unsigned to signed default: drop `identity: null`; add `hardenedRuntime`, `gatekeeperAssess: false`, `entitlements`, `entitlementsInherit`, `mac.binaries: [Contents/Resources/core/skillbox-core]`, `notarize: true`. Keep `appId`, `productName`, `directories`, `files`, `asar`, `extraResources`, `category`, `target`.
- `apps/desktop/package.json` — **Modify.** Bump `version` to `0.1.0`; add signed `package:mac`; rewrite `package:mac:unsigned` to override `identity`/`hardenedRuntime`/`notarize` off.
- `SMOKE.md` — **Modify.** Add a "Signed Packaging Dry-Run (Slice 3B1)" section and a gated "Signed + Notarized Smoke (Slice 3B2 — requires Apple Developer ID)" section.
- `SCAFFOLD.md` — **Modify.** Document signed vs unsigned commands and the 3B2 credential prerequisites.

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core` and `scripts/build-core.mjs` stay exactly as 3A left them.

---

## Tasks

### Task 1: Entitlements plist files

**Files:**
- Create: `apps/desktop/build/entitlements.mac.plist`
- Create: `apps/desktop/build/entitlements.mac.inherit.plist`

- [ ] **Step 1: Create the main-app entitlements**

`apps/desktop/build/entitlements.mac.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.cs.allow-jit</key>
  <true/>
  <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
  <true/>
</dict>
</plist>
```

These are the minimal entitlements Electron's renderer (V8 JIT) needs under hardened runtime. **Do NOT** add `com.apple.security.cs.disable-library-validation`, `allow-dyld-environment-variables`, network, or Keychain entitlements — the spec marks those as "only if a verified failure proves necessary" (the sidecar is a static Go binary spawned as a separate process, so none are expected).

- [ ] **Step 2: Create the inherited entitlements**

`apps/desktop/build/entitlements.mac.inherit.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.cs.allow-jit</key>
  <true/>
  <key>com.apple.security.inherit</key>
  <true/>
</dict>
</plist>
```

- [ ] **Step 3: Lint both plists**

Run: `plutil -lint apps/desktop/build/entitlements.mac.plist apps/desktop/build/entitlements.mac.inherit.plist`
Expected: both report `OK`.

- [ ] **Step 4: Commit**

```bash
git add apps/desktop/build/entitlements.mac.plist apps/desktop/build/entitlements.mac.inherit.plist
git commit -m "feat(3b1): add hardened-runtime entitlements for mac signing"
```

---

### Task 2: Convert electron-builder.yml to the signed default

**Files:**
- Modify: `apps/desktop/electron-builder.yml`

- [ ] **Step 1: Rewrite the `mac` block (and drop `identity: null`)**

Replace the current `mac:` block. Final file:

```yaml
appId: com.astraler.skillbox
productName: Astraler Skillbox
directories:
  output: dist
files:
  - out/**/*
  - package.json
asar: true
extraResources:
  - from: resources/core/skillbox-core
    to: core/skillbox-core
mac:
  category: public.app-category.developer-tools
  hardenedRuntime: true
  gatekeeperAssess: false
  entitlements: build/entitlements.mac.plist
  entitlementsInherit: build/entitlements.mac.inherit.plist
  notarize: true
  binaries:
    - Contents/Resources/core/skillbox-core
  target:
    - target: dmg
      arch:
        - arm64
```

Notes:
- `identity` is **removed** (auto-discovery). The signed default expects a Developer ID identity to exist — that only happens in 3B2. In 3B1 we never run the bare signed path; the dry-run forces `-c.mac.identity=-` and the unsigned command forces `identity=null`.
- `mac.binaries: [Contents/Resources/core/skillbox-core]` — relative to the built `.app` (per spec §3.1). Task 6 confirms electron-builder 26.8.1 actually signs the sidecar at this path; if the packed layout differs, adjust to the implementation-verified equivalent and re-run.
- `notarize: true` is inert without credentials; the dry-run and unsigned commands both pass `-c.mac.notarize=false` to be explicit.

- [ ] **Step 2: Readability/sanity check (NOT a config validation)**

This is a quick visual sanity check, not a validation — the **only** authoritative parse/config-load check is the electron-builder run in Task 6.

Run: `git --no-pager diff apps/desktop/electron-builder.yml` and eyeball that the `mac:` block matches Step 1 (keys present, indentation consistent, `identity: null` removed). Optionally confirm indentation with `grep -n "  " apps/desktop/electron-builder.yml`.
Expected: the diff reflects exactly the Step 1 changes and nothing else.

- [ ] **Step 3: Commit**

```bash
git add apps/desktop/electron-builder.yml
git commit -m "feat(3b1): make electron-builder.yml signed-default with hardened runtime + sidecar signing"
```

---

### Task 3: package.json — version bump + signed/unsigned scripts

**Files:**
- Modify: `apps/desktop/package.json`

- [ ] **Step 1: Bump the version**

```json
  "version": "0.1.0",
```

(First meaningful release version; 3A artifacts were `0.0.0`. No reason found to keep `0.0.0`.)

- [ ] **Step 2: Rewrite `package:mac:unsigned` and add signed `package:mac`**

In `"scripts"`, replace the current `package:mac:unsigned` line and add `package:mac`:

```json
    "package:mac": "pnpm build:core && pnpm build && electron-builder --mac dmg",
    "package:mac:unsigned": "pnpm build:core && pnpm build && CSC_IDENTITY_AUTO_DISCOVERY=false electron-builder --mac dmg -c.mac.identity=null -c.mac.hardenedRuntime=false -c.mac.notarize=false",
```

Rationale:
- `package:mac` (signed) is the 3B2 entrypoint — it signs + notarizes only when Developer ID + notarization credentials are present in the environment. Running it on this machine (no cert) is **expected to fail or skip**; it is **not** a 3B1 gate.
- `package:mac:unsigned` now overrides the signed defaults: `CSC_IDENTITY_AUTO_DISCOVERY=false` + `identity=null` (skip signing) **and** `hardenedRuntime=false` (hardened runtime on an unsigned build is invalid) + `notarize=false`. The inline env-var prefix works because packaging is macOS-only and scripts run via the shell.

- [ ] **Step 3: Confirm scripts are well-formed**

Run: `(cd apps/desktop && node -e "const p=require('./package.json'); console.log(p.version, Object.keys(p.scripts).filter(s=>s.startsWith('package:')))")`
Expected: `0.1.0 [ 'package:mac', 'package:mac:unsigned' ]`.

- [ ] **Step 4: Commit**

```bash
git add apps/desktop/package.json
git commit -m "feat(3b1): add signed package:mac, override unsigned path, bump version to 0.1.0"
```

---

### Task 4: SMOKE.md — dry-run + gated 3B2 sections

**Files:**
- Modify: `SMOKE.md`

- [ ] **Step 1: Append a 3B1 dry-run section**

After the existing "Packaged macOS DMG Smoke (Slice 3A)" section, append the block below (shown inside a four-backtick fence so its nested ```sh fences are unambiguous — insert only the inner Markdown, not the four-backtick wrapper):

````markdown
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
````

- [ ] **Step 2: Append a gated 3B2 section**

Append the block below (four-backtick fence wraps the nested ```sh fence; insert only the inner Markdown):

````markdown
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
spctl -a -vvv -t open "$APP"                # expect: accepted, source=Notarized Developer ID
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
````

- [ ] **Step 3: Commit**

```bash
git add SMOKE.md
git commit -m "docs(3b1): add signed dry-run smoke and gated 3b2 notarization smoke"
```

---

### Task 5: SCAFFOLD.md — packaging commands + 3B2 prerequisites

**Files:**
- Modify: `SCAFFOLD.md`

- [ ] **Step 1: Extend the "Packaging" section**

Update the existing "Packaging (Slice 3A — unsigned macOS DMG)" section to document all three flows. Append (or restructure into) the following:

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add SCAFFOLD.md
git commit -m "docs(3b1): document signed/unsigned packaging commands and 3b2 prerequisites"
```

---

### Task 6: Full verification gauntlet (dry-run)

No code changes — run every gate and the 3B1 dry-run. Commit nothing unless a gate forces a fix (e.g. a `mac.binaries` path correction).

- [ ] **Step 1: Go tests** — `(cd core-go && go test ./...)` → all PASS (sanity; nothing Go changed).
- [ ] **Step 2: Frontend typecheck** — `(cd apps/desktop && pnpm typecheck)` → PASS.
- [ ] **Step 3: Frontend unit tests** — `(cd apps/desktop && pnpm test --run)` → PASS.
- [ ] **Step 4: Contract drift** — `(cd apps/desktop && pnpm check:contracts-drift)` → PASS (no contract changes). If it reports drift: **stop**, run `git diff` (and `git status`) to identify what changed. Only revert changes **this slice made** to contract/schema/generated files; if the drift looks pre-existing or unrelated to 3B1, do **not** revert — ask the lead whether it predates this slice.
- [ ] **Step 5: electron-vite build** — `(cd apps/desktop && pnpm build)` → builds `out/main`, `out/preload`, `out/renderer`.
- [ ] **Step 6: Entitlements lint** — `plutil -lint apps/desktop/build/entitlements.mac.plist apps/desktop/build/entitlements.mac.inherit.plist` → both `OK`.
- [ ] **Step 7: Unsigned regression** — `(cd apps/desktop && pnpm package:mac:unsigned)` → produces `dist/Astraler Skillbox-0.1.0-arm64.dmg`.
- [ ] **Step 8: Ad-hoc signing proof (the key 3B1 check)** — run the "mac.binaries proof" block from SMOKE.md. `codesign -dvvv "$SIDE"` must show a signature **before** any manual codesign, and `codesign --verify --deep --strict` on the `.app` must pass. If the sidecar is unsigned, correct the `mac.binaries` path and re-run.
- [ ] **Step 9: Confirm NOT attempted** — verify the dry-run did not submit to notarytool (no network notarization step) and `pnpm package:mac` (bare signed) was not run on this credential-less machine.

---

## Acceptance Criteria (lead-required — verified in Task 6 / SMOKE.md §3B1)

- [ ] `package:mac:unsigned` still produces a launchable `0.1.0` arm64 DMG (3A regression guard).
- [ ] Both entitlements plists pass `plutil -lint`.
- [ ] Signed-default `electron-builder.yml` packs under an ad-hoc identity with no config error.
- [ ] electron-builder's own ad-hoc signing signs `Contents/Resources/core/skillbox-core` — verified by `codesign -dvvv` **before** any manual codesign; `codesign --verify --deep --strict` on the `.app` passes.
- [ ] No notarytool submission occurred (no credentials present; `-c.mac.notarize=false` on dry-run/unsigned).
- [ ] All gates green: `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change (contract-drift clean).

---

## Out of Scope — MUST NOT Touch

- **Product features / UI / RPC behavior** — no renderer, main, or `core-go` logic changes.
- **Schema / migrations / JSON-RPC contracts** — `shared/api-contracts`, `shared/generated`, `core-go/migrations` untouched; contract-drift must stay clean.
- **3B2 work** — no Developer ID cert handling, no real notarization (`mac.notarize` stays inert/off in 3B1), no stapling, no Gatekeeper-clean download test, no credential files committed or referenced as present.
- **`build:core` / `scripts/build-core.mjs`** — unchanged from 3A; the sidecar is built exactly as before and signed by electron-builder, never by the build script.
- **CI release automation, auto-update / `electron-updater`** — not added.
- **Universal binary, `amd64`/`lipo`, Windows, Linux, `.pkg`/Mac App Store** — not added.
- **Secrets** — never commit certificates, `.p12`, `.p8`, passwords, Team ID, or Apple IDs.

---

## Cleanup Expectations

- Build outputs stay gitignored (3A already ignores `apps/desktop/dist/` and `apps/desktop/resources/core/`). Confirm `git status --porcelain apps/desktop/dist apps/desktop/resources/core` is empty after packaging.
- Ad-hoc-signed `dist/mac-arm64/` output is a throwaway dry-run artifact — do not commit it.
- The only committed changes are: two plist files, `electron-builder.yml`, `package.json`, `SMOKE.md`, `SCAFFOLD.md`.
- If Task 6 forces a `mac.binaries` path fix, amend Task 2's config and re-verify; commit the corrected config with a clear message.

---

## Notes for the Executing Agent

- This slice changes **no** application code. If `pnpm check:contracts-drift` reports drift, **stop** and inspect `git diff` / `git status`. Revert only the changes **this slice introduced**; never blanket-revert the worktree, since it may contain unrelated or pre-existing user changes. If the drift is not from this slice, ask the lead before touching it.
- `mac.binaries` path is the single most likely thing to get wrong. Treat Task 6 Step 8 as the gate: the sidecar must be signed by electron-builder itself, observed **before** any manual `codesign`.
- Do not "fix" a missing sidecar signature with `codesign -s - --deep --force` — that masks the real config bug. Fix the path instead.
- Running the bare signed `pnpm package:mac` on a machine without a Developer ID cert is expected to fail and is **not** a gate. The 3B1 proof is the ad-hoc (`-c.mac.identity=-`) path only.
