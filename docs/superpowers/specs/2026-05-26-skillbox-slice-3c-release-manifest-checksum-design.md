# Slice 3C: macOS Release Manifest + Checksum Generation — Design

Date: 2026-05-26
Status: lead-approved (scope + constraints recorded below)
Depends on: Slice 3B2C (`pnpm release:mac:full` = check → package → verify, committed and approved)
Relates to: Slice 3B2 (real Developer ID signing + notarization — blocked on Apple credentials, not required here)

## 0. PM Decision Recorded

3B2C produces a verified `.dmg` but ends there: the artifact ships with **no integrity contract** a customer can check against. 3C adds the standalone command `pnpm release:mac:manifest <dmg>` and wires manifest generation into `release:mac:full` **after** a successful `release:mac:verify`.

It is **credential-free** to author, unit-test, and run today: hashing and metadata extraction work on **any** `.dmg` regardless of signing status, so it lands against the 3A/3B1 unsigned/ad-hoc DMG without Apple credentials. When credentials arrive and a real notarized DMG exists, the same command emits the customer-facing integrity artifacts for it unchanged.

3C performs **file reads + two atomic file writes** (`dist/<artifact>.manifest.json`, `dist/SHA256SUMS`). It never builds, signs, notarizes, staples, reads credentials, touches the keychain, or makes a network request.

## 1. Purpose and Non-Goals

### Purpose
Add a customer-facing **artifact integrity contract** for the shipped DMG without needing Apple credentials. Given the exact path to a built `.dmg`, compute its SHA-256 and emit:
- `dist/<artifact>.manifest.json` — a structured manifest describing the artifact.
- `dist/SHA256SUMS` — a coreutils-compatible checksum line customers verify with `shasum -a 256 -c` / `sha256sum -c`.

Integrate the same generation step into `release:mac:full` so a full release run leaves the verified DMG **plus** its integrity artifacts in `dist/`.

### Non-Goals
- **No signing of the manifest or checksums** — no GPG, no detached signature, no `.asc`/`.sig`, no signing of `SHA256SUMS` itself.
- **No notarization / keychain / credential reading** — 3C never reads `.p8`/`.p12`/passwords/Apple env, never calls `notarytool`/`codesign -s`/`security`.
- **No network / upload / publish** — no release upload, no GitHub Release, no S3, no `latest-mac.yml` / auto-update feed.
- **No latest-DMG discovery or fuzzy selection** — the command hashes exactly the path passed in (see §2).
- **No product / RPC / schema / migration / contract changes.**
- **No build or packaging** — it never runs `build:core`, `electron-vite build`, or `electron-builder`.

## 2. Command / Input Resolution

- **Command:** `pnpm release:mac:manifest <dmg>` (added to `apps/desktop/package.json` `scripts`), invoked from `apps/desktop/`. The script resolves repo paths relative to its own location so cwd does not matter.
- **Input is a required explicit path.** The command hashes **exactly the DMG path passed in**. There is **no** `dist/` auto-discovery, no "latest DMG" heuristic, no glob, no fuzzy selection.
  - No argument → FAIL with usage (`release:mac:manifest <path-to-dmg>`); exit non-zero.
  - Path does not exist, is not a regular file, or does not end in `.dmg` → FAIL with a clear message; exit non-zero.
- **Output location:** both outputs are written to the **`dist/` directory of the desktop app** (`apps/desktop/dist/`), the same directory `package:mac` writes to — not next to an arbitrary input path. `<artifact>` is the **basename** of the supplied DMG (e.g. `Astraler Skillbox-0.1.0-arm64.dmg` → `Astraler Skillbox-0.1.0-arm64.dmg.manifest.json`). The command may hash an explicit DMG path outside `dist/` for development/testing, but customer verification assumes the DMG, manifest, and `SHA256SUMS` are co-located in `apps/desktop/dist/`; the release orchestrator passes the selected `dist/` DMG path.

## 3. Outputs

### 3.1 `dist/<artifact>.manifest.json`

A single JSON object with **exactly** these required fields (stable key order as listed):

