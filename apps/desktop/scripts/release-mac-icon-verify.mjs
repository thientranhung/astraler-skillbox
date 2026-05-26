/**
 * Packaged-app icon verification (Slice 3I).
 * Offline, read-only. No Apple credentials, no network, no keychain.
 * Asserts the .app ships a customized, valid .icns (not the default Electron icon).
 *
 * Usage (from apps/desktop/):
 *   pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"
 *   pnpm release:mac:icon-verify          # defaults to dist/mac-arm64/Astraler Skillbox.app
 */
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  resolveInfoPlist,
  resolveIconResource,
  assertIconFacts,
} from "./release-mac-icon-verify.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");

function run(cmd, args) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return null;
  }
}

const appArg = process.argv[2] || path.join("dist", "mac-arm64", "Astraler Skillbox.app");
const appPath = path.isAbsolute(appArg) ? appArg : path.join(desktop, appArg);

if (!existsSync(appPath)) {
  console.error(`ERROR: app bundle not found: ${appArg}`);
  console.error("Build it first (e.g. pnpm package:mac:unsigned), then pass the .app path.");
  process.exit(1);
}

const infoPlist = resolveInfoPlist(appPath);
if (!existsSync(infoPlist)) {
  console.error(`ERROR: Info.plist not found under ${appArg}/Contents`);
  process.exit(1);
}

const iconFile = run("/usr/bin/plutil", ["-extract", "CFBundleIconFile", "raw", infoPlist]) ?? "";
const resourcePath = iconFile ? resolveIconResource(appPath, iconFile) : null;
const resourceExists = resourcePath ? existsSync(resourcePath) : false;
const sha256 = resourceExists ? createHash("sha256").update(readFileSync(resourcePath)).digest("hex") : null;
const fileType = resourceExists ? run("/usr/bin/file", [resourcePath]) : null;

try {
  assertIconFacts({ iconFile, resourceExists, sha256, fileType });
} catch (err) {
  console.error(err.message);
  process.exit(1);
}

console.log("Icon verification PASS");
console.log(`  CFBundleIconFile: ${iconFile}`);
console.log(`  resource:         Contents/Resources/${iconFile} (present)`);
console.log(`  sha256:           ${sha256}`);
console.log("  bytes differ from the default Electron icon; resource is a valid .icns");
process.exit(0);
