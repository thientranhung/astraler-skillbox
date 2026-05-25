# Slice 3B2A: macOS Release Preflight / Credential Doctor — Design

Date: 2026-05-26
Status: draft (for lead review)
Depends on: Slice 3B1 (signed-default config + entitlements + dry-run, lead-approved)
Relates to: Slice 3B2 (real Developer ID signing + notarization + stapling — blocked on Apple credentials)

## 0. PM Decision Recorded

3B2 (the real notarized release) is blocked only on Apple credentials: this machine has **0** valid Developer ID Application identities and **no** notarization env vars set, while every other prerequisite is in place (verified 2026-05-26: `xcrun notarytool`, `xcrun stapler`, `codesign`, `spctl`, `plutil` all present; electron-builder 26.8.1; the 3B1 signed-default config landed).

Rather than serialize, we run two tracks in parallel:

- **Track 1 (user-owned, schedule-critical):** ask the user to procure Apple credentials (Developer Program enrollment, Developer ID Application cert + key, Team ID, a notarization credential group). Enrollment approval is the long pole and is entirely on the user.
- **Track 2 (this slice, 3B2A):** build a small, unblocked, credential-free **release preflight / credential doctor** that deterministically reports what is still missing before anyone runs the slow `pnpm package:mac` notarized build.

3B2A is strictly **read-only, offline, no secrets leaked, and free of signing / notarization / build side effects.**

## 1. Goal and Non-Goals

### Goal
Add a single fast command — `pnpm release:mac:check` — that, in under a second and without any Apple credentials, gathers facts about the local machine and the repo and reports a grouped PASS / FAIL / WARN / INFO readiness summary for a customer-ready notarized macOS DMG. It ends in an actionable "missing for a notarized DMG" list and exits non-zero on any hard failure. It is the local gate a developer runs **before** the real `pnpm package:mac` (3B2) so credential or config gaps surface in <1s instead of minutes into a build.

### Non-Goals
- **No signing, no notarization submission, no stapling, no DMG build** — this is a checker, not a packager. It never runs `build:core` or `electron-vite build`.
- **No online Apple validation** — no network calls; no App Store Connect lookup; no certificate revocation check.
- **No keychain mutation** — read-only identity inspection only; never import, delete, or unlock.
- **No printing of secret values** — credentials are reported presence-only.
- **No changes to 3B1/3B2 packaging behavior** — `package:mac` and `package:mac:unsigned` are untouched; the preflight is standalone and is not auto-chained into `package:mac` in this slice.
- **No new runtime/product behavior** — no renderer, Electron main, `core-go`, schema, migration, or JSON-RPC contract changes.
- **Not a secret scanner** — the secret-file hygiene check is a narrow tracked-file check, not a content scanner.

## 2. Why This Helps Customer Release and Is Not Busywork

- **Turns prose prerequisites into an executable gate.** 3B1 spec §7 and SCAFFOLD's "Signed + notarized" section list the 3B2 prerequisites as prose. 3B2A makes them runnable: the user satisfies items, re-runs `release:mac:check` until green, then runs `package:mac` once with confidence.
- **Replaces a slow failure loop with instant feedback.** Today the only way to learn a credential is missing is to run `package:mac` (`build:core` + `electron-vite build` + electron-builder, several minutes) and watch it die near the signing/notarize step with a cryptic message. The preflight returns the same verdict in <1s before any build.
- **Locks config invariants that otherwise regress silently.** It asserts hardened runtime is on, `mac.notarize` is true, entitlements files exist and lint, and `mac.binaries` still points at `Contents/Resources/core/skillbox-core` — the documented #1 notarization hazard (3B1 spec §5). Without the gate these only surface as a notarization rejection.
- **Durable, not throwaway.** It is run on every release, not once. That, plus asserting invariants no existing gate covers, is what keeps it from being busywork. The anti-creep rule: if it ever grows an online validator or a real content secret scanner, that is out of scope and should be rejected.

## 3. Command / API Shape

