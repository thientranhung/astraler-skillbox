/**
 * macOS DMG Mount-and-Launch Smoke (Slice 3G).
 *
 * Proves the actual distributable artifact boots: mounts the produced DMG
 * read-only, finds the top-level "Astraler Skillbox.app", copies it to a
 * temp install dir (never /Applications) via ditto, launches the copied app
 * with a temp --user-data-dir and SKILLBOX_DB_PATH, waits for the bundled
 * Go core to be ready, shuts down, asserts no orphaned skillbox-core from
 * the copied bundle, detaches the DMG, and cleans temp dirs.
 *
 * Does NOT package, sign, notarize, call Apple services, read the keychain,
 * or run release:mac:check / release:mac:full.
 *
 * Run from apps/desktop/:  pnpm release:mac:dmg-smoke [path/to/artifact.dmg]
 *
 * Exits non-zero on any failure.
 */
import { spawn, spawnSync, execFileSync } from "node:child_process";
import { existsSync, readdirSync, mkdtempSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  resolveCopiedApp,
  buildAttachArgs,
  buildDittoArgs,
  assertExpectedAppBundle,
  finalizeDetach,
} from "./release-mac-dmg-smoke.lib.mjs";
import {
  isReadyLine,
  isFailureLine,
  extractFailureDiagnostic,
  buildLaunchEnv,
  detectOrphanedSidecar,
} from "./release-mac-launch-smoke.lib.mjs";
import { discoverDmg, pickTopLevelApp } from "./release-mac-verify.parse.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");
const dist = path.join(desktop, "dist");

const READY_TIMEOUT_MS = 30_000;
const SHUTDOWN_WAIT_MS = 3_000;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function hdiutilRun(args) {
  const r = spawnSync("/usr/bin/hdiutil", args, { encoding: "utf8" });
  return { status: typeof r.status === "number" ? r.status : 1 };
}

function listSidecarProcesses() {
  try {
    const out = execFileSync("ps", ["-axo", "pid=,command="], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    });
    return out
      .split("\n")
      .filter((line) => line.includes("skillbox-core"))
      .map((line) => {
        const m = line.match(/^\s*(\d+)\s+(.*)\s*$/);
        if (!m) return null;
        return { pid: parseInt(m[1], 10), exe: m[2].trim() };
      })
      .filter((p) => p !== null && !isNaN(p.pid));
  } catch {
    return [];
  }
}

function removeTmpDirSync(dir) {
  if (!dir) return;
  try {
    rmSync(dir, { recursive: true, force: true });
  } catch {
    // best-effort
  }
}

function prefixLines(stream, prefix, sink, onLine) {
  let buf = "";
  stream.on("data", (chunk) => {
    buf += chunk.toString();
    const lines = buf.split("\n");
    buf = lines.pop();
    for (const line of lines) {
      sink.write(`${prefix} ${line}\n`);
      if (onLine) onLine(line);
    }
  });
  stream.on("end", () => {
    if (buf) {
      sink.write(`${prefix} ${buf}\n`);
      if (onLine) onLine(buf);
    }
  });
}

// ---------------------------------------------------------------------------
// Shutdown + orphan check helper (runs on every exit path)
// ---------------------------------------------------------------------------

async function shutdownAndCheckOrphans(appProcess, copiedAppPath, shutdownWaitMs) {
  try { appProcess.kill("SIGTERM"); } catch { /* already exited */ }
  await new Promise((res) => setTimeout(res, shutdownWaitMs));
  try { appProcess.kill("SIGKILL"); } catch { /* already exited */ }
  await new Promise((res) => setTimeout(res, 500));

  const procs = listSidecarProcesses();
  const { hasOrphan, orphans } = detectOrphanedSidecar(procs, copiedAppPath);

  if (hasOrphan) {
    process.stderr.write(
      `\n[release:mac:dmg-smoke] FAILED: ${orphans.length} orphaned skillbox-core process(es) remain:\n`
    );
    for (const p of orphans) {
      process.stderr.write(`  pid=${p.pid}  exe=${p.exe}\n`);
      try { process.kill(p.pid, "SIGKILL"); } catch { /* best-effort */ }
    }
  } else {
    process.stdout.write("[release:mac:dmg-smoke] no orphaned sidecar — clean shutdown.\n");
  }

  return { hadOrphan: hasOrphan };
}

// ---------------------------------------------------------------------------
// Resolve DMG path
// ---------------------------------------------------------------------------

process.stdout.write(
  "\n" +
    "=".repeat(72) + "\n" +
    "[release:mac:dmg-smoke] STARTING dmg mount-and-launch smoke\n" +
    "=".repeat(72) + "\n\n"
);

const argv = process.argv.slice(2);
const pathArg = argv.find((x) => !x.startsWith("--")) ?? null;

