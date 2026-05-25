import { describe, it, expect } from "vitest";
import {
  parseCodesign,
  parseSpctl,
  parseStapler,
  parseEntitlementKeys,
  pickTopLevelApp,
  discoverDmg,
} from "./release-mac-verify.parse.mjs";

const DEVID_APP = `Identifier=com.astraler.skillbox
Format=app bundle with Mach-O thin (arm64)
CodeDirectory v=20500 size=1234 flags=0x10000(runtime) hashes=10+7
Authority=Developer ID Application: Astraler Inc (AB12CD34EF)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
TeamIdentifier=AB12CD34EF`;

const ADHOC_APP = `Identifier=com.astraler.skillbox
Format=app bundle with Mach-O thin (arm64)
CodeDirectory v=20400 size=1234 flags=0x10002(adhoc,runtime) hashes=10+7
Signature=adhoc
TeamIdentifier=not set`;

const NO_RUNTIME_APP = `Identifier=com.astraler.skillbox
CodeDirectory v=20400 size=1234 flags=0x0(none) hashes=10+7
Authority=Developer ID Application: Astraler Inc (AB12CD34EF)
TeamIdentifier=AB12CD34EF`;

const SPCTL_ACCEPTED = `/Volumes/x/Astraler Skillbox.app: accepted
source=Notarized Developer ID
origin=Developer ID Application: Astraler Inc (AB12CD34EF)`;

const SPCTL_REJECTED = `/Volumes/x/Astraler Skillbox.app: rejected
source=no usable signature`;

