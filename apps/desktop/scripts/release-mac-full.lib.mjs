/**
 * Pure helpers for the macOS release orchestrator (Slice 3B2C).
 * No I/O, no child_process. All side-effecting code lives in release-mac-full.mjs.
 */

/**
 * @typedef {{ path: string, size: number, mtimeMs: number, isFile: boolean }} StatRecord
 * @typedef {{ ok: true, dmgPath: string, reason: 'created'|'modified' } | { ok: false, error: string }} SelectResult
 */

/**
 * Select the one DMG that was created or modified between two dist/ snapshots.
 *
 * A file is a candidate when:
 *   - created: its path was absent in `before`
 *   - modified: its path existed in `before` and size or mtimeMs changed (same-name overwrite)
 * Stale unchanged DMGs (path present, size and mtimeMs identical) are ignored.
 * Non-regular files (.dmg directories, symlinks) are excluded via isFile.
 *
 * `packageStartMs` is accepted for completeness but the before/after snapshot comparison
 * already isolates changes to the package run; filename-based detection is never used.
 *
 * @param {StatRecord[]} before - snapshot of dist/*.dmg before package:mac
 * @param {StatRecord[]} after  - snapshot of dist/*.dmg after package:mac
 * @param {number} _packageStartMs - timestamp when package:mac was started (accepted, not required)
 * @returns {SelectResult}
 */
export function selectChangedDmg(before, after, _packageStartMs) {
  const afterDmgs = after.filter((r) => r.isFile && r.path.endsWith(".dmg"));

  const beforeMap = new Map(
    before.filter((r) => r.isFile && r.path.endsWith(".dmg")).map((r) => [r.path, r])
  );

  const candidates = [];

  for (const a of afterDmgs) {
    const b = beforeMap.get(a.path);
    if (!b) {
      // Created: path absent before
      candidates.push({ ...a, reason: "created" });
    } else if (a.size !== b.size || a.mtimeMs !== b.mtimeMs) {
      // Modified: same-name overwrite detected via metadata
      candidates.push({ ...a, reason: "modified" });
    }
    // Stale unchanged: size and mtimeMs identical → skip
  }

  if (candidates.length === 0) {
    return {
      ok: false,
      error:
        "no .dmg was created or modified during package:mac — cannot select a DMG to verify",
    };
  }

  if (candidates.length > 1) {
    const paths = candidates.map((c) => c.path).join(", ");
    return {
      ok: false,
      error: `multiple changed/new .dmg files found (${candidates.length}): ${paths} — pass an explicit path to release:mac:verify`,
    };
  }

  return { ok: true, dmgPath: candidates[0].path, reason: candidates[0].reason };
}

/**
 * Injectable orchestrator for the full macOS release flow.
 *
 * Flow: preflight → snapshot before → package → snapshot after → select dmg → verify → manifest
 *
 * All I/O is injected so unit tests can run without spawning real processes.
 *
 * @param {{
 *   runStage: (stage: string, args: string[]) => Promise<{code: number, manifestPath?: string, sha256sumsPath?: string}>,
 *   snapshotDist: () => Promise<StatRecord[]>,
 *   now: () => number,
 * }} deps
 * @returns {Promise<{ exitCode: number, failedStage?: string, dmgError?: string, dmgPath?: string, dmgReason?: string, manifestPath?: string, sha256sumsPath?: string }>}
 */
export async function runReleaseMacFull({ runStage, snapshotDist, now }) {
  // Stage 1: preflight
  const preflightResult = await runStage("preflight", ["release:mac:check"]);
  if (preflightResult.code !== 0) {
    return { exitCode: preflightResult.code, failedStage: "preflight" };
  }

  // Snapshot dist before package
  const beforeSnapshot = await snapshotDist();
  const packageStartMs = now();

  // Stage 2: package
  const packageResult = await runStage("package", ["package:mac"]);
  if (packageResult.code !== 0) {
    return { exitCode: packageResult.code, failedStage: "package" };
  }

  // Snapshot dist after package and select the one changed DMG
  const afterSnapshot = await snapshotDist();
  const selected = selectChangedDmg(beforeSnapshot, afterSnapshot, packageStartMs);

  if (!selected.ok) {
    return { exitCode: 1, failedStage: "dmg-selection", dmgError: selected.error };
  }

  // Stage 3: verify — pass explicit DMG path, never --allow-adhoc
  const verifyResult = await runStage("verify", [
    "release:mac:verify",
    selected.dmgPath,
  ]);
  if (verifyResult.code !== 0) {
    return { exitCode: verifyResult.code, failedStage: "verify" };
  }

  // Stage 4: manifest — only after successful verify, using the same selected DMG path
  const manifestResult = await runStage("manifest", [
    "release:mac:manifest",
    selected.dmgPath,
  ]);
  if (manifestResult.code !== 0) {
    return { exitCode: manifestResult.code, failedStage: "manifest" };
  }

  return {
    exitCode: 0,
    dmgPath: selected.dmgPath,
    dmgReason: selected.reason,
    manifestPath: manifestResult.manifestPath,
    sha256sumsPath: manifestResult.sha256sumsPath,
  };
}
