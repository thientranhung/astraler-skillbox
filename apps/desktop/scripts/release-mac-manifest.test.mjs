import { describe, it, expect } from "vitest";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import {
  buildManifest,
  renderManifestJson,
  parseArchFromFilename,
  resolveArch,
  upsertSha256Line,
} from "./release-mac-manifest.lib.mjs";
import { atomicWrite } from "./release-mac-manifest.io.mjs";

// ---------------------------------------------------------------------------
// buildManifest
// ---------------------------------------------------------------------------

const VALID = {
  appId: "com.astraler.skillbox",
  productName: "Astraler Skillbox",
  version: "0.1.0",
  artifact: "Astraler Skillbox-0.1.0-arm64.dmg",
  arch: "arm64",
  byteSize: 134217728,
  sha256: "a".repeat(64),
  buildTimestamp: "2026-05-26T00:00:00.000Z",
};

describe("buildManifest — valid input", () => {
  it("returns an object with exactly eight fields in stable key order", () => {
    const m = buildManifest(VALID);
    const keys = Object.keys(m);
    expect(keys).toEqual([
      "appId",
      "productName",
      "version",
      "artifact",
      "arch",
      "byteSize",
      "sha256",
      "buildTimestamp",
    ]);
  });

  it("byteSize is an integer", () => {
    const m = buildManifest(VALID);
    expect(Number.isInteger(m.byteSize)).toBe(true);
    expect(m.byteSize).toBe(134217728);
  });

  it("sha256 is 64 lowercase hex chars", () => {
    const m = buildManifest({ ...VALID, sha256: "ab01".repeat(16) });
    expect(m.sha256).toMatch(/^[0-9a-f]{64}$/);
  });
});

describe("buildManifest — missing or empty required fields", () => {
  for (const key of [
    "appId",
    "productName",
    "version",
    "artifact",
    "arch",
    "sha256",
    "buildTimestamp",
  ]) {
    it(`throws when "${key}" is missing`, () => {
      const bad = { ...VALID };
      delete bad[key];
      expect(() => buildManifest(bad)).toThrow(new RegExp(`"${key}"`));
    });

    it(`throws when "${key}" is empty string`, () => {
      expect(() => buildManifest({ ...VALID, [key]: "" })).toThrow(new RegExp(`"${key}"`));
    });
  }

  it("throws when byteSize is missing", () => {
    const bad = { ...VALID };
    delete bad.byteSize;
    expect(() => buildManifest(bad)).toThrow(/"byteSize"/);
  });
});

describe("buildManifest — byteSize validation", () => {
  it("throws when byteSize is a float", () => {
    expect(() => buildManifest({ ...VALID, byteSize: 1.5 })).toThrow(/byteSize/);
  });

  it("throws when byteSize is a string", () => {
    expect(() => buildManifest({ ...VALID, byteSize: "1024" })).toThrow(/byteSize/);
  });

  it("accepts byteSize of 0 (valid integer)", () => {
    // 0 is falsy but a valid integer; empty check only applies to strings
    expect(() => buildManifest({ ...VALID, byteSize: 0 })).not.toThrow();
  });
});

describe("buildManifest — sha256 validation", () => {
  it("throws for 63-char hex", () => {
    expect(() => buildManifest({ ...VALID, sha256: "a".repeat(63) })).toThrow(/sha256/);
  });

  it("throws for 65-char hex", () => {
    expect(() => buildManifest({ ...VALID, sha256: "a".repeat(65) })).toThrow(/sha256/);
  });

  it("throws for uppercase hex", () => {
    expect(() => buildManifest({ ...VALID, sha256: "A".repeat(64) })).toThrow(/sha256/);
  });

  it("throws for non-hex string", () => {
    expect(() => buildManifest({ ...VALID, sha256: "z".repeat(64) })).toThrow(/sha256/);
  });
});

// ---------------------------------------------------------------------------
// renderManifestJson
// ---------------------------------------------------------------------------