| Field | Type | Semantics |
|-------|------|-----------|
| `appId` | string | Application bundle id. Sourced from `electron-builder.yml` `appId` (e.g. `com.astraler.skillbox`). |
| `productName` | string | Human product name. Sourced from `electron-builder.yml` `productName` (e.g. `Astraler Skillbox`). |
| `version` | string | Release version. Sourced from `apps/desktop/package.json` `version` (e.g. `0.1.0`). |
| `artifact` | string | **Basename** of the supplied DMG (no directory component). |
| `arch` | string | CPU architecture of the artifact (e.g. `arm64`). Sourced from `electron-builder.yml` `mac.target[].arch` where unambiguous; **filename parsing is the fallback** used only to obtain `arch` if config is ambiguous (see §4). |
| `byteSize` | integer | Exact size of the DMG in bytes, as an integer (not a string, no units). |
| `sha256` | string | SHA-256 of the DMG, **lowercase hex**, 64 chars. |
| `buildTimestamp` | string | Generation time as a **UTC ISO-8601** string (`YYYY-MM-DDTHH:mm:ss.sssZ`). This is the only wall-clock field; it is explicit and injected (see §6) so the rest of the manifest is deterministic. |

No additional fields in this slice. The object is serialized with a trailing newline and 2-space indentation.

### 3.2 `dist/SHA256SUMS`

Coreutils / BSD-compatible checksum file. Each line:

```
<sha256-lowercase-hex><two spaces><artifact-basename>
```

- Exactly **two spaces** between hash and name (the format `shasum -a 256 -c` and `sha256sum -c` expect for binary/text default).
- The name is the **artifact basename only** — never a path.
- File ends with a trailing newline.

**Deterministic update behavior (append-or-replace, never duplicate):**
- `SHA256SUMS` is treated as a multi-line accumulator that may already contain lines for other artifacts (e.g. a future `.zip`, or a prior version).
- On each run, the line for the **current artifact basename** is upserted: if a line with that exact basename already exists, it is **replaced** in place (no stale duplicate line for the same basename); otherwise the new line is appended.
- Basenames are **normalized** (trimmed, compared exactly) so the same artifact never yields two lines that differ only by surrounding whitespace.
- Other artifacts' lines are preserved and their relative order is kept; the upserted line keeps its existing position when replaced, or goes last when newly appended. Output ordering is otherwise stable across runs for identical inputs.

## 4. Metadata Sourcing Rules

Order of preference, **config first, filename parsing last and only where stated**:
- `appId`, `productName` → read from `apps/desktop/electron-builder.yml` (`appId`, `productName`). These are not derivable from the filename and must come from config.
- `version` → read from `apps/desktop/package.json` `version`.
- `arch` → prefer `electron-builder.yml` `mac.target`'s declared `arch`. If exactly one arch is declared (current config: `arm64`), use it. **Filename parsing is used only for `arch`** and only as a fallback when config does not yield a single unambiguous arch (e.g. multiple arches configured) — parse the trailing `-<arch>.dmg` token. No other field is ever derived from the filename.
- `artifact` → basename of the supplied path (string manipulation, not config).
- `byteSize`, `sha256` → from the file itself (§5).
- `buildTimestamp` → injected clock (§6).

If a required config value cannot be read (missing/un-parseable `electron-builder.yml` or `package.json`, or `arch` cannot be resolved by config **or** filename) → FAIL with a clear message; **do not** emit a manifest with guessed/empty fields.

## 5. Hashing

- The IO shell **streams** the DMG through a SHA-256 hasher (Node `crypto.createHash("sha256")` fed by a read stream) — never loads the whole file into memory.
- `byteSize` is taken from `fs.stat` (or accumulated from the stream) as an integer number of bytes.
- Hash digest is emitted as **lowercase hex**.

## 6. Architecture (pure core + thin IO shell)

Mirrors 3B2A/3B2B/3B2C "pure core + thin IO shell" discipline. Modules under `apps/desktop/scripts/`:

