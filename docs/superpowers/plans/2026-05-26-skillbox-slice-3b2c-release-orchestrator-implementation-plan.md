# Slice 3B2C: macOS Release Orchestrator — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Use `/goal` only for the implementation/verification loop, not for plan review.

**Goal:** Add `pnpm release:mac:full` as the canonical customer-release command for macOS. It composes the existing release gates in the only safe order: preflight first, signed packaging second, artifact verification last.

**Architecture:** Add a small Node ESM orchestrator under `apps/desktop/scripts/`, following the existing pure helper + thin IO shell pattern from 3B2A/3B2B. The pure helper owns stage decisions and DMG selection; the shell owns filesystem snapshots and child process spawning.

**HARD CONSTRAINTS (do not violate):**
- Do not modify app behavior, DB, contracts, Go core, Electron IPC, or provider logic.
- Do not call `package:mac:unsigned` from this command.
- Do not read, store, or print secret environment values.
- Do not delete stale `dist/` artifacts automatically.
- Do not pass `--allow-adhoc` to `release:mac:verify`.

---

## File Structure

- **Create** `apps/desktop/scripts/release-mac-full.lib.mjs` — pure: stage result helpers and `selectChangedDmg(before, after, packageStartMs)`.
- **Create** `apps/desktop/scripts/release-mac-full.mjs` — thin IO shell: snapshot `dist/*.dmg`, run child stages, stream prefixed output, exit wiring.
- **Create** `apps/desktop/scripts/release-mac-full.test.mjs` — Vitest coverage for pure helpers and injectable orchestration flow.
- **Modify** `apps/desktop/package.json` — add `"release:mac:full": "node scripts/release-mac-full.mjs"`.
- **Modify** `SMOKE.md` — document `pnpm release:mac:full` as the signed/notarized release path and current credential-less fail-fast check.
- **Modify** `SCAFFOLD.md` — add the release orchestrator to the packaging/release command list.

---

## Task 1: Pure DMG Selection Helper

**Files:**
- Create: `apps/desktop/scripts/release-mac-full.lib.mjs`
- Test: `apps/desktop/scripts/release-mac-full.test.mjs`

- [ ] Write failing tests for `selectChangedDmg(before, after, packageStartMs)`.
- [ ] Implement selection semantics:
  - Inputs are arrays of stat records: `{ path, size, mtimeMs, isFile }`.
  - Only regular files with `.dmg` suffix are candidates.
  - A candidate is **created** when its path was absent before.
  - A candidate is **modified** when its path existed before and `size` or `mtimeMs` changed.
  - `packageStartMs` may be used to require changed/new candidates to have `mtimeMs >= packageStartMs` where available, but must not make same-name overwrite detection rely on filename changes.
  - Return success only when exactly one candidate is created or modified.
  - Return clear errors for zero candidates and multiple candidates.
  - Stale unchanged DMGs are ignored.
- [ ] Cover one new DMG, same-name overwrite, stale unchanged DMGs, multiple changed/new DMGs, non-regular `.dmg`, and zero changed DMGs.

## Task 2: Orchestration Flow

**Files:**
- Add tests to `apps/desktop/scripts/release-mac-full.test.mjs`
- Implement in `apps/desktop/scripts/release-mac-full.lib.mjs`

- [ ] Add an injectable `runReleaseMacFull({ runStage, snapshotDist, now })` pure/near-pure orchestrator so unit tests do not spawn real builds.
- [ ] Flow:
  1. Run stage `preflight`: `pnpm release:mac:check`.
  2. If non-zero, stop with non-zero and do not snapshot/package/verify beyond any initial read needed to prove no changes.
  3. Snapshot `dist/*.dmg` before package.
  4. Record `packageStartMs = now()`.
  5. Run stage `package`: `pnpm package:mac`.
  6. If non-zero, stop and do not verify.
  7. Snapshot `dist/*.dmg` after package and select exactly one changed DMG.
  8. Run stage `verify`: `pnpm release:mac:verify <selected-dmg>`.
  9. Exit 0 only if all stages pass.
- [ ] Tests cover preflight failure, package failure, verify failure, selected same-name overwrite, and selected new DMG.
- [ ] Tests cover successful package followed by DMG selection failure:
  - zero changed/new DMGs exits non-zero and does not invoke verify.
  - multiple changed/new DMGs exits non-zero and does not invoke verify.

## Task 3: Thin IO Shell

**Files:**
- Create: `apps/desktop/scripts/release-mac-full.mjs`
- Modify: `apps/desktop/package.json`

- [ ] Implement filesystem snapshot of `dist/*.dmg` using `fs.promises.readdir` + `stat`.
- [ ] Treat missing `apps/desktop/dist/` as an empty snapshot (`[]`) so a clean checkout can still package. Other filesystem errors must fail clearly.
- [ ] Spawn sibling pnpm scripts using `child_process.spawn("pnpm", ["release:mac:check"], { stdio: ["ignore", "pipe", "pipe"] })` and equivalent commands.
- [ ] Handle child process `error` events as failed stages with clear messages, covering missing `pnpm` or spawn permission errors.
- [ ] Stream stdout/stderr line-by-line with readable stage prefixes such as `[release:mac:check]`.
- [ ] Normalize any failed stage to a non-zero exit with a clear message naming the failed stage.
- [ ] Ensure `release:mac:verify` receives the explicit selected DMG path and no `--allow-adhoc`.

## Task 4: Docs and Verification

**Files:**
- Modify: `SMOKE.md`
- Modify: `SCAFFOLD.md`

- [ ] Document `pnpm release:mac:full` as the canonical signed release orchestrator.
- [ ] Document current expected result on machines without Apple credentials: fails during preflight and does not package.
- [ ] Run targeted tests:
  - `cd apps/desktop && pnpm exec vitest run scripts/release-mac-full.test.mjs`
  - `cd apps/desktop && pnpm test`
  - `cd apps/desktop && pnpm typecheck`
  - `cd apps/desktop && pnpm check:contracts-drift`
  - `cd core-go && go test ./...`
  - `cd apps/desktop && pnpm build`
- [ ] Run live fail-fast smoke:
  - Snapshot `apps/desktop/dist/` before.
  - `cd apps/desktop && pnpm release:mac:full`
  - Confirm it exits non-zero at `release:mac:check`, does not run `package:mac`, and leaves `dist/` unchanged.

---

## Risks / Open Questions

- Same-name overwrites are the key risk; tests must prove metadata-based detection, not filename-only diffing.
- If electron-builder produces multiple DMGs in one run, this command should fail and require manual cleanup or explicit verification outside the orchestrator.
- We cannot verify the successful signed/notarized path on this machine until Apple signing and notarization credentials are installed. That is acceptable for this slice because fail-fast behavior is fully testable now.

**Recommendation:** Ready for implementation after lead review. Scope is small, additive, and release-hardening only.
