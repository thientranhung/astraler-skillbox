# Slice 3I — Production-Candidate macOS App Icon + Deterministic Icon Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the default Electron app icon with a hand-authored, deterministically generated macOS app icon, gated by a release check (D8) and an offline packaged-app verification command.

**Architecture:** A single committed text source (`apps/desktop/build/icon.svg`) is rasterized at build time by `@resvg/resvg-js`, downscaled to the 10 standard iconset PNGs via `sips`, and packed into `build/icon.icns` via `iconutil`. The `.icns` and intermediate PNGs are untracked build artifacts. `electron-builder` consumes `mac.icon: build/icon.icns`. The preflight (`release:mac:check`) adds D8 to assert icon config/source/artifact state; a standalone `release:mac:icon-verify` inspects the packaged `.app` (CFBundleIconFile, resource presence, bytes ≠ default, valid `.icns`) with no Apple credentials.

**Tech Stack:** Node ESM scripts (`.mjs`), `@resvg/resvg-js@2.6.2`, macOS `sips`/`iconutil`/`plutil`/`file`, `electron-builder`, Vitest (all `scripts/*.test.mjs` use `describe/it/expect`, per `vitest.config.ts`), pnpm.

---

## Grounded facts (verified against the repo on 2026-05-26)

- Current bundle (`dist/mac-arm64/Astraler Skillbox.app`): `CFBundleIconFile = electron.icns`, only `Contents/Resources/electron.icns` present, **default SHA-256 `5a9a78d54c157f55672afea37037464858a87fd5f276fc8206787f366ed684cf`**, `file` reports `Mac OS X icon`. This is the anchor used by D8 / icon-verify to detect "still default".
- `electron-builder.yml` has **no `mac.icon`** today. Setting `mac.icon: build/icon.icns` makes electron-builder copy it to `Contents/Resources/icon.icns` and set `CFBundleIconFile = icon.icns` (confirm filename empirically on first build — the verifier does not hardcode `icon.icns`, it only asserts ≠ `electron.icns` + bytes ≠ default).
- Tools available offline: `/usr/bin/iconutil`, `/usr/bin/sips`, `/usr/bin/plutil`, `/usr/bin/file`. No `rsvg-convert`/`inkscape` (hence resvg).
- Root `.gitignore` already ignores `apps/desktop/dist/` and `apps/desktop/resources/core/`. There is **no** `apps/desktop/.gitignore`; add icon ignores at repo root.
- `release-mac-check.mjs` gathers facts and delegates verdicts to the pure `release-mac-check.lib.mjs#evaluate`. Config checks live in category `config` (D1–D7). The renderer groups by category in `CATEGORY_ORDER`.
- `release-mac-check.lib.mjs` exports `isSet`, `checkConfig`, `checkBundleMetadata`, `evaluate`, `render`. `evaluate` composes results; add `checkIcon` after `checkBundleMetadata`.
- Existing smoke libs (`release-mac-launch-smoke.lib.mjs`, `release-mac-dmg-smoke.lib.mjs`) are pure (no I/O); their `.mjs` siblings do the spawning. Mirror that split for icon-verify.
- `package.json` scripts of interest: `package:mac`, `package:mac:unsigned`, `build:core`, `build`, the `release:mac:*` family. `release:mac:full` composes preflight → signed package → verify → manifest; **do not change its composition**. `package:mac`/`package:mac:unsigned` gain the icon step in Task 5, while `release:mac:dry-run` gets its own orchestrator stage in Task 5B because it does not call those package scripts.
- **`release:mac:dry-run` does NOT invoke `package:mac:unsigned`.** Its orchestrator (`runReleaseMacDryRun` in `release-mac-dry-run.lib.mjs`) runs stages directly: `build:core` → `build` → `electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false` (via `runStage`) → select dmg → verify → manifest → checksum. With `mac.icon: build/icon.icns` set and no generated `.icns` on a clean checkout, that direct electron-builder call would fail. The dry-run orchestrator must therefore generate the icon itself (Task 5B), independent of the `package:mac` wiring. The `runStage(stage, args)` helper runs `pnpm <args…>` for non-`electron-builder` args, so a `generate:icon` stage needs no change to `runStage`.
- **Test harness:** `apps/desktop/vitest.config.ts` `include` lists `scripts/**/*.test.mjs`, and **every existing `scripts/*.test.mjs` uses Vitest** (`import { describe, it, expect } from "vitest"`). `pnpm test` is `vitest run`. New `scripts/*.test.mjs` files MUST therefore be Vitest too — a `node:test` file under `scripts/` would be collected by `pnpm test` and fail. This supersedes the original draft's `node:test` usage for the new files (see "Lead-review revisions" below).
- Release runbook is repo-root `RELEASE.md` (sections 1–9).

---

## File structure

| File | Responsibility | Action |
| --- | --- | --- |
| `apps/desktop/build/icon.svg` | The only committed icon source (hand-authored, font-free, 1024 viewBox) | Create |
| `apps/desktop/scripts/generate-icon.lib.mjs` | Pure helpers: iconset entry table, sips/iconutil arg builders, `assertSvgUsable` | Create |
| `apps/desktop/scripts/generate-icon.mjs` | I/O orchestration: svg → resvg master PNG → sips iconset → iconutil `.icns` | Create |
| `apps/desktop/scripts/generate-icon.test.mjs` | Unit tests for the lib | Create |
| `apps/desktop/electron-builder.yml` | Add `mac.icon: build/icon.icns` | Modify |
| `apps/desktop/package.json` | `generate:icon` script, wire into `package:mac`/`package:mac:unsigned`, `release:mac:icon-verify`, devDep | Modify |
| `.gitignore` (repo root) | Ignore generated `build/icon.icns` + `build/.gen/` | Modify |
| `apps/desktop/pnpm-lock.yaml` | Lockfile entry for `@resvg/resvg-js@2.6.2` | Modify (via `pnpm install`) |
| `apps/desktop/scripts/release-mac-dry-run.lib.mjs` | Add a `generate-icon` stage (stage 0) to the dry-run orchestrator | Modify |
| `apps/desktop/scripts/release-mac-dry-run.mjs` | Add the `generate-icon` failed-stage message branch | Modify |
| `apps/desktop/scripts/release-mac-dry-run.test.mjs` | Tests: icon generated first; failure short-circuits before build:core | Modify |
| `apps/desktop/scripts/release-mac-check.lib.mjs` | Add pure `checkIcon`; wire into `evaluate` | Modify |
| `apps/desktop/scripts/release-mac-check.test.mjs` | Tests for `checkIcon` / D8 semantics | Modify |
| `apps/desktop/scripts/release-mac-check.mjs` | Gather icon facts (svg present/usable, icns present/valid) | Modify |
| `apps/desktop/scripts/release-mac-icon-verify.lib.mjs` | Pure: default anchors + `assertIconFacts` + path resolvers | Create |
| `apps/desktop/scripts/release-mac-icon-verify.mjs` | I/O: read Info.plist, hash resource, `file` type, assert, report | Create |
| `apps/desktop/scripts/release-mac-icon-verify.test.mjs` | Unit tests for the lib | Create |
| `RELEASE.md` (repo root) | Document icon generation + icon-verify step | Modify |

