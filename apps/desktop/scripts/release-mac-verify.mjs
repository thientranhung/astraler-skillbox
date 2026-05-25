/**
 * macOS Release Artifact Verification Harness (Slice 3B2B).
 * Inspects a built .app/.dmg and reports whether it is a customer-ready notarized
 * release. Read-only except for a read-only DMG mount/detach. NEVER builds, signs,
 * notarizes, staples, calls the network, or mutates the keychain.
 *
 * Run from apps/desktop/:
 *   pnpm release:mac:verify [path]            # release mode (default)
 *   pnpm release:mac:verify --allow-adhoc [path]
 * Optional: SKILLBOX_EXPECTED_TEAM_ID=ABCDE12345 pins both app + sidecar to that team.
 */
import { spawnSync } from "node:child_process";
import { existsSync, statSync, readFileSync, readdirSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  parseCodesign,
  parseSpctl,
  parseStapler,
  parseEntitlementKeys,
  pickTopLevelApp,
  discoverDmg,
} from "./release-mac-verify.parse.mjs";
import { evaluate, render } from "./release-mac-verify.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, ".."); // apps/desktop
const SIDECAR_REL = "Contents/Resources/core/skillbox-core";

const argv = process.argv.slice(2);
const mode = argv.includes("--allow-adhoc") ? "adhoc" : "release";
const pathArg = argv.find((x) => !x.startsWith("--")) ?? null;
const expectedTeamId = (process.env.SKILLBOX_EXPECTED_TEAM_ID || "").trim() || null;

/** Spawn read-only; capture stdout+stderr+exit. Never throws on non-zero. */
function run(cmd, args) {
  const r = spawnSync(cmd, args, { encoding: "utf8" });
  return { text: `${r.stdout ?? ""}\n${r.stderr ?? ""}`, code: typeof r.status === "number" ? r.status : 1 };
}

let mountPoint = null; // set ONLY after a successful attach
let detachFailed = false;
function detach() {
  if (!mountPoint) return;
  let r = run("/usr/bin/hdiutil", ["detach", mountPoint]);
  if (r.code !== 0) r = run("/usr/bin/hdiutil", ["detach", "-force", mountPoint]);
  if (r.code !== 0) {
    // The volume may STILL be mounted. Do NOT rmSync the mountpoint (that could delete into a
    // live volume) and do NOT clear mountPoint. Record the failure so the process exits non-zero
    // — a mount must never be leaked silently (spec §3 / §10: internal error => non-zero).
    detachFailed = true;
    console.error(
      `ERROR: failed to detach ${mountPoint} (hdiutil detach exit ${r.code}); the volume may still be mounted. ` +
        `Detach it manually:  hdiutil detach -force "${mountPoint}"`
    );
    return;
  }
  // Detach succeeded: hdiutil removes the mountpoint dir; rmSync only cleans a stray empty dir.
  try {
    rmSync(mountPoint, { recursive: true, force: true });
  } catch {
    /* already gone */
  }
  mountPoint = null;
}

/** Print a single S1 FAIL and exit 1 (used before any artifact checks run). */
function failInput(message) {
  detach();
  console.log(render([{ id: "S1", category: "input", status: "FAIL", message }], [message]));
  process.exit(1);
}

/**
 * Read the EXPECTED entitlement keys from a committed plist. A missing or key-less plist is an
 * internal/config error (it would make the ENT subset check vacuously pass), so throw — never
 * return an empty expected set. The throw propagates past the `finally` (which still detaches),
 * yielding a clear message + non-zero exit.
 */
function expectedKeysFrom(rel) {
  const p = path.join(desktop, rel);
  if (!existsSync(p)) throw new Error(`expected entitlements file missing: ${rel} — cannot verify entitlements`);
  const keys = parseEntitlementKeys(readFileSync(p, "utf8"));
  if (keys.length === 0) throw new Error(`expected entitlements file has no <key> entries: ${rel}`);
  return keys;
}

