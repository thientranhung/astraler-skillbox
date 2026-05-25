/**
 * Pure helpers for the macOS release dry-run orchestrator (Slice 3E).
 * No I/O, no child_process. All side-effecting code lives in release-mac-dry-run.mjs.
 */

/**
 * @typedef {{ path: string, size: number, mtimeMs: number, isFile: boolean }} StatRecord
 */

import { selectChangedDmg as _selectChangedDmg } from "./release-mac-full.lib.mjs";
export { selectChangedDmg } from "./release-mac-full.lib.mjs";

const CREDENTIAL_PREFIXES = ["CSC_", "APPLE_", "NOTARYTOOL_"];

/**
 * Returns a copy of the given env with signing/notarization credential vars removed.
 * Strips all keys whose names start with CSC_, APPLE_, or NOTARYTOOL_.
 *
 * @param {Record<string, string|undefined>} env
 * @returns {Record<string, string|undefined>}
 */
export function scrubEnv(env) {
  return Object.fromEntries(
    Object.entries(env).filter(
      ([key]) => !CREDENTIAL_PREFIXES.some((prefix) => key.startsWith(prefix))
    )
  );
}

/**
 * Injectable orchestrator for the macOS release dry-run flow.
 *
 * Flow: snapshot before -> build:core -> build -> ad-hoc electron-builder ->
 *       snapshot after -> select dmg -> verify --allow-adhoc -> manifest -> checksum
 *
 * No preflight (release:mac:check) is invoked - this is a no-credential local harness.
 * The packaged DMG is NON-DISTRIBUTABLE, AD-HOC signed, and NOT NOTARIZED.
 *
 * @param {{
 *   runStage: (stage: string, args: string[]) => Promise<{code: number, manifestPath?: string, sha256sumsPath?: string}>,
 *   snapshotDist: () => Promise<StatRecord[]>,
 *   verifyChecksum: (dmgPath: string) => Promise<{code: number}>,
 *   now: () => number,
 * }} deps
 * @returns {Promise<{ exitCode: number, failedStage?: string, dmgError?: string, dmgPath?: string, dmgReason?: string, manifestPath?: string, sha256sumsPath?: string }>}
 */
export async function runReleaseMacDryRun({ runStage, snapshotDist, verifyChecksum, now }) {
  // Snapshot dist/*.dmg before packaging
  const beforeSnapshot = await snapshotDist();
  const packageStartMs = now();

  // Stage 1: build Go core
  const buildCoreResult = await runStage("build:core", ["build:core"]);
  if (buildCoreResult.code !== 0) {
    return { exitCode: buildCoreResult.code, failedStage: "build:core" };
  }

  // Stage 2: build JS/renderer
  const buildResult = await runStage("build", ["build"]);
  if (buildResult.code !== 0) {
    return { exitCode: buildResult.code, failedStage: "build" };
  }

  // Stage 3: ad-hoc package - identity=- (ad-hoc), notarize=false.
  // Do NOT pass -c.mac.hardenedRuntime=false - hardened runtime stays enabled.
  const pkgResult = await runStage("package-dmg", [
    "electron-builder",
    "--mac",
    "dmg",
    "-c.mac.identity=-",
    "-c.mac.notarize=false",
  ]);
  if (pkgResult.code !== 0) {
    return { exitCode: pkgResult.code, failedStage: "package-dmg" };
  }

  // Snapshot dist after packaging and select the one changed DMG
  const afterSnapshot = await snapshotDist();
  const selected = _selectChangedDmg(beforeSnapshot, afterSnapshot, packageStartMs);
  if (!selected.ok) {
    return { exitCode: 1, failedStage: "dmg-selection", dmgError: selected.error };
  }

  // Stage 4: verify with --allow-adhoc (ad-hoc signature accepted for dry-run)
  const verifyResult = await runStage("verify", [
    "release:mac:verify",
    "--allow-adhoc",
    selected.dmgPath,
  ]);
  if (verifyResult.code !== 0) {
    return { exitCode: verifyResult.code, failedStage: "verify" };
  }

  // Stage 5: manifest - only after successful verify
  const manifestResult = await runStage("manifest", [
    "release:mac:manifest",
    selected.dmgPath,
  ]);
  if (manifestResult.code !== 0) {
    return { exitCode: manifestResult.code, failedStage: "manifest" };
  }

  // Stage 6: checksum - only after successful manifest
  const checksumResult = await verifyChecksum(selected.dmgPath);
  if (checksumResult.code !== 0) {
    return { exitCode: checksumResult.code, failedStage: "checksum" };
  }

  return {
    exitCode: 0,
    dmgPath: selected.dmgPath,
    dmgReason: selected.reason,
    manifestPath: manifestResult.manifestPath,
    sha256sumsPath: manifestResult.sha256sumsPath,
  };
}