---

## Lead-review revisions (2026-05-26)

This plan was revised after lead review. Changes from the first draft:

1. **BLOCKER — dry-run did not generate the icon.** Added **Task 5B**: the `release:mac:dry-run` orchestrator gains a `generate-icon` stage (stage 0, before `build:core`), with a failed-stage message branch and Vitest coverage. This guarantees `build/icon.icns` exists before the dry-run's direct `electron-builder` call on a clean checkout. (`release:mac:full` is unaffected: it packages via `package:mac`, already wired in Task 5.)
2. **BLOCKER — wrong test harness for `release-mac-check.test.mjs`.** Task 6's D8 tests are written in **Vitest** (`describe/it/expect`) and run with `pnpm exec vitest run scripts/release-mac-check.test.mjs`, matching that file's existing style.
3. **Harness consistency (resolves finding #3 at the root).** Finding #3 asked to *split* focused test commands (node:test for new files, Vitest for `release-mac-check`). Instead, the new test files (`generate-icon.test.mjs`, `release-mac-icon-verify.test.mjs`) are now **Vitest** as well — because `vitest.config.ts` includes `scripts/**/*.test.mjs` and every existing sibling is Vitest, so `node:test` files would break `pnpm test`. With all three files on Vitest there is no harness mixing, so the focused command is a single Vitest run rather than a split. **This deliberately supersedes the literal "node --test split" in finding #3; flag for the lead if a split is still desired (it would require excluding the new files from `vitest.config.ts`).**
4. **Non-blocking:** lockfile path stated as `apps/desktop/pnpm-lock.yaml`; a no-arg `release:mac:icon-verify` verification step added after dry-run (the script defaults to the staged `.app`); icon verifier remains **`.app`-only for 3I** (no DMG-mount support).

---

## ⛔ Lead-review checkpoint (before any implementation)

- [ ] **Lead review of this plan.** Before Task 1, the implementing engineer pauses for a lead/PM review of:
  - The **art direction** in Task 2 (does the SVG concept match PM intent? colors restrained/high-contrast, no purple/blue one-note, legible at 16px?).
  - The **D8 semantics** (FAIL on missing/wrong `mac.icon`; FAIL on missing/malformed svg; WARN on missing generated `.icns`; FAIL on invalid `.icns`).
  - The **dependency** addition `@resvg/resvg-js@2.6.2`.
  - Confirmation that `release:mac:full` composition stays unchanged (`package:mac` generates the icon transitively) and that preflight does **not** FAIL solely on a missing generated `.icns`.

  Do not proceed until the lead approves. Record approval in the PR/commit description at the first PM checkpoint.

---

## Task 1: Add the `@resvg/resvg-js@2.6.2` dev dependency

**Files:**
- Modify: `apps/desktop/package.json` (devDependencies)
- Modify: `apps/desktop/pnpm-lock.yaml` (generated by install)

- [ ] **Step 1: Add the devDependency entry**

In `apps/desktop/package.json`, inside `"devDependencies"`, add (keep alphabetical ordering with the existing entries):

```json
"@resvg/resvg-js": "2.6.2",
```

- [ ] **Step 2: Install to update the lockfile**

Run: `cd apps/desktop && pnpm install`
Expected: install succeeds; `@resvg/resvg-js@2.6.2` appears in the lockfile; `node_modules/@resvg/resvg-js` exists.

- [ ] **Step 3: Verify the package resolves**

Run: `cd apps/desktop && node -e "import('@resvg/resvg-js').then(m => console.log(typeof m.Resvg))"`
Expected: prints `function`

- [ ] **Step 4: PM checkpoint (commit)** — *checkpoint only; do not run `git add`/`git commit`.*

Proposed message: `build(3i): add @resvg/resvg-js@2.6.2 for deterministic icon rasterization`
Proposed paths: `apps/desktop/package.json`, `apps/desktop/pnpm-lock.yaml`

---

## Task 2: Author the committed SVG source

**Files:**
- Create: `apps/desktop/build/icon.svg`

PM-approved concept: dark graphite rounded-square background with subtle depth; three modular skill tiles; one cyan-to-teal constellation path; one amber accent node. No text/font elements. Legible at 16px, polished at 1024px. Restrained, high-contrast; avoid one-note purple/blue gradients.

- [ ] **Step 1: Create the SVG file**

Write `apps/desktop/build/icon.svg` exactly as below. (Deterministic: fixed coordinates, no `<text>`/`<tspan>`, no external refs, no embedded raster.)

```svg
<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024" fill="none">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="0" y2="1024" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#2a2f37"/>
      <stop offset="1" stop-color="#171a1f"/>
    </linearGradient>
    <linearGradient id="tile" x1="0" y1="0" x2="0" y2="1" gradientUnits="objectBoundingBox">
      <stop offset="0" stop-color="#3a414c"/>
      <stop offset="1" stop-color="#2c313a"/>
    </linearGradient>
    <linearGradient id="link" x1="324" y1="700" x2="700" y2="372" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#22d3ee"/>
      <stop offset="1" stop-color="#14b8a6"/>
    </linearGradient>
  </defs>

  <!-- rounded-square background with subtle depth -->
  <rect x="96" y="96" width="832" height="832" rx="184" fill="url(#bg)"/>
  <rect x="96.5" y="96.5" width="831" height="831" rx="183.5" fill="none" stroke="#000000" stroke-opacity="0.28" stroke-width="3"/>
  <rect x="99" y="99" width="826" height="826" rx="181" fill="none" stroke="#ffffff" stroke-opacity="0.06" stroke-width="2"/>

  <!-- constellation path: cyan-to-teal -->
  <path d="M324 700 L512 512 L700 372" stroke="url(#link)" stroke-width="26" stroke-linecap="round" stroke-linejoin="round" fill="none"/>

  <!-- three modular skill tiles -->
  <rect x="268" y="644" width="112" height="112" rx="26" fill="url(#tile)" stroke="#4b5563" stroke-opacity="0.55" stroke-width="2"/>
  <rect x="456" y="456" width="112" height="112" rx="26" fill="url(#tile)" stroke="#4b5563" stroke-opacity="0.55" stroke-width="2"/>
  <rect x="644" y="316" width="112" height="112" rx="26" fill="url(#tile)" stroke="#4b5563" stroke-opacity="0.55" stroke-width="2"/>

  <!-- cyan/teal nodes on the lower two tiles -->
  <circle cx="324" cy="700" r="20" fill="#22d3ee"/>
  <circle cx="512" cy="512" r="20" fill="#2dd4bf"/>

  <!-- amber accent node (top of the path) -->
  <circle cx="700" cy="372" r="24" fill="#f59e0b"/>
  <circle cx="700" cy="372" r="24" fill="none" stroke="#fbbf24" stroke-opacity="0.65" stroke-width="3"/>
</svg>
```

- [ ] **Step 2: Sanity-check it is well-formed XML**

Run: `cd apps/desktop && node -e "const s=require('fs').readFileSync('build/icon.svg','utf8'); if(!s.includes('<svg')||!s.includes('</svg>')) throw new Error('bad svg'); if(/<text|<tspan/.test(s)) throw new Error('font element present'); console.log('svg ok', s.length)"`
Expected: prints `svg ok <n>` (no throw)

- [ ] **Step 3: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): add hand-authored app icon SVG source`
Proposed paths: `apps/desktop/build/icon.svg`

---

## Task 3: Icon generation pure library (TDD)

**Files:**
- Create: `apps/desktop/scripts/generate-icon.lib.mjs`
- Test: `apps/desktop/scripts/generate-icon.test.mjs`

The 10 standard iconset entries (exact filenames required by `iconutil`):

| filename | pixels |
| --- | --- |
| `icon_16x16.png` | 16 |
| `icon_16x16@2x.png` | 32 |
| `icon_32x32.png` | 32 |
| `icon_32x32@2x.png` | 64 |
| `icon_128x128.png` | 128 |
| `icon_128x128@2x.png` | 256 |
| `icon_256x256.png` | 256 |
| `icon_256x256@2x.png` | 512 |
| `icon_512x512.png` | 512 |
| `icon_512x512@2x.png` | 1024 |

- [ ] **Step 1: Write the failing test**

Create `apps/desktop/scripts/generate-icon.test.mjs` (Vitest — matches the existing `scripts/*.test.mjs` convention and `vitest.config.ts` include):

```js
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/desktop && pnpm exec vitest run scripts/generate-icon.test.mjs`
Expected: FAIL — cannot resolve `./generate-icon.lib.mjs` (module not found).

- [ ] **Step 3: Implement the library**

Create `apps/desktop/scripts/generate-icon.lib.mjs`:

```js
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
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/desktop && pnpm exec vitest run scripts/generate-icon.test.mjs`
Expected: PASS (all tests green).

- [ ] **Step 5: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): add icon generation pure lib + tests`
Proposed paths: `apps/desktop/scripts/generate-icon.lib.mjs`, `apps/desktop/scripts/generate-icon.test.mjs`

---

## Task 4: Icon generation I/O script

**Files:**
- Create: `apps/desktop/scripts/generate-icon.mjs`

- [ ] **Step 1: Implement the generator**

Create `apps/desktop/scripts/generate-icon.mjs`:

```js
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
```

- [ ] **Step 2: Run the generator**

Run: `cd apps/desktop && node scripts/generate-icon.mjs`
Expected: prints `generated build/icon.icns from build/icon.svg`; exit 0.

- [ ] **Step 3: Verify the artifact is a valid, non-default icns**

Run: `cd apps/desktop && file build/icon.icns && [ -s build/icon.icns ] && shasum -a 256 build/icon.icns`
Expected: `file` reports `Mac OS X icon`; file is non-empty; the SHA-256 is **not** `5a9a78d54c157f55672afea37037464858a87fd5f276fc8206787f366ed684cf` (the default Electron icon).

- [ ] **Step 4: PM checkpoint (commit)** — *checkpoint only.* Note: `build/icon.icns` and `build/.gen/` are untracked artifacts (gitignored in Task 6); nothing to stage here beyond the script.

Proposed message: `feat(3i): add icon generation script (svg -> resvg -> sips -> iconutil)`
Proposed paths: `apps/desktop/scripts/generate-icon.mjs`

---

## Task 5: Wire `mac.icon`, `generate:icon`, and gitignore

**Files:**
- Modify: `apps/desktop/electron-builder.yml`
- Modify: `apps/desktop/package.json` (scripts)
- Modify: `.gitignore` (repo root)

- [ ] **Step 1: Add `mac.icon` to electron-builder config**

In `apps/desktop/electron-builder.yml`, under the `mac:` block, add the `icon` key (place it as the first key under `mac:`, before `category:`):

```yaml
mac:
  icon: build/icon.icns
  category: public.app-category.developer-tools
```

(Leave all other `mac:` keys unchanged.)

- [ ] **Step 2: Add and wire the scripts**

In `apps/desktop/package.json` `"scripts"`, add `generate:icon` and `release:mac:icon-verify`, and prepend `pnpm generate:icon &&` to the two packaging scripts. Final values:

```json
"generate:icon": "node scripts/generate-icon.mjs",
"package:mac": "pnpm generate:icon && pnpm build:core && pnpm build && electron-builder --mac dmg",
"package:mac:unsigned": "pnpm generate:icon && pnpm build:core && pnpm build && CSC_IDENTITY_AUTO_DISCOVERY=false electron-builder --mac dmg -c.mac.identity=null -c.mac.hardenedRuntime=false -c.mac.notarize=false",
"release:mac:icon-verify": "node scripts/release-mac-icon-verify.mjs",
```

(Do not modify `release:mac:full` in this task. `release:mac:full` packages through `package:mac`, so it receives the icon generation step transitively without changing full-release composition. `release:mac:dry-run` is handled separately in Task 5B because it invokes electron-builder directly rather than `package:mac:unsigned`.)

- [ ] **Step 3: Ignore generated icon outputs**

In repo-root `.gitignore`, append:

```
apps/desktop/build/icon.icns
apps/desktop/build/.gen/
```

- [ ] **Step 4: Verify the artifacts are untracked/ignored**

Run: `cd /Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox && git check-ignore apps/desktop/build/icon.icns apps/desktop/build/.gen/icon.iconset/icon_16x16.png`
Expected: both paths echoed back (i.e. ignored).

Run: `git status --porcelain -- apps/desktop/build`
Expected: shows `apps/desktop/build/icon.svg` (Task 2) as tracked/added but **not** `icon.icns` or `.gen/`.

- [ ] **Step 5: Confirm electron-builder still parses**

Run: `cd apps/desktop && node -e "const y=require('js-yaml');const c=y.load(require('fs').readFileSync('electron-builder.yml','utf8'));if(c.mac.icon!=='build/icon.icns')throw new Error('mac.icon wrong: '+c.mac.icon);console.log('mac.icon ok')"`
Expected: prints `mac.icon ok`.

- [ ] **Step 6: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): wire mac.icon + generate:icon into packaging; ignore generated icns`
Proposed paths: `apps/desktop/electron-builder.yml`, `apps/desktop/package.json`, `.gitignore`

---

## Task 5B: Generate the icon during `release:mac:dry-run` (TDD)

`release:mac:dry-run` calls `electron-builder` **directly** through its orchestrator (`runReleaseMacDryRun`), not via `package:mac:unsigned`. So the Task 5 packaging wiring does **not** cover dry-run. With `mac.icon: build/icon.icns` set, a clean-checkout dry-run would fail when electron-builder looks for a not-yet-generated `.icns`. This task adds a `generate-icon` stage as the first stage of the dry-run orchestrator. `release:mac:full` is unaffected (it packages via `package:mac`).

**Files:**
- Modify: `apps/desktop/scripts/release-mac-dry-run.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-dry-run.mjs`
- Test: `apps/desktop/scripts/release-mac-dry-run.test.mjs`

- [ ] **Step 1: Write the failing tests and update existing dry-run expectations**

In `apps/desktop/scripts/release-mac-dry-run.test.mjs`, **reuse the existing imports** at the top of the file:

```js
import { describe, it, expect, vi } from "vitest";
import { runReleaseMacDryRun, scrubEnv } from "./release-mac-dry-run.lib.mjs";
```

Do **not** append duplicate import declarations. Append only this `describe` block near the other `runReleaseMacDryRun` stage-order tests:

```js
describe("runReleaseMacDryRun — generate-icon stage (Slice 3I)", () => {
  function makeDeps(overrides = {}) {
    const calls = [];
    const runStage = vi.fn(async (stage) => {
      calls.push(stage);
      const code = overrides.failStage === stage ? 1 : 0;
      if (stage === "manifest") {
        return { code, manifestPath: "/d/x.dmg.manifest.json", sha256sumsPath: "/d/SHA256SUMS" };
      }
      return { code };
    });
    const snapshotDist = vi
      .fn()
      .mockResolvedValueOnce([])
      .mockResolvedValue([{ path: "/d/x.dmg", size: 10, mtimeMs: 100, isFile: true }]);
    const deps = { runStage, snapshotDist, verifyChecksum: vi.fn(async () => ({ code: 0 })), now: () => 0 };
    return { deps, calls, runStage };
  }

  it("runs generate-icon before build:core on the happy path", async () => {
    const { deps, calls } = makeDeps();
    await runReleaseMacDryRun(deps);
    expect(calls.indexOf("generate-icon")).toBe(0);
    expect(calls.indexOf("generate-icon")).toBeLessThan(calls.indexOf("build:core"));
  });

  it("invokes generate-icon via `pnpm generate:icon` (not electron-builder)", async () => {
    const { deps, runStage } = makeDeps();
    await runReleaseMacDryRun(deps);
    expect(runStage).toHaveBeenCalledWith("generate-icon", ["generate:icon"]);
  });

  it("short-circuits with failedStage 'generate-icon' and never builds when icon generation fails", async () => {
    const { deps, calls } = makeDeps({ failStage: "generate-icon" });
    const result = await runReleaseMacDryRun(deps);
    expect(result.failedStage).toBe("generate-icon");
    expect(result.exitCode).toBe(1);
    expect(calls).toEqual(["generate-icon"]);
    expect(calls).not.toContain("build:core");
  });
});
```

Then update existing dry-run tests that assume `build:core` is the first stage:

- In the success-path order test, expected command order becomes:

```js
expect(calledCmds[0]).toBe("generate:icon");
expect(calledCmds[1]).toBe("build:core");
expect(calledCmds[2]).toBe("build");
expect(calledCmds[3]).toBe("electron-builder");
expect(calledCmds[4]).toBe("release:mac:verify");
expect(calledCmds[5]).toBe("release:mac:manifest");
```

- For explicit `vi.fn().mockResolvedValueOnce(...)` stage sequences, prepend a successful icon-generation result before the existing stage results:

```js
.mockResolvedValueOnce({ code: 0 }) // generate-icon
.mockResolvedValueOnce({ code: 0 }) // build:core
```

Apply that prepend to every existing explicit sequence that starts at `build:core`: manifest failure, build failure, package-dmg failure, DMG selection failures, verify failure, checksum failure, and any same-name/modified DMG path tests. For the existing `build:core failure` test, the first result should be icon success and the second should be `build:core` failure; expected `runStage` calls become 2, with called commands containing `generate:icon` and `build:core` but not `build`.

The helper `makeHappyRunner()` does not need a new branch for `generate:icon`; its default `{ code: 0 }` path is sufficient. `runSuccessPath()` will naturally include the new stage once the orchestrator changes.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-dry-run.test.mjs`
Expected: FAIL — `generate-icon` is never run, so the new tests fail and the updated success-path order still starts at `build:core`.

- [ ] **Step 3: Add the `generate-icon` stage to the orchestrator**

In `apps/desktop/scripts/release-mac-dry-run.lib.mjs`, inside `runReleaseMacDryRun`, insert the stage immediately **before** the `// Stage 1: build Go core` block (i.e. after `const packageStartMs = now();`):

```js
  // Stage 0: generate the macOS app icon (build/icon.icns) before electron-builder.
  // release:mac:dry-run packages via electron-builder directly (not package:mac:unsigned),
  // so it must generate the icon itself; mac.icon: build/icon.icns would otherwise fail on
  // a clean checkout. runStage runs `pnpm generate:icon` (non-electron-builder branch).
  const generateIconResult = await runStage("generate-icon", ["generate:icon"]);
  if (generateIconResult.code !== 0) {
    return { exitCode: generateIconResult.code, failedStage: "generate-icon" };
  }
```

Also update the flow comment in the function's JSDoc to: `Flow: generate-icon -> build:core -> build -> ad-hoc electron-builder -> snapshot after -> select dmg -> verify --allow-adhoc -> manifest -> checksum`.

- [ ] **Step 4: Add the failed-stage message branch in the runner**

In `apps/desktop/scripts/release-mac-dry-run.mjs`, add a branch as the **first** `if` in the result-handling chain, before `if (result.failedStage === "build:core")`:

```js
if (result.failedStage === "generate-icon") {
  process.stderr.write(
    "\n[release:mac:dry-run] STOPPED: generate:icon failed - icon not generated, build not started.\n"
  );
} else if (result.failedStage === "build:core") {
```

(Convert the existing `if (result.failedStage === "build:core")` into the `} else if (...)` shown above; leave the remaining branches unchanged.)

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-dry-run.test.mjs`
Expected: PASS (new generate-icon tests + all pre-existing dry-run tests green).

- [ ] **Step 6: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): generate app icon as first stage of release:mac:dry-run`
Proposed paths: `apps/desktop/scripts/release-mac-dry-run.lib.mjs`, `apps/desktop/scripts/release-mac-dry-run.mjs`, `apps/desktop/scripts/release-mac-dry-run.test.mjs`

---

## Task 6: D8 icon check in the release preflight (TDD)

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.mjs`

D8 is modeled as three `config`-category results so it renders under "electron-builder config":
- `D8` — `mac.icon`: PASS iff exactly `build/icon.icns`; FAIL if unset (default Electron icon) or any other value.
- `D8-source` — `build/icon.svg`: FAIL if missing; FAIL if malformed (generator cannot run); PASS otherwise.
- `D8-artifact` — generated `build/icon.icns`: WARN if absent (package step generates it); FAIL if present but empty/invalid/unreadable; PASS if present and valid.

- [ ] **Step 1: Write the failing test**

Append to `apps/desktop/scripts/release-mac-check.test.mjs`. This file uses **Vitest** (`import { describe, it, expect } from "vitest"` is already at the top) — add `checkIcon` to the existing import from `./release-mac-check.lib.mjs` and append the `describe` block below:

```js
// add to the existing import:
//   import { checkPlatform, /* … */, checkIcon } from "./release-mac-check.lib.mjs";

