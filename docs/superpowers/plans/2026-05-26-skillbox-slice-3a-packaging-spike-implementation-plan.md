# Slice 3A Packaging Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a self-contained, unsigned macOS `.dmg` that runs from `/Applications` with a bundled `darwin/arm64` `skillbox-core` sidecar, requiring no Go toolchain, repo checkout, or dev PATH.

**Architecture:** Electron main resolves the sidecar spawn command from a single pure function that branches on `app.isPackaged` — `go run` in dev, a bundled binary under `process.resourcesPath` in packaged mode. The sidecar is compiled by a prepackage script and bundled outside ASAR via electron-builder `extraResources`. The DB path is already cwd-independent (`os.UserHomeDir()`) and needs no change.

**Tech Stack:** Electron + electron-vite, electron-builder (new), Go (`go build` darwin/arm64), Vitest, Node ESM build script.

**Spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3a-packaging-spike-design.md` (approved, follow-up commit `192d8e2`).

---

## File Structure

- `apps/desktop/electron/main/core-process/core-go-path.ts` — **Modify.** Keep `resolveCoreGoPath` (dev cwd). Add a pure `resolveCoreSpawn(opts)` returning `{ command, args, cwd }` for dev vs packaged. This is the unit-tested decision point.
- `apps/desktop/electron/main/core-process/__tests__/core-go-path.test.ts` — **Modify.** Add tests for `resolveCoreSpawn` (dev + packaged branches).
- `apps/desktop/electron/main/core-process/manager.ts` — **Modify.** Call `resolveCoreSpawn` with `app.isPackaged` / `process.resourcesPath`; spawn the resolved command; log the resolved command path so the packaged sidecar is observable.
- `apps/desktop/scripts/build-core.mjs` — **Create.** Compiles `skillbox-core` for `darwin/arm64`, stages it at `apps/desktop/resources/core/skillbox-core`, and `chmod 0755`.
- `apps/desktop/electron-builder.yml` — **Create.** electron-builder config: mac/dmg/arm64, ASAR on, sidecar via `extraResources`, signing disabled.
- `apps/desktop/package.json` — **Modify.** Add `version`, `build:core` + `package:mac:unsigned` scripts, `electron-builder` devDependency.
- `SMOKE.md` — **Modify.** Add a "Packaged macOS DMG smoke" section.
- `SCAFFOLD.md` — **Modify.** Document the packaging build commands and output path.

The `resources/core/skillbox-core` artifact is build output — it MUST NOT be committed. `.gitignore` is updated in Task 3, Step 1, **before** the binary is ever produced.

---

## Tasks

### Task 1: Pure spawn-spec resolver (TDD)

**Files:**
- Modify: `apps/desktop/electron/main/core-process/core-go-path.ts`
- Test: `apps/desktop/electron/main/core-process/__tests__/core-go-path.test.ts`

- [ ] **Step 1: Write the failing tests**

Append to `core-go-path.test.ts`:

```ts
import { resolveCoreSpawn } from "../core-go-path.js";

