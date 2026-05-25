# Slice 3B2B: macOS Release Artifact Verification Harness — Design

Date: 2026-05-26
Status: revised (lead review round 2 — addresses 4 findings: DMG top-level app discovery, Team ID equality, DMG Gatekeeper assessment, entitlement verification)
Depends on: Slice 3B1 (signed-default config + ad-hoc dry-run, lead-approved), Slice 3B2A (release preflight / credential doctor, lead-approved)
Relates to: Slice 3B2 (real Developer ID signing + notarization + stapling — blocked on Apple credentials)

## 0. PM Decision Recorded

3B2A made the **pre-build** prerequisites executable (`pnpm release:mac:check` runs *before* `pnpm package:mac`). The symmetric **post-build** question — "is this built `.app`/`.dmg` actually customer-ready?" — is still only prose (3B1 §8.2, SMOKE.md "Signed + Notarized Smoke"). 3B2B turns that checklist into a deterministic gate.

It is credential-free to author and unit-test, and it can run **partially now** against the 3B1 ad-hoc dry-run bundle (`--allow-adhoc`), so it lands without Apple credentials. When credentials arrive, 3B2 shrinks to: set env → `pnpm package:mac` → `pnpm release:mac:verify` in release mode → done.

3B2B is **read-only with one bounded local side effect** (mounting/detaching a DMG read-only). It never builds, signs, notarizes, staples, calls the network, or mutates the keychain.

## 1. Purpose and Non-Goals

### Purpose
Add `pnpm release:mac:verify [path]` — a command that inspects an already-built `.app` or `.dmg` and reports whether it meets the bar for a customer-ready notarized release: Developer ID signature on **both** the app and the nested Go sidecar, **the same Team ID across both**, hardened runtime, the **expected hardened-runtime entitlements**, Gatekeeper acceptance of **both the app and the DMG container**, and a stapled notarization ticket on both. It complements 3B2A: 3B2A checks *can we build a good artifact?*; 3B2B checks *did we?*

### Non-Goals
- **No building, signing, notarizing, or stapling** — it inspects an existing artifact; it never runs `build:core`, `electron-vite build`, `electron-builder`, `codesign -s`, `notarytool`, or `stapler staple`.
- **No network / Apple online calls** — no notarytool history lookup, no revocation check.
- **No keychain mutation** — it reads artifacts, not the keychain. (Unlike 3B2A it does not even call `security find-identity`.)
- **No product / RPC / schema / migration / contract changes.**
- **Not a substitute for the manual launch smoke** — the quarantine-launch / "no Gatekeeper prompt" step (SMOKE §8.2) stays manual; 3B2B verifies signature/notarization/stapling facts, not GUI launch.

## 2. Command / Input Resolution

- **Command:** `pnpm release:mac:verify [path]` (added to `apps/desktop/package.json` `scripts`), invoked from `apps/desktop/` like the other repo scripts. The script resolves repo paths relative to its own location so cwd does not matter.
- **Input resolution (check S1):**
  - **Explicit `.app`** → verify that bundle directly; DMG-only checks (ST2 stapling, GK2 DMG Gatekeeper) are reported INFO (not checked — no DMG supplied).
  - **Explicit `.dmg`** → mount read-only, locate the **single top-level `.app`** (see §3), verify it, and additionally verify the DMG's own stapled ticket (ST2) and Gatekeeper acceptance (GK2).
  - **No argument** → auto-discover in `apps/desktop/dist/`: exactly **one** `*.dmg` → use it; **zero** → FAIL (S1) with remediation to build or pass a path; **more than one** → FAIL (S1) listing the matches and requiring an explicit path (never guess).
- A path that does not exist, or is neither a `.app` directory nor a `.dmg` file → FAIL (S1).
- **Top-level `.app` resolution inside a DMG (S1):** the mounted volume must contain **exactly one `.app` at its root** (a direct child of the mountpoint). Nested `.app` bundles **inside** another bundle — e.g. Electron helper apps under `…/Contents/Frameworks/` — are **excluded** and must never be selected or counted. Zero top-level apps → FAIL; more than one top-level app → FAIL listing them and requiring disambiguation. (Electron DMGs ship one top-level app plus several nested helper apps; a naive recursive `*.app` search would false-fail with "multiple apps" — this rule prevents that.)

## 3. DMG Strategy (read-only mount)