function gatherSigning(target, deep) {
  const dvvv = run("/usr/bin/codesign", ["-dvvv", target]);
  const verifyArgs = deep
    ? ["--verify", "--deep", "--strict", "--verbose=2", target]
    : ["--verify", "--strict", "--verbose=2", target];
  const verify = run("/usr/bin/codesign", verifyArgs);
  const ent = run("/usr/bin/codesign", ["-d", "--entitlements", ":-", target]);
  return { parsed: parseCodesign(dvvv.text), verifyExit: verify.code, entitlementKeys: parseEntitlementKeys(ent.text) };
}

// --- Resolve input ---
let appPath = null;
let appName = null;
let dmgPath = null;
let dmgName = null;

if (pathArg) {
  const abs = path.resolve(process.cwd(), pathArg);
  if (!existsSync(abs)) failInput(`input path does not exist: ${pathArg}`);
  if (abs.endsWith(".app")) {
    if (!statSync(abs).isDirectory()) failInput(`.app input must be a directory (app bundle): ${pathArg}`);
    appPath = abs;
    appName = path.basename(abs);
  } else if (abs.endsWith(".dmg")) {
    if (!statSync(abs).isFile()) failInput(`.dmg input must be a regular file: ${pathArg}`);
    dmgPath = abs;
    dmgName = path.basename(abs);
  } else {
    failInput(`input is neither a .app nor a .dmg: ${pathArg}`);
  }
} else {
  const dist = path.join(desktop, "dist");
  const d = discoverDmg(existsSync(dist) ? readdirSync(dist) : []);
  if (d.error) failInput(d.error);
  dmgPath = path.join(dist, d.dmg);
  dmgName = d.dmg;
}

try {
  if (dmgPath) {
    const mp = mkdtempSync(path.join(tmpdir(), "skillbox-verify-"));
    const att = run("/usr/bin/hdiutil", ["attach", "-readonly", "-nobrowse", "-mountpoint", mp, dmgPath]);
    if (att.code !== 0) {
      // Nothing mounted — remove the empty temp dir and fail. Do NOT set mountPoint, so detach()
      // never runs a misleading detach against an unmounted path.
      try {
        rmSync(mp, { recursive: true, force: true });
      } catch {
        /* ignore */
      }
      failInput(`failed to mount DMG read-only (hdiutil attach exit ${att.code})`);
    }
    mountPoint = mp; // only now is a real mount present
    const pick = pickTopLevelApp(readdirSync(mountPoint)); // non-recursive: ignores nested helper apps
    if (pick.error) failInput(pick.error);
    appPath = path.join(mountPoint, pick.app);
    appName = pick.app;
  }

  const app = gatherSigning(appPath, true);
  const sidePath = path.join(appPath, SIDECAR_REL);
  const sidecar = existsSync(sidePath)
    ? { present: true, ...gatherSigning(sidePath, false) }
    : { present: false, parsed: null, verifyExit: null, entitlementKeys: [] };

  const spA = run("/usr/sbin/spctl", ["-a", "-vvv", "-t", "exec", appPath]);
  const stA = run("/usr/bin/xcrun", ["stapler", "validate", appPath]);
  let spctlDmg = null;
  let staplerDmg = null;
  if (dmgPath) {
    const spD = run("/usr/sbin/spctl", ["-a", "-vvv", "-t", "open", dmgPath]);
    const stD = run("/usr/bin/xcrun", ["stapler", "validate", dmgPath]);
    spctlDmg = parseSpctl(spD.text, spD.code);
    staplerDmg = parseStapler(stD.text, stD.code);
  }

  const { results, missing, exitCode } = evaluate({
    mode,
    expectedTeamId,
    expectedEntitlements: {
      app: expectedKeysFrom("build/entitlements.mac.plist"),
      sidecar: expectedKeysFrom("build/entitlements.mac.inherit.plist"),
    },
    input: { dmgName, appName },
    app,
    sidecar,
    spctlApp: parseSpctl(spA.text, spA.code),
    spctlDmg,
    staplerApp: parseStapler(stA.text, stA.code),
    staplerDmg,
  });

  console.log(render(results, missing));
  process.exitCode = exitCode;
} finally {
  detach();
}

// A failed detach must never resolve as success. Surface it AFTER the verify report has printed,
// as a non-zero internal error (spec §3 / §10) — never leak a mounted volume silently.
if (detachFailed) {
  throw new Error("DMG detach failed; the read-only mount was not cleaned up (see ERROR above).");
}