describe("resolveCoreSpawn", () => {
  it("uses `go run` from repo core-go in dev mode", () => {
    const spec = resolveCoreSpawn({
      isPackaged: false,
      baseDir: "/Users/dev/astraler-skillbox/apps/desktop/out/main",
      resourcesPath: "/ignored",
    });
    expect(spec.command).toBe("go");
    expect(spec.args).toEqual(["run", "./cmd/skillbox-core"]);
    expect(spec.cwd).toBe(path.normalize("/Users/dev/astraler-skillbox/core-go"));
  });

  it("uses bundled binary from resourcesPath in packaged mode", () => {
    const resources = "/Applications/Astraler Skillbox.app/Contents/Resources";
    const spec = resolveCoreSpawn({
      isPackaged: true,
      baseDir: "/ignored",
      resourcesPath: resources,
    });
    const expected = path.join(resources, "core", "skillbox-core");
    expect(spec.command).toBe(expected);
    expect(spec.args).toEqual([]);
    expect(spec.cwd).toBe(path.dirname(expected));
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `(cd apps/desktop && pnpm test --run core-go-path)`
Expected: FAIL — `resolveCoreSpawn is not a function` / import error.

- [ ] **Step 3: Implement `resolveCoreSpawn`**

In `core-go-path.ts`, keep the existing `resolveCoreGoPath` and add:

```ts
export interface CoreSpawnSpec {
  command: string;
  args: string[];
  cwd: string;
}

/**
 * Resolves how to spawn the Go sidecar.
 * Dev: `go run ./cmd/skillbox-core` from the repo core-go dir.
 * Packaged: the bundled binary under process.resourcesPath (outside ASAR),
 * so it needs no `go`, repo checkout, or dev PATH.
 */
export function resolveCoreSpawn(opts: {
  isPackaged: boolean;
  baseDir: string;
  resourcesPath: string;
}): CoreSpawnSpec {
  if (opts.isPackaged) {
    const bin = path.join(opts.resourcesPath, "core", "skillbox-core");
    return { command: bin, args: [], cwd: path.dirname(bin) };
  }
  const cwd = resolveCoreGoPath(opts.baseDir);
  return { command: "go", args: ["run", "./cmd/skillbox-core"], cwd };
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `(cd apps/desktop && pnpm test --run core-go-path)`
Expected: PASS — both `resolveCoreGoPath` and `resolveCoreSpawn` tests green.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/electron/main/core-process/core-go-path.ts \
        apps/desktop/electron/main/core-process/__tests__/core-go-path.test.ts
git commit -m "feat(3a): add dev/packaged sidecar spawn resolver"
```

---

### Task 2: Wire manager to the resolver with observable logging

**Files:**
- Modify: `apps/desktop/electron/main/core-process/manager.ts`

- [ ] **Step 1: Import the resolver and `app`**

`manager.ts` already imports `app` from `electron` and `resolveCoreGoPath`. Replace the `resolveCoreGoPath` import with `resolveCoreSpawn`:

```ts
import { resolveCoreSpawn } from "./core-go-path.js";
```

- [ ] **Step 2: Replace the spawn block**

In `spawnGoCore`, replace these current lines:

```ts
    const cwd = resolveCoreGoPath(__dirname);
    process.stderr.write(`[manager] spawning Go core from ${cwd}\n`);

    const child = spawn("go", ["run", "./cmd/skillbox-core"], {
      cwd,
      stdio: ["pipe", "pipe", "pipe"],
    });
```

with:

```ts
    const spec = resolveCoreSpawn({
      isPackaged: app.isPackaged,
      baseDir: __dirname,
      resourcesPath: process.resourcesPath,
    });
    process.stderr.write(
      `[manager] spawning Go core: ${spec.command} ${spec.args.join(" ")} (cwd=${spec.cwd})\n`
    );

    const child = spawn(spec.command, spec.args, {
      cwd: spec.cwd,
      stdio: ["pipe", "pipe", "pipe"],
    });
```

The rest of `spawnGoCore` (ready timeout, restart policy, exit handling) and `shutdownGoCore` (SIGTERM → 3s → SIGKILL) are unchanged.

- [ ] **Step 3: Typecheck**

Run: `(cd apps/desktop && pnpm typecheck)`
Expected: PASS, no unused-import error for the removed `resolveCoreGoPath`.

- [ ] **Step 4: Run unit tests**

Run: `(cd apps/desktop && pnpm test --run)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/electron/main/core-process/manager.ts
git commit -m "feat(3a): spawn sidecar via resolver and log resolved path"
```

---

### Task 3: Build script for the bundled sidecar

**Files:**
- Modify: `.gitignore` (repo root)
- Create: `apps/desktop/scripts/build-core.mjs`

- [ ] **Step 1: Ignore build artifacts FIRST (before any binary is produced)**

This step MUST land before `pnpm build:core` runs in this task, so the generated `skillbox-core` binary can never be staged or committed. Add to the repo-root `.gitignore` (skip any line already present):

```
apps/desktop/dist/
apps/desktop/resources/core/
```

Then verify the ignore is effective immediately:

Run: `git check-ignore apps/desktop/resources/core/skillbox-core apps/desktop/dist`
Expected: both paths echoed back (proves they are ignored before the binary exists).

- [ ] **Step 2: Write the build script**

```js
import { execFileSync } from "node:child_process";
import { mkdirSync, chmodSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");          // apps/desktop
const repoRoot = path.resolve(desktop, "../..");   // repo root
const coreGo = path.join(repoRoot, "core-go");
const outDir = path.join(desktop, "resources", "core");
const outBin = path.join(outDir, "skillbox-core");

mkdirSync(outDir, { recursive: true });

// Pure-Go SQLite (modernc.org/sqlite) => CGO_ENABLED=0, no toolchain at runtime.
execFileSync("go", ["build", "-o", outBin, "./cmd/skillbox-core"], {
  cwd: coreGo,
  stdio: "inherit",
  env: { ...process.env, GOOS: "darwin", GOARCH: "arm64", CGO_ENABLED: "0" },
});

chmodSync(outBin, 0o755);
console.log(`[build:core] built ${outBin}`);
```

- [ ] **Step 3: Add the `build:core` script to package.json**

In `apps/desktop/package.json` `"scripts"`, add:

```json
    "build:core": "node scripts/build-core.mjs",
```

- [ ] **Step 4: Run it and verify the artifact**

Run: `(cd apps/desktop && pnpm build:core)`
Then: `file apps/desktop/resources/core/skillbox-core && test -x apps/desktop/resources/core/skillbox-core && echo EXECUTABLE`
Expected: `Mach-O 64-bit executable arm64` and `EXECUTABLE`.
Then confirm it is still ignored (nothing to stage): `git status --porcelain apps/desktop/resources/` returns no output.

- [ ] **Step 5: Commit (gitignore + script only, never the binary)**

```bash
git add .gitignore apps/desktop/scripts/build-core.mjs apps/desktop/package.json
git commit -m "feat(3a): add build:core script for bundled darwin/arm64 sidecar"
```

---

### Task 4: electron-builder config + packaging script

**Files:**
- Create: `apps/desktop/electron-builder.yml`
- Modify: `apps/desktop/package.json`

(Build-artifact `.gitignore` entries were already added in Task 3, Step 1.)

- [ ] **Step 1: Write `apps/desktop/electron-builder.yml`**

```yaml
appId: com.astraler.skillbox
productName: Astraler Skillbox
directories:
  output: dist
files:
  - out/**/*
  - package.json
asar: true
extraResources:
  - from: resources/core/skillbox-core
    to: core/skillbox-core
mac:
  identity: null            # unsigned spike (signing is Slice 3B)
  category: public.app-category.developer-tools
  target:
    - target: dmg
      arch:
        - arm64
```

`asar: true` packs `out/**` into `app.asar`, while `extraResources` keeps the sidecar at `Contents/Resources/core/skillbox-core` — outside ASAR and directly executable. `identity: null` disables signing.

- [ ] **Step 2: Add `version`, the package script, and the devDependency to package.json**

Add a top-level `"version"` (electron-builder requires it):

```json
  "version": "0.0.0",
```

Add to `"scripts"`:

```json
    "package:mac:unsigned": "pnpm build:core && pnpm build && electron-builder --mac dmg",
```

Add to `"devDependencies"`:

```json
    "electron-builder": "latest",
```

Order: `build:core` (sidecar staged) → `build` (electron-vite renderer/main bundle) → `electron-builder` packs. electron-builder MUST NOT run before the binary exists.

- [ ] **Step 3: Install the new devDependency**

Run: `(cd apps/desktop && pnpm install)`
Expected: `electron-builder` resolves and installs with no error.

- [ ] **Step 4: Commit (config only)**

```bash
git add apps/desktop/electron-builder.yml apps/desktop/package.json \
        apps/desktop/pnpm-lock.yaml
git commit -m "feat(3a): add unsigned macOS dmg packaging config and script"
```

---

### Task 5: Packaged smoke + scaffold docs

**Files:**
- Modify: `SMOKE.md`
- Modify: `SCAFFOLD.md`

- [ ] **Step 1: Add a packaged smoke section to `SMOKE.md`**

Append a new top-level section:

```markdown
## Packaged macOS DMG Smoke (Slice 3A)

Run from the repo root. Produces and verifies an unsigned arm64 `.dmg`.

### Build
- [ ] `(cd apps/desktop && pnpm package:mac:unsigned)`
- [ ] Confirm artifact exists: `ls "apps/desktop/dist/Astraler Skillbox-0.0.0-arm64.dmg"`

### Install
- [ ] Open the `.dmg`, drag **Astraler Skillbox** to `/Applications`, eject the volume.
- [ ] Clear quarantine (unsigned build): `xattr -dr com.apple.quarantine "/Applications/Astraler Skillbox.app"`

### Launch with observable evidence
- [ ] Launch from a **neutral, non-repo cwd** (e.g. `/tmp`) with stderr captured, to strengthen the "no repo dependency" claim (cwd is inherited by the child, so this proves the sidecar does not rely on being run from the checkout):
  ```sh
  (cd /tmp && "/Applications/Astraler Skillbox.app/Contents/MacOS/Astraler Skillbox" 2> /tmp/skillbox-packaged.log)
  ```
- [ ] `grep "spawning Go core" /tmp/skillbox-packaged.log` shows a path under
  `…/Astraler Skillbox.app/Contents/Resources/core/skillbox-core` (NOT `go run` / a repo path).
- [ ] `grep "Go core ready" /tmp/skillbox-packaged.log` is present (server.ready from the bundled sidecar).
- [ ] Sidecar location/exec bit: `test -x "/Applications/Astraler Skillbox.app/Contents/Resources/core/skillbox-core" && echo OK`
- [ ] **Installed** binary is arm64 (check the bundle, not just the staged artifact):
  ```sh
  file "/Applications/Astraler Skillbox.app/Contents/Resources/core/skillbox-core"
  ```
  Expected: `Mach-O 64-bit executable arm64`.
- [ ] Sidecar is outside ASAR: the path above is a real file, not inside `app.asar`.
- [ ] Live process is the bundled one: `pgrep -fl skillbox-core` shows the in-bundle Resources path.

### Functional smoke (packaged app)
- [ ] DB created under Application Support: `ls ~/Library/Application\ Support/Astraler\ Skillbox/skillbox.db`
- [ ] Host scan succeeds; Skills Library lists host skills.
- [ ] Add a project; project scan succeeds.
- [ ] Install a skill to the project via symlink; then remove it (filesystem + DB reflect both).
- [ ] Dashboard renders aggregated state.

### Shutdown
- [ ] Quit the app (Cmd+Q).
- [ ] No orphaned sidecar: `pgrep -fl skillbox-core` returns nothing.
```

- [ ] **Step 2: Add a packaging section to `SCAFFOLD.md`**

Document the two commands and output path:

```markdown
## Packaging (Slice 3A — unsigned macOS DMG)

- `pnpm build:core` — compiles `core-go` to `apps/desktop/resources/core/skillbox-core` (darwin/arm64, CGO off).
- `pnpm package:mac:unsigned` — runs `build:core`, then `electron-vite build`, then `electron-builder --mac dmg`.
- Output: `apps/desktop/dist/Astraler Skillbox-<version>-arm64.dmg` (unsigned).
- The sidecar is bundled via `extraResources` at `Contents/Resources/core/skillbox-core` (outside ASAR).
- Signing/notarization is deferred to Slice 3B.
```

- [ ] **Step 3: Commit**

```bash
git add SMOKE.md SCAFFOLD.md
git commit -m "docs(3a): add packaged dmg smoke and packaging instructions"
```

---

### Task 6: Full verification gauntlet

No code changes — run every gate and the packaged smoke, then commit nothing unless a gate forces a fix.

- [ ] **Step 1: Go tests**

Run: `(cd core-go && go test ./...)`
Expected: all packages PASS.

- [ ] **Step 2: Frontend typecheck**

Run: `(cd apps/desktop && pnpm typecheck)`
Expected: PASS.

- [ ] **Step 3: Frontend unit tests**

Run: `(cd apps/desktop && pnpm test --run)`
Expected: PASS, including both `resolveCoreSpawn` cases.

- [ ] **Step 4: Contract drift**

Run: `(cd apps/desktop && pnpm check:contracts-drift)`
Expected: PASS (no contract changes in this slice).

- [ ] **Step 5: electron-vite build**

Run: `(cd apps/desktop && pnpm build)`
Expected: builds `out/main`, `out/preload`, `out/renderer` with no error.

- [ ] **Step 6: Package the DMG**

Run: `(cd apps/desktop && pnpm package:mac:unsigned)`
Expected: `apps/desktop/dist/Astraler Skillbox-0.0.0-arm64.dmg` is produced.

- [ ] **Step 7: Run the packaged smoke**

Execute the "Packaged macOS DMG Smoke (Slice 3A)" checklist in `SMOKE.md` end to end. All boxes must pass.

---

## Acceptance Criteria (lead-required — verified in Task 6 / SMOKE.md)

- [ ] **No `go` / repo / dev-PATH dependency:** packaged app launches from `/Applications` (terminal launch with captured stderr) and runs without a Go toolchain or repo checkout. (SMOKE: Launch step.)
- [ ] **`process.resourcesPath` packaged sidecar:** `resolveCoreSpawn` packaged branch resolves `<resourcesPath>/core/skillbox-core` (Task 1 test) and the spawn log shows that in-bundle path (SMOKE: `grep "spawning Go core"`).
- [ ] **Outside ASAR and executable:** sidecar lives at `Contents/Resources/core/skillbox-core` (not in `app.asar`) and `test -x` passes (SMOKE: sidecar location/exec-bit checks; config via `extraResources` + `chmod 0755`).
- [ ] **Observable manager logs:** `[manager] spawning Go core: <bundled path> …` and `[manager] Go core ready` both appear in captured stderr (SMOKE: two `grep` checks).
- [ ] **No orphaned sidecar process:** after Cmd+Q, `pgrep -fl skillbox-core` returns nothing (SMOKE: Shutdown step).
- [ ] **DB created/migrated under Application Support** and functional smoke (host scan, skill list, project add/scan, symlink install/remove, Dashboard) passes.
- [ ] **All gates green:** `go test ./...`, `pnpm typecheck`, `pnpm test --run`, `pnpm check:contracts-drift`, `pnpm build`, `pnpm package:mac:unsigned`.

---

## Notes for the Executing Agent

- This slice changes **no** JSON-RPC contracts, schema, or product features. If `pnpm check:contracts-drift` reports drift, you changed something out of scope — revert it.
- `resolveCoreGoPath` stays as-is; `resolveCoreSpawn` is additive and is the only decision point. Do not duplicate the dev/packaged branch in `manager.ts`.
- Never commit `apps/desktop/resources/core/skillbox-core` or `apps/desktop/dist/` — they are build outputs (gitignored in Task 3, Step 1, before the binary is built).
- The DB path is already correct in `core-go/cmd/skillbox-core/main.go` (`resolveDBPath` via `os.UserHomeDir()`); do not modify it.
- Signing, notarization, stapling, auto-update, universal binary, Windows/Linux, and CI release automation are explicitly **out of scope** (Slice 3B and later).