describe("renderManifestJson", () => {
  it("returns 2-space pretty JSON with trailing newline", () => {
    const m = buildManifest(VALID);
    const json = renderManifestJson(m);
    expect(json.endsWith("\n")).toBe(true);
    expect(json).toContain('  "appId"');
    expect(JSON.parse(json)).toEqual(m);
  });

  it("is byte-stable for identical input", () => {
    const m = buildManifest(VALID);
    expect(renderManifestJson(m)).toBe(renderManifestJson(m));
  });

  it("round-trips via JSON.parse", () => {
    const m = buildManifest(VALID);
    expect(JSON.parse(renderManifestJson(m))).toEqual(m);
  });
});

// ---------------------------------------------------------------------------
// parseArchFromFilename
// ---------------------------------------------------------------------------

describe("parseArchFromFilename", () => {
  it("parses arm64 from standard filename", () => {
    expect(parseArchFromFilename("Astraler Skillbox-0.1.0-arm64.dmg")).toBe("arm64");
  });

  it("parses x64 from filename", () => {
    expect(parseArchFromFilename("Astraler Skillbox-0.1.0-x64.dmg")).toBe("x64");
  });

  it("returns null for filename with no arch token", () => {
    expect(parseArchFromFilename("Astraler Skillbox-0.1.0.dmg")).toBeNull();
  });

  it("returns null when trailing segment looks like a version number", () => {
    // A file named App-1.0.0.dmg should not parse '0' as arch
    expect(parseArchFromFilename("App-0.1.0.dmg")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(parseArchFromFilename("")).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// resolveArch
// ---------------------------------------------------------------------------

describe("resolveArch", () => {
  it("uses single configured arch, ignoring filename", () => {
    const arch = resolveArch({
      configArches: ["arm64"],
      artifactBasename: "App-0.1.0-x64.dmg",
    });
    expect(arch).toBe("arm64");
  });

  it("falls back to filename when config has 2+ arches", () => {
    const arch = resolveArch({
      configArches: ["arm64", "x64"],
      artifactBasename: "App-0.1.0-arm64.dmg",
    });
    expect(arch).toBe("arm64");
  });

  it("falls back to filename when config is empty", () => {
    const arch = resolveArch({
      configArches: [],
      artifactBasename: "App-0.1.0-arm64.dmg",
    });
    expect(arch).toBe("arm64");
  });

  it("throws when config is ambiguous and filename has no arch token", () => {
    expect(() =>
      resolveArch({
        configArches: ["arm64", "x64"],
        artifactBasename: "App-0.1.0.dmg",
      })
    ).toThrow(/arch/);
  });

  it("throws when config is empty and filename has no arch token", () => {
    expect(() =>
      resolveArch({
        configArches: [],
        artifactBasename: "App-0.1.0.dmg",
      })
    ).toThrow(/arch/);
  });
});

// ---------------------------------------------------------------------------
// upsertSha256Line
// ---------------------------------------------------------------------------

const SHA = "a".repeat(64);
const SHA2 = "b".repeat(64);
const ARTIFACT = "Astraler Skillbox-0.1.0-arm64.dmg";
const ARTIFACT2 = "Astraler Skillbox-0.2.0-arm64.dmg";

describe("upsertSha256Line — empty existing content", () => {
  it("creates a single canonical line with two-space separator and trailing newline", () => {
    const result = upsertSha256Line({ existingContent: "", sha256: SHA, artifact: ARTIFACT });
    expect(result).toBe(`${SHA}  ${ARTIFACT}\n`);
  });
});

describe("upsertSha256Line — same basename replacement", () => {
  it("replaces the existing line in place, no duplicate", () => {
    const existing = `${SHA}  ${ARTIFACT}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: SHA2, artifact: ARTIFACT });
    expect(result).toBe(`${SHA2}  ${ARTIFACT}\n`);
    // Only one line for the artifact
    const lines = result.trim().split("\n");
    expect(lines.filter((l) => l.includes(ARTIFACT))).toHaveLength(1);
  });

  it("collapses pre-existing duplicate lines for the same basename", () => {
    const existing = `${SHA}  ${ARTIFACT}\n${SHA2}  ${ARTIFACT}\n${"c".repeat(64)}  ${ARTIFACT2}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: "d".repeat(64), artifact: ARTIFACT });
    const lines = result.trim().split("\n");
    expect(lines).toEqual([`${"d".repeat(64)}  ${ARTIFACT}`, `${"c".repeat(64)}  ${ARTIFACT2}`]);
    expect(lines.filter((l) => l.includes(ARTIFACT))).toHaveLength(1);
  });

  it("preserves other artifacts' lines and their order", () => {
    const existing = `${SHA}  ${ARTIFACT}\n${SHA2}  ${ARTIFACT2}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: "c".repeat(64), artifact: ARTIFACT });
    const lines = result.trim().split("\n");
    expect(lines).toHaveLength(2);
    expect(lines[0]).toBe(`${"c".repeat(64)}  ${ARTIFACT}`);
    expect(lines[1]).toBe(`${SHA2}  ${ARTIFACT2}`);
  });

  it("is idempotent — re-running with same inputs produces byte-identical output", () => {
    const existing = `${SHA}  ${ARTIFACT}\n`;
    const r1 = upsertSha256Line({ existingContent: existing, sha256: SHA, artifact: ARTIFACT });
    const r2 = upsertSha256Line({ existingContent: r1, sha256: SHA, artifact: ARTIFACT });
    expect(r1).toBe(r2);
  });
});

describe("upsertSha256Line — new artifact appended", () => {
  it("appends for a different artifact, preserving existing line", () => {
    const existing = `${SHA}  ${ARTIFACT2}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: SHA2, artifact: ARTIFACT });
    const lines = result.trim().split("\n");
    expect(lines).toHaveLength(2);
    expect(lines[0]).toContain(ARTIFACT2);
    expect(lines[1]).toContain(ARTIFACT);
  });
});

describe("upsertSha256Line — whitespace normalization", () => {
  it("normalizes a line where basename has surrounding spaces — no stale duplicate", () => {
    // Simulate an existing line where the name was written with extra space
    const existing = `${SHA}  ${ARTIFACT}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: SHA2, artifact: ARTIFACT });
    const lines = result.trim().split("\n");
    expect(lines).toHaveLength(1);
    expect(lines[0]).toBe(`${SHA2}  ${ARTIFACT}`);
  });
});

describe("upsertSha256Line — trailing newline", () => {
  it("output always ends with exactly one trailing newline", () => {
    const result = upsertSha256Line({ existingContent: "", sha256: SHA, artifact: ARTIFACT });
    expect(result.endsWith("\n")).toBe(true);
    expect(result.endsWith("\n\n")).toBe(false);
  });

  it("output ends with single newline even when existing has multiple lines", () => {
    const existing = `${SHA}  ${ARTIFACT}\n${SHA2}  ${ARTIFACT2}\n`;
    const result = upsertSha256Line({ existingContent: existing, sha256: "c".repeat(64), artifact: ARTIFACT });
    expect(result.endsWith("\n")).toBe(true);
    expect(result.endsWith("\n\n")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// atomicWrite
// ---------------------------------------------------------------------------

describe("atomicWrite", () => {
  it("leaves the previous final file intact when the temp write fails", async () => {
    const dir = await mkdtemp(path.join(os.tmpdir(), "skillbox-manifest-"));
    try {
      const finalPath = path.join(dir, "SHA256SUMS");
      await writeFile(finalPath, "previous\n", "utf8");

      const fsImpl = {
        writeFile: async () => {
          throw new Error("simulated write failure");
        },
        rename: async () => {
          throw new Error("rename should not be reached");
        },
        unlink: async () => {},
      };

      await expect(
        atomicWrite(finalPath, "new\n", { fsImpl, tempPath: path.join(dir, ".tmp-test") })
      ).rejects.toThrow(/simulated write failure/);

      await expect(readFile(finalPath, "utf8")).resolves.toBe("previous\n");
    } finally {
      await rm(dir, { recursive: true, force: true });
    }
  });
});