- **Command:** `pnpm release:mac:check` (added to `apps/desktop/package.json` `scripts`).
- **Script:** `apps/desktop/scripts/release-mac-check.mjs` — Node ESM, matching the existing `scripts/generate-contracts.mjs` and `scripts/build-core.mjs` convention.
- **Naming rationale:** `release:mac:check` reads as a gate/doctor; a `package:*` name would imply it builds. One name only (no `:doctor` alias — YAGNI).
- **Standalone in 3B2A:** it is *not* wired as a pre-step of `package:mac`. `package:mac` is the 3B1-defined 3B2 entrypoint; auto-gating it is a 3B2 decision. Docs tell the user to run the check first.
- **Invocation context:** run from `apps/desktop/` (like all repo `pnpm` commands). The script resolves repo paths relative to its own location so it works regardless of cwd.

## 4. Check Categories and Statuses

Status vocabulary:
- **PASS** — requirement satisfied.
- **FAIL** — hard blocker for a notarized DMG; forces a non-zero exit.
- **WARN** — not a hard blocker now but worth surfacing (e.g. a build step will produce the missing item later).
- **INFO** — neutral fact (e.g. which credential group was detected).

| # | Category | Check | PASS condition | Status if not met |
|---|----------|-------|----------------|-------------------|
| A1 | Platform | `process.platform === 'darwin'` | true | FAIL |
| A2 | Tooling | `xcrun -f notarytool` resolves | found (altool retired 2023-11-01; notarytool is the only accepted path) | FAIL |
| A3 | Tooling | `xcrun -f stapler` resolves | found | FAIL |
| A4 | Tooling | `codesign`, `spctl`, `plutil` on PATH | all found | FAIL |
| B1 | Signing credentials | either a keychain Developer ID Application identity **or** a `.p12` via `CSC_LINK` + `CSC_KEY_PASSWORD` (see §5.1) | ≥1 acceptable signing path satisfied | FAIL |
| C1 | Notarization credentials | **at least one complete** credential group present (see §5.2) | ≥1 complete group | FAIL |
| D1 | Config invariant | `mac.hardenedRuntime === true` | true | FAIL |
| D2 | Config invariant | `mac.notarize === true` | true | FAIL |
| D3 | Config invariant | `mac.entitlements` + `mac.entitlementsInherit` files exist and pass `plutil -lint` | both OK | FAIL |
| D4 | Config invariant | `mac.binaries` includes `Contents/Resources/core/skillbox-core` | present | FAIL |
| D5 | Config invariant | `mac.target` includes a dmg target with arch arm64 | present | FAIL |
| E1 | Sidecar staging | staged `apps/desktop/resources/core/skillbox-core` is a Mach-O **arm64** with the exec bit set | present + arm64 + executable | **WARN if absent** (built by `package:mac` via `build:core`); **FAIL if present but wrong arch / not executable** |
| F1 | Artifact hygiene | `git status --porcelain --untracked-files=no -- apps/desktop/dist apps/desktop/resources/core` shows no **tracked** build artifacts (untracked `??` entries excluded by `--untracked-files=no`; a present-but-untracked `dist/` is expected and must not fail) | no tracked-status entries | FAIL |
| F2 | Secret-file hygiene | no **tracked** `*.p12` / `*.p8` under `apps/desktop` (narrow tracked-file check, not a content scanner) | none tracked | FAIL |
| G1 | Version | `package.json` `version` is a real release string (not `0.0.0`) | non-`0.0.0` | WARN |

The report closes with a **"Missing for a customer-ready notarized DMG:"** section that lists each FAIL with a one-line remediation (e.g. "Install a Developer ID Application certificate into the login keychain"; "export APPLE_API_KEY_ID=…").

## 5. Signing and Notarization Credential Logic

A notarized release needs **both** (a) a way to sign with a Developer ID Application identity and (b) notarization credentials. These are evaluated independently — §5.1 covers signing (check B1), §5.2 covers notarization (check C1). All evaluation is **presence/existence only**; the preflight **never reads or prints** any credential value, env value, or file path (see §6).

### 5.1 Signing credentials (B1)

Existing project docs (3B1 spec §7, SCAFFOLD "Signed + notarized") allow a Developer ID Application identity in the login keychain **or** a `.p12` supplied via `CSC_LINK` + `CSC_KEY_PASSWORD`. B1 must therefore PASS when **either** path is satisfied:

- **Path A — keychain identity:** `security find-identity -v -p codesigning` yields ≥1 `Developer ID Application:` line (`-v` already filters to valid/non-expired). Report the matched identity **names** only (non-secret; see §6).
- **Path B — `.p12` via env:** `CSC_LINK` is set **and** `CSC_KEY_PASSWORD` is set. If `CSC_LINK` looks like a **local file path** (not a `https://` URL or base64 blob), additionally require that the file **exists and is readable**; if it is a URL/base64 form, presence of the env var is sufficient (the preflight does not fetch or decode it). Report only `CSC_LINK set` + (for local paths) `exists`/`readable`, and `CSC_KEY_PASSWORD set` — **never the path or value**.

Rules:
- **Either path satisfied** → B1 PASS; emit an INFO line naming which path was detected (e.g. "Signing: keychain Developer ID Application identity", or "Signing: CSC_LINK + CSC_KEY_PASSWORD").
- **Both satisfied** → B1 PASS; emit INFO noting the keychain identity is electron-builder's default discovery path when no explicit `CSC_LINK` override is forced. Do not fail.
- **Path B partial** (e.g. `CSC_LINK` set but `CSC_KEY_PASSWORD` missing, or a local `CSC_LINK` path that does not exist/readable) and **no keychain identity** → B1 FAIL, naming exactly what is missing (e.g. "CSC_LINK is set but CSC_KEY_PASSWORD is missing", or "CSC_LINK points to a local file that is missing or unreadable") — without printing the path.
- **Neither path satisfied** → B1 FAIL: "No signing credential found. Provide a Developer ID Application identity in the login keychain, or set CSC_LINK + CSC_KEY_PASSWORD."

### 5.2 Notarization credential groups (C1)

A notarized release needs **at least one complete** notarization credential group. The preflight evaluates groups on **presence only** (env var set and non-empty; referenced files exist and are readable) and **never reads or prints their values**.

- **Group 1 — App Store Connect API key (preferred):**
  `APPLE_API_KEY` (path to the `.p8`; the file must exist and be readable) + `APPLE_API_KEY_ID` + `APPLE_API_ISSUER`.
- **Group 2 — Apple ID + app-specific password:**
  `APPLE_ID` + `APPLE_APP_SPECIFIC_PASSWORD` + `APPLE_TEAM_ID`.
- **Group 3 — notarytool keychain profile (optional, INFO):**
  A stored `notarytool store-credentials` profile referenced via `APPLE_KEYCHAIN_PROFILE`. electron-builder's built-in `mac.notarize` consumes the API-key or Apple-ID env groups directly; a keychain profile is a notarytool convenience and is **not** a first-class electron-builder env input, so it is reported as **INFO** ("a notarytool keychain profile env var is set, but electron-builder's `mac.notarize` uses Group 1 or Group 2") and does **not** by itself satisfy C1. Document it; do not let it pass the gate.

Rules:
- **At least one complete group** → C1 PASS; emit an INFO line naming which group was detected (e.g. "Detected notarization credentials: API key (Group 1)").
- **Multiple complete groups present** → C1 still PASS (never a failure); emit a **WARN** about precedence: Group 1 (API key) is preferred, and electron-builder/notarytool will use one — surface which to avoid ambiguity (e.g. "Both Group 1 and Group 2 are complete; the API key (Group 1) is preferred and will be used"). Do **not** fail on redundancy.
- **No group present** → C1 FAIL: "No notarization credential group found. Provide Group 1 (APPLE_API_KEY + APPLE_API_KEY_ID + APPLE_API_ISSUER) or Group 2 (APPLE_ID + APPLE_APP_SPECIFIC_PASSWORD + APPLE_TEAM_ID)."
- **Partial group** (and no other group complete) → C1 FAIL naming exactly the **missing variable names** of the group the user has started (e.g. "APPLE_API_KEY and APPLE_API_KEY_ID are set; missing APPLE_API_ISSUER"). Never print the value of any set variable — only its name and `set`/`missing` state.
- If a referenced file (`APPLE_API_KEY` `.p8`, or `CSC_LINK` local path) is set but missing/unreadable → FAIL reporting only the variable name plus `missing`/`unreadable`, **never the path or file contents** (see §6).

## 6. Secret Hygiene and Side-Effect Boundaries

