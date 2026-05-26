/**
 * Deterministic macOS icon generator (Slice 3I).
 * build/icon.svg --resvg--> build/.gen/icon-1024.png --sips--> build/.gen/icon.iconset/*
 *   --iconutil--> build/icon.icns
 *
 * The .gen directory and build/icon.icns are untracked build artifacts.
 * Run from apps/desktop/:  pnpm generate:icon
 */
import { Resvg } from "@resvg/resvg-js";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, rmSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  ICONSET_ENTRIES,
  MASTER_PX,
  sipsResizeArgs,
  iconutilArgs,
  assertSvgUsable,
} from "./generate-icon.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, ".."); // apps/desktop
const buildDir = path.join(desktop, "build");
const svgPath = path.join(buildDir, "icon.svg");
const genDir = path.join(buildDir, ".gen");
const iconsetDir = path.join(genDir, "icon.iconset");
const masterPng = path.join(genDir, `icon-${MASTER_PX}.png`);
const outIcns = path.join(buildDir, "icon.icns");

if (!existsSync(svgPath)) {
  console.error(`ERROR: icon source not found: ${path.relative(desktop, svgPath)}`);
  process.exit(1);
}

const svg = readFileSync(svgPath, "utf8");
assertSvgUsable(svg);

// Fresh output each run -> deterministic, no stale reps.
rmSync(genDir, { recursive: true, force: true });
mkdirSync(iconsetDir, { recursive: true });

// 1) SVG -> master PNG (deterministic; shapes-only source, pinned resvg version).
const resvg = new Resvg(svg, { fitTo: { mode: "width", value: MASTER_PX } });
writeFileSync(masterPng, resvg.render().asPng());

// 2) master PNG -> 10 iconset reps via sips (downscale only).
for (const { name, px } of ICONSET_ENTRIES) {
  execFileSync("/usr/bin/sips", sipsResizeArgs(masterPng, px, path.join(iconsetDir, name)), {
    stdio: "ignore",
  });
}

// 3) iconset -> .icns
execFileSync("/usr/bin/iconutil", iconutilArgs(iconsetDir, outIcns), { stdio: "inherit" });

console.log(`generated ${path.relative(desktop, outIcns)} from ${path.relative(desktop, svgPath)}`);
