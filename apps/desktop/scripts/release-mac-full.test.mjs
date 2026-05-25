import { describe, it, expect, vi } from "vitest";
import { selectChangedDmg, runReleaseMacFull } from "./release-mac-full.lib.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function dmg(p, size = 100, mtimeMs = 1000) {
  return { path: p, size, mtimeMs, isFile: true };
}

function notFile(p) {
  return { path: p, size: 0, mtimeMs: 1000, isFile: false };
}

// ---------------------------------------------------------------------------
// selectChangedDmg
// ---------------------------------------------------------------------------

describe("selectChangedDmg — zero candidates", () => {
  it("returns error when before and after are both empty", () => {
    const r = selectChangedDmg([], [], 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/no .dmg/);
  });

  it("returns error when only stale unchanged DMGs exist", () => {
    const before = [dmg("/dist/App-1.0.dmg", 500, 2000)];
    const after = [dmg("/dist/App-1.0.dmg", 500, 2000)]; // identical
    const r = selectChangedDmg(before, after, 1000);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/no .dmg/);
  });

  it("ignores non-.dmg files", () => {
    const before = [];
    const after = [{ path: "/dist/App.zip", size: 100, mtimeMs: 1000, isFile: true }];
    const r = selectChangedDmg(before, after, 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/no .dmg/);
  });

  it("ignores non-regular .dmg entries (isFile=false)", () => {
    const before = [];
    const after = [notFile("/dist/App.dmg")];
    const r = selectChangedDmg(before, after, 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/no .dmg/);
  });
});

describe("selectChangedDmg — one new DMG", () => {
  it("returns the new DMG path with reason=created", () => {
    const before = [];
    const after = [dmg("/dist/App-1.0.dmg")];
    const r = selectChangedDmg(before, after, 500);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/App-1.0.dmg", reason: "created" });
  });

  it("ignores stale unchanged DMG alongside the new one — picks only the new one", () => {
    const before = [dmg("/dist/Old.dmg", 200, 1000)];
    const after = [dmg("/dist/Old.dmg", 200, 1000), dmg("/dist/New.dmg", 300, 2000)];
    const r = selectChangedDmg(before, after, 1500);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/New.dmg", reason: "created" });
  });
});

describe("selectChangedDmg — same-name overwrite (modified)", () => {
  it("detects size change as a modification", () => {
    const before = [dmg("/dist/App.dmg", 100, 1000)];
    const after = [dmg("/dist/App.dmg", 200, 1000)]; // size changed
    const r = selectChangedDmg(before, after, 900);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/App.dmg", reason: "modified" });
  });

  it("detects mtimeMs change as a modification", () => {
    const before = [dmg("/dist/App.dmg", 100, 1000)];
    const after = [dmg("/dist/App.dmg", 100, 2000)]; // mtime changed
    const r = selectChangedDmg(before, after, 900);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/App.dmg", reason: "modified" });
  });

  it("detects both size and mtime change", () => {
    const before = [dmg("/dist/App.dmg", 100, 1000)];
    const after = [dmg("/dist/App.dmg", 999, 9999)];
    const r = selectChangedDmg(before, after, 900);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/App.dmg", reason: "modified" });
  });
});

describe("selectChangedDmg — multiple changed/new DMGs", () => {
  it("returns error when two new DMGs appear", () => {
    const before = [];
    const after = [dmg("/dist/A.dmg"), dmg("/dist/B.dmg")];
    const r = selectChangedDmg(before, after, 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/multiple/);
    expect(r.error).toMatch(/2/);
    expect(r.error).toMatch(/A\.dmg/);
    expect(r.error).toMatch(/B\.dmg/);
  });

  it("returns error when one new + one modified DMG", () => {
    const before = [dmg("/dist/Old.dmg", 100, 1000)];
    const after = [dmg("/dist/Old.dmg", 999, 2000), dmg("/dist/New.dmg")];
    const r = selectChangedDmg(before, after, 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/multiple/);
  });

  it("returns error when two modified DMGs (both overwritten)", () => {
    const before = [dmg("/dist/A.dmg", 100, 1000), dmg("/dist/B.dmg", 200, 2000)];
    const after = [dmg("/dist/A.dmg", 101, 1001), dmg("/dist/B.dmg", 201, 2001)];
    const r = selectChangedDmg(before, after, 0);
    expect(r.ok).toBe(false);
    expect(r.error).toMatch(/multiple/);
  });
});

describe("selectChangedDmg — stale unchanged artifacts are ignored", () => {
  it("many pre-existing unchanged DMGs + one new → picks the new one", () => {
    const before = [
      dmg("/dist/Old1.dmg", 100, 1000),
      dmg("/dist/Old2.dmg", 200, 2000),
      dmg("/dist/Old3.dmg", 300, 3000),
    ];
    const after = [
      dmg("/dist/Old1.dmg", 100, 1000),
      dmg("/dist/Old2.dmg", 200, 2000),
      dmg("/dist/Old3.dmg", 300, 3000),
      dmg("/dist/New.dmg", 999, 9999),
    ];
    const r = selectChangedDmg(before, after, 5000);
    expect(r).toEqual({ ok: true, dmgPath: "/dist/New.dmg", reason: "created" });
  });
});