- **`release-mac-manifest.lib.mjs` (pure)** — no process/fs/crypto/clock. Functions:
  - `buildManifest({ appId, productName, version, artifact, arch, byteSize, sha256, buildTimestamp })` → the ordered manifest object (validates presence/shape of each field; throws a clear error on a missing/empty required field). Guarantees stable key ordering and integer `byteSize`.
  - `renderManifestJson(manifest)` → deterministic JSON string (2-space indent, trailing newline).
  - `resolveArch({ configArches, artifactBasename })` → the single arch string, applying the §4 config-first / filename-fallback rule, or an error.
  - `upsertSha256Line({ existingContent, sha256, artifact })` → new `SHA256SUMS` content string implementing the §3.2 normalized append-or-replace (no stale duplicate, preserves other lines/order, two-space separator, trailing newline).
  - `parseArchFromFilename(basename)` → arch token or null (used only by `resolveArch` fallback).
- **`release-mac-manifest.io.mjs` (small IO helper)** — exports the testable atomic-write primitive used by the shell. It writes a temp sibling file then renames over the final path, with dependency injection in tests to simulate write failure.
- **`release-mac-manifest.mjs` (thin IO shell)** — argument/path validation (§2), reads `electron-builder.yml` + `package.json`, streams the DMG to compute `sha256` + `byteSize` (§5), gets `buildTimestamp` from `new Date().toISOString()`, calls the pure functions, performs the **atomic writes** (§7), wires exit codes. Not unit-tested directly (covered by the SMOKE line + acceptance run).

## 7. Atomic Writes (no truncated output)

Both `dist/<artifact>.manifest.json` and `dist/SHA256SUMS` are written **atomically**:
- Write full content to a temp file in the **same directory** (`dist/`) — e.g. `mkstemp`-style sibling — then `fs.rename` over the final path. `rename` within one filesystem is atomic, so a reader never observes a partial file and a crash mid-write never leaves a truncated/corrupt target.
- For `SHA256SUMS`, the existing file (if any) is read first, the new content is computed in memory via `upsertSha256Line`, then the whole content is written to the temp file and renamed — so a failed write **leaves the previous `SHA256SUMS` intact** (no in-place truncation).
- On any write error, the command exits non-zero and leaves no partial/temp file as the visible output (best-effort temp cleanup).

## 8. Integration into `release:mac:full`

- A new **`manifest` stage** is added to `runReleaseMacFull` **after** the `verify` stage, and runs **only when `verify` succeeded** for the selected DMG. Ordering: `preflight → package → select-dmg → verify → manifest`.
- The manifest stage is invoked as `release:mac:manifest <selected-dmg>`, reusing the **exact** DMG path the orchestrator already selected and verified (the `selectChangedDmg` result). It performs **no** independent discovery — consistent with §2.
- Fail-fast preserved: if `manifest` exits non-zero, `release:mac:full` stops with `failedStage: "manifest"` and a non-zero exit, with a clear stopped-message; the success message is extended to report the manifest + `SHA256SUMS` paths.
- Because credentials are not yet installed, a real `release:mac:full` still stops at `preflight` today; the `manifest` stage wiring is exercised by unit tests (injected `runStage`) and by the **standalone** command against an unsigned/ad-hoc DMG.

## 9. Secret Hygiene and Side-Effect Boundaries

- **No secrets involved.** 3C reads a DMG, `electron-builder.yml`, and `package.json` — never credentials, `.p8`/`.p12`, passwords, or Apple env. Nothing read here is secret; nothing secret is printed.
- **Side effects: two atomic file writes into `dist/`** (§7) plus file reads. The script does **not**: build, sign, notarize, staple, run `build:core`/`electron-vite build`/`electron-builder`, make any network request, call any Apple service, or mutate the keychain.

## 10. Output and Exit Codes

- Human-readable progress to stdout: resolved artifact basename, computed sha256, byteSize, and the two output paths written.
- **Exit codes:** `0` = manifest + `SHA256SUMS` written successfully; non-zero = bad/missing argument, unreadable DMG, un-resolvable required metadata, or a write failure. No other codes; internal errors throw with a clear message (still non-zero).

## 11. Test Strategy

Unit-tested with Vitest on the **pure lib** (`release-mac-manifest.lib.mjs`). The IO shell (path validation, streaming hash, atomic writes) is covered by the SMOKE line + acceptance run, not unit-tested directly.