When the input is a `.dmg`, the shell mounts it read-only, inspects the contained app, and always detaches:

```sh
hdiutil attach -readonly -nobrowse -mountpoint "$MNT" "$DMG"
# select the single TOP-LEVEL *.app (direct child of "$MNT"); see §2 S1.
# NEVER recurse into a bundle — nested Electron helper .apps must be ignored.
# verify it in place (read-only mount)
hdiutil detach "$MNT"        # fallback: hdiutil detach -force "$MNT"
```

- **Top-level app selection is a non-recursive listing of `$MNT` filtered to entries ending in `.app`** — it does **not** walk into `Contents/`. This is extracted into a pure helper `pickTopLevelApp(entries)` (see §4) so the "one top-level app, ignore nested helpers" rule is unit-tested without a real mount.
- The mountpoint is a fresh `mkdtemp` dir; detach runs in a `finally` so a mid-verify error still unmounts. Detach failure retries once with `-force`, then surfaces as a non-zero script error (with a clear message) so we never leak a mounted volume silently.
- **This is the one allowed local side effect.** It is documented as such. It is still forbidden to build, sign, notarize, staple, hit the network, or mutate the keychain.

**Why mount over inspecting an adjacent extracted `.app`:** the `.dmg` is the artifact customers actually download, and (a) verifying the app *as it ships inside the DMG* is the only way to confirm the shipped copy is signed/notarized, and (b) the DMG carries its own stapled ticket (ST2) that an extracted app cannot show. A read-only mount guarantees zero mutation while giving us the true distributed bytes; an adjacent `.app` could diverge from DMG contents and cannot verify DMG stapling. (If a bare `.app` is all that is available, the explicit-`.app` path above still works, with ST2 reported INFO.)

## 4. Architecture (pure parser + pure evaluator + thin IO shell)

Mirrors 3B2A's "pure core + thin IO shell" discipline. Three modules under `apps/desktop/scripts/`:

- **`release-mac-verify.parse.mjs` (pure)** — turns raw tool text into structured signals, and holds the pure selection/normalization helpers. No process/fs/env. Functions:
  - `parseCodesign(text)` → `{ adhoc, developerId, teamId, hardenedRuntime }`
  - `parseSpctl(text, exitCode)` → `{ accepted, source }`
  - `parseStapler(text, exitCode)` → `{ stapled }`
  - `parseEntitlementKeys(text)` → `string[]` — the entitlement key names present in a plist/XML blob (from `codesign -d --entitlements :-` on the artifact **or** from a committed `entitlements.*.plist`). Used for both the expected set and the embedded set.
  - `pickTopLevelApp(entries)` → `{ app }` or `{ error }` — given a **non-recursive** listing of the DMG mount root, returns the single top-level `.app`, or an error for zero / multiple (finding 1).
