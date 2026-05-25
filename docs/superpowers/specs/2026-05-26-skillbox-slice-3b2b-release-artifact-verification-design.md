# Slice 3B2B: macOS Release Artifact Verification Harness — Design

Date: 2026-05-26
Status: draft (for lead review)
Depends on: Slice 3B1 (signed-default config + ad-hoc dry-run, lead-approved), Slice 3B2A (release preflight / credential doctor, lead-approved)
Relates to: Slice 3B2 (real Developer ID signing + notarization + stapling — blocked on Apple credentials)

## 0. PM Decision Recorded

3B2A made the **pre-build** prerequisites executable (`pnpm release:mac:check` runs *before* `pnpm package:mac`). The symmetric **post-build** question — "is this built `.app`/`.dmg` actually customer-ready?" — is still only prose (3B1 §8.2, SMOKE.md "Signed + Notarized Smoke"). 3B2B turns that checklist into a deterministic gate.

It is credential-free to author and unit-test, and it can run **partially now** against the 3B1 ad-hoc dry-run bundle (`--allow-adhoc`), so it lands without Apple credentials. When credentials arrive, 3B2 shrinks to: set env → `pnpm package:mac` → `pnpm release:mac:verify` in release mode → done.

3B2B is **read-only with one bounded local side effect** (mounting/detaching a DMG read-only). It never builds, signs, notarizes, staples, calls the network, or mutates the keychain.

## 1. Purpose and Non-Goals

### Purpose
Add `pnpm release:mac:verify [path]` — a command that inspects an already-built `.app` or `.dmg` and reports whether it meets the bar for a customer-ready notarized release: Developer ID signature on **both** the app and the nested Go sidecar, hardened runtime, Gatekeeper acceptance, and a stapled notarization ticket. It complements 3B2A: 3B2A checks *can we build a good artifact?*; 3B2B checks *did we?*

### Non-Goals
- **No building, signing, notarizing, or stapling** — it inspects an existing artifact; it never runs `build:core`, `electron-vite build`, `electron-builder`, `codesign -s`, `notarytool`, or `stapler staple`.
- **No network / Apple online calls** — no notarytool history lookup, no revocation check.
- **No keychain mutation** — it reads artifacts, not the keychain. (Unlike 3B2A it does not even call `security find-identity`.)
- **No product / RPC / schema / migration / contract changes.**
- **Not a substitute for the manual launch smoke** — the quarantine-launch / "no Gatekeeper prompt" step (SMOKE §8.2) stays manual; 3B2B verifies signature/notarization/stapling facts, not GUI launch.

## 2. Command / Input Resolution

- **Command:** `pnpm release:mac:verify [path]` (added to `apps/desktop/package.json` `scripts`), invoked from `apps/desktop/` like the other repo scripts. The script resolves repo paths relative to its own location so cwd does not matter.
- **Input resolution (check S1):**
  - **Explicit `.app`** → verify that bundle directly; DMG-stapling (ST2) is reported INFO (not checked — no DMG supplied).
  - **Explicit `.dmg`** → mount read-only, verify the contained `.app`, and additionally verify the DMG's own stapled ticket (ST2).
  - **No argument** → auto-discover in `apps/desktop/dist/`: exactly **one** `*.dmg` → use it; **zero** → FAIL (S1) with remediation to build or pass a path; **more than one** → FAIL (S1) listing the matches and requiring an explicit path (never guess).
- A path that does not exist, or is neither a `.app` directory nor a `.dmg` file → FAIL (S1).

## 3. DMG Strategy (read-only mount)

When the input is a `.dmg`, the shell mounts it read-only, inspects the contained app, and always detaches:

```sh
hdiutil attach -readonly -nobrowse -mountpoint "$MNT" "$DMG"
# locate exactly one *.app under "$MNT"; verify it in place (read-only mount)
hdiutil detach "$MNT"        # fallback: hdiutil detach -force "$MNT"
```