let dmgPath;
if (pathArg) {
  dmgPath = path.isAbsolute(pathArg) ? pathArg : path.resolve(process.cwd(), pathArg);
  if (!existsSync(dmgPath)) {
    process.stderr.write(
      `\n[release:mac:dmg-smoke] ERROR: DMG not found: ${dmgPath}\n`
    );
    process.exit(1);
  }
} else {
  const entries = existsSync(dist) ? readdirSync(dist) : [];
  const discovery = discoverDmg(entries);
  if (discovery.error) {
    process.stderr.write(`\n[release:mac:dmg-smoke] ERROR: ${discovery.error}\n`);
    process.exit(1);
  }
  dmgPath = path.join(dist, discovery.dmg);
}

process.stdout.write(`[release:mac:dmg-smoke] DMG: ${dmgPath}\n`);

// ---------------------------------------------------------------------------
// Create temp dirs: mountPoint, installRoot, userDataDir
// ---------------------------------------------------------------------------

let mountPoint = null; // only set after successful attach
let installRoot = null;
let userDataDir = null;

try {
  mountPoint = mkdtempSync(path.join(os.tmpdir(), "skillbox-dmgsmoke-mnt-"));
  installRoot = mkdtempSync(path.join(os.tmpdir(), "skillbox-dmgsmoke-app-"));
  userDataDir = mkdtempSync(path.join(os.tmpdir(), "skillbox-dmgsmoke-ud-"));
} catch (err) {
  process.stderr.write(
    `\n[release:mac:dmg-smoke] ERROR: could not create temp dirs: ${err.message}\n`
  );
  // Clean up any that were created
  removeTmpDirSync(mountPoint);
  removeTmpDirSync(installRoot);
  removeTmpDirSync(userDataDir);
  process.exit(1);
}

process.stdout.write(`[release:mac:dmg-smoke] mount point  : ${mountPoint}\n`);
process.stdout.write(`[release:mac:dmg-smoke] install root : ${installRoot}\n`);
process.stdout.write(`[release:mac:dmg-smoke] user-data-dir: ${userDataDir}\n`);
process.stdout.write(
  `[release:mac:dmg-smoke] SKILLBOX_DB_PATH: ${path.join(userDataDir, "skillbox.db")}\n\n`
);

// ---------------------------------------------------------------------------
// Mount DMG read-only
// ---------------------------------------------------------------------------

{
  const r = spawnSync("/usr/bin/hdiutil", buildAttachArgs(mountPoint, dmgPath), {
    encoding: "utf8",
  });
  if ((typeof r.status === "number" ? r.status : 1) !== 0) {
    process.stderr.write(
      `\n[release:mac:dmg-smoke] ERROR: hdiutil attach failed (exit ${r.status})\n` +
        (r.stderr ? r.stderr : "")
    );
    // Mount failed — no real mount present; remove the empty mount temp dir.
    removeTmpDirSync(mountPoint);
    mountPoint = null;
    removeTmpDirSync(installRoot);
    removeTmpDirSync(userDataDir);
    process.exit(1);
  }
}

process.stdout.write(`[release:mac:dmg-smoke] DMG mounted at: ${mountPoint}\n`);

// ---------------------------------------------------------------------------
// From here on, every exit path must detach + cleanup.
// We use a top-level async IIFE so await is available.
// ---------------------------------------------------------------------------