describe("checkIcon (D8)", () => {
  const goodIcon = { svgPresent: true, svgUsable: true, icnsPresent: true, icnsValid: true };
  const cfg = (icon) => ({ mac: { icon } });

  it("PASS path: mac.icon correct, svg usable, icns valid", () => {
    const r = checkIcon(cfg("build/icon.icns"), goodIcon);
    expect(r.find((x) => x.id === "D8").status).toBe("PASS");
    expect(r.find((x) => x.id === "D8-source").status).toBe("PASS");
    expect(r.find((x) => x.id === "D8-artifact").status).toBe("PASS");
    expect(r.every((x) => x.category === "config")).toBe(true);
  });

  it("D8 FAIL when mac.icon is unset (default Electron icon)", () => {
    const d8 = checkIcon({ mac: {} }, goodIcon).find((x) => x.id === "D8");
    expect(d8.status).toBe("FAIL");
    expect(d8.message).toMatch(/not set|default/i);
    expect(d8.remediation).toContain("build/icon.icns");
  });

  it("D8 FAIL when mac.icon points elsewhere", () => {
    const r = checkIcon(cfg("build/other.icns"), goodIcon);
    expect(r.find((x) => x.id === "D8").status).toBe("FAIL");
  });

  it("D8-source FAIL when svg missing", () => {
    const r = checkIcon(cfg("build/icon.icns"), { ...goodIcon, svgPresent: false });
    expect(r.find((x) => x.id === "D8-source").status).toBe("FAIL");
  });

  it("D8-source FAIL when svg malformed (generator cannot run)", () => {
    const src = checkIcon(cfg("build/icon.icns"), { ...goodIcon, svgUsable: false }).find((x) => x.id === "D8-source");
    expect(src.status).toBe("FAIL");
    expect(src.message).toMatch(/malformed|usable|generator/i);
  });

  it("D8-artifact WARN when generated icns absent on clean checkout", () => {
    const art = checkIcon(cfg("build/icon.icns"), { ...goodIcon, icnsPresent: false, icnsValid: false }).find((x) => x.id === "D8-artifact");
    expect(art.status).toBe("WARN");
    expect(art.message).toMatch(/generate:icon|package/i);
  });

  it("D8-artifact FAIL when icns present but invalid", () => {
    const art = checkIcon(cfg("build/icon.icns"), { ...goodIcon, icnsValid: false }).find((x) => x.id === "D8-artifact");
    expect(art.status).toBe("FAIL");
    expect(art.message).toMatch(/invalid|empty|unreadable/i);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs`
Expected: FAIL — `checkIcon` is not exported / not a function.

- [ ] **Step 3: Implement `checkIcon` and wire it into `evaluate`**

In `apps/desktop/scripts/release-mac-check.lib.mjs`, add the function (place it after `checkBundleMetadata`):

```js
const EXPECTED_ICON = "build/icon.icns";

/**
 * @param {any} config
 * @param {{svgPresent:boolean,svgUsable:boolean,icnsPresent:boolean,icnsValid:boolean}} icon
 */
export function checkIcon(config, icon) {
  const out = [];
  const macIcon = config && config.mac && config.mac.icon;

  // D8 — electron-builder mac.icon config
  if (macIcon === EXPECTED_ICON) {
    out.push({ id: "D8", category: "config", status: "PASS", message: `mac.icon: ${EXPECTED_ICON}` });
  } else if (!isSet(macIcon)) {
    out.push({
      id: "D8",
      category: "config",
      status: "FAIL",
      message: "mac.icon is not set (bundle would ship the default Electron icon)",
      remediation: `Set mac.icon: ${EXPECTED_ICON} in electron-builder.yml`,
    });
  } else {
    out.push({
      id: "D8",
      category: "config",
      status: "FAIL",
      message: `mac.icon is ${JSON.stringify(macIcon)}, expected ${EXPECTED_ICON}`,
      remediation: `Set mac.icon: ${EXPECTED_ICON} in electron-builder.yml`,
    });
  }

  // D8-source — committed SVG source
  if (!icon.svgPresent) {
    out.push({
      id: "D8-source",
      category: "config",
      status: "FAIL",
      message: "icon source build/icon.svg is missing",
      remediation: "Restore build/icon.svg (the only committed icon asset)",
    });
  } else if (!icon.svgUsable) {
    out.push({
      id: "D8-source",
      category: "config",
      status: "FAIL",
      message: "build/icon.svg is malformed; the generator cannot run",
      remediation: "Fix build/icon.svg (well-formed <svg>, no <text>/<tspan>); see pnpm generate:icon",
    });
  } else {
    out.push({ id: "D8-source", category: "config", status: "PASS", message: "icon source build/icon.svg present and usable" });
  }

  // D8-artifact — generated .icns
  if (!icon.icnsPresent) {
    out.push({
      id: "D8-artifact",
      category: "config",
      status: "WARN",
      message: "generated build/icon.icns absent (package:mac runs generate:icon before packaging)",
    });
  } else if (!icon.icnsValid) {
    out.push({
      id: "D8-artifact",
      category: "config",
      status: "FAIL",
      message: "build/icon.icns exists but is empty/invalid/unreadable",
      remediation: "Rerun pnpm generate:icon to regenerate build/icon.icns",
    });
  } else {
    out.push({ id: "D8-artifact", category: "config", status: "PASS", message: "generated build/icon.icns present and valid" });
  }

  return out;
}
```

Then wire it into `evaluate` (insert after the `checkBundleMetadata` spread):

```js
    ...checkBundleMetadata(facts.config),
    ...checkIcon(facts.config, facts.icon),
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs`
Expected: PASS (new D8 tests + all pre-existing tests green).

- [ ] **Step 5: Gather icon facts in the runner**

In `apps/desktop/scripts/release-mac-check.mjs`:

(a) Extend the import from the lib:

```js
import { evaluate, render } from "./release-mac-check.lib.mjs";
import { assertSvgUsable } from "./generate-icon.lib.mjs";
```

(b) After the "Staged sidecar" block and before the "Hygiene" block, add icon fact gathering:

```js
// Icon assets (Slice 3I) — source committed, .icns is a generated artifact.
const iconSvgPath = path.join(desktop, "build", "icon.svg");
const iconIcnsPath = path.join(desktop, "build", "icon.icns");
const svgPresent = existsSync(iconSvgPath);
let svgUsable = false;
if (svgPresent) {
  try {
    assertSvgUsable(readFileSync(iconSvgPath, "utf8"));
    svgUsable = true;
  } catch {
    svgUsable = false;
  }
}
const icnsPresent = existsSync(iconIcnsPath);
let icnsValid = false;
if (icnsPresent) {
  const typeOk = (run("/usr/bin/file", [iconIcnsPath]) ?? "").includes("Mac OS X icon");
  let nonEmpty = false;
  try {
    nonEmpty = statSync(iconIcnsPath).size > 0;
  } catch {
    nonEmpty = false;
  }
  icnsValid = typeOk && nonEmpty && readableFile(iconIcnsPath);
}
const icon = { svgPresent, svgUsable, icnsPresent, icnsValid };
```

(c) Add `icon` to the `facts` object:

```js
const facts = {
  platform: process.platform,
  tools,
  identityNames,
  env,
  fileProbes,
  config,
  entitlements,
  sidecar,
  icon,
  trackedArtifacts,
  trackedSecretFiles,
  version: pkg.version,
};
```

- [ ] **Step 6: Run the preflight end-to-end (artifact present after Task 4)**

Run: `cd apps/desktop && pnpm generate:icon && pnpm release:mac:check; echo "exit=$?"`
Expected: under "electron-builder config", `D8`, `D8-source`, `D8-artifact` all show `PASS`. Overall `exit` may still be `1` if Apple credentials are absent (signing/notarization FAIL) — that is expected and unrelated to D8.

- [ ] **Step 7: Confirm the clean-checkout WARN path**

Run: `cd apps/desktop && rm -f build/icon.icns && pnpm release:mac:check | grep -A1 "icon"; pnpm generate:icon`
Expected: `D8-artifact` shows `WARN` (absent) while `D8` and `D8-source` stay `PASS`. The trailing `generate:icon` restores the artifact.

- [ ] **Step 8: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): add D8 icon checks to release:mac:check`
Proposed paths: `apps/desktop/scripts/release-mac-check.lib.mjs`, `apps/desktop/scripts/release-mac-check.test.mjs`, `apps/desktop/scripts/release-mac-check.mjs`

---

## Task 7: Icon-verify pure library (TDD)

**Files:**
- Create: `apps/desktop/scripts/release-mac-icon-verify.lib.mjs`
- Test: `apps/desktop/scripts/release-mac-icon-verify.test.mjs`

- [ ] **Step 1: Write the failing test**

Create `apps/desktop/scripts/release-mac-icon-verify.test.mjs` (Vitest — matches the existing `scripts/*.test.mjs` convention and `vitest.config.ts` include):

```js
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
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-icon-verify.test.mjs`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the library**

Create `apps/desktop/scripts/release-mac-icon-verify.lib.mjs`:

```js
/**
 * Pure helpers for the packaged-app icon verification (Slice 3I).
 * No I/O, no child_process. All side-effecting code lives in release-mac-icon-verify.mjs.
 */
import path from "node:path";

/** The default Electron app icon — the thing we must NOT ship. */
export const DEFAULT_ICON_FILE = "electron.icns";
/** SHA-256 of the default Electron icon bytes (anchor captured 2026-05-26). */
export const DEFAULT_ICON_SHA256 =
  "5a9a78d54c157f55672afea37037464858a87fd5f276fc8206787f366ed684cf";

/** @param {string} appPath @param {string} iconFile */
export function resolveIconResource(appPath, iconFile) {
  return path.join(appPath, "Contents", "Resources", iconFile);
}

/** @param {string} appPath */
export function resolveInfoPlist(appPath) {
  return path.join(appPath, "Contents", "Info.plist");
}

/**
 * Throw with all collected problems if the packaged icon is missing/default/invalid.
 * @param {{iconFile:string, resourceExists:boolean, sha256:string|null, fileType:string|null}} facts
 */
export function assertIconFacts({ iconFile, resourceExists, sha256, fileType }) {
  const problems = [];

  if (!iconFile || iconFile.trim() === "") {
    problems.push("CFBundleIconFile is not set in Info.plist");
  } else if (iconFile === DEFAULT_ICON_FILE) {
    problems.push(`CFBundleIconFile is the default ${DEFAULT_ICON_FILE} (app icon was not customized)`);
  }

  if (!resourceExists) {
    problems.push(`icon resource Contents/Resources/${iconFile || "<unset>"} is missing`);
  }

  if (sha256 && sha256 === DEFAULT_ICON_SHA256) {
    problems.push("icon bytes are identical to the default Electron icon");
  }

  if (resourceExists && fileType && !/Mac OS X icon/.test(fileType)) {
    problems.push(`icon resource is not a valid .icns (file type: ${fileType})`);
  }

  if (problems.length > 0) {
    const err = new Error("Icon verification failed:\n  - " + problems.join("\n  - "));
    err.problems = problems;
    throw err;
  }
  return { ok: true, iconFile };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-icon-verify.test.mjs`
Expected: PASS.

- [ ] **Step 5: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): add icon-verify pure lib + tests`
Proposed paths: `apps/desktop/scripts/release-mac-icon-verify.lib.mjs`, `apps/desktop/scripts/release-mac-icon-verify.test.mjs`

---

## Task 8: Icon-verify I/O script

**Scope note (3I):** the verifier inspects a staged `.app` bundle **only**. DMG-mount support is explicitly out of scope for 3I — the existing `release:mac:dmg-smoke` already mounts the DMG and boots the copied app, so icon-on-DMG coverage is achieved by running this verifier against `dist/mac-arm64/...app` after a dry-run/package. Do not add `hdiutil`/mount logic here.

**Files:**
- Create: `apps/desktop/scripts/release-mac-icon-verify.mjs`

- [ ] **Step 1: Implement the verifier**

Create `apps/desktop/scripts/release-mac-icon-verify.mjs`:

```js
/**
 * Packaged-app icon verification (Slice 3I).
 * Offline, read-only. No Apple credentials, no network, no keychain.
 * Asserts the .app ships a customized, valid .icns (not the default Electron icon).
 *
 * Usage (from apps/desktop/):
 *   pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"
 *   pnpm release:mac:icon-verify          # defaults to dist/mac-arm64/Astraler Skillbox.app
 */
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  resolveInfoPlist,
  resolveIconResource,
  assertIconFacts,
} from "./release-mac-icon-verify.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");

function run(cmd, args) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return null;
  }
}

