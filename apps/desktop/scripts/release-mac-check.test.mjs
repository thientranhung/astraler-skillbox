import { describe, it, expect } from "vitest";
import {
  isSet,
  checkPlatform,
  checkTooling,
  checkSigning,
  checkNotarization,
} from "./release-mac-check.lib.mjs";

describe("isSet", () => {
  it("treats non-empty trimmed strings as set", () => {
    expect(isSet("x")).toBe(true);
    expect(isSet("  ")).toBe(false);
    expect(isSet("")).toBe(false);
    expect(isSet(undefined)).toBe(false);
  });
});

describe("checkPlatform", () => {
  it("passes on darwin", () => {
    expect(checkPlatform("darwin").status).toBe("PASS");
  });
  it("fails off darwin", () => {
    const r = checkPlatform("linux");
    expect(r.status).toBe("FAIL");
    expect(r.remediation).toBeTruthy();
  });
});

describe("checkTooling", () => {
  const all = { notarytool: true, stapler: true, codesign: true, spctl: true, plutil: true };
  it("passes when every tool is present", () => {
    const rows = checkTooling(all);
    expect(rows.every((r) => r.status === "PASS")).toBe(true);
  });
  it("fails the missing tool only", () => {
    const rows = checkTooling({ ...all, notarytool: false });
    const fail = rows.filter((r) => r.status === "FAIL");
    expect(fail).toHaveLength(1);
    expect(fail[0].message).toMatch(/notarytool/);
  });
});

describe("checkSigning (B1)", () => {
  const none = { identityNames: [], env: {}, fileProbes: { cscLink: null, appleApiKey: null } };

  it("fails with neither keychain identity nor CSC env", () => {
    const r = checkSigning(none);
    expect(r.status).toBe("FAIL");
    expect(r.remediation).toMatch(/Developer ID Application/);
  });

  it("passes Path A on a keychain identity", () => {
    expect(checkSigning({ ...none, identityNames: ["Developer ID Application: Acme (TEAMID)"] }).status).toBe("PASS");
  });

  it("passes Path B with CSC_LINK local path + password", () => {
    const r = checkSigning({
      identityNames: [],
      env: { CSC_LINK: "/SENTINEL/cert.p12", CSC_KEY_PASSWORD: "pw" },
      fileProbes: { cscLink: { isLocalPath: true, exists: true, readable: true }, appleApiKey: null },
    });
    expect(r.status).toBe("PASS");
  });

  it("passes Path B with a URL/base64 CSC_LINK (no fetch/decode)", () => {
    const r = checkSigning({
      identityNames: [],
      env: { CSC_LINK: "https://example/cert.p12", CSC_KEY_PASSWORD: "pw" },
      fileProbes: { cscLink: { isLocalPath: false, exists: false, readable: false }, appleApiKey: null },
    });
    expect(r.status).toBe("PASS");
  });

  it("fails when local CSC_LINK is missing/unreadable and no keychain", () => {
    const r = checkSigning({
      identityNames: [],
      env: { CSC_LINK: "/SENTINEL/cert.p12", CSC_KEY_PASSWORD: "pw" },
      fileProbes: { cscLink: { isLocalPath: true, exists: false, readable: false }, appleApiKey: null },
    });
    expect(r.status).toBe("FAIL");
    expect(r.message).toMatch(/missing or unreadable/);
  });

  it("fails naming the missing var when CSC_LINK set but password missing", () => {
    const r = checkSigning({
      identityNames: [],
      env: { CSC_LINK: "/SENTINEL/cert.p12" },
      fileProbes: { cscLink: { isLocalPath: true, exists: true, readable: true }, appleApiKey: null },
    });
    expect(r.status).toBe("FAIL");
    expect(r.message).toMatch(/CSC_KEY_PASSWORD is missing/);
  });

  it("passes when both keychain identity and CSC env are present", () => {
    const r = checkSigning({
      identityNames: ["Developer ID Application: Acme (TEAMID)"],
      env: { CSC_LINK: "/SENTINEL/cert.p12", CSC_KEY_PASSWORD: "pw" },
      fileProbes: { cscLink: { isLocalPath: true, exists: true, readable: true }, appleApiKey: null },
    });
    expect(r.status).toBe("PASS");
  });
});

describe("checkNotarization (C1)", () => {
  const probesOk = { cscLink: null, appleApiKey: { exists: true, readable: true } };
  const g1 = { APPLE_API_KEY: "/SENTINEL/key.p8", APPLE_API_KEY_ID: "KID", APPLE_API_ISSUER: "ISS" };
  const g2 = { APPLE_ID: "a@b.c", APPLE_APP_SPECIFIC_PASSWORD: "pw", APPLE_TEAM_ID: "TEAMID" };
  const c1 = (rows) => rows.find((r) => r.id === "C1");

  it("fails with no group", () => {
    expect(c1(checkNotarization({}, { cscLink: null, appleApiKey: null })).status).toBe("FAIL");
  });
  it("fails a partial Group 1, naming the missing var", () => {
    const rows = checkNotarization({ APPLE_API_KEY: "/x.p8", APPLE_API_KEY_ID: "KID" }, probesOk);
    const r = c1(rows);
    expect(r.status).toBe("FAIL");
    expect(r.message).toMatch(/APPLE_API_ISSUER/);
  });
  it("fails Group 1 when the .p8 file is missing/unreadable", () => {
    const rows = checkNotarization(g1, { cscLink: null, appleApiKey: { exists: false, readable: false } });
    expect(c1(rows).status).toBe("FAIL");
    expect(c1(rows).message).toMatch(/\.p8 file is missing or unreadable/);
  });
  it("surfaces a bad APPLE_API_KEY .p8 even when other Group 1 vars are missing (no path printed)", () => {
    const rows = checkNotarization(
      { APPLE_API_KEY: "/SENTINEL/key.p8" }, // only the key path set, and it is bad
      { cscLink: null, appleApiKey: { exists: false, readable: false } }
    );
    const r = c1(rows);
    expect(r.status).toBe("FAIL");
    expect(r.message).toMatch(/\.p8 file is missing or unreadable/);
    expect(r.message).toMatch(/also missing APPLE_API_KEY_ID, APPLE_API_ISSUER/);
    expect(r.message).not.toContain("/SENTINEL/key.p8");
  });
  it("passes complete Group 1", () => {
    expect(c1(checkNotarization(g1, probesOk)).status).toBe("PASS");
  });
  it("passes complete Group 2", () => {
    expect(c1(checkNotarization(g2, { cscLink: null, appleApiKey: null })).status).toBe("PASS");
  });
  it("passes with a WARN (never FAIL) when both groups are complete", () => {
    const rows = checkNotarization({ ...g1, ...g2 }, probesOk);
    expect(c1(rows).status).toBe("PASS");
    expect(rows.some((r) => r.status === "WARN" && /preferred/.test(r.message))).toBe(true);
    expect(rows.some((r) => r.status === "FAIL")).toBe(false);
  });
  it("emits INFO and still FAILs for keychain-profile-only", () => {
    const rows = checkNotarization({ APPLE_KEYCHAIN_PROFILE: "prof" }, { cscLink: null, appleApiKey: null });
    expect(c1(rows).status).toBe("FAIL");
    expect(rows.some((r) => r.status === "INFO" && /APPLE_KEYCHAIN_PROFILE/.test(r.message))).toBe(true);
  });
});
