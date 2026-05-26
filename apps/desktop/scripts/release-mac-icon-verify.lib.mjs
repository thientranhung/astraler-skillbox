/**
 * Pure helpers for the packaged-app icon verification (Slice 3I).
 * No I/O, no child_process. All side-effecting code lives in release-mac-icon-verify.mjs.
 */
import path from "node:path";

/** The default Electron app icon — the thing we must NOT ship. */
export const DEFAULT_ICON_FILE = "electron.icns";
/** SHA-256 of the default Electron icon bytes (anchor captured 2026-05-26). */
export const DEFAULT_ICON_SHA256 =
  "5a9a78d54c157f55672afea37037464858a87fd5f276fc8206787f366ed684cf";

/** @param {string} appPath @param {string} iconFile */
export function resolveIconResource(appPath, iconFile) {
  return path.join(appPath, "Contents", "Resources", iconFile);
}

/** @param {string} appPath */
export function resolveInfoPlist(appPath) {
  return path.join(appPath, "Contents", "Info.plist");
}

/**
 * Throw with all collected problems if the packaged icon is missing/default/invalid.
 * @param {{iconFile:string, resourceExists:boolean, sha256:string|null, fileType:string|null}} facts
 */
export function assertIconFacts({ iconFile, resourceExists, sha256, fileType }) {
  const problems = [];

  if (!iconFile || iconFile.trim() === "") {
    problems.push("CFBundleIconFile is not set in Info.plist");
  } else if (iconFile === DEFAULT_ICON_FILE) {
    problems.push(`CFBundleIconFile is the default ${DEFAULT_ICON_FILE} (app icon was not customized)`);
  }

  if (!resourceExists) {
    problems.push(`icon resource Contents/Resources/${iconFile || "<unset>"} is missing`);
  }

  if (sha256 && sha256 === DEFAULT_ICON_SHA256) {
    problems.push("icon bytes are identical to the default Electron icon");
  }

  if (resourceExists) {
    if (!fileType || fileType.trim() === "") {
      problems.push("could not determine icon file type (file command returned no output)");
    } else if (!/Mac OS X icon/.test(fileType)) {
      problems.push(`icon resource is not a valid .icns (file type: ${fileType})`);
    }
  }

  if (problems.length > 0) {
    const err = new Error("Icon verification failed:\n  - " + problems.join("\n  - "));
    err.problems = problems;
    throw err;
  }
  return { ok: true, iconFile };
}
