/**
 * Pure helpers for the deterministic macOS icon pipeline (Slice 3I).
 * No I/O, no child_process. All side-effecting code lives in generate-icon.mjs.
 */

/** Largest iconset rep; the master PNG is rendered at this width so sips only downscales. */
export const MASTER_PX = 1024;

/** The 10 standard .iconset entries. Filenames are mandated by iconutil. */
export const ICONSET_ENTRIES = [
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
];

/** @param {string} srcPng @param {number} px @param {string} outPng */
export function sipsResizeArgs(srcPng, px, outPng) {
  return ["-z", String(px), String(px), srcPng, "--out", outPng];
}

/** @param {string} iconsetDir @param {string} outIcns */
export function iconutilArgs(iconsetDir, outIcns) {
  return ["-c", "icns", iconsetDir, "-o", outIcns];
}

/**
 * Throw if the SVG cannot be fed to the generator. Guards the font-free invariant
 * (text/tspan rasterize differently across machines) and basic well-formedness.
 * @param {string} text
 */
export function assertSvgUsable(text) {
  if (typeof text !== "string" || text.trim() === "") {
    throw new Error("icon source is empty");
  }
  if (!text.includes("<svg") || !text.includes("</svg>")) {
    throw new Error("icon source is not an svg (missing <svg>…</svg>)");
  }
  if (/<text[\s>]/.test(text) || /<tspan[\s>]/.test(text)) {
    throw new Error("icon source contains <text>/<tspan> (font rendering is non-deterministic); use paths only");
  }
}
