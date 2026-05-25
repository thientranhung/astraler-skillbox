import { describe, it, expect, vi } from "vitest";
import { runReleaseMacDryRun, scrubEnv } from "./release-mac-dry-run.lib.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function dmg(p, size = 100, mtimeMs = 1000) {
  return { path: p, size, mtimeMs, isFile: true };
}

/** Returns a snapshotDist that yields [] before, then [dmg(p)] after. */
function makeSnapshotWithOneDmg(p) {
  let calls = 0;
  return vi.fn().mockImplementation(() => {
    calls++;
    if (calls === 1) return Promise.resolve([]);
    return Promise.resolve([dmg(p)]);
  });
}

/** Happy-path runStage: every script returns code 0. */
function makeHappyRunner(extras = {}) {
  return vi.fn().mockImplementation(async (_stage, args) => {
    const key = args[0];
    if (key in extras) return extras[key];
    return { code: 0 };
  });
}

/** Happy-path verifyChecksum: returns code 0. */
function happyVerify() {
  return vi.fn().mockResolvedValue({ code: 0 });
}

// ---------------------------------------------------------------------------
// Safety constraints - forbidden commands must never be called
// ---------------------------------------------------------------------------

async function runSuccessPath() {
  const snapshotDist = makeSnapshotWithOneDmg("/dist/App-1.0-arm64.dmg");
  const runStage = makeHappyRunner({
    "release:mac:manifest": {
      code: 0,
      manifestPath: "/dist/App-1.0-arm64.dmg.manifest.json",
      sha256sumsPath: "/dist/SHA256SUMS",
    },
  });
  const verifyChecksum = happyVerify();
  const result = await runReleaseMacDryRun({
    runStage,
    snapshotDist,
    verifyChecksum,
    now: () => 500,
  });
  return { result, runStage, verifyChecksum };
}

describe("runReleaseMacDryRun - forbidden commands", () => {
  it("never calls release:mac:check", async () => {
    const { runStage } = await runSuccessPath();
    const allCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(allCmds).not.toContain("release:mac:check");
  });

  it("never calls package:mac", async () => {
    const { runStage } = await runSuccessPath();
    const allCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(allCmds).not.toContain("package:mac");
  });

  it("never calls package:mac:unsigned", async () => {
    const { runStage } = await runSuccessPath();
    const allCmds = runStage.mock.calls.flatMap((c) => c[1]);
    expect(allCmds).not.toContain("package:mac:unsigned");
  });
});

// ---------------------------------------------------------------------------
// Ad-hoc package flags - exact electron-builder invocation
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - ad-hoc package flags", () => {
  it("calls electron-builder with --mac dmg -c.mac.identity=- -c.mac.notarize=false", async () => {
    const { runStage } = await runSuccessPath();
    const pkgCall = runStage.mock.calls.find((c) => c[1][0] === "electron-builder");
    expect(pkgCall).toBeDefined();
    const args = pkgCall[1];
    expect(args).toContain("--mac");
    expect(args).toContain("dmg");
    expect(args).toContain("-c.mac.identity=-");
    expect(args).toContain("-c.mac.notarize=false");
  });

  it("does NOT pass -c.mac.hardenedRuntime=false", async () => {
    const { runStage } = await runSuccessPath();
    const allArgs = runStage.mock.calls.flatMap((c) => c[1]);
    expect(allArgs).not.toContain("-c.mac.hardenedRuntime=false");
    // Also check no variant of hardenedRuntime=false
    const hardenedFalse = allArgs.filter((a) => a.includes("hardenedRuntime=false"));
    expect(hardenedFalse).toHaveLength(0);
  });

  it("does NOT pass CSC_IDENTITY_AUTO_DISCOVERY or identity=null anywhere", async () => {
    const { runStage } = await runSuccessPath();
    const allArgs = runStage.mock.calls.flatMap((c) => c[1]);
    expect(allArgs.some((a) => a.includes("CSC_IDENTITY_AUTO_DISCOVERY"))).toBe(false);
    expect(allArgs.some((a) => a.includes("identity=null"))).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Verify uses --allow-adhoc
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - verify --allow-adhoc", () => {
  it("passes --allow-adhoc as the first argument after release:mac:verify", async () => {
    const { runStage } = await runSuccessPath();
    const verifyCall = runStage.mock.calls.find((c) => c[1][0] === "release:mac:verify");
    expect(verifyCall).toBeDefined();
    expect(verifyCall[1][1]).toBe("--allow-adhoc");
  });

  it("passes the selected DMG path after --allow-adhoc", async () => {
    const { runStage } = await runSuccessPath();
    const verifyCall = runStage.mock.calls.find((c) => c[1][0] === "release:mac:verify");
    expect(verifyCall[1][2]).toBe("/dist/App-1.0-arm64.dmg");
  });
});

