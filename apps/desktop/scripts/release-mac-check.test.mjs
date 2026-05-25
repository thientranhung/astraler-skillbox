import { describe, it, expect } from "vitest";
import {
  isSet,
  checkPlatform,
  checkTooling,
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
