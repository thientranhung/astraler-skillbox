import { describe, it, expect } from "vitest";
import path from "node:path";
import {
  resolveAppExecutable,
  isReadyLine,
  isFailureLine,
  extractFailureDiagnostic,
  buildLaunchEnv,
  isTimedOut,
  detectOrphanedSidecar,
} from "./release-mac-launch-smoke.lib.mjs";

// ---------------------------------------------------------------------------
// resolveAppExecutable
// ---------------------------------------------------------------------------

describe("resolveAppExecutable", () => {
  it("returns ok=true with correct app and exec paths", () => {
    const result = resolveAppExecutable("/some/desktop");
    expect(result.ok).toBe(true);
    expect(result.appPath).toBe(
      path.join("/some/desktop", "dist", "mac-arm64", "Astraler Skillbox.app")
    );
    expect(result.execPath).toBe(
      path.join(
        "/some/desktop",
        "dist",
        "mac-arm64",
        "Astraler Skillbox.app",
        "Contents",
        "MacOS",
        "Astraler Skillbox"
      )
    );
  });
});

// ---------------------------------------------------------------------------
// isReadyLine
// ---------------------------------------------------------------------------

describe("isReadyLine", () => {
  it("returns true for Go core ready line", () => {
    expect(isReadyLine("[manager] Go core ready")).toBe(true);
    expect(isReadyLine("some prefix [manager] Go core ready extra")).toBe(true);
  });

  it("returns false for unrelated lines", () => {
    expect(isReadyLine("[manager] spawning Go core")).toBe(false);
    expect(isReadyLine("[manager] FATAL: something")).toBe(false);
    expect(isReadyLine("")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// isFailureLine
// ---------------------------------------------------------------------------

describe("isFailureLine", () => {
  it("detects Library not loaded", () => {
    expect(isFailureLine("Library not loaded: /path/to/Electron Framework")).toBe(true);
  });

  it("detects not valid for use in process", () => {
    expect(isFailureLine("Electron Framework not valid for use in process using Library Validation: mapped file has no Team ID and this process does not allow using entitlements with no Team ID")).toBe(true);
  });

  it("detects server.ready timeout", () => {
    expect(isFailureLine("server.ready timeout waiting for Go core")).toBe(true);
  });

  it("detects [manager] FATAL", () => {
    expect(isFailureLine("[manager] FATAL: Go core crashed too many times")).toBe(true);
  });

  it("returns false for normal output", () => {
    expect(isFailureLine("[manager] spawning Go core")).toBe(false);
    expect(isFailureLine("[manager] Go core ready")).toBe(false);
    expect(isFailureLine("")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// extractFailureDiagnostic
// ---------------------------------------------------------------------------

describe("extractFailureDiagnostic", () => {
  it("returns library-validation diagnostic for Library not loaded", () => {
    const result = extractFailureDiagnostic("Library not loaded: /path/to/Electron Framework");
    expect(result).toContain("Library not loaded");
    expect(result).toContain("hardened runtime");
  });

  it("returns Team ID diagnostic for not valid for use in process", () => {
    const result = extractFailureDiagnostic("Electron Framework not valid for use in process");
    expect(result).toContain("Team ID");
  });

  it("returns timeout diagnostic for server.ready timeout", () => {
    const result = extractFailureDiagnostic("server.ready timeout");
    expect(result).toContain("timeout");
  });

  it("returns fatal diagnostic for [manager] FATAL", () => {
    const result = extractFailureDiagnostic("[manager] FATAL: crash");
    expect(result).toContain("fatal");
  });

  it("returns trimmed line for unknown failures", () => {
    const result = extractFailureDiagnostic("  some unknown error  ");
    expect(result).toBe("some unknown error");
  });
});

// ---------------------------------------------------------------------------
// buildLaunchEnv
// ---------------------------------------------------------------------------

describe("buildLaunchEnv", () => {
  it("sets SKILLBOX_DB_PATH inside tmpDir", () => {
    const env = buildLaunchEnv({}, "/tmp/smoke-123");
    expect(env["SKILLBOX_DB_PATH"]).toBe(path.join("/tmp/smoke-123", "skillbox.db"));
  });

  it("strips CSC_ prefixed keys", () => {
    const env = buildLaunchEnv({ CSC_LINK: "cert", CSC_KEY_PASSWORD: "pass" }, "/tmp/x");
    expect(env["CSC_LINK"]).toBeUndefined();
    expect(env["CSC_KEY_PASSWORD"]).toBeUndefined();
  });

  it("strips APPLE_ prefixed keys", () => {
    const env = buildLaunchEnv({ APPLE_ID: "user@example.com", APPLE_APP_SPECIFIC_PASSWORD: "pass" }, "/tmp/x");
    expect(env["APPLE_ID"]).toBeUndefined();
    expect(env["APPLE_APP_SPECIFIC_PASSWORD"]).toBeUndefined();
  });

  it("strips NOTARYTOOL_ prefixed keys", () => {
    const env = buildLaunchEnv({ NOTARYTOOL_API_KEY: "key" }, "/tmp/x");
    expect(env["NOTARYTOOL_API_KEY"]).toBeUndefined();
  });

  it("preserves non-credential keys", () => {
    const env = buildLaunchEnv({ HOME: "/Users/test", PATH: "/usr/bin" }, "/tmp/x");
    expect(env["HOME"]).toBe("/Users/test");
    expect(env["PATH"]).toBe("/usr/bin");
  });

  it("overrides any existing SKILLBOX_DB_PATH", () => {
    const env = buildLaunchEnv({ SKILLBOX_DB_PATH: "/old/path.db" }, "/tmp/smoke-456");
    expect(env["SKILLBOX_DB_PATH"]).toBe(path.join("/tmp/smoke-456", "skillbox.db"));
  });
});

// ---------------------------------------------------------------------------
// isTimedOut
// ---------------------------------------------------------------------------

describe("isTimedOut", () => {
  it("returns false when elapsed < timeout", () => {
    expect(isTimedOut(4999, 5000)).toBe(false);
  });

  it("returns true when elapsed >= timeout", () => {
    expect(isTimedOut(5000, 5000)).toBe(true);
    expect(isTimedOut(6000, 5000)).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// detectOrphanedSidecar
// ---------------------------------------------------------------------------

describe("detectOrphanedSidecar", () => {
  const appPath = "/Users/dev/astraler-skillbox/apps/desktop/dist/mac-arm64/Astraler Skillbox.app";
  const sidecarExe = path.join(appPath, "Contents", "Resources", "core", "skillbox-core");

  it("detects a skillbox-core inside the staged app as an orphan", () => {
    const result = detectOrphanedSidecar([{ pid: 1234, exe: sidecarExe }], appPath);
    expect(result.hasOrphan).toBe(true);
    expect(result.orphans).toHaveLength(1);
    expect(result.orphans[0].pid).toBe(1234);
  });

  it("returns no orphans when process list is empty", () => {
    const result = detectOrphanedSidecar([], appPath);
    expect(result.hasOrphan).toBe(false);
    expect(result.orphans).toHaveLength(0);
  });

  it("ignores skillbox-core outside the staged app path", () => {
    const externalSidecar = "/usr/local/bin/skillbox-core";
    const result = detectOrphanedSidecar([{ pid: 999, exe: externalSidecar }], appPath);
    expect(result.hasOrphan).toBe(false);
  });

  it("ignores processes inside the app that are NOT skillbox-core", () => {
    const otherExe = path.join(appPath, "Contents", "MacOS", "Astraler Skillbox");
    const result = detectOrphanedSidecar([{ pid: 5678, exe: otherExe }], appPath);
    expect(result.hasOrphan).toBe(false);
  });

  it("detects multiple orphans", () => {
    const procs = [
      { pid: 100, exe: sidecarExe },
      { pid: 101, exe: sidecarExe },
    ];
    const result = detectOrphanedSidecar(procs, appPath);
    expect(result.hasOrphan).toBe(true);
    expect(result.orphans).toHaveLength(2);
  });

  it("detects orphan when exe path contains spaces (e.g. 'Astraler Skillbox.app')", () => {
    // Simulates a ps-parsed entry where the full path with spaces is preserved intact.
    // Previously, pgrep output split by whitespace would truncate the path at the first
    // space, turning ".../Astraler Skillbox.app/..." into ".../Astraler" and missing the orphan.
    const spacyExe = "/Users/dev/repo/apps/desktop/dist/mac-arm64/Astraler Skillbox.app/Contents/Resources/core/skillbox-core";
    const spacyAppPath = "/Users/dev/repo/apps/desktop/dist/mac-arm64/Astraler Skillbox.app";
    const result = detectOrphanedSidecar([{ pid: 9999, exe: spacyExe }], spacyAppPath);
    expect(result.hasOrphan).toBe(true);
    expect(result.orphans[0].pid).toBe(9999);
  });

  it("does not detect orphan when truncated path (old bug) is passed as exe", () => {
    // Confirms that a path truncated at the first space does NOT falsely match.
    const truncatedExe = "/Users/dev/repo/apps/desktop/dist/mac-arm64/Astraler";
    const spacyAppPath = "/Users/dev/repo/apps/desktop/dist/mac-arm64/Astraler Skillbox.app";
    const result = detectOrphanedSidecar([{ pid: 9998, exe: truncatedExe }], spacyAppPath);
    expect(result.hasOrphan).toBe(false);
  });
});