**Secret hygiene (hard rules):**
- Credentials are reported **presence-only** — by variable **name** plus one state token (`set` / `missing` / `exists` / `readable` / `unreadable`), **never the value**.
- For **`APPLE_API_KEY` and `CSC_LINK` specifically**, the preflight must **not print the env value or the file path** — only the variable name and its state token. (The path itself can leak a username or directory layout, so it is treated as sensitive.) Files behind these vars are reported as `exists`/`readable`/`missing`/`unreadable`, never by content and never by path.
- The identity check prints only matched identity **names** from `security find-identity`. Identity names and Team IDs are **not secrets** (they are embedded in every shipped notarized app and visible via `codesign -dvvv`); the cert private key, passwords, and `.p8` body are the secrets and are never read or printed.
- No `set -x`, no echoing of any command line that could carry a secret.

**Side-effect boundaries (this slice does NOT):**
- run any network request;
- submit to notarytool or call any Apple online service;
- sign, re-sign, or staple any artifact;
- build a DMG, run `build:core`, or run `electron-vite build`;
- mutate the keychain (import / delete / unlock) — `security find-identity` is read-only;
- write, move, or delete any file (the script only reads).

## 7. Output Format and Exit Code Rules

**Output:** human-readable, grouped by category, one line per check with a leading status token, e.g.:

```
Platform & tooling
  PASS  macOS (darwin)
  PASS  xcrun notarytool found
  ...
Signing credentials
  FAIL  no Developer ID Application identity in login keychain, and CSC_LINK + CSC_KEY_PASSWORD not set
Notarization credentials
  FAIL  no complete credential group (see remediation)
electron-builder config
  PASS  hardenedRuntime: true
  ...

Missing for a customer-ready notarized DMG:
  - Signing credential: a Developer ID Application identity in the login keychain, OR CSC_LINK + CSC_KEY_PASSWORD
  - One notarization credential group (Group 1: APPLE_API_KEY + APPLE_API_KEY_ID + APPLE_API_ISSUER, or Group 2: APPLE_ID + APPLE_APP_SPECIFIC_PASSWORD + APPLE_TEAM_ID)
```

**Exit codes:**
- `0` — no FAIL (WARN / INFO allowed). The machine is ready for `pnpm package:mac`.
- `1` — one or more FAIL. The "Missing…" list is non-empty.
- A reserved non-`0`/`1` code (e.g. `2`) is **not** used; any internal script error should surface as a thrown exception with a clear message (still non-zero). Keep it simple: ready vs not-ready.

Optional `--json` flag (nice-to-have, may be deferred): emit the structured result array for CI consumption. Not required for acceptance.

## 8. Test Strategy

The logic is nontrivial (credential-group completeness, config parsing, status aggregation, redaction), so it is unit-tested. Architecture mirrors the repo's existing "pure core + thin IO shell" discipline:

- **Pure evaluator** — a pure function that takes plain inputs and returns a structured result `{ id, category, status, message, remediation? }[]` plus an overall exit code. Inputs:
  - an env map (`{ APPLE_API_KEY?: string, ... }`),
  - a parsed electron-builder config object,
  - a list of identity **names** (strings),
  - a tool-presence map (`{ notarytool: boolean, ... }`),
  - file-probe results (sidecar arch/exec, entitlements lint outcome, tracked-artifact list, and — for local-path `CSC_LINK` / `APPLE_API_KEY` — an `exists`/`readable`/`isLocalPath` flag set, with the path itself **not** carried into the evaluator output).
  No process spawning, no filesystem, no env access inside the evaluator.
- **Thin IO shell** — gathers the facts (spawns `security` / `xcrun` / `plutil`, reads/parses `electron-builder.yml`, runs `git status --porcelain`, stats the sidecar), passes them to the evaluator, renders the report, sets the exit code. Kept minimal and not unit-tested directly (covered by the manual SMOKE line).