const appArg = process.argv[2] || path.join("dist", "mac-arm64", "Astraler Skillbox.app");
const appPath = path.isAbsolute(appArg) ? appArg : path.join(desktop, appArg);

if (!existsSync(appPath)) {
  console.error(`ERROR: app bundle not found: ${appArg}`);
  console.error("Build it first (e.g. pnpm package:mac:unsigned), then pass the .app path.");
  process.exit(1);
}

const infoPlist = resolveInfoPlist(appPath);
if (!existsSync(infoPlist)) {
  console.error(`ERROR: Info.plist not found under ${appArg}/Contents`);
  process.exit(1);
}

const iconFile = run("/usr/bin/plutil", ["-extract", "CFBundleIconFile", "raw", infoPlist]) ?? "";
const resourcePath = iconFile ? resolveIconResource(appPath, iconFile) : null;
const resourceExists = resourcePath ? existsSync(resourcePath) : false;
const sha256 = resourceExists ? createHash("sha256").update(readFileSync(resourcePath)).digest("hex") : null;
const fileType = resourceExists ? run("/usr/bin/file", [resourcePath]) : null;

try {
  assertIconFacts({ iconFile, resourceExists, sha256, fileType });
} catch (err) {
  console.error(err.message);
  process.exit(1);
}

console.log("Icon verification PASS");
console.log(`  CFBundleIconFile: ${iconFile}`);
console.log(`  resource:         Contents/Resources/${iconFile} (present)`);
console.log(`  sha256:           ${sha256}`);
console.log("  bytes differ from the default Electron icon; resource is a valid .icns");
process.exit(0);
```

- [ ] **Step 2: Verify against a packaged app (if one exists from a prior build)**

If `dist/mac-arm64/Astraler Skillbox.app` predates this slice (still default icon), the verifier should correctly FAIL:

Run: `cd apps/desktop && pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"; echo "exit=$?"`
Expected (pre-rebuild bundle): prints `Icon verification failed:` listing the default-icon problems; `exit=1`. (A correct PASS is produced after Task 10 rebuilds the bundle with the new icon.)

