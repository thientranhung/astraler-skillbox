import { describe, it, expect } from "vitest";
import {
  isSet,
  checkPlatform,
  checkTooling,
  checkSigning,
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