async function main() {
  let appProcess = null;
  let copiedAppPath = null;
  let exitCode = 0;

  try {
    // ---- Select top-level app ----
    const entries = readdirSync(mountPoint);
    const pick = pickTopLevelApp(entries);
    if (pick.error) {
      process.stderr.write(`\n[release:mac:dmg-smoke] ERROR: ${pick.error}\n`);
      exitCode = 1;
      return;
    }

    let appBundleName;
    try {
      appBundleName = assertExpectedAppBundle(pick.app);
    } catch (err) {
      process.stderr.write(`\n[release:mac:dmg-smoke] ERROR: ${err.message}\n`);
      exitCode = 1;
      return;
    }

    const srcAppPath = path.join(mountPoint, appBundleName);
    const { appPath: destAppPath, execPath } = resolveCopiedApp(installRoot, appBundleName);
    copiedAppPath = destAppPath;

    process.stdout.write(`[release:mac:dmg-smoke] app bundle   : ${appBundleName}\n`);
    process.stdout.write(`[release:mac:dmg-smoke] copied to    : ${destAppPath}\n`);

    // ---- Copy via ditto ----
    {
      const r = spawnSync("/usr/bin/ditto", buildDittoArgs(srcAppPath, destAppPath), {
        encoding: "utf8",
      });
      if ((typeof r.status === "number" ? r.status : 1) !== 0) {
        process.stderr.write(
          `\n[release:mac:dmg-smoke] ERROR: ditto copy failed (exit ${r.status})\n` +
            (r.stderr ? r.stderr : "")
        );
        exitCode = 1;
        return;
      }
    }

    process.stdout.write(`[release:mac:dmg-smoke] ditto copy   : OK\n`);

    // ---- Verify copied executable exists ----
    if (!existsSync(execPath)) {
      process.stderr.write(
        `\n[release:mac:dmg-smoke] ERROR: copied app executable not found: ${execPath}\n`
      );
      exitCode = 1;
      return;
    }

    process.stdout.write(`[release:mac:dmg-smoke] exec path    : ${execPath}\n\n`);

    // ---- Launch from a neutral cwd (temp root, not repo, not mount) ----
    const launchEnv = buildLaunchEnv(process.env, userDataDir);

    try {
      appProcess = spawn(execPath, [`--user-data-dir=${userDataDir}`], {
        cwd: os.tmpdir(),
        env: launchEnv,
        stdio: ["ignore", "pipe", "pipe"],
        detached: false,
      });
    } catch (err) {
      process.stderr.write(
        `\n[release:mac:dmg-smoke] ERROR: failed to launch app: ${err.message}\n`
      );
      exitCode = 1;
      return;
    }

    // ---- Stream output; watch for readiness ----
    let ready = false;
    let failureDiagnostic = null;
    let readyResolve;
    const readyPromise = new Promise((res) => { readyResolve = res; });

    prefixLines(appProcess.stdout, "[app]", process.stdout, null);
    prefixLines(appProcess.stderr, "[app][err]", process.stderr, (line) => {
      if (!ready && isReadyLine(line)) {
        ready = true;
        readyResolve({ ok: true });
      }
      if (!ready && isFailureLine(line)) {
        failureDiagnostic = extractFailureDiagnostic(line);
        readyResolve({ ok: false, diagnostic: failureDiagnostic });
      }
    });

    appProcess.on("close", (code) => {
      if (!ready && !failureDiagnostic) {
        readyResolve({
          ok: false,
          diagnostic: `App exited prematurely with code ${code ?? "null"}`,
        });
      }
    });

    appProcess.on("error", (err) => {
      if (!ready && !failureDiagnostic) {
        readyResolve({ ok: false, diagnostic: `App spawn error: ${err.message}` });
      }
    });

    const timeoutHandle = setTimeout(() => {
      if (!ready && !failureDiagnostic) {
        readyResolve({
          ok: false,
          diagnostic: "Timed out waiting for '[manager] Go core ready' signal",
        });
      }
    }, READY_TIMEOUT_MS);

    const readinessResult = await readyPromise;
    clearTimeout(timeoutHandle);

    if (!readinessResult.ok) {
      process.stderr.write(
        `\n[release:mac:dmg-smoke] FAILED: ${readinessResult.diagnostic}\n`
      );
      if (
        readinessResult.diagnostic &&
        (readinessResult.diagnostic.includes("exited prematurely") ||
          readinessResult.diagnostic.includes("Timed out"))
      ) {
        process.stderr.write(
          "[release:mac:dmg-smoke] NOTE: Electron requires a display session. " +
            "If running headless, ensure a display is available (e.g., via a virtual display or GUI session).\n"
        );
      }
      exitCode = 1;
      return;
    }

    process.stdout.write(
      "\n[release:mac:dmg-smoke] Go core ready — app launched successfully.\n\n"
    );

    // ---- Shutdown + orphan check ----
    process.stdout.write("[release:mac:dmg-smoke] sending SIGTERM to quit app…\n");
    const { hadOrphan } = await shutdownAndCheckOrphans(
      appProcess,
      copiedAppPath,
      SHUTDOWN_WAIT_MS
    );
    appProcess = null; // already handled

    if (hadOrphan) {
      exitCode = 1;
      return;
    }
  } finally {
    // ---- Shutdown if still running (failure paths that returned early) ----
    if (appProcess) {
      await shutdownAndCheckOrphans(appProcess, copiedAppPath, SHUTDOWN_WAIT_MS);
    }

    // ---- Detach DMG ----
    if (mountPoint) {
      const detachResult = finalizeDetach(mountPoint, hdiutilRun);
      if (detachResult.detachFailed) {
        process.stderr.write(`\n${detachResult.message}\n`);
        // Do NOT remove the mount point — it may be live
      } else {
        // hdiutil removes the dir on success; clean up stray empty dir just in case
        removeTmpDirSync(mountPoint);
        process.stdout.write(`[release:mac:dmg-smoke] DMG detached successfully.\n`);
      }

      // Cleanup install root and user-data dir (best-effort)
      removeTmpDirSync(installRoot);
      removeTmpDirSync(userDataDir);

      // After cleanup report: if detach failed, exit non-zero
      if (detachResult.detachFailed) {
        throw new Error("DMG detach failed; the read-only mount was not cleaned up (see ERROR above).");
      }
    } else {
      removeTmpDirSync(installRoot);
      removeTmpDirSync(userDataDir);
    }
  }

  if (exitCode !== 0) {
    process.exit(exitCode);
  }

  process.stdout.write(
    "\n" +
      "=".repeat(72) + "\n" +
      "[release:mac:dmg-smoke] OK: app launched from mounted DMG, Go core ready, no orphaned sidecar, clean detach\n" +
      `  dmg  : ${dmgPath}\n` +
      "=".repeat(72) + "\n"
  );

  process.exit(0);
}

main().catch((err) => {
  process.stderr.write(`\n[release:mac:dmg-smoke] FATAL: ${err.message}\n`);
  process.exit(1);
});
