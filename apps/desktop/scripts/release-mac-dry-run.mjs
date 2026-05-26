/**
 * macOS Release Dry-Run (Slice 3E).
 * NON-DISTRIBUTABLE - AD-HOC SIGNED - NOT NOTARIZED
 *
 * Local end-to-end harness: build:core -> build -> ad-hoc electron-builder DMG ->
 * release:mac:verify --allow-adhoc -> release:mac:manifest -> shasum checksum.
 *
 * Never calls release:mac:check, signed package:mac, notarization, keychain,
 * network, upload, or credential reads. Does not set hardenedRuntime=false.
 *
 * Run from apps/desktop/:  pnpm release:mac:dry-run
 *
 * Exits non-zero at the first failed stage.
 */
import { spawn } from "node:child_process";
import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { runReleaseMacDryRun, scrubEnv } from "./release-mac-dry-run.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");
const distDir = path.join(desktop, "dist");

const DRY_RUN_BANNER =
  "!! NON-DISTRIBUTABLE  |  AD-HOC SIGNED  |  NOT NOTARIZED\n" +
  "   This artifact is for local chain validation only.\n" +
  "   Gatekeeper/customer distribution requires real signing + notarization.\n";

/**
 * Snapshot all regular .dmg files in dist/ using lstat (symlinks are excluded).
 * Missing dist/ is treated as [] (clean checkout can still package).
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
      continue; // disappeared between readdir and lstat
    }
    if (!st.isFile()) continue;
    records.push({ path: full, size: st.size, mtimeMs: st.mtimeMs, isFile: true });
  }
  return records;
}

/**
 * Spawn a sub-command, streaming output with a stage prefix.
 * For pnpm scripts (build:core, build, release:mac:verify, release:mac:manifest):
 *   spawn("pnpm", args)
 * For electron-builder:
 *   spawn("pnpm", ["exec", ...args])
 *
 * Returns { code } - never throws on non-zero child exit.
 * For the manifest stage, also resolves manifest/sums paths from the selected DMG arg.
 */
function runStage(stage, args) {
  const isElectronBuilder = args[0] === "electron-builder";
  const cmdArgs = isElectronBuilder ? ["exec", ...args] : args;
  const label = isElectronBuilder ? "[electron-builder]" : `[${args[0]}]`;
  const errLabel = isElectronBuilder ? "[electron-builder][err]" : `[${args[0]}][err]`;

  return new Promise((resolve) => {
    let child;
    try {
      child = spawn("pnpm", cmdArgs, {
        cwd: desktop,
        stdio: ["ignore", "pipe", "pipe"],
        env: scrubEnv(process.env),
      });
    } catch (err) {
      process.stderr.write(
        `\n[release:mac:dry-run] ERROR: failed to spawn '${cmdArgs.join(" ")}': ${err.message}\n`
      );
      resolve({ code: 1 });
      return;
    }

    let spawnError = null;
    child.on("error", (err) => {
      spawnError = err;
    });

    // Capture manifest/sums paths from the manifest stage
    let manifestPath;
    let sha256sumsPath;
    if (stage === "manifest" && args[1]) {
      const artifactBasename = path.basename(args[1]);
      manifestPath = path.join(distDir, `${artifactBasename}.manifest.json`);
      sha256sumsPath = path.join(distDir, "SHA256SUMS");
    }

    function prefixLines(stream, prefix, sink) {
      let buf = "";
      stream.on("data", (chunk) => {
        buf += chunk.toString();
        const lines = buf.split("\n");
        buf = lines.pop();
        for (const line of lines) sink.write(`${prefix} ${line}\n`);
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
          `\n[release:mac:dry-run] ERROR: failed to spawn '${cmdArgs.join(" ")}': ${spawnError.message}\n`
        );
        resolve({ code: 1 });
        return;
      }
      resolve({ code: code ?? 1, manifestPath, sha256sumsPath });
    });
  });
}

/**
 * Verify only the selected artifact's SHA256SUMS entry.
 *
 * Reads dist/SHA256SUMS, extracts the line matching the DMG basename, writes it to
 * a temporary file, runs `shasum -a 256 -c <tmpfile>` from dist/, then removes the
 * temp file. This preserves the shared SHA256SUMS and avoids failing on unrelated
 * stale lines from other artifacts.
 *
 * @param {string} dmgPath - absolute path to the selected DMG
 * @returns {Promise<{code: number}>}
 */