- The mountpoint is a fresh `mkdtemp` dir; detach runs in a `finally` so a mid-verify error still unmounts. Detach failure retries once with `-force`, then surfaces as a non-zero script error (with a clear message) so we never leak a mounted volume silently.
- **This is the one allowed local side effect.** It is documented as such. It is still forbidden to build, sign, notarize, staple, hit the network, or mutate the keychain.

**Why mount over inspecting an adjacent extracted `.app`:** the `.dmg` is the artifact customers actually download, and (a) verifying the app *as it ships inside the DMG* is the only way to confirm the shipped copy is signed/notarized, and (b) the DMG carries its own stapled ticket (ST2) that an extracted app cannot show. A read-only mount guarantees zero mutation while giving us the true distributed bytes; an adjacent `.app` could diverge from DMG contents and cannot verify DMG stapling. (If a bare `.app` is all that is available, the explicit-`.app` path above still works, with ST2 reported INFO.)

## 4. Architecture (pure parser + pure evaluator + thin IO shell)

Mirrors 3B2A's "pure core + thin IO shell" discipline. Three modules under `apps/desktop/scripts/`:

- **`release-mac-verify.parse.mjs` (pure)** — turns raw tool text into structured signals. No process/fs/env. Functions:
  - `parseCodesign(text)` → `{ adhoc, developerId, teamId, hardenedRuntime }`
  - `parseSpctl(text, exitCode)` → `{ accepted, source }`
  - `parseStapler(text, exitCode)` → `{ stapled }`