// ---------------------------------------------------------------------------
// runReleaseMacFull — orchestration flow
// ---------------------------------------------------------------------------

function makeRunner(steps) {
  // steps: Map of scriptArgs[0] -> { code }
  return async (_stage, args) => steps[args[0]] ?? { code: 0 };
}

describe("runReleaseMacFull — preflight failure", () => {
  it("stops without snapshotting or packaging when preflight fails", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn().mockResolvedValueOnce({ code: 1 }); // preflight fails
    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 1000 });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("preflight");
    expect(snapshotDist).not.toHaveBeenCalled();
    expect(runStage).toHaveBeenCalledTimes(1);
    expect(runStage.mock.calls[0][1][0]).toBe("release:mac:check");
  });
});

describe("runReleaseMacFull — package failure", () => {
  it("stops without verifying when package:mac fails", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight ok
      .mockResolvedValueOnce({ code: 2 }); // package fails
    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 1000 });

    expect(result.exitCode).toBe(2);
    expect(result.failedStage).toBe("package");
    expect(runStage).toHaveBeenCalledTimes(2);
    // verify must not be called
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacFull — DMG selection failure (zero changed DMGs)", () => {
  it("exits non-zero and does not invoke verify when no DMG changed", async () => {
    const staleSnapshot = [dmg("/dist/App.dmg", 100, 1000)];
    const snapshotDist = vi.fn().mockResolvedValue(staleSnapshot); // same before and after
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight ok
      .mockResolvedValueOnce({ code: 0 }); // package ok

    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 500 });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("dmg-selection");
    expect(result.dmgError).toMatch(/no .dmg/);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacFull — DMG selection failure (multiple changed DMGs)", () => {
  it("exits non-zero and does not invoke verify when multiple DMGs changed", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve([]); // before: empty
      // after: two new DMGs
      return Promise.resolve([dmg("/dist/A.dmg"), dmg("/dist/B.dmg")]);
    });
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight ok
      .mockResolvedValueOnce({ code: 0 }); // package ok

    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 500 });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("dmg-selection");
    expect(result.dmgError).toMatch(/multiple/);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacFull — verify failure", () => {
  it("exits non-zero when verify fails", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve([]); // before: empty
      return Promise.resolve([dmg("/dist/App-1.0.dmg")]); // after: one new DMG
    });
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight ok
      .mockResolvedValueOnce({ code: 0 }) // package ok
      .mockResolvedValueOnce({ code: 1 }); // verify fails

    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 500 });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("verify");
  });
});

describe("runReleaseMacFull — success with new DMG", () => {
  it("calls verify with the selected DMG path and returns exitCode 0", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve([]);
      return Promise.resolve([dmg("/dist/App-1.0-arm64.dmg")]);
    });
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight
      .mockResolvedValueOnce({ code: 0 }) // package
      .mockResolvedValueOnce({ code: 0 }); // verify

    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 500 });

    expect(result.exitCode).toBe(0);
    expect(result.dmgPath).toBe("/dist/App-1.0-arm64.dmg");
    expect(result.dmgReason).toBe("created");

    const verifyCall = runStage.mock.calls[2];
    expect(verifyCall[1][0]).toBe("release:mac:verify");
    expect(verifyCall[1][1]).toBe("/dist/App-1.0-arm64.dmg");
    // Must not include --allow-adhoc
    expect(verifyCall[1]).not.toContain("--allow-adhoc");
  });
});

describe("runReleaseMacFull — success with same-name overwrite (modified)", () => {
  it("calls verify with the overwritten DMG and returns exitCode 0", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.resolve([dmg("/dist/App.dmg", 100, 1000)]);
      }
      return Promise.resolve([dmg("/dist/App.dmg", 999, 9999)]); // overwritten
    });
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 })
      .mockResolvedValueOnce({ code: 0 })
      .mockResolvedValueOnce({ code: 0 });

    const result = await runReleaseMacFull({ runStage, snapshotDist, now: () => 500 });

    expect(result.exitCode).toBe(0);
    expect(result.dmgPath).toBe("/dist/App.dmg");
    expect(result.dmgReason).toBe("modified");

    const verifyCall = runStage.mock.calls[2];
    expect(verifyCall[1][1]).toBe("/dist/App.dmg");
    expect(verifyCall[1]).not.toContain("--allow-adhoc");
  });
});

describe("runReleaseMacFull — stage command names", () => {
  it("calls preflight with release:mac:check", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn().mockResolvedValueOnce({ code: 1 }); // stop early
    await runReleaseMacFull({ runStage, snapshotDist, now: () => 0 });
    expect(runStage.mock.calls[0][1]).toContain("release:mac:check");
  });

  it("calls package with package:mac", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // preflight ok
      .mockResolvedValueOnce({ code: 1 }); // package fails - stop
    await runReleaseMacFull({ runStage, snapshotDist, now: () => 0 });
    expect(runStage.mock.calls[1][1]).toContain("package:mac");
  });
});