Pure-lib cases:
- `buildManifest`: all eight fields present → object with **exact key order** `appId, productName, version, artifact, arch, byteSize, sha256, buildTimestamp`; `byteSize` is an integer; missing/empty any required field → throws; non-integer `byteSize` → throws; non-lowercase-hex or wrong-length `sha256` → throws.
- `renderManifestJson`: byte-stable output across repeated calls for identical input (2-space indent, trailing newline); round-trips via `JSON.parse`.
- `resolveArch`: single configured arch (`["arm64"]`) → `arm64` (config wins, filename ignored); ambiguous config (`["arm64","x64"]`) → falls back to filename token; config empty + filename `…-arm64.dmg` → `arm64`; neither resolvable → error.
- `parseArchFromFilename`: `Astraler Skillbox-0.1.0-arm64.dmg` → `arm64`; name with no arch token → null.
- `upsertSha256Line`:
  - empty existing content + one artifact → single line, two-space separator, trailing newline.
  - existing content already containing **the same basename** → line **replaced in place**, no duplicate, other lines untouched and in original order.
  - existing content with **a different artifact's** line → current line appended, prior line preserved.
  - existing line for the same basename with trailing/leading whitespace differences → normalized to exactly one canonical line (no stale duplicate).
  - re-running with identical inputs → byte-identical output (idempotent).
- Orchestrator wiring (`release-mac-full.test.mjs`, extended): with injected `runStage`, a successful `verify` triggers a `manifest` stage invoked with the **selected DMG path**; a failing `verify` **never** runs `manifest`; a failing `manifest` yields `failedStage: "manifest"` and non-zero exit; full success reports manifest paths.

**Manual smoke (SMOKE.md):** build the unsigned/ad-hoc DMG (3A/3B1), run `pnpm release:mac:manifest "dist/<that>.dmg"`, then verify integrity from the shell:
```sh
cd apps/desktop/dist && shasum -a 256 -c SHA256SUMS   # and: sha256sum -c SHA256SUMS
```
Confirm the line passes, `manifest.json` contains all eight fields with a UTC ISO-8601 `buildTimestamp` and lowercase-hex `sha256`, and a second run does not add a duplicate line.

## 12. Acceptance Criteria (current machine, no credentials)