// ---------------------------------------------------------------------------
// Success path - stage order and return value
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - success path", () => {
  it("calls build:core, build, electron-builder, verify, manifest, checksum in order", async () => {
    const { runStage, verifyChecksum } = await runSuccessPath();
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds[0]).toBe("build:core");
    expect(calledCmds[1]).toBe("build");
    expect(calledCmds[2]).toBe("electron-builder");
    expect(calledCmds[3]).toBe("release:mac:verify");
    expect(calledCmds[4]).toBe("release:mac:manifest");
    expect(verifyChecksum).toHaveBeenCalledTimes(1);
    expect(verifyChecksum).toHaveBeenCalledWith("/dist/App-1.0-arm64.dmg");
  });

  it("returns exitCode=0 and dmgPath/reason/manifestPath/sha256sumsPath on success", async () => {
    const { result } = await runSuccessPath();
    expect(result.exitCode).toBe(0);
    expect(result.dmgPath).toBe("/dist/App-1.0-arm64.dmg");
    expect(result.dmgReason).toBe("created");
    expect(result.manifestPath).toBe("/dist/App-1.0-arm64.dmg.manifest.json");
    expect(result.sha256sumsPath).toBe("/dist/SHA256SUMS");
  });

  it("takes snapshots before and after the package stages", async () => {
    const snapshotDist = makeSnapshotWithOneDmg("/dist/App.dmg");
    const runStage = makeHappyRunner();
    const verifyChecksum = happyVerify();
    await runReleaseMacDryRun({ runStage, snapshotDist, verifyChecksum, now: () => 0 });
    // snapshotDist is called twice: before package and after
    expect(snapshotDist).toHaveBeenCalledTimes(2);
  });
});

// ---------------------------------------------------------------------------
// Manifest is called only after verify passes
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - manifest only after verify", () => {
  it("calls manifest with the selected DMG path after verify", async () => {
    const { runStage } = await runSuccessPath();
    const manifestCall = runStage.mock.calls.find((c) => c[1][0] === "release:mac:manifest");
    expect(manifestCall).toBeDefined();
    expect(manifestCall[1][1]).toBe("/dist/App-1.0-arm64.dmg");
  });

  it("does NOT call manifest when verify fails", async () => {
    const snapshotDist = makeSnapshotWithOneDmg("/dist/App.dmg");
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 0 }) // electron-builder
      .mockResolvedValueOnce({ code: 1 }); // verify fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("verify");
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:manifest");
  });
});

// ---------------------------------------------------------------------------
// Failure boundaries - fail-fast at each stage
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - build:core failure", () => {
  it("stops at build:core and does not invoke build or package", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn().mockResolvedValueOnce({ code: 2 }); // build:core fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(2);
    expect(result.failedStage).toBe("build:core");
    expect(runStage).toHaveBeenCalledTimes(1);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("build");
    expect(calledCmds).not.toContain("electron-builder");
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacDryRun - build failure", () => {
  it("stops after build:core and does not invoke electron-builder or verify", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core ok
      .mockResolvedValueOnce({ code: 1 }); // build fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("build");
    expect(runStage).toHaveBeenCalledTimes(2);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("electron-builder");
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacDryRun - package-dmg failure", () => {
  it("stops before DMG selection and verify when electron-builder fails", async () => {
    const snapshotDist = vi.fn().mockResolvedValue([]);
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 3 }); // electron-builder fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(3);
    expect(result.failedStage).toBe("package-dmg");
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
    expect(calledCmds).not.toContain("release:mac:manifest");
  });
});

