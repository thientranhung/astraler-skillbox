# Slice 3E: No-Credential macOS Release Dry-Run — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Use `/goal` only for the implementation/verification loop after lead approval.

**Goal:** Add `pnpm release:mac:dry-run`, a non-distributable no-credential end-to-end harness that proves the local build → ad-hoc sign → verify → manifest/checksum chain works before Apple signing/notarization credentials are installed.

**Lead-approved correction:** Do **not** use `package:mac:unsigned`; it disables signing/hardened runtime and will not satisfy `release:mac:verify --allow-adhoc`. Dry-run packaging must use ad-hoc signing with `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`, preserving hardened runtime, entitlements, and `mac.binaries`.

**Hard Constraints:**
- Do not invoke `release:mac:check`, signed `package:mac`, notarization, keychain, network, upload, or credential reads.
- Do not call `package:mac:unsigned`.
- Output must loudly label artifacts `NON-DISTRIBUTABLE`, `AD-HOC`, and `NOT NOTARIZED`.
- Select exactly one newly created/modified DMG using before/after metadata.
- Do not blindly delete `dist/` or shared `SHA256SUMS`; preserve user/customer artifacts.

---

## File Structure

- **Create** `apps/desktop/scripts/release-mac-dry-run.lib.mjs` — pure orchestration helpers.
- **Create** `apps/desktop/scripts/release-mac-dry-run.mjs` — IO shell that spawns stages, snapshots `dist/`, and verifies checksum.
- **Create** `apps/desktop/scripts/release-mac-dry-run.test.mjs` — Vitest coverage for orchestration and safety constraints.
- **Modify** `apps/desktop/package.json` — add `release:mac:dry-run`.
- **Modify** `SMOKE.md`, `SCAFFOLD.md`, and `RELEASE.md` — document the dry-run command and its non-distributable purpose.

---

## Task 1: Pure Orchestrator

**Files:**
- Create `apps/desktop/scripts/release-mac-dry-run.lib.mjs`
- Create `apps/desktop/scripts/release-mac-dry-run.test.mjs`

- [ ] Reuse/import `selectChangedDmg` from `release-mac-full.lib.mjs` for before/after DMG selection.
- [ ] Add `runReleaseMacDryRun({ runStage, snapshotDist, verifyChecksum, now })`.
- [ ] Flow:
  1. snapshot `dist/*.dmg` before packaging.
  2. run ad-hoc package stage.
  3. snapshot `dist/*.dmg` after packaging.
  4. select exactly one created/modified DMG.
  5. run `release:mac:verify --allow-adhoc <selected-dmg>`.
  6. run `release:mac:manifest <selected-dmg>` only after verify passes.
  7. run checksum verification for only the selected artifact's `SHA256SUMS` line by filtering that line into a temporary check input and running `shasum -a 256 -c` from `dist/`.
- [ ] Preserve fail-fast:
  - package failure stops before verify/manifest/checksum.
  - DMG selection failure stops before verify.
  - verify failure stops before manifest.
  - manifest failure stops before checksum.
  - checksum failure fails the dry-run.
- [ ] Tests cover success order, each failure boundary, selected DMG path propagation, `--allow-adhoc` on verify, no `release:mac:check`, no `package:mac`, no `package:mac:unsigned`, exact ad-hoc package flags, no `-c.mac.hardenedRuntime=false`, and manifest only after verify.

## Task 2: IO Shell and Ad-Hoc Package Stage

**Files:**
- Create `apps/desktop/scripts/release-mac-dry-run.mjs`
- Modify `apps/desktop/package.json`

- [ ] Add script: `"release:mac:dry-run": "node scripts/release-mac-dry-run.mjs"`.
- [ ] Implement snapshot of regular `dist/*.dmg` using `lstat` so symlink DMGs are ignored.
- [ ] Missing `dist/` snapshots as `[]`.
- [ ] Spawn package stage as:
  - `pnpm build:core`
  - `pnpm build`
  - `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false`
- [ ] Do **not** set `mac.hardenedRuntime=false`; hardened runtime must remain enabled.
- [ ] Do **not** use `CSC_IDENTITY_AUTO_DISCOVERY=false ... identity=null`.
- [ ] Spawn verify as `pnpm release:mac:verify --allow-adhoc <selected-dmg>`.
- [ ] Spawn manifest as `pnpm release:mac:manifest <selected-dmg>`.
- [ ] Verify checksum from `apps/desktop/dist/` by extracting only the line whose basename matches the selected DMG into a temporary check file, then running `shasum -a 256 -c <temp-check-file>`. This preserves shared `SHA256SUMS` and avoids failing on unrelated stale lines.
- [ ] Stream stage output with clear prefixes.
- [ ] Print start/success/failure banners that include `NON-DISTRIBUTABLE`, `AD-HOC`, and `NOT NOTARIZED`.

## Task 3: Deterministic Reruns / Artifact Handling

**Files:**
- Implement in `release-mac-dry-run.mjs` and test pure behavior where practical.

- [ ] Prefer before/after metadata selection over manual empty `dist/`.
- [ ] Do not delete all of `dist/`.
- [ ] Do not delete or truncate `SHA256SUMS`; `release:mac:manifest` owns deterministic upsert.
- [ ] If multiple DMGs are created/modified, fail clearly and ask the operator to clean conflicting build outputs.
- [ ] If no DMG is created/modified, fail clearly.
- [ ] Do not introduce upload/distribution behavior.

## Task 4: Documentation

**Files:**
- Modify `SMOKE.md`
- Modify `SCAFFOLD.md`
- Modify `RELEASE.md`

- [ ] Document `pnpm release:mac:dry-run` as a no-credential, non-distributable end-to-end local harness.
- [ ] Explicitly distinguish it from:
  - `release:mac:full` — customer release path gated by credentials.
  - `package:mac:unsigned` — packaging-only smoke that does not prove verifier/manifest chain.
- [ ] Include expected output/artifacts and warning that Gatekeeper/customer distribution still requires real signing + notarization.

## Task 5: Verification

- [ ] `cd apps/desktop && pnpm exec vitest run scripts/release-mac-dry-run.test.mjs scripts/release-mac-full.test.mjs scripts/release-mac-manifest.test.mjs`
- [ ] `cd apps/desktop && pnpm test`
- [ ] `cd apps/desktop && pnpm typecheck`
- [ ] `cd apps/desktop && pnpm check:contracts-drift`
- [ ] `cd core-go && go test ./...`
- [ ] `cd apps/desktop && pnpm build`
- [ ] `cd apps/desktop && pnpm release:mac:dry-run`
- [ ] Confirm dry-run exits `0`, emits `NON-DISTRIBUTABLE` / `AD-HOC` / `NOT NOTARIZED`, produces a DMG, verifies it with `--allow-adhoc`, writes manifest + `SHA256SUMS`, and verifies only the selected artifact's checksum line with `shasum -a 256 -c`.
- [ ] `cd apps/desktop && pnpm release:mac:full` still exits at preflight on this no-credential machine and does not package.

## Risks / Notes

- This does **not** prove Developer ID signing, notarization, stapling, or Gatekeeper acceptance for customer distribution. It proves the local release chain excluding Apple credentials.
- The ad-hoc package step is heavier than unit tests and may take minutes; unit tests should mock spawns, while the real dry-run is a smoke/manual gate.
- If electron-builder emits multiple changed DMGs, the command should fail rather than guessing.

**Ready for lead review.**
