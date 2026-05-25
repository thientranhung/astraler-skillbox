# Slice 3B2A Implementation Plan: macOS Release Preflight / Credential Doctor

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only, offline `pnpm release:mac:check` command that reports macOS notarized-release readiness (signing credentials, notarization credential groups, electron-builder config invariants, sidecar/artifact/secret hygiene, version) and exits non-zero when a hard blocker is present — without signing, notarizing, building, mutating the keychain, calling Apple, or printing any secret value or path.

**Architecture:** A **pure evaluator** module (`release-mac-check.lib.mjs`) takes plain facts (env values, parsed config, identity names, tool/file probes) and returns a structured `{ results, missing, exitCode }` plus a `render()` string — no process, filesystem, or env access inside it, so it is fully unit-testable and structurally cannot leak. A **thin IO shell** (`release-mac-check.mjs`) gathers those facts (spawns read-only `security`/`xcrun`/`plutil`/`file`/`git`, reads `package.json` and parses `electron-builder.yml`), calls the evaluator, prints the report, and sets the exit code.

**Tech Stack:** Node ESM (`.mjs`, matching `scripts/build-core.mjs` / `scripts/generate-contracts.mjs`), Vitest (node environment), `js-yaml` for config parsing.

**Spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3b2a-release-preflight-design.md` (lead-approved, commit `1e5bc81`).

**Starting state (verified 2026-05-26):** `apps/desktop/scripts/` holds `build-core.mjs` + `generate-contracts.mjs` (ESM, not typechecked). `vitest.config.ts` `include` lists only `electron/**/__tests__/**/*.test.ts` and `renderer/**/__tests__/**/*.test.{ts,tsx}` — no `scripts/` glob. `js-yaml@4.1.1` exists in the pnpm store transitively but is **not** a direct dependency. `electron-builder.yml` is the 3B1 signed default (hardenedRuntime/notarize true, entitlements, `mac.binaries: [Contents/Resources/core/skillbox-core]`, dmg/arm64). `package.json` version is `0.1.0`. This machine has 0 Developer ID identities and no notarization env vars.

---

## File Structure

- `apps/desktop/scripts/release-mac-check.lib.mjs` — **Create.** Pure evaluator: `isSet`, per-category check functions (`checkPlatform`, `checkTooling`, `checkSigning`, `checkNotarization`, `checkConfig`, `checkSidecar`, `checkHygiene`, `checkVersion`), `evaluate(facts)`, and `render(results, missing)`. No I/O.
- `apps/desktop/scripts/release-mac-check.test.mjs` — **Create.** Vitest unit tests for every check + redaction + exit-code mapping.
- `apps/desktop/scripts/release-mac-check.mjs` — **Create.** Thin IO shell: gathers facts, calls `evaluate`, prints `render`, `process.exit(exitCode)`.
- `apps/desktop/vitest.config.ts` — **Modify.** Add `scripts/**/*.test.mjs` to `test.include`.
- `apps/desktop/package.json` — **Modify.** Add `release:mac:check` script; add `js-yaml` devDependency. (No version change; `package:mac` / `package:mac:unsigned` untouched.)
- `SMOKE.md` — **Modify.** Add a short "Release Preflight (Slice 3B2A)" section.
- `SCAFFOLD.md` — **Modify.** Document `pnpm release:mac:check` as the pre-release gate.

No files under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/` change. `build:core`, `scripts/build-core.mjs`, `electron-builder.yml`, and the entitlements plists are untouched.

**Shared types (JSDoc, defined once; referenced by all tasks):**

```js
/** @typedef {'PASS'|'FAIL'|'WARN'|'INFO'} Status */
/** @typedef {{ id: string, category: string, status: Status, message: string, remediation?: string }} CheckResult */
/**
 * @typedef {Object} Facts
 * @property {string} platform                         // process.platform
 * @property {{notarytool:boolean,stapler:boolean,codesign:boolean,spctl:boolean,plutil:boolean}} tools
 * @property {string[]} identityNames                  // "Developer ID Application: …" names only (never secret)
 * @property {Record<string,string|undefined>} env     // raw env VALUES; evaluator computes presence, never prints values
 * @property {{cscLink:{isLocalPath:boolean,exists:boolean,readable:boolean}|null, appleApiKey:{exists:boolean,readable:boolean}|null}} fileProbes
 * @property {any} config                              // parsed electron-builder.yml
 * @property {{mainExists:boolean,mainLintOk:boolean,inheritExists:boolean,inheritLintOk:boolean}} entitlements
 * @property {{present:boolean,arch:string|null,executable:boolean}} sidecar
 * @property {string[]} trackedArtifacts               // tracked dist/ or resources/core paths (in-git, not secret)
 * @property {string[]} trackedSecretFiles             // tracked .p12/.p8 filenames (in-git, not secret)
 * @property {string} version                          // package.json version
 */
```

---

## Task 1: Evaluator scaffold + platform/tooling checks (A1–A4)

**Files:**
- Create: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Create: `apps/desktop/scripts/release-mac-check.test.mjs`
- Modify: `apps/desktop/vitest.config.ts`

- [ ] **Step 1: Add the scripts glob to Vitest include**

In `apps/desktop/vitest.config.ts`, change the `include` array to add the `.mjs` scripts glob (keep the existing entries):

```ts
    include: [
      "electron/**/__tests__/**/*.test.ts",
      "renderer/**/__tests__/**/*.test.ts",
      "renderer/**/__tests__/**/*.test.tsx",
      "scripts/**/*.test.mjs",
    ],
```