const ENT_XML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
\t<key>com.apple.security.cs.allow-jit</key>
\t<true/>
\t<key>com.apple.security.cs.allow-unsigned-executable-memory</key>
\t<true/>
</dict>
</plist>`;

describe("parseCodesign", () => {
  it("classifies a Developer ID signature", () => {
    expect(parseCodesign(DEVID_APP)).toEqual({
      adhoc: false,
      developerId: true,
      teamId: "AB12CD34EF",
      hardenedRuntime: true,
    });
  });
  it("classifies an ad-hoc signature with no team id", () => {
    expect(parseCodesign(ADHOC_APP)).toEqual({
      adhoc: true,
      developerId: false,
      teamId: null,
      hardenedRuntime: true,
    });
  });
  it("detects a missing hardened runtime", () => {
    expect(parseCodesign(NO_RUNTIME_APP).hardenedRuntime).toBe(false);
  });
});

describe("parseSpctl", () => {
  it("accepted + notarized source", () => {
    expect(parseSpctl(SPCTL_ACCEPTED, 0)).toEqual({ accepted: true, source: "Notarized Developer ID" });
  });
  it("rejected", () => {
    expect(parseSpctl(SPCTL_REJECTED, 3)).toEqual({ accepted: false, source: "no usable signature" });
  });
});

describe("parseStapler", () => {
  it("stapled when exit 0", () => {
    expect(parseStapler("The validate action worked!", 0)).toEqual({ stapled: true });
  });
  it("not stapled when non-zero", () => {
    expect(parseStapler("does not have a ticket stapled to it.", 65)).toEqual({ stapled: false });
  });
});

describe("parseEntitlementKeys", () => {
  it("extracts <key> names from an entitlements plist/XML blob", () => {
    expect(parseEntitlementKeys(ENT_XML)).toEqual([
      "com.apple.security.cs.allow-jit",
      "com.apple.security.cs.allow-unsigned-executable-memory",
    ]);
  });
  it("returns [] for an empty/unsigned blob", () => {
    expect(parseEntitlementKeys("")).toEqual([]);
  });
});

describe("pickTopLevelApp", () => {
  it("picks the single top-level app, ignoring non-app entries", () => {
    expect(pickTopLevelApp(["Astraler Skillbox.app", "Applications", ".background"])).toEqual({
      app: "Astraler Skillbox.app",
    });
  });
  it("errors on zero apps", () => {
    expect(pickTopLevelApp(["Applications"]).error).toMatch(/no top-level/i);
  });
  it("errors on multiple top-level apps", () => {
    const r = pickTopLevelApp(["A.app", "B.app"]);
    expect(r.error).toMatch(/multiple top-level/i);
  });
});

describe("discoverDmg", () => {
  it("picks the single dist dmg", () => {
    expect(discoverDmg(["Astraler Skillbox-0.1.0-arm64.dmg", "mac-arm64"])).toEqual({
      dmg: "Astraler Skillbox-0.1.0-arm64.dmg",
    });
  });
  it("errors on zero dmgs", () => {
    expect(discoverDmg(["mac-arm64"]).error).toMatch(/no \.dmg/i);
  });
  it("errors on multiple dmgs", () => {
    expect(discoverDmg(["a.dmg", "b.dmg"]).error).toMatch(/multiple \.dmg/i);
  });
});

// ---- Evaluator + Render tests (Task 2) ----

import { evaluate, render } from "./release-mac-verify.lib.mjs";

const EXPECTED_ENT = {
  app: ["com.apple.security.cs.allow-jit", "com.apple.security.cs.allow-unsigned-executable-memory"],
  sidecar: ["com.apple.security.cs.allow-jit", "com.apple.security.inherit"],
};

/** Build a signals object with sensible "good release" defaults, overridable per test. */
function signals(over = {}) {
  const base = {
    mode: "release",
    expectedTeamId: null,
    expectedEntitlements: EXPECTED_ENT,
    input: { dmgName: "Astraler Skillbox-0.1.0-arm64.dmg", appName: "Astraler Skillbox.app" },
    app: {
      parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true },
      verifyExit: 0,
      entitlementKeys: EXPECTED_ENT.app,
    },
    sidecar: {
      present: true,
      parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true },
      verifyExit: 0,
      entitlementKeys: EXPECTED_ENT.sidecar,
    },
    spctlApp: { accepted: true, source: "Notarized Developer ID" },
    spctlDmg: { accepted: true, source: "Notarized Developer ID" },
    staplerApp: { stapled: true },
    staplerDmg: { stapled: true },
  };
  return { ...base, ...over };
}
const status = (results, id) => results.find((r) => r.id === id)?.status;

describe("evaluate — release mode, fully customer-ready", () => {
  it("all PASS, exit 0", () => {
    const { results, exitCode } = evaluate(signals());
    expect(exitCode).toBe(0);
    expect(results.every((r) => r.status === "PASS")).toBe(true);
  });
});

describe("evaluate — release mode, ad-hoc artifact", () => {
  const adhoc = signals({
    app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
    sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
    spctlApp: { accepted: false, source: "no usable signature" },
    spctlDmg: { accepted: false, source: "no usable signature" },
    staplerApp: { stapled: false },
    staplerDmg: { stapled: false },
  });
  it("FAILs signature/team/gatekeeper/staple, exit 1; entitlements still PASS", () => {
    const { results, exitCode } = evaluate(adhoc);
    expect(exitCode).toBe(1);
    expect(status(results, "APP2")).toBe("FAIL");
    expect(status(results, "SID3")).toBe("FAIL");
    expect(status(results, "TID1")).toBe("FAIL");
    expect(status(results, "GK1")).toBe("FAIL");
    expect(status(results, "GK2")).toBe("FAIL");
    expect(status(results, "ST1")).toBe("FAIL");
    expect(status(results, "ST2")).toBe("FAIL");
    expect(status(results, "ENT1")).toBe("PASS");
    expect(status(results, "ENT2")).toBe("PASS");
  });
});

describe("evaluate — --allow-adhoc, ad-hoc artifact", () => {
  const adhoc = signals({
    mode: "adhoc",
    app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
    sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
    spctlApp: { accepted: false, source: null },
    spctlDmg: { accepted: false, source: null },
    staplerApp: { stapled: false },
    staplerDmg: { stapled: false },
  });
  it("PASSes signature-class/verify/runtime/entitlements; soft checks INFO; exit 0", () => {
    const { results, exitCode } = evaluate(adhoc);
    expect(exitCode).toBe(0);
    // APP2/SID3 PASS in --allow-adhoc: ad-hoc is an accepted signature class (spec §6).
    for (const id of ["APP1", "APP2", "APP3", "SID2", "SID3", "SID4", "ENT1", "ENT2"]) expect(status(results, id)).toBe("PASS");
    for (const id of ["GK1", "GK2", "ST1", "ST2", "APP4", "SID5", "TID1"]) expect(status(results, id)).toBe("INFO");
  });
});

describe("evaluate — sidecar + runtime + entitlement failures", () => {
  it("missing sidecar FAILs SID1 in release", () => {
    const r = evaluate(signals({ sidecar: { present: false, parsed: null, verifyExit: null, entitlementKeys: [] } }));
    expect(status(r.results, "SID1")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
  it("missing app runtime FAILs APP3 in --allow-adhoc too", () => {
    const r = evaluate(signals({ mode: "adhoc", app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: false }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app } }));
    expect(status(r.results, "APP3")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
  it("app entitlements missing a key FAILs ENT1 in both modes", () => {
    const r = evaluate(signals({ app: { parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: ["com.apple.security.cs.allow-jit"] } }));
    expect(status(r.results, "ENT1")).toBe("FAIL");
  });
  it("extra entitlement keys still PASS (subset semantics)", () => {
    const r = evaluate(signals({ app: { parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: [...EXPECTED_ENT.app, "com.apple.security.network.client"] } }));
    expect(status(r.results, "ENT1")).toBe("PASS");
  });
  it("empty expected app entitlements FAIL ENT1 (no vacuous pass) — finding 3", () => {
    const r = evaluate(signals({ expectedEntitlements: { app: [], sidecar: EXPECTED_ENT.sidecar } }));
    expect(status(r.results, "ENT1")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
  it("empty expected sidecar entitlements FAIL ENT2 — finding 3", () => {
    const r = evaluate(signals({ expectedEntitlements: { app: EXPECTED_ENT.app, sidecar: [] } }));
    expect(status(r.results, "ENT2")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
});

describe("evaluate — Team ID equality (finding 2)", () => {
  it("FAILs TID1 when app and sidecar teams differ", () => {
    const r = evaluate(signals({ sidecar: { present: true, parsed: { adhoc: false, developerId: true, teamId: "ZZ99ZZ99ZZ", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar } }));
    expect(status(r.results, "TID1")).toBe("FAIL");
    expect(status(r.results, "APP4")).toBe("PASS");
    expect(status(r.results, "SID5")).toBe("PASS");
  });
  it("PASSes TID1 when equal and no expected env", () => {
    expect(status(evaluate(signals()).results, "TID1")).toBe("PASS");
  });
  it("PASSes TID1 when both match expected env", () => {
    expect(status(evaluate(signals({ expectedTeamId: "AB12CD34EF" })).results, "TID1")).toBe("PASS");
  });
  it("FAILs TID1 when expected env differs", () => {
    expect(status(evaluate(signals({ expectedTeamId: "WRONG12345" })).results, "TID1")).toBe("FAIL");
  });
});

describe("evaluate — DMG-only checks (findings 3, input)", () => {
  it("FAILs GK2 when DMG Gatekeeper rejects", () => {
    const r = evaluate(signals({ spctlDmg: { accepted: false, source: "no usable signature" } }));
    expect(status(r.results, "GK2")).toBe("FAIL");
  });
  it("bare .app input → GK2 and ST2 INFO", () => {
    const r = evaluate(signals({ input: { dmgName: null, appName: "Astraler Skillbox.app" }, spctlDmg: null, staplerDmg: null }));
    expect(status(r.results, "GK2")).toBe("INFO");
    expect(status(r.results, "ST2")).toBe("INFO");
  });
});

describe("render", () => {
  it("groups by category and lists the missing remediations", () => {
    const { results, missing } = evaluate(signals({
      app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
      sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
      spctlApp: { accepted: false, source: null }, spctlDmg: { accepted: false, source: null },
      staplerApp: { stapled: false }, staplerDmg: { stapled: false },
    }));
    const out = render(results, missing);
    expect(out).toMatch(/App signature/);
    expect(out).toMatch(/Sidecar \(core\/skillbox-core\)/);
    expect(out).toMatch(/Missing for a customer-ready release:/);
  });
});
