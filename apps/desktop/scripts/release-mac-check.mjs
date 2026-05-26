/**
 * macOS Release Preflight / Credential Doctor (Slice 3B2A).
 * Read-only, offline. Gathers facts and delegates all verdicts to the pure
 * evaluator. NEVER signs, notarizes, builds, mutates the keychain, calls the
 * network, or prints any credential value or file path.
 *
 * Run from apps/desktop/:  pnpm release:mac:check
 */
import { execFileSync } from "node:child_process";
import { existsSync, accessSync, statSync, readFileSync, constants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import yaml from "js-yaml";
import { evaluate, render } from "./release-mac-check.lib.mjs";
import { assertSvgUsable } from "./generate-icon.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, ".."); // apps/desktop
const repoRoot = path.resolve(desktop, "..", "..");

/** Run a command read-only; return trimmed stdout or null on any failure. */
function run(cmd, args, opts = {}) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"], cwd: opts.cwd }).trim();
  } catch {
    return null;
  }
}

function commandExists(name) {
  return run("/usr/bin/which", [name]) !== null;
}

function xcrunFinds(tool) {
  return run("/usr/bin/xcrun", ["-f", tool]) !== null;
}

function readableFile(p) {
  try {
    accessSync(p, constants.R_OK);
    return statSync(p).isFile();
  } catch {
    return false;
  }
}

function plutilLintOk(p) {
  if (!existsSync(p)) return false;
  return run("/usr/bin/plutil", ["-lint", p]) !== null; // non-zero exit -> null -> false
}

// --- Gather facts (all read-only) ---
const env = {
  APPLE_API_KEY: process.env.APPLE_API_KEY,
  APPLE_API_KEY_ID: process.env.APPLE_API_KEY_ID,
  APPLE_API_ISSUER: process.env.APPLE_API_ISSUER,
  APPLE_ID: process.env.APPLE_ID,
  APPLE_APP_SPECIFIC_PASSWORD: process.env.APPLE_APP_SPECIFIC_PASSWORD,
  APPLE_TEAM_ID: process.env.APPLE_TEAM_ID,
  APPLE_KEYCHAIN_PROFILE: process.env.APPLE_KEYCHAIN_PROFILE,
  CSC_LINK: process.env.CSC_LINK,
  CSC_KEY_PASSWORD: process.env.CSC_KEY_PASSWORD,
};

const tools = {
  notarytool: xcrunFinds("notarytool"),
  stapler: xcrunFinds("stapler"),
  codesign: commandExists("codesign"),
  spctl: commandExists("spctl"),
  plutil: commandExists("plutil"),
};

// Identity NAMES only (non-secret); read-only keychain query.
const identityOut = run("/usr/bin/security", ["find-identity", "-v", "-p", "codesigning"]) ?? "";
const identityNames = [...identityOut.matchAll(/"(Developer ID Application:[^"]*)"/g)].map((m) => m[1]);

// File probes — derived flags only; the path itself is never put into facts beyond env.
function cscLinkProbe(v) {
  if (typeof v !== "string" || v.trim() === "") return null;
  // Non-local: explicit URL, or pure base64 blob (only base64 alphabet chars — no dots or other path chars).
  // Everything else — absolute (/...), tilde (~), dot-relative (./), AND bare relative (cert.p12) — is a local path.
  const isLocalPath = !/^https?:\/\//i.test(v) && !/^[A-Za-z0-9+/]+=*$/.test(v);
  if (!isLocalPath) return { isLocalPath: false, exists: false, readable: false };
  return { isLocalPath: true, exists: existsSync(v), readable: readableFile(v) };
}
function apiKeyProbe(v) {
  if (typeof v !== "string" || v.trim() === "") return null;
  return { exists: existsSync(v), readable: readableFile(v) };
}
const fileProbes = { cscLink: cscLinkProbe(env.CSC_LINK), appleApiKey: apiKeyProbe(env.APPLE_API_KEY) };