describe("runReleaseMacDryRun - DMG selection failure (no DMG changed)", () => {
  it("exits non-zero without calling verify when no DMG is created/modified", async () => {
    const staleSnapshot = [dmg("/dist/App.dmg", 100, 1000)];
    const snapshotDist = vi.fn().mockResolvedValue(staleSnapshot); // identical before and after
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 0 }); // electron-builder

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("dmg-selection");
    expect(result.dmgError).toMatch(/no .dmg/);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacDryRun - DMG selection failure (multiple changed DMGs)", () => {
  it("exits non-zero without calling verify when multiple DMGs changed", async () => {
    let callCount = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.resolve([]);
      return Promise.resolve([dmg("/dist/A.dmg"), dmg("/dist/B.dmg")]);
    });
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 })
      .mockResolvedValueOnce({ code: 0 })
      .mockResolvedValueOnce({ code: 0 });

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("dmg-selection");
    expect(result.dmgError).toMatch(/multiple/);
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:verify");
  });
});

describe("runReleaseMacDryRun - verify failure", () => {
  it("exits non-zero and stops before manifest when verify fails", async () => {
    const snapshotDist = makeSnapshotWithOneDmg("/dist/App.dmg");
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 0 }) // electron-builder
      .mockResolvedValueOnce({ code: 1 }); // verify fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum: happyVerify(),
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("verify");
    const calledCmds = runStage.mock.calls.map((c) => c[1][0]);
    expect(calledCmds).not.toContain("release:mac:manifest");
  });
});

describe("runReleaseMacDryRun - manifest failure", () => {
  it("exits non-zero and stops before checksum when manifest fails", async () => {
    const snapshotDist = makeSnapshotWithOneDmg("/dist/App.dmg");
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 0 }) // electron-builder
      .mockResolvedValueOnce({ code: 0 }) // verify ok
      .mockResolvedValueOnce({ code: 1 }); // manifest fails
    const verifyChecksum = happyVerify();

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum,
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("manifest");
    expect(verifyChecksum).not.toHaveBeenCalled();
  });
});

describe("runReleaseMacDryRun - checksum failure", () => {
  it("exits non-zero when checksum verification fails", async () => {
    const snapshotDist = makeSnapshotWithOneDmg("/dist/App.dmg");
    const runStage = vi.fn()
      .mockResolvedValueOnce({ code: 0 }) // build:core
      .mockResolvedValueOnce({ code: 0 }) // build
      .mockResolvedValueOnce({ code: 0 }) // electron-builder
      .mockResolvedValueOnce({ code: 0 }) // verify ok
      .mockResolvedValueOnce({ code: 0 }); // manifest ok
    const verifyChecksum = vi.fn().mockResolvedValue({ code: 1 }); // checksum fails

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum,
      now: () => 0,
    });

    expect(result.exitCode).toBe(1);
    expect(result.failedStage).toBe("checksum");
    expect(verifyChecksum).toHaveBeenCalledTimes(1);
    expect(verifyChecksum).toHaveBeenCalledWith("/dist/App.dmg");
  });
});

// ---------------------------------------------------------------------------
// scrubEnv - credential variable removal
// ---------------------------------------------------------------------------