async function verifyChecksum(dmgPath) {
  const basename = path.basename(dmgPath);
  const sha256sumsPath = path.join(distDir, "SHA256SUMS");

  let content;
  try {
    content = await fs.readFile(sha256sumsPath, "utf8");
  } catch (err) {
    process.stderr.write(
      `\n[release:mac:dry-run] ERROR: SHA256SUMS not found at ${sha256sumsPath}: ${err.message}\n`
    );
    return { code: 1 };
  }

  // Find the line for this artifact (format: `<hash>  <basename>`)
  const lines = content.split("\n").filter((l) => l.trim());
  const matchingLine = lines.find((l) => {
    const parts = l.match(/^([0-9a-fA-F]{64})  (.+)$/);
    return parts && parts[2].trim() === basename;
  });

  if (!matchingLine) {
    process.stderr.write(
      `\n[release:mac:dry-run] ERROR: no SHA256SUMS entry found for ${basename}\n`
    );
    return { code: 1 };
  }

  // Write only the matching line to a temp file in dist/
  const tmpFile = path.join(os.tmpdir(), `.sha256check-${Date.now()}-${process.pid}.tmp`);
  try {
    await fs.writeFile(tmpFile, matchingLine + "\n");
  } catch (err) {
    process.stderr.write(
      `\n[release:mac:dry-run] ERROR: could not write temp checksum file: ${err.message}\n`
    );
    return { code: 1 };
  }

  try {
    return await new Promise((resolve) => {
      let child;
      try {
        child = spawn("shasum", ["-a", "256", "-c", tmpFile], {
          cwd: distDir,
          stdio: ["ignore", "pipe", "pipe"],
          env: scrubEnv(process.env),
        });
      } catch (err) {
        process.stderr.write(
          `\n[release:mac:dry-run] ERROR: failed to spawn shasum: ${err.message}\n`
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
          const parts = buf.split("\n");
          buf = parts.pop();
          for (const line of parts) sink.write(`${prefix} ${line}\n`);
        });
        stream.on("end", () => {
          if (buf) sink.write(`${prefix} ${buf}\n`);
        });
      }

      prefixLines(child.stdout, "[shasum]", process.stdout);
      prefixLines(child.stderr, "[shasum][err]", process.stderr);

      child.on("close", (code) => {
        if (spawnError) {
          process.stderr.write(
            `\n[release:mac:dry-run] ERROR: shasum spawn error: ${spawnError.message}\n`
          );
          resolve({ code: 1 });
          return;
        }
        resolve({ code: code ?? 1 });
      });
    });
  } finally {
    await fs.unlink(tmpFile).catch(() => {});
  }
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

process.stdout.write(
  "\n" +
    "=".repeat(72) + "\n" +
    "[release:mac:dry-run] STARTING dry-run\n" +
    DRY_RUN_BANNER +
    "=".repeat(72) + "\n\n"
);

const result = await runReleaseMacDryRun({
  runStage,
  snapshotDist,
  verifyChecksum,
  now: () => Date.now(),
});

if (result.failedStage === "generate-icon") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: generate:icon failed - icon not generated, build not started.\n"
  );
} else if (result.failedStage === "build:core") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: build:core failed - build not started.\n"
  );
} else if (result.failedStage === "build") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: build failed - electron-builder not started.\n"
  );
} else if (result.failedStage === "package-dmg") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: ad-hoc electron-builder failed - artifact not produced.\n"
  );
} else if (result.failedStage === "dmg-selection") {
  process.stderr.write(
    `\n[release:mac:dry-run] STOPPED: DMG selection failed - ${result.dmgError}\n`
  );
} else if (result.failedStage === "verify") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: release:mac:verify --allow-adhoc failed - dry-run incomplete.\n"
  );
} else if (result.failedStage === "manifest") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: release:mac:manifest failed - integrity artifacts not written.\n"
  );
} else if (result.failedStage === "checksum") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: checksum verification failed - artifact may be corrupted.\n"
  );
} else if (result.exitCode === 0) {
  process.stdout.write(
    "\n" +
      "=".repeat(72) + "\n" +
      "[release:mac:dry-run] OK: all stages passed\n" +
      DRY_RUN_BANNER +
      `  dmg      : ${result.dmgPath} (${result.dmgReason})\n` +
      (result.manifestPath ? `  manifest : ${result.manifestPath}\n` : "") +
      (result.sha256sumsPath ? `  sums     : ${result.sha256sumsPath}\n` : "") +
      "=".repeat(72) + "\n"
  );
}

process.exit(result.exitCode);
