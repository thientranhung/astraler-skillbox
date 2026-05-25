/**
 * Pure helpers for the macOS packaged app launch smoke (Slice 3F).
 * No I/O, no child_process. All side-effecting code lives in release-mac-launch-smoke.mjs.
 */

import path from "node:path";
import os from "node:os";

/**
 * Resolve the staged .app executable path from the dist/mac-arm64 directory.
 *
 * @param {string} desktopDir - absolute path to the apps/desktop directory
 * @returns {{ ok: true, execPath: string, appPath: string } | { ok: false, error: string }}
 */
export function resolveAppExecutable(desktopDir) {
  const appPath = path.join(desktopDir, "dist", "mac-arm64", "Astraler Skillbox.app");
  const execPath = path.join(appPath, "Contents", "MacOS", "Astraler Skillbox");
  return { ok: true, execPath, appPath };
}

/**
 * Returns true if the stderr line indicates Go core is ready.
 *
 * @param {string} line
 * @returns {boolean}
 */
export function isReadyLine(line) {
  return line.includes("[manager] Go core ready");
}

/**
 * Returns true if the stderr line indicates a known startup failure.
 *
 * @param {string} line
 * @returns {boolean}
 */
export function isFailureLine(line) {
  return (
    line.includes("Library not loaded") ||
    line.includes("not valid for use in process") ||
    line.includes("server.ready timeout") ||
    line.includes("[manager] FATAL")
  );
}

/**
 * Extract a short diagnostic from a failure line for human-readable output.
 *
 * @param {string} line
 * @returns {string}
 */
export function extractFailureDiagnostic(line) {
  if (line.includes("Library not loaded")) return "Library not loaded (hardened runtime / library validation issue)";
  if (line.includes("not valid for use in process")) return "Not valid for use in process (Team ID mismatch)";
  if (line.includes("server.ready timeout")) return "server.ready timeout (Go core did not start in time)";
  if (line.includes("[manager] FATAL")) return "Go core manager fatal error";
  return line.trim();
}

/**
 * Build a sanitized environment for launching the app in smoke mode.
 * Strips credential vars, sets SKILLBOX_DB_PATH to the temp db path.
 *
 * @param {Record<string, string|undefined>} env - process.env
 * @param {string} tmpDir - absolute path to the temp directory
 * @returns {Record<string, string|undefined>}
 */
export function buildLaunchEnv(env, tmpDir) {
  const CREDENTIAL_PREFIXES = ["CSC_", "APPLE_", "NOTARYTOOL_"];
  const cleaned = Object.fromEntries(
    Object.entries(env).filter(
      ([key]) => !CREDENTIAL_PREFIXES.some((prefix) => key.startsWith(prefix))
    )
  );
  cleaned["SKILLBOX_DB_PATH"] = path.join(tmpDir, "skillbox.db");
  return cleaned;
}

/**
 * Determines whether to declare a timeout given elapsed ms and a limit.
 *
 * @param {number} elapsedMs
 * @param {number} timeoutMs
 * @returns {boolean}
 */
export function isTimedOut(elapsedMs, timeoutMs) {
  return elapsedMs >= timeoutMs;
}

/**
 * Check whether a list of process entries contains an orphaned skillbox-core
 * whose executable path is inside the staged app.
 *
 * @param {Array<{pid: number, exe: string}>} procs - list of running process entries
 * @param {string} appPath - absolute path to the staged .app bundle
 * @returns {{ hasOrphan: boolean, orphans: Array<{pid: number, exe: string}> }}
 */
export function detectOrphanedSidecar(procs, appPath) {
  const normalizedApp = path.normalize(appPath);
  const orphans = procs.filter((p) => {
    const normalizedExe = path.normalize(p.exe);
    return (
      normalizedExe.startsWith(normalizedApp + path.sep) &&
      path.basename(normalizedExe) === "skillbox-core"
    );
  });
  return { hasOrphan: orphans.length > 0, orphans };
}