describe("scrubEnv - credential prefix removal", () => {
  it("removes all CSC_ prefixed vars", () => {
    const env = { CSC_LINK: "cert", CSC_KEY_PASSWORD: "pass", CSC_NAME: "id", PATH: "/usr/bin" };
    const result = scrubEnv(env);
    expect(result).not.toHaveProperty("CSC_LINK");
    expect(result).not.toHaveProperty("CSC_KEY_PASSWORD");
    expect(result).not.toHaveProperty("CSC_NAME");
    expect(result).toHaveProperty("PATH", "/usr/bin");
  });

  it("removes all APPLE_ prefixed vars", () => {
    const env = { APPLE_ID: "user@example.com", APPLE_TEAM_ID: "ABC123", APPLE_KEYCHAIN: "kc", HOME: "/Users/dev" };
    const result = scrubEnv(env);
    expect(result).not.toHaveProperty("APPLE_ID");
    expect(result).not.toHaveProperty("APPLE_TEAM_ID");
    expect(result).not.toHaveProperty("APPLE_KEYCHAIN");
    expect(result).toHaveProperty("HOME", "/Users/dev");
  });

  it("removes all NOTARYTOOL_ prefixed vars", () => {
    const env = { NOTARYTOOL_CREDENTIALS: "secret", NOTARYTOOL_PROFILE: "profile", NODE_ENV: "test" };
    const result = scrubEnv(env);
    expect(result).not.toHaveProperty("NOTARYTOOL_CREDENTIALS");
    expect(result).not.toHaveProperty("NOTARYTOOL_PROFILE");
    expect(result).toHaveProperty("NODE_ENV", "test");
  });

  it("preserves PATH and all non-credential vars", () => {
    const env = {
      PATH: "/usr/local/bin:/usr/bin",
      HOME: "/Users/dev",
      NODE_ENV: "development",
      PNPM_HOME: "/usr/local/pnpm",
      CSC_LINK: "x",
      APPLE_ID: "y",
    };
    const result = scrubEnv(env);
    expect(result.PATH).toBe("/usr/local/bin:/usr/bin");
    expect(result.HOME).toBe("/Users/dev");
    expect(result.NODE_ENV).toBe("development");
    expect(result.PNPM_HOME).toBe("/usr/local/pnpm");
    expect(result).not.toHaveProperty("CSC_LINK");
    expect(result).not.toHaveProperty("APPLE_ID");
  });

  it("removes all specific credential vars listed in the spec", () => {
    const credVars = [
      "CSC_LINK", "CSC_KEY_PASSWORD", "CSC_NAME", "CSC_IDENTITY_AUTO_DISCOVERY",
      "CSC_INSTALLER_LINK", "CSC_INSTALLER_KEY_PASSWORD", "CSC_INSTALLER_NAME",
      "APPLE_ID", "APPLE_APP_SPECIFIC_PASSWORD", "APPLE_TEAM_ID", "APPLE_API_KEY",
      "APPLE_API_KEY_ID", "APPLE_API_ISSUER", "APPLE_KEYCHAIN", "APPLE_KEYCHAIN_PROFILE",
      "NOTARYTOOL_PROFILE", "NOTARYTOOL_CREDENTIALS",
    ];
    const env = Object.fromEntries([
      ...credVars.map((k) => [k, "secret"]),
      ["PATH", "/usr/bin"],
    ]);
    const result = scrubEnv(env);
    for (const key of credVars) {
      expect(result).not.toHaveProperty(key);
    }
    expect(result).toHaveProperty("PATH", "/usr/bin");
  });

  it("does not mutate the original env object", () => {
    const env = { CSC_LINK: "cert", PATH: "/usr/bin" };
    scrubEnv(env);
    expect(env).toHaveProperty("CSC_LINK", "cert");
  });
});

// ---------------------------------------------------------------------------
// Selected DMG path propagation
// ---------------------------------------------------------------------------

describe("runReleaseMacDryRun - selected DMG path propagation", () => {
  it("passes the exact selected DMG path to verify, manifest, and verifyChecksum", async () => {
    const dmgPath = "/dist/Astraler Skillbox-0.1.0-arm64.dmg";
    const snapshotDist = makeSnapshotWithOneDmg(dmgPath);
    const runStage = makeHappyRunner({
      "release:mac:manifest": {
        code: 0,
        manifestPath: `${dmgPath}.manifest.json`,
        sha256sumsPath: "/dist/SHA256SUMS",
      },
    });
    const verifyChecksum = happyVerify();

    await runReleaseMacDryRun({ runStage, snapshotDist, verifyChecksum, now: () => 0 });

    const verifyCall = runStage.mock.calls.find((c) => c[1][0] === "release:mac:verify");
    expect(verifyCall[1]).toContain(dmgPath);

    const manifestCall = runStage.mock.calls.find((c) => c[1][0] === "release:mac:manifest");
    expect(manifestCall[1]).toContain(dmgPath);

    expect(verifyChecksum).toHaveBeenCalledWith(dmgPath);
  });

  it("works with a same-name overwrite (modified DMG)", async () => {
    let calls = 0;
    const snapshotDist = vi.fn().mockImplementation(() => {
      calls++;
      if (calls === 1) return Promise.resolve([dmg("/dist/App.dmg", 100, 1000)]);
      return Promise.resolve([dmg("/dist/App.dmg", 999, 9999)]); // overwritten
    });
    const runStage = makeHappyRunner();
    const verifyChecksum = happyVerify();

    const result = await runReleaseMacDryRun({
      runStage,
      snapshotDist,
      verifyChecksum,
      now: () => 0,
    });

    expect(result.exitCode).toBe(0);
    expect(result.dmgPath).toBe("/dist/App.dmg");
    expect(result.dmgReason).toBe("modified");
  });
});