- **`release-mac-verify.lib.mjs` (pure)** — `evaluate(signals)` takes the parsed signals + `mode` + `expectedTeamId` + `expectedEntitlements` (`{ app: string[], sidecar: string[] }`) + input metadata and returns `{ results: CheckResult[], missing, exitCode }`, plus `render(results, missing)` (reuses 3B2A's status-token + grouped-category output shape). No process/fs/env.
- **`release-mac-verify.mjs` (thin IO shell)** — resolves the input (§2), mounts the DMG if needed (§3) and applies `pickTopLevelApp` to the non-recursive mount listing, reads the **committed** `build/entitlements.mac.plist` / `entitlements.mac.inherit.plist` for the expected key sets, spawns `codesign` / `spctl` / `stapler`, captures **both stdout and stderr** (these tools write to stderr) and the **exit code**, feeds raw text+exit to the parsers, runs `evaluate`, prints `render`, sets the exit code. Not unit-tested directly (covered by the SMOKE line).

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
- entitlement key names (e.g. `com.apple.security.cs.allow-jit`) extracted from the `codesign -d --entitlements :-` plist/XML blob.

Commands the shell runs (read-only):
- App: `codesign -dvvv "$APP"`, `codesign --verify --deep --strict --verbose=2 "$APP"`, `codesign -d --entitlements :- "$APP"`, `spctl -a -vvv -t exec "$APP"`, `xcrun stapler validate "$APP"`.
- Sidecar (`$APP/Contents/Resources/core/skillbox-core`): `codesign -dvvv "$SIDE"`, `codesign --verify --strict --verbose=2 "$SIDE"` (direct — **not** relying on the app's `--deep` walk), `codesign -d --entitlements :- "$SIDE"`.
- DMG (when input is a `.dmg`): `spctl -a -vvv -t open "$DMG"` (finding 3) and `xcrun stapler validate "$DMG"`.

> **Assessment types are deliberate:** `spctl -t exec` is the correct type for an **executable app bundle** (GK1); `spctl -t open` is the correct type for the **DMG container** that the user double-clicks (GK2). The 3B1 SMOKE used `-t open` against the *app*; SMOKE.md is corrected so the app uses `-t exec` and the DMG uses `-t open` as part of this slice's doc update.

## 6. Check Table

| ID | Category | Check | Release-mode PASS | `--allow-adhoc` PASS |
|----|----------|-------|-------------------|----------------------|
| S1 | input | Input resolved to one `.app`, or a `.dmg` with exactly one **top-level** `.app` (§2) | resolved | resolved |
| APP1 | app | `codesign --verify --deep --strict` exit 0 | yes | yes |
| APP2 | app | Signature class | `Authority=Developer ID Application` present, **not** ad-hoc | ad-hoc **or** Developer ID accepted |
| APP3 | app | Hardened runtime (`flags=…(runtime)`) | present | present |
| APP4 | app | `TeamIdentifier` present | present | INFO only (ad-hoc has none) |
| ENT1 | entitlements | App embedded entitlements include **every** key from committed `build/entitlements.mac.plist` | all present | all present |
| SID1 | sidecar | Sidecar exists at `Contents/Resources/core/skillbox-core` | present | present |
| SID2 | sidecar | Sidecar `codesign --verify --strict` exit 0 (direct) | yes | yes |
| SID3 | sidecar | Sidecar signature class | Developer ID, not ad-hoc | ad-hoc or Developer ID |
| SID4 | sidecar | Sidecar hardened runtime | present | present |
| SID5 | sidecar | Sidecar `TeamIdentifier` present | present | INFO only |
| ENT2 | entitlements | Sidecar embedded entitlements include **every** key from committed `build/entitlements.mac.inherit.plist` | all present | all present |
| TID1 | identity | App and sidecar `TeamIdentifier` are **both present and equal** (and **both** `== SKILLBOX_EXPECTED_TEAM_ID` when that env is set) — finding 2 | present + equal (+ match if env set) | INFO only (ad-hoc has none) |
| GK1 | gatekeeper | `spctl -t exec "$APP"` accepted + `source=Notarized Developer ID` | accepted + notarized | rejection/unsigned ⇒ **INFO** |
| GK2 | gatekeeper | `spctl -t open "$DMG"` accepted (only when DMG input) — finding 3 | accepted | **INFO**; app-only input ⇒ INFO |
| ST1 | staple | `stapler validate "$APP"` succeeds | stapled | missing ⇒ **INFO** |
| ST2 | staple | `stapler validate "$DMG"` succeeds (only when DMG input) | stapled | missing ⇒ **INFO**; app-only input ⇒ INFO |

Statuses follow 3B2A: **FAIL** forces a non-zero exit; **WARN/INFO** do not. The report ends with a "Missing for a customer-ready release:" section listing each FAIL with a one-line remediation.

**Entitlements rationale (ENT1/ENT2):** these are checked in **both** modes because the embedded entitlements come from the same committed plists during *any* signing pass (the 3B1 ad-hoc dry-run keeps `hardenedRuntime: true` and applies the entitlements; only identity and notarize differ). A shipped app missing `com.apple.security.cs.allow-jit` / `allow-unsigned-executable-memory` would crash the Electron renderer under hardened runtime, so a missing expected key is a hard FAIL regardless of mode. The check is a **subset** assertion (artifact must contain every expected key; extra keys are not failed) so it stays robust to electron-builder injecting additional standard entitlements.

## 7. Modes

- **Release mode (default):** the artifact must be genuinely shippable. PASS-required: APP1–APP4, ENT1, SID1–SID5, ENT2, TID1, GK1, ST1. DMG-only checks GK2 and ST2 are PASS-required when a `.dmg` was supplied and INFO when input was a bare `.app` (with a prominent note that full verification should target the `.dmg`). Any FAIL ⇒ exit 1.
- **`--allow-adhoc` (dry-run, for the 3B1 ad-hoc artifact):** intended for the bundle produced by `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`. Exit 0 requires app + sidecar **codesign verification (APP1, SID1, SID2)**, **hardened-runtime (APP3, SID4)**, **entitlements (ENT1, ENT2)** pass, and the signature is at least ad-hoc (APP2, SID3). Because an ad-hoc bundle is neither notarized nor stapled and has no Team ID, **GK1, GK2, ST1, ST2, APP4, SID5, TID1 are reported INFO, never WARN/FAIL** in this mode. This proves `mac.binaries` reached the sidecar (the #1 notarization hazard) **and** that the expected entitlements were applied — all without credentials.

The two modes share one evaluator; `mode` only changes whether notarization/stapling/Team-ID gaps are FAIL (release) or INFO (`--allow-adhoc`). Entitlement checks (ENT1/ENT2) are FAIL-on-miss in **both** modes (see §6 rationale).

## 8. Team ID Assertion (finding 2)

- **App ↔ sidecar equality is required by default (TID1).** Release mode FAILs unless **both** the app and the sidecar carry a `TeamIdentifier` **and the two are equal**. Requiring mere presence on each independently (the previous APP4/SID5-only design) would false-pass a mixed-identity artifact where the app is signed by one team and the bundled sidecar by another — exactly the kind of build mistake this gate must catch. APP4/SID5 remain as presence checks (clearer per-target diagnostics); TID1 is the cross-check that they match.
- **Identity-agnostic by default:** with no env set, TID1 pins app and sidecar to *each other*, not to a specific value.
- **`SKILLBOX_EXPECTED_TEAM_ID`:** when set, TID1 additionally FAILs unless **both** the app's and the sidecar's `TeamIdentifier` equal the expected value (catches a wrong-cert build, not just an internal mismatch). The env var holds a Team ID, which is **not a secret** (it is embedded in every shipped notarized app and visible via `codesign -dvvv`), so it may appear in output. It has no effect in `--allow-adhoc` mode (ad-hoc artifacts carry no Team ID; TID1 is INFO there).

## 9. Secret Hygiene and Side-Effect Boundaries

- **No secrets are involved.** 3B2B reads built artifacts and the committed entitlements plists, not credentials — no passwords, `.p8`, `.p12`, or env credential values are read or printed. Identity names and Team IDs surfaced from `codesign -dvvv` are non-secret (per 3B2A §6). Reading `build/entitlements.mac.*.plist` is read-only; the report lists entitlement **key names** (non-secret) and never dumps the full embedded entitlements blob or full env.
- **Side effects: exactly one, bounded** — read-only `hdiutil attach`/`detach` of a supplied DMG (§3). The script otherwise only reads. It does **not**: build, sign, notarize, staple, run `build:core`/`electron-vite build`/`electron-builder`, make any network request, call notarytool/any Apple service, or mutate the keychain or any file outside its own temp mountpoint.

## 10. Output and Exit Codes

Human-readable, grouped by category, one line per check with a leading status token — identical shape to 3B2A's `render`. Example (release mode, ad-hoc artifact, correctly failing):

```
Input
  PASS  resolved DMG: Astraler Skillbox-0.1.0-arm64.dmg (one top-level app: Astraler Skillbox.app)
App signature
  PASS  codesign --verify --deep --strict
  FAIL  signature is ad-hoc, expected Developer ID Application
  PASS  hardened runtime enabled
  FAIL  no TeamIdentifier (ad-hoc)
  PASS  entitlements include allow-jit, allow-unsigned-executable-memory
Sidecar (core/skillbox-core)
  PASS  present
  PASS  codesign --verify --strict
  FAIL  signature is ad-hoc, expected Developer ID Application
  PASS  hardened runtime enabled
  FAIL  no TeamIdentifier (ad-hoc)
  PASS  entitlements include allow-jit, inherit
Identity
  FAIL  app/sidecar TeamIdentifier not present, cannot confirm a single team
Gatekeeper
  FAIL  spctl -t exec (app) rejected (not notarized)
  FAIL  spctl -t open (dmg) rejected (not notarized)
Stapling
  FAIL  app has no stapled ticket
  FAIL  dmg has no stapled ticket

Missing for a customer-ready release:
  - App and sidecar must be signed with a Developer ID Application identity (not ad-hoc)
  - App and sidecar must share one Team ID (set SKILLBOX_EXPECTED_TEAM_ID to also pin it)
  - Artifact must be notarized (spctl: source=Notarized Developer ID) for both the app and the dmg
  - Notarization ticket must be stapled to the app and the dmg
```

(Entitlement checks PASS even on the ad-hoc bundle — electron-builder applies the committed entitlements during the ad-hoc dry-run, so only the Developer-ID / notarization / Team-ID facts are missing.)

**Exit codes:** `0` = verified for the requested mode (WARN/INFO allowed); `1` = one or more FAIL. No other codes; internal errors throw with a clear message (still non-zero).

## 11. Test Strategy

Unit-tested with Vitest on the **pure parser and evaluator**, using captured tool-output fixtures (real `codesign -dvvv` / `spctl` / `stapler` text, ad-hoc and Developer ID variants). The IO shell + DMG mount are not unit-tested (covered by the SMOKE line).

Parser cases:
- `parseCodesign`: ad-hoc app/sidecar output → `{ adhoc:true, developerId:false, teamId:null, hardenedRuntime:true }`; Developer ID app/sidecar output → `{ adhoc:false, developerId:true, teamId:"<id>", hardenedRuntime:true }`; missing-runtime output → `hardenedRuntime:false`.
- `parseSpctl`: accepted+notarized output → `{ accepted:true, source:"Notarized Developer ID" }`; rejected output → `{ accepted:false }`.
- `parseStapler`: success → `{ stapled:true }`; missing-ticket → `{ stapled:false }`.
- `parseEntitlementKeys` (finding 4): app `codesign -d --entitlements :-` blob → `["com.apple.security.cs.allow-jit", "com.apple.security.cs.allow-unsigned-executable-memory"]`; sidecar blob → `["com.apple.security.cs.allow-jit", "com.apple.security.inherit"]`; the committed `entitlements.mac.plist` / `.inherit.plist` parse to the same expected sets; an empty/`not signed` blob → `[]`.
- `pickTopLevelApp` (finding 1): listing `["Astraler Skillbox.app", "Applications"]` (symlink) → picks `Astraler Skillbox.app`; a listing where the helper apps would appear only via recursion is **not** passed in (the shell lists non-recursively) — and a listing with **two** top-level `.app` entries → `{ error }`; **zero** `.app` entries → `{ error }`.

Evaluator cases (per §6/§7):
- Release mode, Developer ID app+sidecar with **equal** Team IDs, expected entitlements present, accepted spctl (app + dmg), stapled app+dmg, matching/absent expected Team ID → all PASS, exit 0.
- Release mode, ad-hoc artifact → APP2/SID3/TID1/GK1/GK2/ST1/ST2 FAIL (ENT1/ENT2 still PASS), exit 1 (proves the gate distinguishes customer-ready from not).
- `--allow-adhoc`, ad-hoc artifact → APP1/APP3/SID2/SID4/ENT1/ENT2 PASS, GK1/GK2/ST1/ST2/APP4/SID5/TID1 INFO, exit 0.
- Missing sidecar (SID1) → FAIL in both modes.
- Missing hardened runtime (APP3/SID4) → FAIL in both modes.
- **Entitlements missing an expected key (finding 4):** app blob missing `allow-jit` → ENT1 FAIL in **both** modes; sidecar blob missing `inherit` → ENT2 FAIL in both modes; artifact with **extra** non-expected keys but all expected present → ENT PASS (subset semantics).
- **Team ID equality (finding 2):** Developer ID app+sidecar with **different** Team IDs → TID1 FAIL in release mode (even though APP4/SID5 each PASS on presence); both present + equal, no env → TID1 PASS; `SKILLBOX_EXPECTED_TEAM_ID` set and **both** match → TID1 PASS; env set and either app or sidecar differs from it → TID1 FAIL; ad-hoc (no Team ID) → TID1 INFO in `--allow-adhoc`, FAIL in release.
- **DMG Gatekeeper (finding 3):** DMG `spctl -t open` rejected, app otherwise notarized → GK2 FAIL in release mode; DMG accepted → GK2 PASS; bare `.app` input → GK2 INFO (not applicable).
- Input resolution: zero DMGs → S1 FAIL; multiple DMGs → S1 FAIL listing matches; explicit `.app` → GK2/ST2 INFO (DMG-only checks not run).

**Manual smoke (SMOKE.md):** build the 3B1 ad-hoc bundle, run `pnpm release:mac:verify --allow-adhoc <bundle>` and confirm exit 0 **with ENT1/ENT2 PASS** (the ad-hoc bundle carries the expected entitlements), then run release mode against the same bundle and confirm it FAILs on Developer ID / Team-ID equality / notarization (app + dmg) / stapling (exit 1). For the DMG path, confirm `hdiutil info` shows no leftover mount afterward.

## 12. Acceptance Criteria (current machine, no credentials)

- [ ] `pnpm release:mac:verify` is added and runnable from `apps/desktop/`.
- [ ] Unit tests pass for the parser and evaluator across all §11 fixtures (ad-hoc and Developer ID), including `parseEntitlementKeys` and `pickTopLevelApp`.
- [ ] **`--allow-adhoc` against a freshly built 3B1 ad-hoc bundle exits 0**: APP1/APP3/SID2/SID4 **and ENT1/ENT2** PASS, sidecar present, GK1/GK2/ST1/ST2/TID1 INFO.
- [ ] **Release mode against the same ad-hoc bundle exits 1**, FAILing on signature class (APP2/SID3), Team-ID equality (TID1), Gatekeeper (GK1 + GK2), and stapling (ST1/ST2), while ENT1/ENT2 still PASS — proving the gate distinguishes customer-ready from not.
- [ ] **A DMG containing nested Electron helper `.app` bundles resolves to the single top-level app** (S1 PASS) and does not false-fail with "multiple apps" (finding 1).
- [ ] DMG input is mounted read-only and **always detached** (verified by `hdiutil info` showing no leftover mount after a run, including an error path).
- [ ] Multiple `dist/*.dmg` → S1 FAIL requiring an explicit path; zero → S1 FAIL; one → auto-discovered.
- [ ] No build, no sign/notarize/staple, no network, no keychain mutation, no file writes outside the temp mountpoint (verified by code review and absence of those calls).
- [ ] No secret value or credential path appears in output.
- [ ] All existing gates stay green: `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change.

## 13. Files Expected to Change (for the implementation plan, not this pass)

- `apps/desktop/scripts/release-mac-verify.mjs` — **Create.** Thin IO shell: input resolution, read-only DMG mount/detach + top-level-app selection, reading committed entitlements plists for expected keys, tool spawning (stdout+stderr+exit capture), report render + exit wiring.
- `apps/desktop/scripts/release-mac-verify.parse.mjs` — **Create.** Pure parsers/helpers: `parseCodesign`, `parseSpctl`, `parseStapler`, `parseEntitlementKeys`, `pickTopLevelApp`.
- `apps/desktop/scripts/release-mac-verify.lib.mjs` — **Create.** Pure `evaluate` + `render`. (Exact split with `parse.mjs` confirmed at implementation; requirement: pure logic importable and unit-tested independent of process/fs.)
- `apps/desktop/scripts/release-mac-verify.test.mjs` (Vitest) — **Create.** Parser + evaluator cases per §11, including fixtures.
- `apps/desktop/package.json` — **Modify.** Add the `release:mac:verify` script entry. (No version change; `package:mac` / `package:mac:unsigned` / `release:mac:check` untouched.)
- `SMOKE.md` — **Modify.** Add a "Release Artifact Verification (Slice 3B2B)" section; correct the `spctl` assessment types (app → `-t exec`, DMG → `-t open`) in the 3B1/3B2 sections.
- `SCAFFOLD.md` — **Modify.** Document `pnpm release:mac:verify` as the post-build verification gate (the bookend to `release:mac:check`).

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core`, `build-core.mjs`, `electron-builder.yml`, the entitlements plists, and the 3B2A `release-mac-check.*` scripts are untouched.

## 14. Out of Scope — MUST NOT Touch

- **3B2 execution** — no real Developer ID signing, notarization, stapling, or `package:mac` run; no credential files committed or referenced as present.
- **Packaging / config** — `package:mac`, `package:mac:unsigned`, `electron-builder.yml`, entitlements plists, `build:core` unchanged.
- **Apple credentials / network / keychain** — never required, fetched, printed, or mutated.
- **Product / RPC / schema / migrations** — no renderer, main, or `core-go` logic; contract-drift stays clean.
- **CI release automation, auto-update, universal binary, Windows/Linux, `.pkg`/Mac App Store** — not added.
- **GUI launch / Gatekeeper-prompt verification** — stays a manual SMOKE step; 3B2B verifies signature/notarization/stapling facts only.