- [ ] **Step 2: Write the failing test for platform + tooling**

Create `apps/desktop/scripts/release-mac-check.test.mjs`:

```js
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
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — cannot resolve `./release-mac-check.lib.mjs` (module does not exist yet).

- [ ] **Step 4: Create the library with the scaffold + these two checks**

Create `apps/desktop/scripts/release-mac-check.lib.mjs`:

```js
/**
 * Pure evaluator for the macOS release preflight (Slice 3B2A).
 * NO process / filesystem / env access here — callers pass plain facts.
 * Never emit any credential VALUE or file PATH; only variable names + state tokens.
 *
 * See the plan header for the CheckResult / Facts typedefs.
 */

/** @param {unknown} v */
export function isSet(v) {
  return typeof v === "string" && v.trim().length > 0;
}

/** @param {string} platform @returns {import('./release-mac-check.lib.mjs').CheckResult} */
export function checkPlatform(platform) {
  if (platform === "darwin") {
    return { id: "A1", category: "platform", status: "PASS", message: "macOS (darwin)" };
  }
  return {
    id: "A1",
    category: "platform",
    status: "FAIL",
    message: `unsupported platform: ${platform} (macOS required)`,
    remediation: "Run on macOS; packaging is macOS-only.",
  };
}

/** @param {Record<string,boolean>} tools */
export function checkTooling(tools) {
  const defs = [
    ["A2", "notarytool", "xcrun notarytool"],
    ["A3", "stapler", "xcrun stapler"],
    ["A4a", "codesign", "codesign"],
    ["A4b", "spctl", "spctl"],
    ["A4c", "plutil", "plutil"],
  ];
  return defs.map(([id, key, label]) =>
    tools[key]
      ? { id, category: "platform", status: "PASS", message: `${label} found` }
      : {
          id,
          category: "platform",
          status: "FAIL",
          message: `${label} not found`,
          remediation: `Install Xcode Command Line Tools (xcode-select --install) to provide ${label}.`,
        }
  );
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS (all `isSet`, `checkPlatform`, `checkTooling` cases green).

- [ ] **Step 6: Commit**

```bash
git add apps/desktop/vitest.config.ts apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): preflight evaluator scaffold + platform/tooling checks"
```

---

## Task 2: Signing credentials check (B1) — keychain OR CSC_LINK

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`

- [ ] **Step 1: Write the failing tests for `checkSigning`**

Add to `release-mac-check.test.mjs` (extend the import line and append the describe block):

```js
import {
  isSet,
  checkPlatform,
  checkTooling,
  checkSigning,
} from "./release-mac-check.lib.mjs";
```

```js
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — `checkSigning` is not exported.

- [ ] **Step 3: Implement `checkSigning`**

Append to `release-mac-check.lib.mjs`:

```js
/** @param {{identityNames:string[], env:Record<string,string|undefined>, fileProbes:any}} facts */
export function checkSigning({ identityNames, env, fileProbes }) {
  const hasKeychain = identityNames.length > 0;
  const cscLinkSet = isSet(env.CSC_LINK);
  const cscPwSet = isSet(env.CSC_KEY_PASSWORD);

  let pathB = false;
  let pathBProblem = null;
  if (cscLinkSet && cscPwSet) {
    const p = fileProbes.cscLink;
    if (p && p.isLocalPath) {
      if (p.exists && p.readable) pathB = true;
      else pathBProblem = "CSC_LINK points to a local file that is missing or unreadable";
    } else {
      pathB = true; // URL/base64 form — presence is sufficient; never fetched/decoded
    }
  } else if (cscLinkSet && !cscPwSet) {
    pathBProblem = "CSC_LINK is set but CSC_KEY_PASSWORD is missing";
  } else if (!cscLinkSet && cscPwSet) {
    pathBProblem = "CSC_KEY_PASSWORD is set but CSC_LINK is missing";
  }

  if (hasKeychain) {
    const note = pathB ? " (CSC_LINK + CSC_KEY_PASSWORD also present)" : "";
    return { id: "B1", category: "signing", status: "PASS", message: `Developer ID Application identity in keychain${note}` };
  }
  if (pathB) {
    return { id: "B1", category: "signing", status: "PASS", message: "CSC_LINK + CSC_KEY_PASSWORD present" };
  }
  const detail = pathBProblem ? ` (${pathBProblem})` : "";
  return {
    id: "B1",
    category: "signing",
    status: "FAIL",
    message: `no signing credential${detail}`,
    remediation:
      "Signing credential: a Developer ID Application identity in the login keychain, OR CSC_LINK + CSC_KEY_PASSWORD",
  };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): signing credential check (keychain or CSC_LINK)"
```

---

## Task 3: Notarization credential groups (C1)

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`

- [ ] **Step 1: Write the failing tests for `checkNotarization`**

Add `checkNotarization` to the import, then append:

```js
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — `checkNotarization` is not exported.

- [ ] **Step 3: Implement `checkNotarization`**

Append to `release-mac-check.lib.mjs`:

```js
/** @param {Record<string,string|undefined>} env @param {any} fileProbes */
export function checkNotarization(env, fileProbes) {
  const out = [];
  const g1Vars = ["APPLE_API_KEY", "APPLE_API_KEY_ID", "APPLE_API_ISSUER"];
  const g2Vars = ["APPLE_ID", "APPLE_APP_SPECIFIC_PASSWORD", "APPLE_TEAM_ID"];
  const setOf = (vars) => vars.filter((v) => isSet(env[v]));
  const missingOf = (vars) => vars.filter((v) => !isSet(env[v]));

  const g1Set = setOf(g1Vars);
  const g1Missing = missingOf(g1Vars);
  const g2Set = setOf(g2Vars);
  const g2Missing = missingOf(g2Vars);

  const apiKeyFileOk = fileProbes.appleApiKey ? fileProbes.appleApiKey.exists && fileProbes.appleApiKey.readable : false;
  const g1AllSet = g1Missing.length === 0;
  const g1Complete = g1AllSet && apiKeyFileOk;
  const g2Complete = g2Missing.length === 0;
  const profileSet = isSet(env.APPLE_KEYCHAIN_PROFILE);

  const REMEDIATION =
    "One notarization credential group (Group 1: APPLE_API_KEY + APPLE_API_KEY_ID + APPLE_API_ISSUER, or Group 2: APPLE_ID + APPLE_APP_SPECIFIC_PASSWORD + APPLE_TEAM_ID)";

  if (g1Complete && g2Complete) {
    out.push({ id: "C1", category: "notarization", status: "PASS", message: "notarization credentials present" });
    out.push({
      id: "C1-precedence",
      category: "notarization",
      status: "WARN",
      message: "both Group 1 (API key) and Group 2 (Apple ID) are complete; Group 1 (API key) is preferred and will be used",
    });
    return out;
  }
  if (g1Complete) {
    out.push({ id: "C1", category: "notarization", status: "PASS", message: "API key (Group 1) detected" });
    return out;
  }
  if (g2Complete) {
    out.push({ id: "C1", category: "notarization", status: "PASS", message: "Apple ID (Group 2) detected" });
    return out;
  }

  let msg;
  if (g1AllSet && !apiKeyFileOk) {
    msg = "APPLE_API_KEY, APPLE_API_KEY_ID, APPLE_API_ISSUER are set, but the APPLE_API_KEY .p8 file is missing or unreadable";
  } else if (g1Set.length >= g2Set.length && g1Set.length > 0) {
    msg = `Group 1 partially set; missing ${g1Missing.join(", ")}`;
  } else if (g2Set.length > 0) {
    msg = `Group 2 partially set; missing ${g2Missing.join(", ")}`;
  } else {
    msg = "no complete credential group";
  }
  out.push({ id: "C1", category: "notarization", status: "FAIL", message: msg, remediation: REMEDIATION });
  if (profileSet) {
    out.push({
      id: "C1-profile",
      category: "notarization",
      status: "INFO",
      message: "APPLE_KEYCHAIN_PROFILE is set, but electron-builder mac.notarize uses Group 1 or Group 2; a keychain profile alone does not satisfy this gate",
    });
  }
  return out;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): notarization credential group check (>=1 complete, precedence WARN)"
```

---

## Task 4: Config invariants (D1–D5) + sidecar staging (E1)

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`

- [ ] **Step 1: Write the failing tests**

Add `checkConfig, checkSidecar` to the import, then append:

```js
const GOOD_CONFIG = {
  mac: {
    hardenedRuntime: true,
    notarize: true,
    binaries: ["Contents/Resources/core/skillbox-core"],
    target: [{ target: "dmg", arch: ["arm64"] }],
  },
};
const GOOD_ENT = { mainExists: true, mainLintOk: true, inheritExists: true, inheritLintOk: true };

describe("checkConfig (D1–D5)", () => {
  const ids = (rows) => Object.fromEntries(rows.map((r) => [r.id, r.status]));
  it("passes the committed signed-default shape", () => {
    expect(ids(checkConfig(GOOD_CONFIG, GOOD_ENT))).toEqual({ D1: "PASS", D2: "PASS", D3: "PASS", D4: "PASS", D5: "PASS" });
  });
  it("fails D1 when hardenedRuntime is off", () => {
    const cfg = { mac: { ...GOOD_CONFIG.mac, hardenedRuntime: false } };
    expect(ids(checkConfig(cfg, GOOD_ENT)).D1).toBe("FAIL");
  });
  it("fails D2 when notarize is off", () => {
    const cfg = { mac: { ...GOOD_CONFIG.mac, notarize: false } };
    expect(ids(checkConfig(cfg, GOOD_ENT)).D2).toBe("FAIL");
  });
  it("fails D3 when an entitlement fails lint", () => {
    expect(ids(checkConfig(GOOD_CONFIG, { ...GOOD_ENT, inheritLintOk: false })).D3).toBe("FAIL");
  });
  it("fails D4 when mac.binaries omits the sidecar path", () => {
    const cfg = { mac: { ...GOOD_CONFIG.mac, binaries: [] } };
    expect(ids(checkConfig(cfg, GOOD_ENT)).D4).toBe("FAIL");
  });
  it("fails D5 when no dmg/arm64 target", () => {
    const cfg = { mac: { ...GOOD_CONFIG.mac, target: [{ target: "dmg", arch: ["x64"] }] } };
    expect(ids(checkConfig(cfg, GOOD_ENT)).D5).toBe("FAIL");
  });
});

describe("checkSidecar (E1)", () => {
  it("warns when absent", () => {
    expect(checkSidecar({ present: false, arch: null, executable: false }).status).toBe("WARN");
  });
  it("passes arm64 + executable", () => {
    expect(checkSidecar({ present: true, arch: "arm64", executable: true }).status).toBe("PASS");
  });
  it("fails wrong arch", () => {
    expect(checkSidecar({ present: true, arch: "x86_64", executable: true }).status).toBe("FAIL");
  });
  it("fails not executable", () => {
    expect(checkSidecar({ present: true, arch: "arm64", executable: false }).status).toBe("FAIL");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — `checkConfig` / `checkSidecar` not exported.

- [ ] **Step 3: Implement `checkConfig` and `checkSidecar`**

Append to `release-mac-check.lib.mjs`:

```js
const SIDECAR_BUNDLE_PATH = "Contents/Resources/core/skillbox-core";

/** @param {any} config @param {{mainExists:boolean,mainLintOk:boolean,inheritExists:boolean,inheritLintOk:boolean}} entitlements */
export function checkConfig(config, entitlements) {
  const mac = (config && config.mac) || {};
  const out = [];

  out.push(
    mac.hardenedRuntime === true
      ? { id: "D1", category: "config", status: "PASS", message: "mac.hardenedRuntime: true" }
      : { id: "D1", category: "config", status: "FAIL", message: `mac.hardenedRuntime is not true (got ${JSON.stringify(mac.hardenedRuntime)})`, remediation: "Set mac.hardenedRuntime: true in electron-builder.yml" }
  );
  out.push(
    mac.notarize === true
      ? { id: "D2", category: "config", status: "PASS", message: "mac.notarize: true" }
      : { id: "D2", category: "config", status: "FAIL", message: `mac.notarize is not true (got ${JSON.stringify(mac.notarize)})`, remediation: "Set mac.notarize: true in electron-builder.yml" }
  );

  const entOk = entitlements.mainExists && entitlements.mainLintOk && entitlements.inheritExists && entitlements.inheritLintOk;
  out.push(
    entOk
      ? { id: "D3", category: "config", status: "PASS", message: "entitlements present and lint OK" }
      : { id: "D3", category: "config", status: "FAIL", message: "entitlements missing or failed plutil -lint", remediation: "Ensure build/entitlements.mac.plist and .inherit.plist exist and pass plutil -lint" }
  );

  const bins = Array.isArray(mac.binaries) ? mac.binaries : [];
  out.push(
    bins.includes(SIDECAR_BUNDLE_PATH)
      ? { id: "D4", category: "config", status: "PASS", message: `mac.binaries includes ${SIDECAR_BUNDLE_PATH}` }
      : { id: "D4", category: "config", status: "FAIL", message: `mac.binaries does not include ${SIDECAR_BUNDLE_PATH}`, remediation: `Add ${SIDECAR_BUNDLE_PATH} to mac.binaries in electron-builder.yml` }
  );

  const targets = Array.isArray(mac.target) ? mac.target : [];
  const hasDmgArm64 = targets.some((t) => t && t.target === "dmg" && Array.isArray(t.arch) && t.arch.includes("arm64"));
  out.push(
    hasDmgArm64
      ? { id: "D5", category: "config", status: "PASS", message: "mac.target includes dmg/arm64" }
      : { id: "D5", category: "config", status: "FAIL", message: "mac.target does not include a dmg/arm64 target", remediation: "Add a dmg target with arch arm64 to mac.target" }
  );

  return out;
}

/** @param {{present:boolean,arch:string|null,executable:boolean}} sidecar */
export function checkSidecar(sidecar) {
  if (!sidecar || !sidecar.present) {
    return { id: "E1", category: "sidecar", status: "WARN", message: "staged sidecar absent (will be built by package:mac via build:core)" };
  }
  if (sidecar.arch !== "arm64") {
    return { id: "E1", category: "sidecar", status: "FAIL", message: `staged sidecar arch is ${sidecar.arch ?? "unknown"}, expected arm64`, remediation: "Rebuild the sidecar for arm64 (pnpm build:core)" };
  }
  if (!sidecar.executable) {
    return { id: "E1", category: "sidecar", status: "FAIL", message: "staged sidecar is not executable", remediation: "Restore the exec bit (rerun pnpm build:core)" };
  }
  return { id: "E1", category: "sidecar", status: "PASS", message: "staged sidecar is arm64 + executable" };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): electron-builder config invariants + sidecar staging checks"
```

---

## Task 5: Hygiene (F1/F2) + version (G1)

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`

- [ ] **Step 1: Write the failing tests**

Add `checkHygiene, checkVersion` to the import, then append:

```js
describe("checkHygiene (F1/F2)", () => {
  const id = (rows, k) => rows.find((r) => r.id === k).status;
  it("passes when nothing tracked", () => {
    const rows = checkHygiene({ trackedArtifacts: [], trackedSecretFiles: [] });
    expect(id(rows, "F1")).toBe("PASS");
    expect(id(rows, "F2")).toBe("PASS");
  });
  it("fails F1 on a tracked build artifact", () => {
    expect(id(checkHygiene({ trackedArtifacts: ["apps/desktop/dist/x.dmg"], trackedSecretFiles: [] }), "F1")).toBe("FAIL");
  });
  it("fails F2 on a tracked credential file", () => {
    expect(id(checkHygiene({ trackedArtifacts: [], trackedSecretFiles: ["apps/desktop/cert.p8"] }), "F2")).toBe("FAIL");
  });
});

describe("checkVersion (G1)", () => {
  it("passes a real version", () => {
    expect(checkVersion("0.1.0").status).toBe("PASS");
  });
  it("warns on 0.0.0", () => {
    expect(checkVersion("0.0.0").status).toBe("WARN");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — `checkHygiene` / `checkVersion` not exported.

- [ ] **Step 3: Implement `checkHygiene` and `checkVersion`**

Append to `release-mac-check.lib.mjs`:

```js
/** @param {{trackedArtifacts:string[], trackedSecretFiles:string[]}} facts */
export function checkHygiene({ trackedArtifacts, trackedSecretFiles }) {
  const out = [];
  out.push(
    trackedArtifacts.length === 0
      ? { id: "F1", category: "hygiene", status: "PASS", message: "no tracked build artifacts under dist/ or resources/core" }
      : { id: "F1", category: "hygiene", status: "FAIL", message: `tracked build artifacts present: ${trackedArtifacts.join(", ")}`, remediation: "git rm --cached the tracked dist/ or resources/core artifacts; keep them gitignored" }
  );
  out.push(
    trackedSecretFiles.length === 0
      ? { id: "F2", category: "hygiene", status: "PASS", message: "no tracked .p12/.p8 under apps/desktop" }
      : { id: "F2", category: "hygiene", status: "FAIL", message: `tracked credential file(s): ${trackedSecretFiles.join(", ")}`, remediation: "git rm --cached the tracked .p12/.p8 file(s); never commit credentials" }
  );
  return out;
}

/** @param {string} version */
export function checkVersion(version) {
  return version && version !== "0.0.0"
    ? { id: "G1", category: "version", status: "PASS", message: `version ${version}` }
    : { id: "G1", category: "version", status: "WARN", message: `version is ${version || "unset"} (set a real release version)` };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): artifact/secret-file hygiene + version checks"
```

---

## Task 6: `evaluate()` aggregation + `render()` + redaction + exit codes

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs`
- Modify: `apps/desktop/scripts/release-mac-check.test.mjs`

- [ ] **Step 1: Write the failing tests (aggregation, render, redaction, exit code)**

Add `evaluate, render` to the import, then append. `baseFacts()` builds an all-passing fact set so each test overrides only what it needs:

```js
function baseFacts(overrides = {}) {
  return {
    platform: "darwin",
    tools: { notarytool: true, stapler: true, codesign: true, spctl: true, plutil: true },
    identityNames: ["Developer ID Application: Acme (TEAMID)"],
    env: { APPLE_API_KEY: "/SENTINEL/key.p8", APPLE_API_KEY_ID: "KID", APPLE_API_ISSUER: "ISS" },
    fileProbes: { cscLink: null, appleApiKey: { exists: true, readable: true } },
    config: GOOD_CONFIG,
    entitlements: GOOD_ENT,
    sidecar: { present: true, arch: "arm64", executable: true },
    trackedArtifacts: [],
    trackedSecretFiles: [],
    version: "0.1.0",
    ...overrides,
  };
}

describe("evaluate", () => {
  it("exits 0 when everything passes (WARN/INFO allowed)", () => {
    expect(evaluate(baseFacts()).exitCode).toBe(0);
  });

  it("exits 1 and lists exactly signing + notarization when both are missing", () => {
    const { exitCode, missing, results } = evaluate(
      baseFacts({ identityNames: [], env: {}, fileProbes: { cscLink: null, appleApiKey: null } })
    );
    expect(exitCode).toBe(1);
    expect(missing).toHaveLength(2);
    const failIds = results.filter((r) => r.status === "FAIL").map((r) => r.id);
    expect(failIds).toEqual(["B1", "C1"]);
  });

  it("WARN/INFO never force a non-zero exit", () => {
    // sidecar absent → WARN, version 0.0.0 → WARN, profile-only INFO; still all hard checks pass
    const facts = baseFacts({ sidecar: { present: false, arch: null, executable: false }, version: "0.0.0" });
    expect(evaluate(facts).exitCode).toBe(0);
  });
});

describe("render redaction", () => {
  it("never prints any credential value or file path", () => {
    const facts = baseFacts({
      identityNames: [],
      env: {
        APPLE_API_KEY: "/Users/SENTINEL/key.p8",
        APPLE_API_KEY_ID: "SENTINEL_KEY_ID",
        APPLE_API_ISSUER: "SENTINEL_ISSUER",
        APPLE_APP_SPECIFIC_PASSWORD: "SENTINEL_PW",
        CSC_LINK: "/Users/SENTINEL/cert.p12",
        CSC_KEY_PASSWORD: "SENTINEL_CSC_PW",
      },
      fileProbes: { cscLink: { isLocalPath: true, exists: true, readable: true }, appleApiKey: { exists: false, readable: false } },
    });
    const { results, missing } = evaluate(facts);
    const text = render(results, missing);
    for (const secret of [
      "/Users/SENTINEL/key.p8",
      "SENTINEL_KEY_ID",
      "SENTINEL_ISSUER",
      "SENTINEL_PW",
      "/Users/SENTINEL/cert.p12",
      "SENTINEL_CSC_PW",
    ]) {
      expect(text).not.toContain(secret);
    }
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: FAIL — `evaluate` / `render` not exported.

- [ ] **Step 3: Implement `evaluate` and `render`**

Append to `release-mac-check.lib.mjs`:

```js
/** @param {import('./release-mac-check.lib.mjs').Facts} facts */
export function evaluate(facts) {
  const results = [
    checkPlatform(facts.platform),
    ...checkTooling(facts.tools),
    checkSigning(facts),
    ...checkNotarization(facts.env, facts.fileProbes),
    ...checkConfig(facts.config, facts.entitlements),
    checkSidecar(facts.sidecar),
    ...checkHygiene(facts),
    checkVersion(facts.version),
  ];
  const fails = results.filter((r) => r.status === "FAIL");
  const missing = fails.map((r) => r.remediation ?? r.message);
  const exitCode = fails.length > 0 ? 1 : 0;
  return { results, missing, exitCode };
}

const CATEGORY_ORDER = ["platform", "signing", "notarization", "config", "sidecar", "hygiene", "version"];
const CATEGORY_LABEL = {
  platform: "Platform & tooling",
  signing: "Signing credentials",
  notarization: "Notarization credentials",
  config: "electron-builder config",
  sidecar: "Sidecar staging",
  hygiene: "Artifact & secret hygiene",
  version: "Version",
};

/** @param {import('./release-mac-check.lib.mjs').CheckResult[]} results @param {string[]} missing */
export function render(results, missing) {
  const lines = [];
  for (const cat of CATEGORY_ORDER) {
    const rows = results.filter((r) => r.category === cat);
    if (rows.length === 0) continue;
    lines.push(CATEGORY_LABEL[cat]);
    for (const r of rows) lines.push(`  ${r.status.padEnd(4)}  ${r.message}`);
  }
  if (missing.length > 0) {
    lines.push("");
    lines.push("Missing for a customer-ready notarized DMG:");
    for (const m of missing) lines.push(`  - ${m}`);
  }
  return lines.join("\n");
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)`
Expected: PASS (aggregation, redaction, and exit-code cases all green).

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-check.lib.mjs apps/desktop/scripts/release-mac-check.test.mjs
git commit -m "feat(3b2a): evaluate() aggregation + render() with redaction + exit-code mapping"
```

---

## Task 7: Thin IO shell + `release:mac:check` script + `js-yaml` dep

**Files:**
- Create: `apps/desktop/scripts/release-mac-check.mjs`
- Modify: `apps/desktop/package.json`

- [ ] **Step 1: Add `js-yaml` devDependency and the script entry**

In `apps/desktop/package.json`, add to `"scripts"` (after `package:mac:unsigned`):

```json
    "release:mac:check": "node scripts/release-mac-check.mjs"
```

And add to `"devDependencies"` (alphabetical position near other entries):

```json
    "js-yaml": "^4.1.0",
```

- [ ] **Step 2: Install so `js-yaml` is a resolvable direct dependency**

Run: `(cd apps/desktop && pnpm install)`
Expected: completes; `pnpm-lock.yaml` updates to list `js-yaml` as a direct dep (it is already in the store via electron-builder, so no large download).

- [ ] **Step 3: Create the IO shell**

Create `apps/desktop/scripts/release-mac-check.mjs`. It is the ONLY place that touches the OS, and it is strictly read-only — no signing, notarization, build, network, or keychain mutation:

```js
/**
 * macOS Release Preflight / Credential Doctor (Slice 3B2A).
 * Read-only, offline. Gathers facts and delegates all verdicts to the pure
 * evaluator. NEVER signs, notarizes, builds, mutates the keychain, calls the
 * network, or prints any credential value or file path.
 *
 * Run from apps/desktop/:  pnpm release:mac:check
 */
import { execFileSync } from "node:child_process";
import { existsSync, accessSync, statSync, readFileSync, constants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import yaml from "js-yaml";
import { evaluate, render } from "./release-mac-check.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, ".."); // apps/desktop
const repoRoot = path.resolve(desktop, "..", "..");

/** Run a command read-only; return trimmed stdout or null on any failure. */
function run(cmd, args) {
  try {
    return execFileSync(cmd, args, { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return null;
  }
}

function commandExists(name) {
  return run("/usr/bin/which", [name]) !== null;
}

function xcrunFinds(tool) {
  return run("/usr/bin/xcrun", ["-f", tool]) !== null;
}

function readableFile(p) {
  try {
    accessSync(p, constants.R_OK);
    return statSync(p).isFile();
  } catch {
    return false;
  }
}

function plutilLintOk(p) {
  if (!existsSync(p)) return false;
  return run("/usr/bin/plutil", ["-lint", p]) !== null; // non-zero exit -> null -> false
}

// --- Gather facts (all read-only) ---
const env = {
  APPLE_API_KEY: process.env.APPLE_API_KEY,
  APPLE_API_KEY_ID: process.env.APPLE_API_KEY_ID,
  APPLE_API_ISSUER: process.env.APPLE_API_ISSUER,
  APPLE_ID: process.env.APPLE_ID,
  APPLE_APP_SPECIFIC_PASSWORD: process.env.APPLE_APP_SPECIFIC_PASSWORD,
  APPLE_TEAM_ID: process.env.APPLE_TEAM_ID,
  APPLE_KEYCHAIN_PROFILE: process.env.APPLE_KEYCHAIN_PROFILE,
  CSC_LINK: process.env.CSC_LINK,
  CSC_KEY_PASSWORD: process.env.CSC_KEY_PASSWORD,
};

const tools = {
  notarytool: xcrunFinds("notarytool"),
  stapler: xcrunFinds("stapler"),
  codesign: commandExists("codesign"),
  spctl: commandExists("spctl"),
  plutil: commandExists("plutil"),
};

// Identity NAMES only (non-secret); read-only keychain query.
const identityOut = run("/usr/bin/security", ["find-identity", "-v", "-p", "codesigning"]) ?? "";
const identityNames = [...identityOut.matchAll(/"(Developer ID Application:[^"]*)"/g)].map((m) => m[1]);

// File probes — derived flags only; the path itself is never put into facts beyond env.
function cscLinkProbe(v) {
  if (typeof v !== "string" || v.trim() === "") return null;
  const isLocalPath = /^(\/|\.|~)/.test(v) && !/^https?:\/\//i.test(v);
  if (!isLocalPath) return { isLocalPath: false, exists: false, readable: false };
  return { isLocalPath: true, exists: existsSync(v), readable: readableFile(v) };
}
function apiKeyProbe(v) {
  if (typeof v !== "string" || v.trim() === "") return null;
  return { exists: existsSync(v), readable: readableFile(v) };
}
const fileProbes = { cscLink: cscLinkProbe(env.CSC_LINK), appleApiKey: apiKeyProbe(env.APPLE_API_KEY) };

// electron-builder config
const ebPath = path.join(desktop, "electron-builder.yml");
const config = yaml.load(readFileSync(ebPath, "utf8"));

// Entitlements
const mainPlist = path.join(desktop, "build", "entitlements.mac.plist");
const inheritPlist = path.join(desktop, "build", "entitlements.mac.inherit.plist");
const entitlements = {
  mainExists: existsSync(mainPlist),
  mainLintOk: plutilLintOk(mainPlist),
  inheritExists: existsSync(inheritPlist),
  inheritLintOk: plutilLintOk(inheritPlist),
};

// Staged sidecar
const sidecarPath = path.join(desktop, "resources", "core", "skillbox-core");
let sidecar = { present: false, arch: null, executable: false };
if (existsSync(sidecarPath)) {
  const fileOut = run("/usr/bin/file", [sidecarPath]) ?? "";
  const arch = /arm64/.test(fileOut) ? "arm64" : /x86_64/.test(fileOut) ? "x86_64" : null;
  let executable = false;
  try {
    executable = (statSync(sidecarPath).mode & 0o111) !== 0;
  } catch {
    executable = false;
  }
  sidecar = { present: true, arch, executable };
}

// Hygiene — tracked-status entries only (ignore untracked ??), per spec F1.
const statusOut = run("git", ["status", "--porcelain", "--untracked-files=no", "--", "apps/desktop/dist", "apps/desktop/resources/core"]) ?? "";
const trackedArtifacts = statusOut
  .split("\n")
  .map((l) => l.trim())
  .filter(Boolean)
  .map((l) => l.replace(/^\S+\s+/, "")); // drop the status code, keep the path
const lsFiles = run("git", ["ls-files", "--", "apps/desktop"]) ?? "";
const trackedSecretFiles = lsFiles.split("\n").filter((f) => /\.(p12|p8)$/.test(f));

// Version
const pkg = JSON.parse(readFileSync(path.join(desktop, "package.json"), "utf8"));

const facts = {
  platform: process.platform,
  tools,
  identityNames,
  env,
  fileProbes,
  config,
  entitlements,
  sidecar,
  trackedArtifacts,
  trackedSecretFiles,
  version: pkg.version,
};

const { results, missing, exitCode } = evaluate(facts);
console.log(render(results, missing));
process.exit(exitCode);
```

- [ ] **Step 4: Run the command live (expected NON-ZERO on this machine)**

Run: `(cd apps/desktop && pnpm release:mac:check); echo "exit=$?"`
Expected: prints the grouped report; `Signing credentials` and `Notarization credentials` are FAIL; the "Missing…" list has exactly those two items; `exit=1`. (Platform/tooling/config PASS; sidecar WARN or PASS.)

- [ ] **Step 5: Confirm no secrets leak and no side effects**

Run: `(cd apps/desktop && pnpm release:mac:check 2>&1 | grep -E '/Users/|-----BEGIN' || echo "no secret-looking values in output")`
Expected: `no secret-looking values in output`. (The grep targets real value/path indicators — a home-directory path or a PEM header. It deliberately does NOT grep for `password`/`.p8`/`.p12`, because the report legitimately prints variable *names* like `CSC_KEY_PASSWORD` and generic phrases like ".p8 file"; the authoritative no-leak guarantee is the redaction unit test in Task 6, which asserts actual credential values and paths never appear.) Then confirm the working tree shows only this task's intended changes (`package.json`, `pnpm-lock.yaml`, the new script) and no build/sign artifacts: `git status --porcelain`.

- [ ] **Step 6: Commit**

```bash
git add apps/desktop/package.json apps/desktop/pnpm-lock.yaml apps/desktop/scripts/release-mac-check.mjs
git commit -m "feat(3b2a): add release:mac:check IO shell + js-yaml dep + npm script"
```

---

## Task 8: Docs — SMOKE.md + SCAFFOLD.md

**Files:**
- Modify: `SMOKE.md`
- Modify: `SCAFFOLD.md`

- [ ] **Step 1: Add a Release Preflight section to SMOKE.md**

In `SMOKE.md`, immediately AFTER the "## Signed + Notarized Smoke (Slice 3B2 — requires Apple Developer ID)" section and BEFORE the "## Notes" section, insert:

```markdown
## Release Preflight (Slice 3B2A)

Read-only, offline credential/config doctor. No Apple credentials required; makes no
network call, signs/notarizes/builds nothing, and never mutates the keychain.

- [ ] Run the gate: `(cd apps/desktop && pnpm release:mac:check); echo "exit=$?"`
- [ ] On a machine WITHOUT credentials: `Signing credentials` and `Notarization credentials`
  are FAIL; the "Missing for a customer-ready notarized DMG" list contains exactly those
  two items; `exit=1`. Platform/tooling and electron-builder config invariants are PASS.
- [ ] No secret values/paths in output (targets real path/PEM indicators; variable names like `CSC_KEY_PASSWORD` are expected and fine):
  ```sh
  (cd apps/desktop && pnpm release:mac:check 2>&1 | grep -E '/Users/|-----BEGIN') || echo "clean"
  ```
  Expected: `clean`.
- [ ] When credentials ARE present (3B2), the same command exits `0` — run it before `pnpm package:mac`.
```

- [ ] **Step 2: Add the preflight to SCAFFOLD.md Packaging section**

In `SCAFFOLD.md`, inside the "## Packaging" section, immediately AFTER the "### Signed + notarized (Slice 3B2 — deferred, needs credentials)" block, insert:

```markdown
### Release preflight (Slice 3B2A)
- `pnpm release:mac:check` — read-only, offline gate. Reports signing-credential readiness
  (keychain Developer ID Application identity OR `CSC_LINK` + `CSC_KEY_PASSWORD`), notarization
  credential groups (API key, or Apple ID + app password + Team ID), electron-builder config
  invariants (hardened runtime, notarize, entitlements, `mac.binaries`, dmg/arm64), staged-sidecar
  sanity, and artifact/secret hygiene. Exits non-zero when a hard blocker is present.
- Run it BEFORE `pnpm package:mac` to surface credential/config gaps in <1s instead of minutes
  into a build. It never signs, notarizes, builds, calls Apple, mutates the keychain, or prints
  any secret value or path. See SMOKE.md → "Release Preflight (Slice 3B2A)".
```

- [ ] **Step 3: Commit**

```bash
git add SMOKE.md SCAFFOLD.md
git commit -m "docs(3b2a): document release:mac:check preflight in SMOKE and SCAFFOLD"
```

---

## Task 9: Full verification gauntlet

No code changes. Run every gate and the live preflight; commit nothing unless a gate forces a fix.

- [ ] **Step 1: Targeted preflight unit tests** — `(cd apps/desktop && pnpm exec vitest run scripts/release-mac-check.test.mjs)` → all PASS.
- [ ] **Step 2: Full frontend test suite** — `(cd apps/desktop && pnpm test)` → PASS (the new scripts test is now discovered via the `scripts/**/*.test.mjs` include; no existing test regresses).
- [ ] **Step 3: Typecheck** — `(cd apps/desktop && pnpm typecheck)` → PASS (`.mjs` scripts are not part of the TS build; no type errors introduced).
- [ ] **Step 4: Contract drift** — `(cd apps/desktop && pnpm check:contracts-drift)` → PASS (no contract/schema/generated change in this slice).
- [ ] **Step 5: Go tests (sanity; nothing Go changed)** — `(cd core-go && go test ./...)` → PASS.
- [ ] **Step 6: electron-vite build (sanity; build path unaffected)** — `(cd apps/desktop && pnpm build)` → builds `out/main`, `out/preload`, `out/renderer`.
- [ ] **Step 7: Live preflight verdict** — `(cd apps/desktop && pnpm release:mac:check); echo "exit=$?"` → `exit=1`; FAIL on `Signing credentials` + `Notarization credentials`; the "Missing…" list has exactly those two items; A1–A4 and D1–D5 PASS; E1 WARN or PASS.
- [ ] **Step 8: Redaction grep** — `(cd apps/desktop && pnpm release:mac:check 2>&1 | grep -E '/Users/|-----BEGIN') || echo clean` → `clean`. (Targets real path/PEM indicators; variable names such as `CSC_KEY_PASSWORD` are expected output and must not be flagged.)
- [ ] **Step 9: No side effects** — confirm the command made no build/sign/notarize artifact and mutated nothing: `git status --porcelain` is empty (clean of tracked artifacts), and `apps/desktop/dist` is unchanged. No network was contacted (the script contains no network call by construction).

---

## Acceptance Criteria (lead-required — verified in Task 9)

- [ ] `pnpm release:mac:check` exits **non-zero (1)** on this credential-less machine.
- [ ] The "Missing for a customer-ready notarized DMG" list contains **exactly two** items: signing credentials (B1) and one notarization credential group (C1).
- [ ] Platform/tooling (A1–A4) and config invariants (D1–D5) PASS against the committed `electron-builder.yml`; F1/F2 PASS; G1 PASS (`0.1.0`); E1 WARN or PASS.
- [ ] **No secret value or file path** appears in output (redaction unit test + manual grep).
- [ ] No network call, no build, no signing/notarization, no keychain mutation, no file writes (by construction; verified by review + clean `git status`).
- [ ] All existing gates green: `go test ./...`, `pnpm typecheck`, `pnpm test`, `pnpm check:contracts-drift`, `pnpm build`.
- [ ] No JSON-RPC contract, schema, or product change.

---

## Out of Scope — MUST NOT Touch

- **Real signing / notarization / stapling** — the script never invokes `codesign -s`, `notarytool submit`, `stapler staple`, or any Apple online service.
- **Building** — never runs `build:core`, `electron-vite build`, or `electron-builder`; never produces a DMG.
- **Keychain mutation** — only `security find-identity -v` (read-only); never import/delete/unlock.
- **Network** — no HTTP/notarization/App Store Connect calls; offline only.
- **Secret printing** — never print credential values or the `CSC_LINK` / `APPLE_API_KEY` paths; presence/state tokens only.
- **Packaging behavior** — `package:mac`, `package:mac:unsigned`, `electron-builder.yml`, entitlements plists, `build:core` unchanged.
- **Product / RPC / schema / migrations** — no renderer, Electron main, or `core-go` logic; contract-drift must stay clean.
- **CI release automation, auto-update, universal binary, Windows/Linux, `.pkg`/Mac App Store** — not added.
- **`package.json` version** — not changed; only the `release:mac:check` script and the `js-yaml` devDependency are added.