- [ ] **Step 3: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `feat(3i): add standalone release:mac:icon-verify`
Proposed paths: `apps/desktop/scripts/release-mac-icon-verify.mjs`

---

## Task 9: Document the icon pipeline in the runbook

**Files:**
- Modify: `RELEASE.md` (repo root)

- [ ] **Step 1: Add an icon-generation note to Prerequisites (§1)**

In `RELEASE.md` section `## 1. Prerequisites`, add a bullet:

```markdown
- App icon: `apps/desktop/build/icon.svg` is the only committed icon source. `pnpm generate:icon`
  produces `build/icon.icns` (untracked) via resvg → sips → iconutil. `package:mac` runs it
  automatically before packaging, so the generated `.icns` need not exist on a clean checkout.
```

- [ ] **Step 2: Add a manual generate command near §3 (Preflight)**

After the preflight description in `## 3. Preflight Check`, add:

```markdown
To regenerate the icon manually (e.g. after editing `build/icon.svg`):

\`\`\`bash
cd apps/desktop
pnpm generate:icon
\`\`\`

`release:mac:check` reports the icon under "electron-builder config":
- `D8` — `mac.icon` is `build/icon.icns`.
- `D8-source` — `build/icon.svg` present and usable.
- `D8-artifact` — generated `build/icon.icns` present/valid (WARN if absent on a clean checkout; the package step generates it).
```

