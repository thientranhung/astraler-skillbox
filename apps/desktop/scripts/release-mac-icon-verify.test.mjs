import { describe, it, expect } from "vitest";
import {
  DEFAULT_ICON_FILE,
  DEFAULT_ICON_SHA256,
  resolveIconResource,
  assertIconFacts,
} from "./release-mac-icon-verify.lib.mjs";

describe("release-mac-icon-verify.lib", () => {
  const ok = {
    iconFile: "icon.icns",
    resourceExists: true,
    sha256: "0000000000000000000000000000000000000000000000000000000000000000",
    fileType: 'Mac OS X icon, 123456 bytes, "s8mk" type',
  };

  it("constants pin the known default Electron icon anchors", () => {
    expect(DEFAULT_ICON_FILE).toBe("electron.icns");
    expect(DEFAULT_ICON_SHA256).toBe("5a9a78d54c157f55672afea37037464858a87fd5f276fc8206787f366ed684cf");
  });

  it("resolveIconResource joins app/Contents/Resources/<iconFile>", () => {
    expect(resolveIconResource("/x/My App.app", "icon.icns")).toBe(
      "/x/My App.app/Contents/Resources/icon.icns"
    );
  });

  it("assertIconFacts passes for a customized, valid icns", () => {
    expect(() => assertIconFacts(ok)).not.toThrow();
  });

  it("assertIconFacts fails when CFBundleIconFile is the default electron.icns", () => {
    expect(() => assertIconFacts({ ...ok, iconFile: "electron.icns" })).toThrow(/default|electron\.icns/i);
  });

  it("assertIconFacts fails when CFBundleIconFile is unset", () => {
    expect(() => assertIconFacts({ ...ok, iconFile: "" })).toThrow(/CFBundleIconFile/i);
  });

  it("assertIconFacts fails when the icon resource is missing", () => {
    expect(() => assertIconFacts({ ...ok, resourceExists: false })).toThrow(/missing/i);
  });

  it("assertIconFacts fails when bytes equal the default Electron icon", () => {
    expect(() => assertIconFacts({ ...ok, sha256: DEFAULT_ICON_SHA256 })).toThrow(/identical|default/i);
  });

  it("assertIconFacts fails when the resource is not a valid icns", () => {
    expect(() => assertIconFacts({ ...ok, fileType: "PNG image data" })).toThrow(/valid|icns/i);
  });

  it("assertIconFacts fails when resource exists but fileType is null (file command failed)", () => {
    expect(() => assertIconFacts({ ...ok, fileType: null })).toThrow(/could not determine|file type/i);
  });

  it("assertIconFacts fails when resource exists but fileType is empty string", () => {
    expect(() => assertIconFacts({ ...ok, fileType: "" })).toThrow(/could not determine|file type/i);
  });
});