- **`release-mac-verify.lib.mjs` (pure)** — `evaluate(signals)` takes the parsed signals + `mode` + `expectedTeamId` + input metadata and returns `{ results: CheckResult[], missing, exitCode }`, plus `render(results, missing)` (reuses 3B2A's status-token + grouped-category output shape). No process/fs/env.
- **`release-mac-verify.mjs` (thin IO shell)** — resolves the input (§2), mounts the DMG if needed (§3), spawns `codesign` / `spctl` / `stapler`, captures **both stdout and stderr** (these tools write to stderr) and the **exit code**, feeds raw text+exit to the parser, runs `evaluate`, prints `render`, sets the exit code. Not unit-tested directly (covered by the SMOKE line).

`CheckResult` reuses 3B2A's shape: `{ id, category, status: "PASS"|"FAIL"|"WARN"|"INFO", message, remediation? }`.

## 5. Signals and Parsing Rules

**Exit code is the primary signal; markers classify.** A check FAILs when the relevant command's exit code indicates failure **or** a required marker is absent. Parsed markers (ad-hoc vs Developer ID, hardened runtime, Team ID, Gatekeeper source) are used to *classify* the result, not to override a non-zero exit. Capture stderr too — `codesign -dvvv`, `spctl`, and `stapler` routinely write their human-readable output there.

Stable markers parsed (substring/anchored regex, never positional):
- `Signature=adhoc` → ad-hoc signature.
- `Authority=Developer ID Application` → Developer ID signing authority.
- `TeamIdentifier=XXXXXXXXXX` → Team ID (value `not set` ⇒ absent).
- `flags=…(runtime)` → hardened runtime enabled.
- `source=Notarized Developer ID` (from `spctl`) → notarized Gatekeeper source; plus `accepted`/`rejected`.
- stapler success marker (`The validate action worked!`) / failure marker (`does not have a ticket stapled`) — corroborated by exit code.

Commands the shell runs (read-only):
- App: `codesign -dvvv "$APP"`, `codesign --verify --deep --strict --verbose=2 "$APP"`, `spctl -a -vvv -t exec "$APP"`, `xcrun stapler validate "$APP"`.
- Sidecar (`$APP/Contents/Resources/core/skillbox-core`): `codesign -dvvv "$SIDE"`, `codesign --verify --strict --verbose=2 "$SIDE"` (direct — **not** relying on the app's `--deep` walk).
- DMG (when input is a `.dmg`): `xcrun stapler validate "$DMG"`.

> Note: `spctl -t exec` is the correct assessment type for an executable app bundle. The 3B1 SMOKE used `-t open`; SMOKE.md is aligned to `-t exec` as part of this slice's doc update.

## 6. Check Table

| ID | Category | Check | Release-mode PASS | `--allow-adhoc` PASS |
|----|----------|-------|-------------------|----------------------|
| S1 | input | Input resolved to exactly one `.app`/`.dmg` | resolved | resolved |
| APP1 | app | `codesign --verify --deep --strict` exit 0 | yes | yes |
| APP2 | app | Signature class | `Authority=Developer ID Application` present, **not** ad-hoc | ad-hoc **or** Developer ID accepted |
| APP3 | app | Hardened runtime (`flags=…(runtime)`) | present | present |
| APP4 | app | `TeamIdentifier` present (and `== SKILLBOX_EXPECTED_TEAM_ID` if set) | present/match | INFO only (ad-hoc has none) |
| SID1 | sidecar | Sidecar exists at `Contents/Resources/core/skillbox-core` | present | present |
| SID2 | sidecar | Sidecar `codesign --verify --strict` exit 0 (direct) | yes | yes |
| SID3 | sidecar | Sidecar signature class | Developer ID, not ad-hoc | ad-hoc or Developer ID |
| SID4 | sidecar | Sidecar hardened runtime | present | present |
| SID5 | sidecar | Sidecar `TeamIdentifier` present (and `==` expected if set) | present/match | INFO only |
| GK1 | gatekeeper | `spctl` accepted + `source=Notarized Developer ID` | accepted + notarized | rejection/unsigned ⇒ **INFO** |
| ST1 | staple | `stapler validate "$APP"` succeeds | stapled | missing ⇒ **INFO** |
| ST2 | staple | `stapler validate "$DMG"` succeeds (only when DMG input) | stapled | missing ⇒ **INFO**; app-only input ⇒ INFO |

Statuses follow 3B2A: **FAIL** forces a non-zero exit; **WARN/INFO** do not. The report ends with a "Missing for a customer-ready release:" section listing each FAIL with a one-line remediation.

## 7. Modes

- **Release mode (default):** the artifact must be genuinely shippable. APP1–APP4, SID1–SID5, GK1, ST1 are PASS-required; ST2 is PASS-required when a `.dmg` was supplied and INFO when input was a bare `.app` (with a prominent note that full verification should target the `.dmg`). Any FAIL ⇒ exit 1.
- **`--allow-adhoc` (dry-run, for the 3B1 ad-hoc artifact):** intended for the bundle produced by `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`. Exit 0 requires only that app + sidecar **codesign verification (APP1, SID1, SID2) and hardened-runtime (APP3, SID4)** pass and the signature is at least ad-hoc (APP2, SID3). Because an ad-hoc bundle is neither notarized nor stapled and has no Team ID, **GK1, ST1, ST2, APP4, SID5 are reported INFO, never WARN/FAIL** in this mode. This proves `mac.binaries` reached the sidecar (the #1 notarization hazard) without any credentials.

The two modes share one evaluator; `mode` only changes whether notarization/stapling/Team-ID gaps are FAIL (release) or INFO (`--allow-adhoc`).

## 8. Optional Team ID Assertion

- **Identity-agnostic by default:** with no env set, release mode requires only that a Team ID is *present* (APP4/SID5) — it does not pin a specific value.
- **`SKILLBOX_EXPECTED_TEAM_ID`:** when set, release mode FAILs APP4/SID5 if the app's or sidecar's `TeamIdentifier` differs from it (catches a wrong-cert build). The env var holds a Team ID, which is **not a secret** (it is embedded in every shipped notarized app and visible via `codesign -dvvv`), so it may appear in output. It has no effect in `--allow-adhoc` mode.

## 9. Secret Hygiene and Side-Effect Boundaries

- **No secrets are involved.** 3B2B reads built artifacts, not credentials — no passwords, `.p8`, `.p12`, or env credential values are read or printed. Identity names and Team IDs surfaced from `codesign -dvvv` are non-secret (per 3B2A §6). The script does not dump entitlements blobs or full env.
- **Side effects: exactly one, bounded** — read-only `hdiutil attach`/`detach` of a supplied DMG (§3). The script otherwise only reads. It does **not**: build, sign, notarize, staple, run `build:core`/`electron-vite build`/`electron-builder`, make any network request, call notarytool/any Apple service, or mutate the keychain or any file outside its own temp mountpoint.

## 10. Output and Exit Codes

Human-readable, grouped by category, one line per check with a leading status token — identical shape to 3B2A's `render`. Example (release mode, ad-hoc artifact, correctly failing):

```
Input
  PASS  resolved DMG: Astraler Skillbox-0.1.0-arm64.dmg
App signature
  PASS  codesign --verify --deep --strict
  FAIL  signature is ad-hoc, expected Developer ID Application
  PASS  hardened runtime enabled
  FAIL  no TeamIdentifier (ad-hoc)
Sidecar (core/skillbox-core)
  PASS  present
  PASS  codesign --verify --strict
  FAIL  signature is ad-hoc, expected Developer ID Application
  PASS  hardened runtime enabled
  FAIL  no TeamIdentifier (ad-hoc)
Gatekeeper
  FAIL  spctl rejected (not notarized)
Stapling
  FAIL  app has no stapled ticket
  FAIL  dmg has no stapled ticket

Missing for a customer-ready release:
  - App and sidecar must be signed with a Developer ID Application identity (not ad-hoc)
  - Artifact must be notarized (spctl: source=Notarized Developer ID)
  - Notarization ticket must be stapled to the app and the dmg
```

**Exit codes:** `0` = verified for the requested mode (WARN/INFO allowed); `1` = one or more FAIL. No other codes; internal errors throw with a clear message (still non-zero).

## 11. Test Strategy

Unit-tested with Vitest on the **pure parser and evaluator**, using captured tool-output fixtures (real `codesign -dvvv` / `spctl` / `stapler` text, ad-hoc and Developer ID variants). The IO shell + DMG mount are not unit-tested (covered by the SMOKE line).

Parser cases:
- `parseCodesign`: ad-hoc app/sidecar output → `{ adhoc:true, developerId:false, teamId:null, hardenedRuntime:true }`; Developer ID app/sidecar output → `{ adhoc:false, developerId:true, teamId:"<id>", hardenedRuntime:true }`; missing-runtime output → `hardenedRuntime:false`.
- `parseSpctl`: accepted+notarized output → `{ accepted:true, source:"Notarized Developer ID" }`; rejected output → `{ accepted:false }`.
- `parseStapler`: success → `{ stapled:true }`; missing-ticket → `{ stapled:false }`.

Evaluator cases (per §6/§7):
- Release mode, Developer ID app+sidecar, accepted spctl, stapled app+dmg, matching/absent expected Team ID → all PASS, exit 0.
- Release mode, ad-hoc artifact → APP2/SID3/GK1/ST1/ST2 FAIL, exit 1 (proves the gate distinguishes customer-ready from not).
- `--allow-adhoc`, ad-hoc artifact → APP1/APP3/SID2/SID4 PASS, GK1/ST1/ST2/APP4/SID5 INFO, exit 0.
- Missing sidecar (SID1) → FAIL in both modes.
- Missing hardened runtime (APP3/SID4) → FAIL in both modes.
- `SKILLBOX_EXPECTED_TEAM_ID` set, app Team ID differs → APP4/SID5 FAIL in release mode; no effect in `--allow-adhoc`.
- Input resolution: zero DMGs → S1 FAIL; multiple DMGs → S1 FAIL listing matches; explicit `.app` → ST2 INFO (DMG stapling not checked).

**Manual smoke (SMOKE.md):** build the 3B1 ad-hoc bundle, run `pnpm release:mac:verify --allow-adhoc <bundle>` (exit 0), then run release mode against the same bundle and confirm it FAILs on Developer ID / notarization / stapling (exit 1).

## 12. Acceptance Criteria (current machine, no credentials)

- [ ] `pnpm release:mac:verify` is added and runnable from `apps/desktop/`.
- [ ] Unit tests pass for the parser and evaluator across all §11 fixtures (ad-hoc and Developer ID).
- [ ] **`--allow-adhoc` against a freshly built 3B1 ad-hoc bundle exits 0**: APP1/APP3/SID2/SID4 PASS, sidecar present, GK1/ST1/ST2 INFO.
- [ ] **Release mode against the same ad-hoc bundle exits 1**, FAILing on signature class (APP2/SID3), Gatekeeper (GK1), and stapling (ST1/ST2) — proving the gate distinguishes customer-ready from not.
- [ ] DMG input is mounted read-only and **always detached** (verified by `hdiutil info` showing no leftover mount after a run, including an error path).
- [ ] Multiple `dist/*.dmg` → S1 FAIL requiring an explicit path; zero → S1 FAIL; one → auto-discovered.
- [ ] No build, no sign/notarize/staple, no network, no keychain mutation, no file writes outside the temp mountpoint (verified by code review and absence of those calls).
- [ ] No secret value or credential path appears in output.
- [ ] All existing gates stay green: `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change.

## 13. Files Expected to Change (for the implementation plan, not this pass)

- `apps/desktop/scripts/release-mac-verify.mjs` — **Create.** Thin IO shell: input resolution, read-only DMG mount/detach, tool spawning (stdout+stderr+exit capture), report render + exit wiring.
- `apps/desktop/scripts/release-mac-verify.parse.mjs` — **Create.** Pure parsers for `codesign`/`spctl`/`stapler` output.
- `apps/desktop/scripts/release-mac-verify.lib.mjs` — **Create.** Pure `evaluate` + `render`. (Exact split with `parse.mjs` confirmed at implementation; requirement: pure logic importable and unit-tested independent of process/fs.)
- `apps/desktop/scripts/release-mac-verify.test.mjs` (Vitest) — **Create.** Parser + evaluator cases per §11, including fixtures.
- `apps/desktop/package.json` — **Modify.** Add the `release:mac:verify` script entry. (No version change; `package:mac` / `package:mac:unsigned` / `release:mac:check` untouched.)
- `SMOKE.md` — **Modify.** Add a "Release Artifact Verification (Slice 3B2B)" section; align the 3B1/3B2 `spctl` invocation to `-t exec`.
- `SCAFFOLD.md` — **Modify.** Document `pnpm release:mac:verify` as the post-build verification gate (the bookend to `release:mac:check`).

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core`, `build-core.mjs`, `electron-builder.yml`, the entitlements plists, and the 3B2A `release-mac-check.*` scripts are untouched.

## 14. Out of Scope — MUST NOT Touch

- **3B2 execution** — no real Developer ID signing, notarization, stapling, or `package:mac` run; no credential files committed or referenced as present.
- **Packaging / config** — `package:mac`, `package:mac:unsigned`, `electron-builder.yml`, entitlements plists, `build:core` unchanged.
- **Apple credentials / network / keychain** — never required, fetched, printed, or mutated.
- **Product / RPC / schema / migrations** — no renderer, main, or `core-go` logic; contract-drift stays clean.
- **CI release automation, auto-update, universal binary, Windows/Linux, `.pkg`/Mac App Store** — not added.
- **GUI launch / Gatekeeper-prompt verification** — stays a manual SMOKE step; 3B2B verifies signature/notarization/stapling facts only.