- [ ] **Step 3: Add an icon-verify step to the smoke section (§7)**

In `## 7. No-Credential Release Dry-Run and Launch Smoke`, alongside the launch/dmg smoke commands, add:

```markdown
### Icon verification (offline, no credentials)

\`\`\`bash
pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"
\`\`\`

Reads `CFBundleIconFile` from the packaged `Info.plist` and asserts the bundle ships a customized,
valid `.icns`: the icon file is not `electron.icns`, the resource exists under `Contents/Resources/`,
its bytes are not identical to the default Electron icon, and `file` reports a `Mac OS X icon`.
No Apple services, keychain, or network are touched.
```

Also append `release:mac:icon-verify` to the quick-reference command block at the end of §7:

```markdown
pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"  # confirm the packaged app ships the custom icon
```

- [ ] **Step 4: Verify the doc renders / no broken fences**

Run: `cd /Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox && grep -n "release:mac:icon-verify\|generate:icon\|D8" RELEASE.md`
Expected: matches in §1, §3, and §7 as added above.

- [ ] **Step 5: PM checkpoint (commit)** — *checkpoint only.*

Proposed message: `docs(3i): document icon generation + release:mac:icon-verify`
Proposed paths: `RELEASE.md`

---

## Task 10: Full verification gate

Run the complete verification suite. All commands run from the indicated directory.

