/**
 * macOS Packaged App Launch Smoke (Slice 3F).
 *
 * Proves the staged .app at dist/mac-arm64/Astraler Skillbox.app can boot,
 * starts the bundled Go core (skillbox-core), and shuts down cleanly without
 * leaving an orphaned sidecar process.
 *
 * Does NOT package, sign, notarize, call Apple services, read the keychain,
 * or run release:mac:check / release:mac:full. Requires an already-packaged
 * .app from release:mac:dry-run or electron-builder.
 *
 * Run from apps/desktop/:  pnpm release:mac:launch-smoke
 *
 * Exits non-zero on startup failure, timeout, or orphaned sidecar.
 */
import { spawn, execFileSync } from "node:child_process";
import { promises as fs, existsSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  resolveAppExecutable,
  isReadyLine,
  isFailureLine,
  extractFailureDiagnostic,
  buildLaunchEnv,
  detectOrphanedSidecar,
} from "./release-mac-launch-smoke.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");

const READY_TIMEOUT_MS = 30_000;
const SHUTDOWN_WAIT_MS = 3_000;

// ---------------------------------------------------------------------------
// Process list helper - returns [{pid, exe}] for processes named skillbox-core
// ---------------------------------------------------------------------------

function listSidecarProcesses() {
  try {
    // ps -axo pid=,command= emits lines like "  1234 /full/path with spaces/skillbox-core".
    // The trailing = suppresses headers. We parse with a regex so the full command path
    // (which may contain spaces, e.g. "Astraler Skillbox.app") is preserved.
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

// ---------------------------------------------------------------------------
// Temp dir management
// ---------------------------------------------------------------------------

async function makeTmpDir() {
  return await fs.mkdtemp(path.join(os.tmpdir(), "skillbox-smoke-"));
}

async function removeTmpDir(tmpDir) {
  try {
    await fs.rm(tmpDir, { recursive: true, force: true });
  } catch {
    // best-effort
  }
}

// ---------------------------------------------------------------------------
// Shared shutdown + orphan check + cleanup
//
// Used by both the failure path and the success path so orphan detection
// always runs regardless of how the smoke exits.
//
// Returns { hadOrphan: boolean } — caller decides exit code.
// ---------------------------------------------------------------------------

async function shutdownAndCheckOrphans(appProcess, appPath, tmpDir, shutdownWaitMs) {
  // Terminate the app process.
  try {
    appProcess.kill("SIGTERM");
  } catch {
    // already exited — expected on failure paths
  }
  await new Promise((res) => setTimeout(res, shutdownWaitMs));
  try {
    appProcess.kill("SIGKILL");
  } catch {
    // already exited — expected
  }
  // Brief wait for OS to reap the process.
  await new Promise((res) => setTimeout(res, 500));

  // Check for any staged skillbox-core orphans.
  const procs = listSidecarProcesses();
  const { hasOrphan, orphans } = detectOrphanedSidecar(procs, appPath);

  if (hasOrphan) {
    process.stderr.write(
      `\n[release:mac:launch-smoke] FAILED: ${orphans.length} orphaned skillbox-core process(es) remain:\n`
    );
    for (const p of orphans) {
      process.stderr.write(`  pid=${p.pid}  exe=${p.exe}\n`);
      try {
        process.kill(p.pid, "SIGKILL");
      } catch {
        // best-effort
      }
    }
  }

  await removeTmpDir(tmpDir);
  return { hadOrphan: hasOrphan };
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

process.stdout.write(
  "\n" +
    "=".repeat(72) + "\n" +
    "[release:mac:launch-smoke] STARTING packaged app launch smoke\n" +
    "=".repeat(72) + "\n\n"
);

const resolved = resolveAppExecutable(desktop);

// Verify the staged app exists before attempting launch
if (!existsSync(resolved.appPath)) {
  process.stderr.write(
    `\n[release:mac:launch-smoke] ERROR: staged app not found at:\n  ${resolved.appPath}\n` +
      "  Run 'pnpm release:mac:dry-run' first to produce the staged .app.\n"
  );
  process.exit(1);
}

if (!existsSync(resolved.execPath)) {
  process.stderr.write(
    `\n[release:mac:launch-smoke] ERROR: app executable not found at:\n  ${resolved.execPath}\n`
  );
  process.exit(1);
}

let tmpDir;
try {
  tmpDir = await makeTmpDir();
} catch (err) {
  process.stderr.write(
    `\n[release:mac:launch-smoke] ERROR: could not create temp dir: ${err.message}\n`
  );
  process.exit(1);
}

process.stdout.write(`[release:mac:launch-smoke] temp dir: ${tmpDir}\n`);
process.stdout.write(`[release:mac:launch-smoke] launching: ${resolved.execPath}\n`);
process.stdout.write(
  `[release:mac:launch-smoke] SKILLBOX_DB_PATH: ${path.join(tmpDir, "skillbox.db")}\n\n`
);

const launchEnv = buildLaunchEnv(process.env, tmpDir);

let appProcess;
try {
  appProcess = spawn(resolved.execPath, [`--user-data-dir=${tmpDir}`], {
    stdio: ["ignore", "pipe", "pipe"],
    env: launchEnv,
    detached: false,
  });
} catch (err) {
  process.stderr.write(
    `\n[release:mac:launch-smoke] ERROR: failed to launch app: ${err.message}\n`
  );
  await removeTmpDir(tmpDir);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Stream output with prefixes; watch stderr for readiness / failure signals
// ---------------------------------------------------------------------------

let ready = false;
let failureDiagnostic = null;
let readyResolve;
const readyPromise = new Promise((res) => {
  readyResolve = res;
});

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

// Resolve if process exits before readiness
appProcess.on("close", (code) => {
  if (!ready && !failureDiagnostic) {
    readyResolve({ ok: false, diagnostic: `App exited prematurely with code ${code ?? "null"}` });
  }
});

appProcess.on("error", (err) => {
  if (!ready && !failureDiagnostic) {
    readyResolve({ ok: false, diagnostic: `App spawn error: ${err.message}` });
  }
});

// ---------------------------------------------------------------------------
// Wait for readiness with a bounded timeout
// ---------------------------------------------------------------------------

const timeoutHandle = setTimeout(() => {
  if (!ready && !failureDiagnostic) {
    readyResolve({ ok: false, diagnostic: "Timed out waiting for '[manager] Go core ready' signal" });
  }
}, READY_TIMEOUT_MS);

const readinessResult = await readyPromise;
clearTimeout(timeoutHandle);

if (!readinessResult.ok) {
  process.stderr.write(
    `\n[release:mac:launch-smoke] FAILED: ${readinessResult.diagnostic}\n`
  );

  // Check for display session issues (Electron requires a display)
  if (
    readinessResult.diagnostic &&
    (readinessResult.diagnostic.includes("exited prematurely") ||
      readinessResult.diagnostic.includes("Timed out"))
  ) {
    process.stderr.write(
      "[release:mac:launch-smoke] NOTE: Electron requires a display session. " +
        "If running headless, ensure a display is available (e.g., via a virtual display or GUI session).\n"
    );
  }

  // Always check for staged orphans before exiting, even on startup failure.
  await shutdownAndCheckOrphans(appProcess, resolved.appPath, tmpDir, SHUTDOWN_WAIT_MS);
  process.exit(1);
}

process.stdout.write(
  "\n[release:mac:launch-smoke] Go core ready — app launched successfully.\n\n"
);

// ---------------------------------------------------------------------------
// Quit the app, check for staged orphans, cleanup, and report
// ---------------------------------------------------------------------------

process.stdout.write("[release:mac:launch-smoke] sending SIGTERM to quit app…\n");

const { hadOrphan } = await shutdownAndCheckOrphans(
  appProcess,
  resolved.appPath,
  tmpDir,
  SHUTDOWN_WAIT_MS
);

if (hadOrphan) {
  process.exit(1);
}

process.stdout.write(
  "\n" +
    "=".repeat(72) + "\n" +
    "[release:mac:launch-smoke] OK: app launched, Go core ready, no orphaned sidecar\n" +
    `  app  : ${resolved.appPath}\n` +
    "=".repeat(72) + "\n"
);

process.exit(0);