- [ ] `pnpm release:mac:manifest <dmg>` is added and runnable from `apps/desktop/`.
- [ ] Command hashes **exactly the path passed in**; no argument / non-existent / non-`.dmg` path → non-zero exit with a clear message and **no** discovery/fuzzy fallback.
- [ ] Running against an unsigned/ad-hoc DMG writes `dist/<artifact>.manifest.json` and `dist/SHA256SUMS`.
- [ ] Manifest contains **exactly** `appId, productName, version, artifact, arch, byteSize, sha256, buildTimestamp` with the stated semantics: UTC ISO-8601 `buildTimestamp`, lowercase-hex 64-char `sha256`, integer `byteSize`, basename-only `artifact`, `appId`/`productName`/`version` from config, `arch` from config (filename fallback only).
- [ ] From `apps/desktop/dist/`, `shasum -a 256 -c SHA256SUMS` **and** `sha256sum -c SHA256SUMS` pass against the real artifact.
- [ ] Re-running for the same artifact leaves **no stale duplicate line** in `SHA256SUMS`; a different artifact's line is preserved.
- [ ] `renderManifestJson` output is byte-stable for identical inputs (only `buildTimestamp` varies, and only because the clock advances).
- [ ] Writes are atomic: a simulated/forced write failure leaves the previous `SHA256SUMS` intact and no truncated `manifest.json` (verified by code review of the temp-file + rename path and a unit/integration check).
- [ ] `release:mac:full` runs the `manifest` stage **only after** a successful `release:mac:verify <selected-dmg>`, uses the selected DMG path, fails fast on manifest error (`failedStage: "manifest"`), and reports manifest paths on success (verified via orchestrator unit tests with injected `runStage`).
- [ ] No signing, notarization, keychain, network, upload, credential read, or GPG/signature behavior anywhere in the slice.
- [ ] No secret value or credential path appears in output.
- [ ] All existing gates stay green: `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change.

## 13. Risks / Open Questions

- **Manifest schema longevity.** Adding fields later (e.g. a `.zip` artifact, multiple arches, a `format`/`schemaVersion` field) is likely. Lead decision for 3C: **defer `schemaVersion`** and keep the manifest contract to exactly the eight fields listed in §3. A future schema-versioned manifest should be a deliberate contract change, not a hidden expansion of this slice.
- **One manifest per artifact vs. one combined manifest.** This slice emits one `*.manifest.json` per DMG and a single shared `SHA256SUMS`. If a future release ships multiple artifacts, `SHA256SUMS` already accumulates them; whether to also emit a combined top-level manifest is deferred.
- **`arch` ambiguity.** Current config declares a single `arm64` target, so config-first resolution is unambiguous today. The filename fallback exists only to avoid a hard failure if multi-arch is configured later; if neither resolves, the command fails rather than guessing.
- **Value of the manifest before notarization.** Until Apple credentials land, the only DMG available is unsigned/ad-hoc, so the manifest describes a non-customer-ready artifact. This is acceptable: the slice's value is the **mechanism** (built, tested, wired into `release:mac:full`); the same command emits the real customer-facing integrity artifacts unchanged once a notarized DMG exists. **No credential input is required to complete or merge 3C.**

## 14. Files Expected to Change (for the implementation plan, not this pass)

- `apps/desktop/scripts/release-mac-manifest.mjs` — **Create.** Thin IO shell: arg/path validation, config + package.json reads, streaming SHA-256, atomic writes of `manifest.json` + `SHA256SUMS`, exit wiring.
- `apps/desktop/scripts/release-mac-manifest.lib.mjs` — **Create.** Pure `buildManifest`, `renderManifestJson`, `resolveArch`, `parseArchFromFilename`, `upsertSha256Line`.
- `apps/desktop/scripts/release-mac-manifest.io.mjs` — **Create.** Testable `atomicWrite` helper with injected fs dependency for write-failure tests.
- `apps/desktop/scripts/release-mac-manifest.test.mjs` (Vitest) — **Create.** Pure-lib cases per §11.
- `apps/desktop/scripts/release-mac-full.lib.mjs` — **Modify.** Add the `manifest` stage after `verify` in `runReleaseMacFull` (only on verify success; fail-fast `failedStage: "manifest"`).
- `apps/desktop/scripts/release-mac-full.mjs` — **Modify.** Handle the new `manifest` failed-stage message; extend the success message to report manifest + `SHA256SUMS` paths.
- `apps/desktop/scripts/release-mac-full.test.mjs` — **Modify.** Add manifest-stage wiring cases (success-after-verify, skip-on-verify-fail, fail-fast on manifest error).
- `apps/desktop/package.json` — **Modify.** Add the `release:mac:manifest` script entry. (No version change; `package:mac` / `release:mac:check` / `release:mac:verify` untouched.)
- `SMOKE.md` — **Modify.** Add a "Release Manifest + Checksums (Slice 3C)" section with the `shasum -c` / `sha256sum -c` verification steps.
- `SCAFFOLD.md` — **Modify.** Document `pnpm release:mac:manifest` and the extended `release:mac:full` flow.

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core`, `build-core.mjs`, `electron-builder.yml`, the entitlements plists, and the `release-mac-check.*` / `release-mac-verify.*` scripts are untouched.

## 15. Out of Scope — MUST NOT Touch

- **Signing / notarization / stapling / keychain / Apple credentials / network** — never invoked, read, fetched, printed, or required.
- **GPG / detached signatures / signing of `SHA256SUMS`** — not added.
- **Upload / publish / GitHub Release / auto-update feed (`latest-mac.yml`)** — not added.
- **Latest-DMG discovery / glob / fuzzy artifact selection** — the command hashes only the explicit path.
- **Packaging / config** — `package:mac`, `package:mac:unsigned`, `electron-builder.yml`, entitlements plists, `build:core` unchanged.
- **Product / RPC / schema / migrations** — no renderer, main, or `core-go` logic; contract-drift stays clean.
- **CI release automation, Windows/Linux artifacts, `.pkg`/Mac App Store** — not added.
