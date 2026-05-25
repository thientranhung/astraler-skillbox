# Slice 3C: Release Manifest + Checksum Generation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Use `/goal` only for the implementation/verification loop after this plan is approved.

**Goal:** Add `pnpm release:mac:manifest <dmg>` and extend `release:mac:full` so a verified release DMG also emits customer-checkable integrity artifacts: `dist/<artifact>.manifest.json` and `dist/SHA256SUMS`.

**Spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3c-release-manifest-checksum-design.md`

**Hard Constraints:**
- Explicit DMG path only. No latest-DMG discovery, globbing, or fuzzy selection.
- Manifest has exactly eight fields: `appId`, `productName`, `version`, `artifact`, `arch`, `byteSize`, `sha256`, `buildTimestamp`.
- `SHA256SUMS` uses basename-only entries and deterministic upsert behavior; no stale duplicate line for the same basename.
- Writes are atomic via temp file in `dist/` then rename.
- `release:mac:full` runs manifest generation only after successful `release:mac:verify <selected-dmg>`.
- No signing, notarization, keychain, network, upload, credential reads, GPG, contract, schema, or product changes.

---

## File Structure

- **Create** `apps/desktop/scripts/release-mac-manifest.lib.mjs` — pure manifest/checksum helpers.
- **Create** `apps/desktop/scripts/release-mac-manifest.io.mjs` — small IO helper for testable atomic writes.
- **Create** `apps/desktop/scripts/release-mac-manifest.mjs` — IO shell: explicit path validation, config reads, streaming hash, atomic writes, exit wiring.
- **Create** `apps/desktop/scripts/release-mac-manifest.test.mjs` — Vitest coverage for pure helpers.
- **Modify** `apps/desktop/scripts/release-mac-full.lib.mjs` — add manifest stage after verify.
- **Modify** `apps/desktop/scripts/release-mac-full.mjs` — add manifest failure/success messaging.
- **Modify** `apps/desktop/scripts/release-mac-full.test.mjs` — add manifest-stage orchestration tests.
- **Modify** `apps/desktop/package.json` — add `release:mac:manifest`.
- **Modify** `SMOKE.md` and `SCAFFOLD.md` — document command and smoke checks.

---

## Task 1: Pure Manifest Helpers

**Files:**
- Create `apps/desktop/scripts/release-mac-manifest.lib.mjs`
- Create `apps/desktop/scripts/release-mac-manifest.test.mjs`

- [ ] Write failing tests for:
  - `buildManifest` returns exactly the eight fields in stable key order.
  - missing/empty required fields throw clear errors.
  - `byteSize` must be an integer.
  - `sha256` must be 64 chars of lowercase hex.
  - `renderManifestJson` is 2-space pretty JSON with trailing newline and byte-stable for identical input.
  - `parseArchFromFilename` handles `Astraler Skillbox-0.1.0-arm64.dmg` and returns null for no arch token.
  - `resolveArch` prefers a single config arch, falls back to filename only when config is ambiguous/empty, and errors when unresolved.
  - `upsertSha256Line` creates a canonical two-space line, replaces an existing line for the same basename in place, preserves other artifacts/order, trims basename comparison, appends for new artifact, and is idempotent.
- [ ] Implement the pure helpers only. No `fs`, `process`, `crypto`, clock, env, or child process access.
- [ ] Run targeted tests:
  - `cd apps/desktop && pnpm exec vitest run scripts/release-mac-manifest.test.mjs`

## Task 2: Manifest IO Shell

**Files:**
- Create `apps/desktop/scripts/release-mac-manifest.mjs`
- Modify `apps/desktop/package.json`

- [ ] Add script: `"release:mac:manifest": "node scripts/release-mac-manifest.mjs"`.
- [ ] Validate exactly one required argument:
  - missing arg → usage + non-zero.
  - non-existent path, non-regular file, or non-`.dmg` → clear error + non-zero.
  - use `lstat` intentionally so symlink DMGs are rejected as non-regular files unless a future slice deliberately changes that behavior.
- [ ] Hash the exact supplied path with streaming `crypto.createHash("sha256")`.
- [ ] Get `byteSize` from stat as an integer.
- [ ] Read `apps/desktop/package.json` for `version`.
- [ ] Read `apps/desktop/electron-builder.yml` for `appId`, `productName`, and configured mac target arch; fail clearly if required metadata cannot resolve.
- [ ] Use `new Date().toISOString()` for UTC `buildTimestamp`.
- [ ] Always write outputs to `apps/desktop/dist/`:
  - `dist/<artifact-basename>.manifest.json`
  - `dist/SHA256SUMS`
- [ ] Implement atomic writes:
  - write to a unique temp sibling in `dist/`.
  - rename temp to final.
  - best-effort cleanup temp on failure.
  - read existing `SHA256SUMS`, compute upserted content in memory, then atomically replace.
  - keep the atomic write helper injectable/testable so write-failure behavior is covered by Vitest.
- [ ] Print concise human output: artifact basename, byte size, sha256, manifest path, SHA256SUMS path. Do not print env or credentials.

## Task 3: Integrate With `release:mac:full`

**Files:**
- Modify `apps/desktop/scripts/release-mac-full.lib.mjs`
- Modify `apps/desktop/scripts/release-mac-full.mjs`
- Modify `apps/desktop/scripts/release-mac-full.test.mjs`

- [ ] Extend orchestrator flow to `preflight → package → select-dmg → verify → manifest`.
- [ ] Invoke manifest as `release:mac:manifest <selected-dmg>`, reusing the exact selected/verified DMG path.
- [ ] Preserve fail-fast:
  - verify failure never runs manifest.
  - manifest failure returns non-zero with `failedStage: "manifest"`.
  - success reports the exact manifest path and `SHA256SUMS` path produced by the manifest stage.
- [ ] Add tests:
  - successful verify triggers manifest with selected DMG path.
  - verify failure skips manifest.
  - manifest failure fails the whole orchestrator.
  - `--allow-adhoc` is still never passed anywhere in `release:mac:full`.

## Task 4: Docs and Smoke

**Files:**
- Modify `SMOKE.md`
- Modify `SCAFFOLD.md`

- [ ] Document standalone `pnpm release:mac:manifest <dmg>`.
- [ ] Document extended `release:mac:full` flow including manifest stage after verify.
- [ ] Add manual smoke for unsigned/ad-hoc DMG:
  - `cd apps/desktop && pnpm release:mac:manifest "dist/<artifact>.dmg"`
  - `cd apps/desktop/dist && shasum -a 256 -c SHA256SUMS`
  - `cd apps/desktop/dist && sha256sum -c SHA256SUMS` when available.
  - confirm a second run does not duplicate the same basename line.
- [ ] Note that customer verification assumes the DMG, manifest, and `SHA256SUMS` are co-located in `dist/`.

## Task 5: Verification

- [ ] `cd apps/desktop && pnpm exec vitest run scripts/release-mac-manifest.test.mjs scripts/release-mac-full.test.mjs`
- [ ] `cd apps/desktop && pnpm test`
- [ ] `cd apps/desktop && pnpm typecheck`
- [ ] `cd apps/desktop && pnpm check:contracts-drift`
- [ ] `cd core-go && go test ./...`
- [ ] `cd apps/desktop && pnpm build`
- [ ] Manual manifest smoke against an existing unsigned/ad-hoc DMG in `apps/desktop/dist/`.
- [ ] Atomic-write failure check: simulate or force a write failure and verify the previous `SHA256SUMS` remains intact and no truncated manifest is left visible.
- [ ] Live `pnpm release:mac:full` on this no-credential machine still exits at preflight before package, so manifest stage is not reached live until Apple credentials exist.

## Risks / Notes

- `sha256sum` may not exist on stock macOS. The smoke should require `shasum -a 256 -c` and treat `sha256sum -c` as an additional check when installed.
- The manifest describes unsigned/ad-hoc artifacts until Apple credentials are installed; that is acceptable for validating the mechanism only.
- Do not add `schemaVersion` in 3C. The approved contract is exactly eight fields.

**Ready for lead review.**
