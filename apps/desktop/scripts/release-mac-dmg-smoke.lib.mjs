/**
 * Pure helpers for the macOS DMG mount-and-launch smoke (Slice 3G).
 * No I/O, no child_process. All side-effecting code lives in release-mac-dmg-smoke.mjs.
 */

import path from "node:path";

const EXPECTED_BUNDLE = "Astraler Skillbox.app";

/**
 * Resolve the copied .app executable path inside the temp install directory.
 *
 * @param {string} installDir - absolute path to the temp install root
 * @param {string} appBundleName - the .app bundle name (e.g. "Astraler Skillbox.app")
 * @returns {{ appPath: string, execPath: string }}
 */
export function resolveCopiedApp(installDir, appBundleName) {
  const appPath = path.join(installDir, appBundleName);
  const execPath = path.join(appPath, "Contents", "MacOS", execName(appBundleName));
  return { appPath, execPath };
}

/**
 * Build hdiutil attach args for a read-only, no-browse, fixed mount point.
 *
 * @param {string} mountPoint - absolute path to the temp mount dir
 * @param {string} dmgPath - absolute path to the .dmg file
 * @returns {string[]}
 */
export function buildAttachArgs(mountPoint, dmgPath) {
  return ["attach", "-readonly", "-nobrowse", "-mountpoint", mountPoint, dmgPath];
}

/**
 * Build hdiutil detach args, optionally with -force.
 *
 * @param {string} mountPoint - absolute path to the mount point
 * @param {boolean} force - whether to add -force flag
 * @returns {string[]}
 */
export function buildDetachArgs(mountPoint, force) {
  if (force) return ["detach", "-force", mountPoint];
  return ["detach", mountPoint];
}

/**
 * Build ditto args for copying a .app bundle (preserves symlinks, xattrs, permissions).
 *
 * @param {string} srcAppPath - absolute path to the source .app on the mounted volume
 * @param {string} destAppPath - absolute path to the destination .app in the install dir
 * @returns {string[]}
 */
export function buildDittoArgs(srcAppPath, destAppPath) {
  return [srcAppPath, destAppPath];
}

/**
 * Derive the Mach-O executable name from the bundle name.
 * "Astraler Skillbox.app" → "Astraler Skillbox"
 *
 * @param {string} appBundleName
 * @returns {string}
 */
export function execName(appBundleName) {
  return appBundleName.replace(/\.app$/, "");
}

/**
 * Assert that the bundle name found in the DMG is exactly the expected one.
 * Returns the name on success; throws a descriptive error on mismatch.
 *
 * @param {string} appBundleName - the top-level .app name found in the DMG
 * @param {string} [expected] - the required bundle name (default: "Astraler Skillbox.app")
 * @returns {string}
 */
export function assertExpectedAppBundle(appBundleName, expected = EXPECTED_BUNDLE) {
  if (appBundleName === expected) return appBundleName;
  throw new Error(
    `DMG top-level app is "${appBundleName}" but expected "${expected}". ` +
      `The DMG must contain exactly one top-level bundle named "${expected}".`
  );
}

/**
 * Execute the two-step detach finalization (normal → force) using an injectable runFn.
 *
 * Returns:
 *   { detachFailed: false, mountPointPreserved: false }              — on success
 *   { detachFailed: true,  mountPointPreserved: true, message: str } — on full failure
 *
 * The caller must:
 *   - NOT rmSync the mount point when mountPointPreserved is true
 *   - exit non-zero (or throw) when detachFailed is true, AFTER printing the report
 *
 * @param {string} mountPoint
 * @param {(args: string[]) => { status: number }} runFn - mimics spawnSync result shape
 * @returns {{ detachFailed: boolean, mountPointPreserved: boolean, message?: string }}
 */
export function finalizeDetach(mountPoint, runFn) {
  let r = runFn(buildDetachArgs(mountPoint, false));
  if (r.status === 0) return { detachFailed: false, mountPointPreserved: false };

  r = runFn(buildDetachArgs(mountPoint, true));
  if (r.status === 0) return { detachFailed: false, mountPointPreserved: false };

  const message =
    `ERROR: failed to detach ${mountPoint} (hdiutil detach exit ${r.status}); ` +
    `the volume may still be mounted. ` +
    `Detach it manually:  hdiutil detach -force "${mountPoint}"`;
  return { detachFailed: true, mountPointPreserved: true, message };
}