// electron-builder config
const ebPath = path.join(desktop, "electron-builder.yml");
const config = yaml.load(readFileSync(ebPath, "utf8"));

// Entitlements
const mainPlist = path.join(desktop, "build", "entitlements.mac.plist");
const inheritPlist = path.join(desktop, "build", "entitlements.mac.inherit.plist");
const entitlements = {
  mainExists: existsSync(mainPlist),
  mainLintOk: plutilLintOk(mainPlist),
  inheritExists: existsSync(inheritPlist),
  inheritLintOk: plutilLintOk(inheritPlist),
};

// Staged sidecar
const sidecarPath = path.join(desktop, "resources", "core", "skillbox-core");
let sidecar = { present: false, arch: null, executable: false };
if (existsSync(sidecarPath)) {
  const fileOut = run("/usr/bin/file", [sidecarPath]) ?? "";
  const arch = /arm64/.test(fileOut) ? "arm64" : /x86_64/.test(fileOut) ? "x86_64" : null;
  let executable = false;
  try {
    executable = (statSync(sidecarPath).mode & 0o111) !== 0;
  } catch {
    executable = false;
  }
  sidecar = { present: true, arch, executable };
}

// Icon assets (Slice 3I) — source committed, .icns is a generated artifact.
const iconSvgPath = path.join(desktop, "build", "icon.svg");
const iconIcnsPath = path.join(desktop, "build", "icon.icns");
const svgPresent = existsSync(iconSvgPath);
let svgUsable = false;
if (svgPresent) {
  try {
    assertSvgUsable(readFileSync(iconSvgPath, "utf8"));
    svgUsable = true;
  } catch {
    svgUsable = false;
  }
}
const icnsPresent = existsSync(iconIcnsPath);
let icnsValid = false;
if (icnsPresent) {
  const typeOk = (run("/usr/bin/file", [iconIcnsPath]) ?? "").includes("Mac OS X icon");
  let nonEmpty = false;
  try {
    nonEmpty = statSync(iconIcnsPath).size > 0;
  } catch {
    nonEmpty = false;
  }
  icnsValid = typeOk && nonEmpty && readableFile(iconIcnsPath);
}
const icon = { svgPresent, svgUsable, icnsPresent, icnsValid };

// Hygiene — tracked-status entries only (ignore untracked ??), per spec F1.
// CRITICAL: this command is invoked from apps/desktop (pnpm cwd). Git pathspecs are
// resolved relative to the *current directory*, so a naive `git ls-files -- apps/desktop`
// from apps/desktop would look for apps/desktop/apps/desktop and silently find NOTHING,
// missing tracked .p12/.p8 and dist artifacts. Run every git command from repoRoot with
// repo-root-relative pathspecs so detection is correct regardless of invocation cwd.
const statusOut = run("git", ["status", "--porcelain", "--untracked-files=no", "--", "apps/desktop/dist", "apps/desktop/resources/core"], { cwd: repoRoot }) ?? "";
const trackedArtifacts = statusOut
  .split("\n")
  .map((l) => l.trim())
  .filter(Boolean)
  .map((l) => l.replace(/^\S+\s+/, "")); // drop the status code, keep the path
const lsFiles = run("git", ["ls-files", "--", "apps/desktop"], { cwd: repoRoot }) ?? "";
const trackedSecretFiles = lsFiles.split("\n").filter((f) => /\.(p12|p8)$/.test(f));

// Version
const pkg = JSON.parse(readFileSync(path.join(desktop, "package.json"), "utf8"));

const facts = {
  platform: process.platform,
  tools,
  identityNames,
  env,
  fileProbes,
  config,
  entitlements,
  sidecar,
  icon,
  trackedArtifacts,
  trackedSecretFiles,
  version: pkg.version,
};

const { results, missing, exitCode } = evaluate(facts);
console.log(render(results, missing));
process.exit(exitCode);
