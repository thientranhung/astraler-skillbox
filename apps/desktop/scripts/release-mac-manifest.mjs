/**
 * macOS Release Manifest + Checksum Generator (Slice 3C).
 * Usage: pnpm release:mac:manifest <path-to-dmg>
 *
 * Reads the exact DMG path, computes SHA-256, writes:
 *   dist/<artifact>.manifest.json  (atomic)
 *   dist/SHA256SUMS                (atomic upsert)
 *
 * Never reads credentials, calls Apple services, or makes network requests.
 */

import { createHash } from "node:crypto";
import { createReadStream, promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { load as loadYaml } from "js-yaml";
import {
  buildManifest,
  renderManifestJson,
  resolveArch,
  upsertSha256Line,
} from "./release-mac-manifest.lib.mjs";
import { atomicWrite } from "./release-mac-manifest.io.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");
const distDir = path.join(desktop, "dist");

// ---------------------------------------------------------------------------
// Argument validation
// ---------------------------------------------------------------------------

const args = process.argv.slice(2);
if (args.length !== 1) {
  process.stderr.write(
    "Usage: pnpm release:mac:manifest <path-to-dmg>\n" +
      `Error: expected exactly one DMG path, got ${args.length}\n`
  );
  process.exit(1);
}

const [dmgArg] = args;
const dmgPath = path.resolve(dmgArg);

if (!dmgPath.endsWith(".dmg")) {
  process.stderr.write(
    `Error: path does not end with .dmg — got: ${dmgPath}\n`
  );
  process.exit(1);
}

let dmgStat;
try {
  dmgStat = await fs.lstat(dmgPath);
} catch (err) {
  process.stderr.write(
    `Error: cannot stat DMG — ${err.message}\n` +
      `  Path: ${dmgPath}\n`
  );
  process.exit(1);
}

if (!dmgStat.isFile()) {
  process.stderr.write(
    `Error: DMG path is not a regular file (symlinks are rejected) — ${dmgPath}\n`
  );
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Read config metadata
// ---------------------------------------------------------------------------

let pkgJson;
try {
  const raw = await fs.readFile(path.join(desktop, "package.json"), "utf8");
  pkgJson = JSON.parse(raw);
} catch (err) {
  process.stderr.write(`Error: cannot read package.json — ${err.message}\n`);
  process.exit(1);
}

const version = pkgJson.version;
if (!version) {
  process.stderr.write("Error: package.json has no \"version\" field\n");
  process.exit(1);
}

let builderConfig;
try {
  const raw = await fs.readFile(path.join(desktop, "electron-builder.yml"), "utf8");
  builderConfig = loadYaml(raw);
} catch (err) {
  process.stderr.write(
    `Error: cannot read electron-builder.yml — ${err.message}\n`
  );
  process.exit(1);
}

const appId = builderConfig?.appId;
const productName = builderConfig?.productName;

if (!appId) {
  process.stderr.write("Error: electron-builder.yml has no \"appId\" field\n");
  process.exit(1);
}
if (!productName) {
  process.stderr.write("Error: electron-builder.yml has no \"productName\" field\n");
  process.exit(1);
}

// Collect all declared arches from mac.target entries
const macTargets = builderConfig?.mac?.target ?? [];
const configArches = [];
for (const t of macTargets) {
  const arches = Array.isArray(t?.arch) ? t.arch : t?.arch ? [t.arch] : [];
  configArches.push(...arches);
}

const artifactBasename = path.basename(dmgPath);

let arch;
try {
  arch = resolveArch({ configArches, artifactBasename });
} catch (err) {
  process.stderr.write(`Error: ${err.message}\n`);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Stream SHA-256 + byte size
// ---------------------------------------------------------------------------

function hashFile(filePath) {
  return new Promise((resolve, reject) => {
    const hash = createHash("sha256");
    const stream = createReadStream(filePath);
    stream.on("error", reject);
    stream.on("data", (chunk) => hash.update(chunk));
    stream.on("end", () => resolve(hash.digest("hex")));
  });
}

process.stdout.write(`\nHashing ${artifactBasename}...\n`);

let sha256;
try {
  sha256 = await hashFile(dmgPath);
} catch (err) {
  process.stderr.write(`Error: failed to hash DMG — ${err.message}\n`);
  process.exit(1);
}

const byteSize = dmgStat.size;
const buildTimestamp = new Date().toISOString();

// ---------------------------------------------------------------------------
// Build manifest
// ---------------------------------------------------------------------------

let manifest;
try {
  manifest = buildManifest({
    appId,
    productName,
    version,
    artifact: artifactBasename,
    arch,
    byteSize,
    sha256,
    buildTimestamp,
  });
} catch (err) {
  process.stderr.write(`Error: ${err.message}\n`);
  process.exit(1);
}

const manifestJson = renderManifestJson(manifest);

// ---------------------------------------------------------------------------
// Atomic writes to dist/
// ---------------------------------------------------------------------------

// Ensure dist/ exists
try {
  await fs.mkdir(distDir, { recursive: true });
} catch (err) {
  process.stderr.write(`Error: cannot create dist/ directory — ${err.message}\n`);
  process.exit(1);
}

// Write manifest.json atomically
const manifestPath = path.join(distDir, `${artifactBasename}.manifest.json`);
try {
  await atomicWrite(manifestPath, manifestJson);
} catch (err) {
  process.stderr.write(`Error: failed to write manifest — ${err.message}\n`);
  process.exit(1);
}

// Read existing SHA256SUMS, upsert line, write atomically
const sha256sumsPath = path.join(distDir, "SHA256SUMS");
let existingContent = "";
try {
  existingContent = await fs.readFile(sha256sumsPath, "utf8");
} catch (err) {
  if (err.code !== "ENOENT") {
    process.stderr.write(`Error: failed to read SHA256SUMS — ${err.message}\n`);
    process.exit(1);
  }
}

const updatedSums = upsertSha256Line({ existingContent, sha256, artifact: artifactBasename });

try {
  await atomicWrite(sha256sumsPath, updatedSums);
} catch (err) {
  process.stderr.write(`Error: failed to write SHA256SUMS — ${err.message}\n`);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Success output
// ---------------------------------------------------------------------------

process.stdout.write(
  `\n[release:mac:manifest] OK\n` +
    `  artifact : ${artifactBasename}\n` +
    `  byteSize : ${byteSize}\n` +
    `  sha256   : ${sha256}\n` +
    `  manifest : ${manifestPath}\n` +
    `  sums     : ${sha256sumsPath}\n`
);
