/**
 * macOS Release Orchestrator (Slice 3B2C).
 * Composes: preflight → signed package → artifact verification.
 *
 * Run from apps/desktop/:  pnpm release:mac:full
 *
 * Exits non-zero at the first failed stage. Never reads, stores, or prints secret values.
 * Never passes --allow-adhoc to release:mac:verify. Never calls package:mac:unsigned.
 */
import { spawn } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { runReleaseMacFull } from "./release-mac-full.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");
const distDir = path.join(desktop, "dist");

/**
 * Snapshot all regular .dmg files in dist/.
 * Missing dist/ is treated as [] (clean checkout can still package).
 * Other filesystem errors propagate.
 */
async function snapshotDist() {
  let entries;
  try {
    entries = await fs.readdir(distDir);
  } catch (err) {
    if (err.code === "ENOENT") return [];
    throw new Error(`failed to read dist/: ${err.message}`);
  }

  const records = [];
  for (const name of entries) {
    if (!name.endsWith(".dmg")) continue;
    const full = path.join(distDir, name);
    let st;
    try {
      st = await fs.lstat(full);
    } catch {
      continue; // disappeared between readdir and stat
    }
    if (!st.isFile()) continue;
    records.push({ path: full, size: st.size, mtimeMs: st.mtimeMs, isFile: true });
  }
  return records;
}

/**
 * Spawn a pnpm sub-command, streaming output with a stage prefix.
 * Handles spawn errors (missing pnpm executable, permission denied).
 * Returns { code } — never throws on non-zero child exit.
 */
function runStage(_stage, scriptArgs) {
  const label = `[${scriptArgs[0]}]`;
  const errLabel = `[${scriptArgs[0]}][err]`;

  return new Promise((resolve) => {
    let child;
    try {
      child = spawn("pnpm", scriptArgs, {
        cwd: desktop,
        stdio: ["ignore", "pipe", "pipe"],
      });
    } catch (err) {
      process.stderr.write(
        `\n[release:mac:full] ERROR: failed to spawn 'pnpm ${scriptArgs.join(" ")}': ${err.message}\n`
      );
      resolve({ code: 1 });
      return;
    }

    let spawnError = null;
    child.on("error", (err) => {
      spawnError = err;
    });

    function prefixLines(stream, prefix, sink) {
      let buf = "";
      stream.on("data", (chunk) => {
        buf += chunk.toString();
        const lines = buf.split("\n");
        buf = lines.pop();
        for (const line of lines) {
          sink.write(`${prefix} ${line}\n`);
        }
      });
      stream.on("end", () => {
        if (buf) sink.write(`${prefix} ${buf}\n`);
      });
    }

    prefixLines(child.stdout, label, process.stdout);
    prefixLines(child.stderr, errLabel, process.stderr);

    child.on("close", (code) => {
      if (spawnError) {
        process.stderr.write(
          `\n[release:mac:full] ERROR: failed to spawn 'pnpm ${scriptArgs.join(" ")}': ${spawnError.message}\n`
        );
        resolve({ code: 1 });
        return;
      }
      resolve({ code: code ?? 1 });
    });
  });
}

const result = await runReleaseMacFull({
  runStage,
  snapshotDist,
  now: () => Date.now(),
});

if (result.failedStage === "preflight") {
  process.stderr.write(
    "\n[release:mac:full] STOPPED: preflight (release:mac:check) failed — packaging not started.\n"
  );
} else if (result.failedStage === "package") {
  process.stderr.write(
    "\n[release:mac:full] STOPPED: package:mac failed — artifact verification not started.\n"
  );
} else if (result.failedStage === "dmg-selection") {
  process.stderr.write(
    `\n[release:mac:full] STOPPED: DMG selection failed — ${result.dmgError}\n`
  );
} else if (result.failedStage === "verify") {
  process.stderr.write(
    "\n[release:mac:full] STOPPED: release:mac:verify failed — release not complete.\n"
  );
} else if (result.exitCode === 0) {
  process.stdout.write(
    `\n[release:mac:full] OK: all stages passed — ${result.dmgPath} verified (${result.dmgReason}).\n`
  );
}

process.exit(result.exitCode);
