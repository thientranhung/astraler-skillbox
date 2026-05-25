/**
 * Pure helpers for macOS release manifest + checksum generation (Slice 3C).
 * No fs, process, crypto, clock, env, or child_process access.
 */

const REQUIRED_KEYS = [
  "appId",
  "productName",
  "version",
  "artifact",
  "arch",
  "byteSize",
  "sha256",
  "buildTimestamp",
];

/**
 * Build the manifest object with stable key order.
 * Throws a clear error for missing/empty required fields or invalid shapes.
 *
 * @param {{ appId: string, productName: string, version: string, artifact: string,
 *           arch: string, byteSize: number, sha256: string, buildTimestamp: string }} fields
 * @returns {Record<string, unknown>}
 */
export function buildManifest(fields) {
  for (const key of REQUIRED_KEYS) {
    const val = fields[key];
    if (val === undefined || val === null || val === "") {
      throw new Error(`buildManifest: required field "${key}" is missing or empty`);
    }
  }

  const { appId, productName, version, artifact, arch, byteSize, sha256, buildTimestamp } =
    fields;

  if (!Number.isInteger(byteSize)) {
    throw new Error(
      `buildManifest: "byteSize" must be an integer, got ${typeof byteSize} ${byteSize}`
    );
  }

  if (typeof sha256 !== "string" || !/^[0-9a-f]{64}$/.test(sha256)) {
    throw new Error(
      `buildManifest: "sha256" must be 64 lowercase hex chars, got ${JSON.stringify(sha256)}`
    );
  }

  // Stable key order as specified
  return { appId, productName, version, artifact, arch, byteSize, sha256, buildTimestamp };
}

/**
 * Serialize a manifest to 2-space indented JSON with a trailing newline.
 * Deterministic: byte-stable for identical input objects.
 *
 * @param {Record<string, unknown>} manifest
 * @returns {string}
 */
export function renderManifestJson(manifest) {
  return JSON.stringify(manifest, null, 2) + "\n";
}

/**
 * Parse the arch token from a DMG filename.
 * Expected pattern: `<name>-<version>-<arch>.dmg` where arch is the last dash-segment.
 * Returns null if the pattern doesn't match.
 *
 * @param {string} basename
 * @returns {string | null}
 */
export function parseArchFromFilename(basename) {
  // Match trailing `-<arch>.dmg` — arch is the segment between the last dash and .dmg
  const match = basename.match(/-([^-]+)\.dmg$/i);
  if (!match) return null;
  const token = match[1];
  // Must look like a valid arch token (not a version number like "0.1.0")
  if (/^\d+\.\d+/.test(token)) return null;
  return token;
}

/**
 * Resolve the CPU architecture for the artifact.
 * Config-first: if configArches has exactly one entry, use it.
 * Filename fallback: used only when config is ambiguous (0 or 2+ arches).
 * Throws if neither resolves.
 *
 * @param {{ configArches: string[], artifactBasename: string }} opts
 * @returns {string}
 */
export function resolveArch({ configArches, artifactBasename }) {
  if (Array.isArray(configArches) && configArches.length === 1) {
    return configArches[0];
  }

  // Fallback: parse from filename
  const fromFilename = parseArchFromFilename(artifactBasename);
  if (fromFilename) return fromFilename;

  throw new Error(
    `resolveArch: cannot determine arch — config has ${configArches?.length ?? 0} arches and filename "${artifactBasename}" has no recognizable arch token`
  );
}

/**
 * Upsert a SHA-256 checksum line into SHA256SUMS content.
 * Line format: `<sha256><two spaces><basename>\n`
 * - If a line for this basename exists, it is replaced in place.
 * - If not, a new line is appended.
 * - Other lines are preserved in their original order.
 * - Basename comparison is trimmed/exact.
 * - Always ends with a single trailing newline.
 *
 * @param {{ existingContent: string, sha256: string, artifact: string }} opts
 * @returns {string}
 */
export function upsertSha256Line({ existingContent, sha256, artifact }) {
  const newLine = `${sha256}  ${artifact}`;

  // Split into non-empty lines (trim trailing newline before splitting)
  const raw = existingContent ?? "";
  const lines = raw.length > 0 ? raw.replace(/\n$/, "").split("\n") : [];

  let replaced = false;
  const result = [];
  for (const line of lines) {
    // Parse basename from line — format is `<hash><two spaces><name>`
    const parts = line.match(/^([0-9a-fA-F]{64})  (.+)$/);
    if (parts && parts[2].trim() === artifact.trim()) {
      if (!replaced) {
        replaced = true;
        result.push(newLine);
      }
      continue;
    }
    result.push(line);
  }

  if (!replaced) {
    result.push(newLine);
  }

  return result.join("\n") + "\n";
}
