import { describe, it, expect } from "vitest";
import {
  ICONSET_ENTRIES,
  MASTER_PX,
  sipsResizeArgs,
  iconutilArgs,
  assertSvgUsable,
} from "./generate-icon.lib.mjs";

describe("generate-icon.lib", () => {
  it("ICONSET_ENTRIES has the 10 standard iconset entries with exact names and pixel sizes", () => {
    expect(ICONSET_ENTRIES).toEqual([
      { name: "icon_16x16.png", px: 16 },
      { name: "icon_16x16@2x.png", px: 32 },
      { name: "icon_32x32.png", px: 32 },
      { name: "icon_32x32@2x.png", px: 64 },
      { name: "icon_128x128.png", px: 128 },
      { name: "icon_128x128@2x.png", px: 256 },
      { name: "icon_256x256.png", px: 256 },
      { name: "icon_256x256@2x.png", px: 512 },
      { name: "icon_512x512.png", px: 512 },
      { name: "icon_512x512@2x.png", px: 1024 },
    ]);
  });

  it("MASTER_PX is 1024 (largest rep, no upscaling needed)", () => {
    expect(MASTER_PX).toBe(1024);
    expect(Math.max(...ICONSET_ENTRIES.map((e) => e.px))).toBe(MASTER_PX);
  });

  it("sipsResizeArgs builds a square -z resize with --out", () => {
    expect(sipsResizeArgs("/tmp/master.png", 128, "/tmp/out/icon_128x128.png")).toEqual([
      "-z", "128", "128", "/tmp/master.png", "--out", "/tmp/out/icon_128x128.png",
    ]);
  });

  it("iconutilArgs builds an icns pack command", () => {
    expect(iconutilArgs("/tmp/icon.iconset", "/tmp/icon.icns")).toEqual([
      "-c", "icns", "/tmp/icon.iconset", "-o", "/tmp/icon.icns",
    ]);
  });

  it("assertSvgUsable accepts a minimal valid svg", () => {
    expect(() => assertSvgUsable('<svg xmlns="http://www.w3.org/2000/svg"></svg>')).not.toThrow();
  });

  it("assertSvgUsable throws on empty / non-svg content", () => {
    expect(() => assertSvgUsable("")).toThrow(/empty|svg/i);
    expect(() => assertSvgUsable("not xml")).toThrow(/svg/i);
    expect(() => assertSvgUsable("<svg>")).toThrow(/svg/i); // missing closing tag
  });

  it("assertSvgUsable throws when font/text elements are present (non-deterministic raster)", () => {
    expect(() => assertSvgUsable("<svg><text>x</text></svg>")).toThrow(/text|font/i);
    expect(() => assertSvgUsable("<svg><tspan>x</tspan></svg>")).toThrow(/text|font/i);
  });
});