- [ ] **Step 1: Install (lockfile/devDep)**

Run: `cd apps/desktop && pnpm install`
Expected: clean install; `@resvg/resvg-js@2.6.2` present.

- [ ] **Step 2: Generate the icon**

Run: `cd apps/desktop && pnpm generate:icon`
Expected: `generated build/icon.icns from build/icon.svg`.

- [ ] **Step 3: Focused unit tests (all Vitest — single harness, no mixing)**

Run: `cd apps/desktop && pnpm exec vitest run scripts/generate-icon.test.mjs scripts/release-mac-check.test.mjs scripts/release-mac-icon-verify.test.mjs scripts/release-mac-dry-run.test.mjs`
Expected: all PASS. (All four `scripts/*.test.mjs` files are Vitest, matching `vitest.config.ts`'s `scripts/**/*.test.mjs` include; there is no `node:test` file to run separately.)

- [ ] **Step 4: Full frontend test + typecheck + contracts drift**

Run: `cd apps/desktop && pnpm test && pnpm typecheck && pnpm check:contracts-drift`
Expected: all green; no contract drift.

- [ ] **Step 5: Go tests (no regression)**

Run: `cd core-go && go test ./...`
Expected: all PASS.

- [ ] **Step 6: Preflight shows D8 PASS**

Run: `cd apps/desktop && pnpm release:mac:check; echo "exit=$?"`
Expected: `D8`, `D8-source`, `D8-artifact` all `PASS`. `exit=1` is acceptable **only** if it is due to missing Apple signing/notarization credentials (B1/C1), not D8.

- [ ] **Step 7: Dry-run package (no credentials) — builds the bundle with the new icon**

Run: `cd apps/desktop && pnpm release:mac:dry-run; echo "exit=$?"`
Expected: the first stage logs `[generate:icon]` and produces `build/icon.icns`; the chain then completes the ad-hoc build/verify/manifest stages; `exit=0`. Produces `dist/mac-arm64/Astraler Skillbox.app` and `dist/astraler-skillbox-0.1.0-arm64.dmg`. (Note: `release:mac:dry-run` does **not** call `package:mac:unsigned` — it packages via electron-builder directly, so the icon is generated by the orchestrator's `generate-icon` stage added in Task 5B.)

- [ ] **Step 8: Verify the packaged app ships the custom icon (explicit path)**

Run: `cd apps/desktop && pnpm release:mac:icon-verify "dist/mac-arm64/Astraler Skillbox.app"; echo "exit=$?"`
Expected: `Icon verification PASS`; `exit=0`; printed `CFBundleIconFile` is not `electron.icns` and the sha256 differs from the default anchor.

- [ ] **Step 8b: Verify with the no-arg default (defaults to the staged app)**

Run: `cd apps/desktop && pnpm release:mac:icon-verify; echo "exit=$?"`
Expected: identical PASS — the script defaults `appArg` to `dist/mac-arm64/Astraler Skillbox.app`, so the no-arg form verifies the same staged bundle; `exit=0`.

- [ ] **Step 9: DMG smoke (regression guard for the distributable)**

Run: `cd apps/desktop && pnpm release:mac:dmg-smoke "dist/astraler-skillbox-0.1.0-arm64.dmg"; echo "exit=$?"`
Expected: app boots from the mounted DMG, Go core ready, clean detach; `exit=0`.

- [ ] **Step 10: Leak checks + whitespace hygiene**

Run: `cd /Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox && git diff --check`
Expected: no whitespace errors.

Run: `git status --porcelain -- apps/desktop/build`
Expected: `apps/desktop/build/icon.svg` tracked; `build/icon.icns` and `build/.gen/` **not** listed (gitignored).

Run: `git check-ignore apps/desktop/build/icon.icns apps/desktop/build/.gen/icon.iconset/icon_512x512@2x.png`
Expected: both paths echoed (ignored).

- [ ] **Step 11: Confirm `release:mac:full` composition is unchanged**

Run: `cd /Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox && git diff -- apps/desktop/scripts/release-mac-full.mjs apps/desktop/scripts/release-mac-full.lib.mjs`
Expected: **empty diff** (this slice must not modify the full-release composition).

- [ ] **Step 12: Final PM checkpoint** — *checkpoint only; do not run `git add`/`git commit`.* Confirm with PM, then a single integration commit (or the per-task commits above) at PM discretion.

---

## Self-review (against the brief)

- **Scope coverage:** SVG source (Task 2); `generate-icon.lib/.mjs/.test` (Tasks 3–4); `generate:icon` + packaging wiring (Task 5); dry-run `generate-icon` stage (Task 5B); `@resvg/resvg-js@2.6.2` + `apps/desktop/pnpm-lock.yaml` (Task 1); `.gitignore` (Task 5); `mac.icon` (Task 5); D8 tests + impl (Task 6); standalone `release-mac-icon-verify.lib/.mjs/.test` + `release:mac:icon-verify` script (Tasks 7–8); runbook docs (Task 9). ✅
- **D8 semantics:** FAIL on missing/wrong `mac.icon` (D8); FAIL on missing/malformed svg (D8-source); WARN on missing generated `.icns` (D8-artifact); FAIL on present-but-invalid `.icns` (D8-artifact). ✅
- **Lead-review blockers:** (1) dry-run now generates the icon as stage 0 via Task 5B + tests; (2) D8 tests use Vitest and run via `pnpm exec vitest run`; (3) all new `scripts/*.test.mjs` are Vitest → single focused command, no harness mixing (supersedes the literal node:test split; flagged in "Lead-review revisions"). ✅
- **`release:mac:full` unchanged:** only `package:mac`/`package:mac:unsigned` gain `generate:icon`; `release:mac:dry-run` gains a `generate-icon` orchestrator stage (it does not use `package:mac`); preflight WARNs (not FAILs) on missing artifact; Step 11 asserts an empty diff on the full script. ✅
- **Verification plan:** install, `generate:icon`, focused Vitest tests (incl. dry-run), full `pnpm test`/typecheck/contracts-drift/Go tests, preflight D8 PASS (creds may still exit=1), dry-run (icon stage first), `release:mac:icon-verify` (explicit + no-arg), `release:mac:dmg-smoke`, leak checks + `git diff --check`. ✅
- **Process constraints:** lead-review checkpoint before implementation; checkbox tasks with exact files + commands; commit steps are PM checkpoints only (no `git add`/`commit`); plan saved to the requested path; `/goal` not used. ✅
- **Type/name consistency:** `assertSvgUsable`, `ICONSET_ENTRIES`, `MASTER_PX`, `sipsResizeArgs`, `iconutilArgs` consistent across Tasks 3/4/6; `runReleaseMacDryRun` `generate-icon` stage uses `runStage("generate-icon", ["generate:icon"])` with `failedStage: "generate-icon"` consistent across Task 5B lib/mjs/test; `checkIcon(config, icon)` facts shape `{svgPresent,svgUsable,icnsPresent,icnsValid}` matches the runner in Task 6; `assertIconFacts({iconFile,resourceExists,sha256,fileType})` + `DEFAULT_ICON_SHA256`/`DEFAULT_ICON_FILE`/`resolveIconResource`/`resolveInfoPlist` consistent across Tasks 7/8. ✅
