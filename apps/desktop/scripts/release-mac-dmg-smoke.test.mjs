import { describe, it, expect } from "vitest";
import path from "node:path";

// ---------------------------------------------------------------------------
// Imports under test — these will fail until the lib is created
// ---------------------------------------------------------------------------

import {
  resolveCopiedApp,
  buildAttachArgs,
  buildDetachArgs,
  buildDittoArgs,
  execName,
  assertExpectedAppBundle,
  finalizeDetach,
} from "./release-mac-dmg-smoke.lib.mjs";

// Re-export sanity: these must resolve without error
import { detectOrphanedSidecar, buildLaunchEnv } from "./release-mac-launch-smoke.lib.mjs";
import { discoverDmg, pickTopLevelApp } from "./release-mac-verify.parse.mjs";

// ---------------------------------------------------------------------------
// resolveCopiedApp
// ---------------------------------------------------------------------------

describe("resolveCopiedApp", () => {
  it("returns appPath and execPath inside installDir", () => {
    const result = resolveCopiedApp("/tmp/inst", "Astraler Skillbox.app");
    expect(result.appPath).toBe(path.join("/tmp/inst", "Astraler Skillbox.app"));
    expect(result.execPath).toBe(
      path.join("/tmp/inst", "Astraler Skillbox.app", "Contents", "MacOS", "Astraler Skillbox")
    );
  });
});

// ---------------------------------------------------------------------------
// buildAttachArgs
// ---------------------------------------------------------------------------

describe("buildAttachArgs", () => {
  it("returns hdiutil attach args for read-only nobrowse mount", () => {
    expect(buildAttachArgs("/tmp/mp", "/tmp/x.dmg")).toEqual([
      "attach",
      "-readonly",
      "-nobrowse",
      "-mountpoint",
      "/tmp/mp",
      "/tmp/x.dmg",
    ]);
  });
});

// ---------------------------------------------------------------------------
// buildDetachArgs
// ---------------------------------------------------------------------------

describe("buildDetachArgs", () => {
  it("returns plain detach args when force=false", () => {
    expect(buildDetachArgs("/tmp/mp", false)).toEqual(["detach", "/tmp/mp"]);
  });

  it("returns force detach args when force=true", () => {
    expect(buildDetachArgs("/tmp/mp", true)).toEqual(["detach", "-force", "/tmp/mp"]);
  });
});

// ---------------------------------------------------------------------------
// buildDittoArgs
// ---------------------------------------------------------------------------

describe("buildDittoArgs", () => {
  it("returns [src, dest] for ditto copy", () => {
    expect(buildDittoArgs("/vol/App.app", "/tmp/inst/App.app")).toEqual([
      "/vol/App.app",
      "/tmp/inst/App.app",
    ]);
  });
});

// ---------------------------------------------------------------------------
// execName
// ---------------------------------------------------------------------------

describe("execName", () => {
  it("strips .app suffix to get Mach-O executable name", () => {
    expect(execName("Astraler Skillbox.app")).toBe("Astraler Skillbox");
  });

  it("works for simple bundle names", () => {
    expect(execName("MyApp.app")).toBe("MyApp");
  });
});

// ---------------------------------------------------------------------------
// assertExpectedAppBundle
// ---------------------------------------------------------------------------

describe("assertExpectedAppBundle", () => {
  it("returns the bundle name when it matches the expected default", () => {
    expect(assertExpectedAppBundle("Astraler Skillbox.app")).toBe("Astraler Skillbox.app");
  });

  it("throws a clear error for a differently named bundle", () => {
    expect(() => assertExpectedAppBundle("Other.app")).toThrow("Astraler Skillbox.app");
  });

  it("throws for any name that is not the expected bundle", () => {
    expect(() => assertExpectedAppBundle("SomethingElse.app")).toThrow();
  });
});

// ---------------------------------------------------------------------------
// finalizeDetach — detach finalization helper
//
// Contract:
//   finalizeDetach(mountPoint, runFn) → { detachFailed, mountPointPreserved }
//
//   runFn(args) mimics spawnSync("/usr/bin/hdiutil", args) and returns {status}.
//   When normal detach fails AND forced detach also fails:
//     - detachFailed is true
//     - mountPointPreserved is true (mount point dir must NOT be removed)
//     - the returned message includes the manual-detach hint
//   When normal detach succeeds:
//     - detachFailed is false
//     - mountPointPreserved is false
//   When normal fails but force succeeds:
//     - detachFailed is false
//     - mountPointPreserved is false
// ---------------------------------------------------------------------------

describe("finalizeDetach", () => {
  it("succeeds when normal detach exits 0", () => {
    const calls = [];
    const runFn = (args) => { calls.push(args); return { status: 0 }; };
    const result = finalizeDetach("/tmp/mnt", runFn);
    expect(result.detachFailed).toBe(false);
    expect(result.mountPointPreserved).toBe(false);
    expect(calls).toHaveLength(1);
  });

  it("retries with -force when normal detach fails, succeeds on force", () => {
    let call = 0;
    const runFn = (args) => { call++; return { status: call === 1 ? 1 : 0 }; };
    const result = finalizeDetach("/tmp/mnt", runFn);
    expect(result.detachFailed).toBe(false);
    expect(result.mountPointPreserved).toBe(false);
  });

  it("exits non-zero (detachFailed=true), preserves mount point, includes manual hint when both attempts fail", () => {
    const runFn = () => ({ status: 1 });
    const result = finalizeDetach("/tmp/mnt", runFn);
    expect(result.detachFailed).toBe(true);
    expect(result.mountPointPreserved).toBe(true);
    expect(result.message).toMatch(/hdiutil detach -force/);
  });
});

// ---------------------------------------------------------------------------
// Re-export sanity checks
// ---------------------------------------------------------------------------

describe("re-export sanity", () => {
  it("detectOrphanedSidecar is importable from release-mac-launch-smoke.lib.mjs", () => {
    expect(typeof detectOrphanedSidecar).toBe("function");
  });

  it("buildLaunchEnv is importable from release-mac-launch-smoke.lib.mjs", () => {
    expect(typeof buildLaunchEnv).toBe("function");
  });

  it("discoverDmg is importable from release-mac-verify.parse.mjs", () => {
    expect(typeof discoverDmg).toBe("function");
  });

  it("pickTopLevelApp is importable from release-mac-verify.parse.mjs", () => {
    expect(typeof pickTopLevelApp).toBe("function");
  });
});