**Vitest cases (on the pure evaluator):**
- Notarization groups (C1): none → FAIL; partial Group 1 (missing `APPLE_API_ISSUER`) → FAIL naming the missing var; complete Group 1 → PASS; complete Group 2 → PASS; **both groups complete → PASS + WARN naming Group 1 as preferred (never FAIL)**; keychain-profile-only → C1 FAIL + INFO line.
- Signing credentials (B1): no keychain name and no `CSC_LINK` → FAIL; ≥1 `Developer ID Application` name only → PASS (Path A); no keychain name but `CSC_LINK` + `CSC_KEY_PASSWORD` set with a readable local path → PASS (Path B); `CSC_LINK` set as a URL/base64 + `CSC_KEY_PASSWORD` set → PASS (no fetch/decode); `CSC_LINK` set as a local path that is missing/unreadable, no keychain → FAIL; `CSC_LINK` set but `CSC_KEY_PASSWORD` missing, no keychain → FAIL naming the missing var; both keychain identity and CSC env present → PASS.
- Config invariants: each of `hardenedRuntime` / `notarize` flipped false → FAIL; `mac.binaries` missing the sidecar path → FAIL; all-good → PASS.
- Sidecar staging: absent → WARN; present+arm64+exec → PASS; present+wrong-arch → FAIL.
- Hygiene: tracked `dist/` artifact → F1 FAIL; tracked `.p8` → F2 FAIL; clean → PASS.
- **Redaction:** feed an env map whose credential values are sentinel strings — including a sentinel **file path** for `CSC_LINK` and `APPLE_API_KEY` (e.g. `/Users/SENTINEL/key.p8`); assert the rendered report (and every `message`/`remediation`) contains **none** of those sentinel values **or paths** — only var names and `set`/`missing`/`exists`/`readable` tokens.
- Exit-code mapping: any FAIL → `1`; only WARN/INFO → `0`.

**Manual smoke (SMOKE.md line):** run `pnpm release:mac:check` on this machine; confirm it reports the two missing items and exits non-zero, and `grep` the output to confirm no value-looking secret appears.

## 9. Acceptance Criteria (current machine, no credentials)

- [ ] `pnpm release:mac:check` exits **non-zero** (`1`).
- [ ] The "Missing for a customer-ready notarized DMG" list contains **exactly two** items: signing credentials (B1 — neither a keychain Developer ID Application identity nor `CSC_LINK` + `CSC_KEY_PASSWORD` is present) and one notarization credential group (C1).
- [ ] All platform/tooling checks (A1–A4) PASS.
- [ ] All electron-builder config invariants (D1–D5) PASS against the committed `electron-builder.yml`.
- [ ] Artifact hygiene (F1), secret-file hygiene (F2) PASS; version (G1) PASS (`0.1.0`).
- [ ] Sidecar staging (E1) is WARN or PASS — never an unexpected FAIL on a clean checkout.
- [ ] **No secret value** appears anywhere in the output (verified by the redaction test and the manual `grep`).
- [ ] The command makes **no** network call, performs **no** build, and mutates **no** keychain or file (by design — verified by code review and the absence of those calls).
- [ ] Runs in under ~2 seconds.
- [ ] All existing gates stay green: `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change.

## 10. Files Expected to Change (for the implementation plan, not this pass)

- `apps/desktop/scripts/release-mac-check.mjs` — **Create.** Thin IO shell + report renderer + exit-code wiring.
- `apps/desktop/scripts/release-mac-check.lib.mjs` (or a `lib/` module) — **Create.** Pure evaluator + the types it returns. (Exact split confirmed at implementation; the requirement is that the pure logic is importable and unit-tested independent of process/filesystem.)
- `apps/desktop/scripts/release-mac-check.test.mjs` (Vitest) — **Create.** Unit tests per §8, including the redaction case.
- `apps/desktop/package.json` — **Modify.** Add the `release:mac:check` script entry. (No version change, no change to `package:mac` / `package:mac:unsigned`.)
- `SMOKE.md` — **Modify.** Add a short "Release Preflight (Slice 3B2A)" line under the signing sections.
- `SCAFFOLD.md` — **Modify.** Document `pnpm release:mac:check` as the pre-release gate to run before `pnpm package:mac`.

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core`, `scripts/build-core.mjs`, `electron-builder.yml`, and the entitlements plists are untouched.

## 11. Out of Scope — MUST NOT Touch

- **3B2 work** — no Developer ID cert handling, no real notarization, no stapling, no Gatekeeper test, no credential files committed or referenced as present.
- **Packaging behavior** — `package:mac`, `package:mac:unsigned`, `electron-builder.yml`, entitlements plists, `build:core` unchanged.
- **Product / RPC / schema / migrations** — no renderer, main, or `core-go` logic; contract-drift must stay clean.
- **CI release automation, auto-update, universal binary, Windows/Linux, `.pkg`/Mac App Store** — not added.
- **Secrets** — never read, print, commit, or log certificate/key/password/`.p8`/`.p12` contents.
