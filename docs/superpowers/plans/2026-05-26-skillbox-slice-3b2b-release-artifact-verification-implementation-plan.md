# Slice 3B2B: macOS Release Artifact Verification Harness — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `pnpm release:mac:verify [path]` — a read-only command that verifies whether a built `.app`/`.dmg` is a customer-ready notarized macOS release (Developer ID on app + sidecar, same Team ID, hardened runtime, expected entitlements, Gatekeeper-accepted app + DMG, stapled), with an `--allow-adhoc` dry-run mode usable now against the 3B1 ad-hoc bundle.

**Architecture:** Three modules under `apps/desktop/scripts/`, mirroring 3B2A's "pure core + thin IO shell" split: `release-mac-verify.parse.mjs` (pure parsers + selection helpers), `release-mac-verify.lib.mjs` (pure `evaluate` + `render`), and `release-mac-verify.mjs` (thin IO shell that spawns tools, mounts the DMG read-only, and wires exit codes). Only the two pure modules are unit-tested; the shell is covered by the manual SMOKE line.

**Tech Stack:** Node ESM (`.mjs`), Vitest (`scripts/**/*.test.mjs`, node env), macOS `codesign` / `spctl` / `xcrun stapler` / `hdiutil`.

**Spec:** `docs/superpowers/specs/2026-05-26-skillbox-slice-3b2b-release-artifact-verification-design.md`

**HARD CONSTRAINTS (do not violate):**
- **No product / schema / RPC / migration changes.** Nothing under `electron/`, `renderer/`, `core-go/`, `shared/`, or `migrations/`. No JSON-RPC contract or DB change.
- **No build / sign / notarize / staple / network / keychain mutation** in the harness. The only side effect is a read-only `hdiutil attach` + matching `detach`.
- Do not touch `package:mac`, `package:mac:unsigned`, `release-mac-check.*`, `electron-builder.yml`, or the entitlements plists.

---

## File Structure

- **Create** `apps/desktop/scripts/release-mac-verify.parse.mjs` — pure: `parseCodesign`, `parseSpctl`, `parseStapler`, `parseEntitlementKeys`, `pickTopLevelApp`, `discoverDmg`.
- **Create** `apps/desktop/scripts/release-mac-verify.lib.mjs` — pure: `evaluate(signals)`, `render(results, missing)`.
- **Create** `apps/desktop/scripts/release-mac-verify.mjs` — thin IO shell (input resolution, read-only DMG mount/detach, tool spawning, exit wiring).
- **Create** `apps/desktop/scripts/release-mac-verify.test.mjs` — Vitest for the two pure modules (all spec §11 cases + fixtures).
- **Modify** `apps/desktop/package.json` — add `release:mac:verify` script.
- **Modify** `SMOKE.md` — add "Release Artifact Verification (Slice 3B2B)"; correct `spctl` assessment types.
- **Modify** `SCAFFOLD.md` — document the post-build gate.

---

## Task 1: Pure parsers and selection helpers

**Files:**
- Create: `apps/desktop/scripts/release-mac-verify.parse.mjs`
- Test: `apps/desktop/scripts/release-mac-verify.test.mjs` (parser section)

- [ ] **Step 1: Write the failing tests** (create the test file with the parser suite)

```js
import { describe, it, expect } from "vitest";
import {
  parseCodesign,
  parseSpctl,
  parseStapler,
  parseEntitlementKeys,
  pickTopLevelApp,
  discoverDmg,
} from "./release-mac-verify.parse.mjs";

const DEVID_APP = `Identifier=com.astraler.skillbox
Format=app bundle with Mach-O thin (arm64)
CodeDirectory v=20500 size=1234 flags=0x10000(runtime) hashes=10+7
Authority=Developer ID Application: Astraler Inc (AB12CD34EF)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
TeamIdentifier=AB12CD34EF`;

const ADHOC_APP = `Identifier=com.astraler.skillbox
Format=app bundle with Mach-O thin (arm64)
CodeDirectory v=20400 size=1234 flags=0x10002(adhoc,runtime) hashes=10+7
Signature=adhoc
TeamIdentifier=not set`;

const NO_RUNTIME_APP = `Identifier=com.astraler.skillbox
CodeDirectory v=20400 size=1234 flags=0x0(none) hashes=10+7
Authority=Developer ID Application: Astraler Inc (AB12CD34EF)
TeamIdentifier=AB12CD34EF`;

const SPCTL_ACCEPTED = `/Volumes/x/Astraler Skillbox.app: accepted
source=Notarized Developer ID
origin=Developer ID Application: Astraler Inc (AB12CD34EF)`;

const SPCTL_REJECTED = `/Volumes/x/Astraler Skillbox.app: rejected
source=no usable signature`;

const ENT_XML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
\t<key>com.apple.security.cs.allow-jit</key>
\t<true/>
\t<key>com.apple.security.cs.allow-unsigned-executable-memory</key>
\t<true/>
</dict>
</plist>`;

describe("parseCodesign", () => {
  it("classifies a Developer ID signature", () => {
    expect(parseCodesign(DEVID_APP)).toEqual({
      adhoc: false,
      developerId: true,
      teamId: "AB12CD34EF",
      hardenedRuntime: true,
    });
  });
  it("classifies an ad-hoc signature with no team id", () => {
    expect(parseCodesign(ADHOC_APP)).toEqual({
      adhoc: true,
      developerId: false,
      teamId: null,
      hardenedRuntime: true,
    });
  });
  it("detects a missing hardened runtime", () => {
    expect(parseCodesign(NO_RUNTIME_APP).hardenedRuntime).toBe(false);
  });
});

describe("parseSpctl", () => {
  it("accepted + notarized source", () => {
    expect(parseSpctl(SPCTL_ACCEPTED, 0)).toEqual({ accepted: true, source: "Notarized Developer ID" });
  });
  it("rejected", () => {
    expect(parseSpctl(SPCTL_REJECTED, 3)).toEqual({ accepted: false, source: "no usable signature" });
  });
});

describe("parseStapler", () => {
  it("stapled when exit 0", () => {
    expect(parseStapler("The validate action worked!", 0)).toEqual({ stapled: true });
  });
  it("not stapled when non-zero", () => {
    expect(parseStapler("does not have a ticket stapled to it.", 65)).toEqual({ stapled: false });
  });
});

describe("parseEntitlementKeys", () => {
  it("extracts <key> names from an entitlements plist/XML blob", () => {
    expect(parseEntitlementKeys(ENT_XML)).toEqual([
      "com.apple.security.cs.allow-jit",
      "com.apple.security.cs.allow-unsigned-executable-memory",
    ]);
  });
  it("returns [] for an empty/unsigned blob", () => {
    expect(parseEntitlementKeys("")).toEqual([]);
  });
});

describe("pickTopLevelApp", () => {
  it("picks the single top-level app, ignoring non-app entries", () => {
    expect(pickTopLevelApp(["Astraler Skillbox.app", "Applications", ".background"])).toEqual({
      app: "Astraler Skillbox.app",
    });
  });
  it("errors on zero apps", () => {
    expect(pickTopLevelApp(["Applications"]).error).toMatch(/no top-level/i);
  });
  it("errors on multiple top-level apps", () => {
    const r = pickTopLevelApp(["A.app", "B.app"]);
    expect(r.error).toMatch(/multiple top-level/i);
  });
});

describe("discoverDmg", () => {
  it("picks the single dist dmg", () => {
    expect(discoverDmg(["Astraler Skillbox-0.1.0-arm64.dmg", "mac-arm64"])).toEqual({
      dmg: "Astraler Skillbox-0.1.0-arm64.dmg",
    });
  });
  it("errors on zero dmgs", () => {
    expect(discoverDmg(["mac-arm64"]).error).toMatch(/no \.dmg/i);
  });
  it("errors on multiple dmgs", () => {
    expect(discoverDmg(["a.dmg", "b.dmg"]).error).toMatch(/multiple \.dmg/i);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-verify.test.mjs`
Expected: FAIL — `Failed to resolve import "./release-mac-verify.parse.mjs"`.

- [ ] **Step 3: Write the parser module**

Create `apps/desktop/scripts/release-mac-verify.parse.mjs`:

```js
/**
 * Pure parsers + selection helpers for the macOS release artifact verifier (Slice 3B2B).
 * NO process / filesystem / env access here — callers pass raw tool text + exit codes.
 */

/** @param {string} text codesign -dvvv stdout+stderr */
export function parseCodesign(text) {
  const t = text ?? "";
  const adhoc = /\bSignature=adhoc\b/.test(t);
  const developerId = /Authority=Developer ID Application/.test(t);
  const m = t.match(/^TeamIdentifier=(.+)$/m);
  const raw = m ? m[1].trim() : null;
  const teamId = !raw || raw === "not set" ? null : raw;
  const hardenedRuntime = /flags=[^\s]*runtime/i.test(t);
  return { adhoc, developerId, teamId, hardenedRuntime };
}

/** @param {string} text @param {number} exitCode spctl assessment */
export function parseSpctl(text, exitCode) {
  const t = text ?? "";
  const m = t.match(/^source=(.+)$/m);
  return { accepted: exitCode === 0, source: m ? m[1].trim() : null };
}

/** @param {string} _text @param {number} exitCode stapler validate */
export function parseStapler(_text, exitCode) {
  return { stapled: exitCode === 0 };
}

/** @param {string} text codesign -d --entitlements :- (or a committed plist) */
export function parseEntitlementKeys(text) {
  const t = text ?? "";
  const keys = new Set();
  for (const mm of t.matchAll(/<key>([^<]+)<\/key>/g)) keys.add(mm[1].trim());
  for (const mm of t.matchAll(/^\s*\[Key\]\s+(\S+)/gm)) keys.add(mm[1].trim());
  return [...keys];
}

/** @param {string[]} entries non-recursive listing of the DMG mount root */
export function pickTopLevelApp(entries) {
  const apps = (entries ?? []).filter((e) => e.endsWith(".app"));
  if (apps.length === 1) return { app: apps[0] };
  if (apps.length === 0) return { error: "no top-level .app found at the DMG root" };
  return { error: `multiple top-level .app bundles at the DMG root: ${apps.join(", ")}` };
}

/** @param {string[]} entries listing of apps/desktop/dist */
export function discoverDmg(entries) {
  const dmgs = (entries ?? []).filter((e) => e.endsWith(".dmg"));
  if (dmgs.length === 1) return { dmg: dmgs[0] };
  if (dmgs.length === 0)
    return { error: "no .dmg found in apps/desktop/dist; build one or pass an explicit path" };
  return { error: `multiple .dmg files in dist: ${dmgs.join(", ")}; pass an explicit path` };
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-verify.test.mjs`
Expected: PASS (parser suites green).

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-verify.parse.mjs apps/desktop/scripts/release-mac-verify.test.mjs
git commit -m "feat(3b2b): pure codesign/spctl/stapler/entitlement parsers + app/dmg selection helpers"
```

---

## Task 2: Pure evaluator and renderer

**Files:**
- Create: `apps/desktop/scripts/release-mac-verify.lib.mjs`
- Test: `apps/desktop/scripts/release-mac-verify.test.mjs` (append evaluator section)

- [ ] **Step 1: Append the failing evaluator tests** to `release-mac-verify.test.mjs`

```js
import { evaluate, render } from "./release-mac-verify.lib.mjs";

const EXPECTED_ENT = {
  app: ["com.apple.security.cs.allow-jit", "com.apple.security.cs.allow-unsigned-executable-memory"],
  sidecar: ["com.apple.security.cs.allow-jit", "com.apple.security.inherit"],
};

/** Build a signals object with sensible "good release" defaults, overridable per test. */
function signals(over = {}) {
  const base = {
    mode: "release",
    expectedTeamId: null,
    expectedEntitlements: EXPECTED_ENT,
    input: { dmgName: "Astraler Skillbox-0.1.0-arm64.dmg", appName: "Astraler Skillbox.app" },
    app: {
      parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true },
      verifyExit: 0,
      entitlementKeys: EXPECTED_ENT.app,
    },
    sidecar: {
      present: true,
      parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true },
      verifyExit: 0,
      entitlementKeys: EXPECTED_ENT.sidecar,
    },
    spctlApp: { accepted: true, source: "Notarized Developer ID" },
    spctlDmg: { accepted: true, source: "Notarized Developer ID" },
    staplerApp: { stapled: true },
    staplerDmg: { stapled: true },
  };
  return { ...base, ...over };
}
const status = (results, id) => results.find((r) => r.id === id)?.status;

describe("evaluate — release mode, fully customer-ready", () => {
  it("all PASS, exit 0", () => {
    const { results, exitCode } = evaluate(signals());
    expect(exitCode).toBe(0);
    expect(results.every((r) => r.status === "PASS")).toBe(true);
  });
});

describe("evaluate — release mode, ad-hoc artifact", () => {
  const adhoc = signals({
    app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
    sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
    spctlApp: { accepted: false, source: "no usable signature" },
    spctlDmg: { accepted: false, source: "no usable signature" },
    staplerApp: { stapled: false },
    staplerDmg: { stapled: false },
  });
  it("FAILs signature/team/gatekeeper/staple, exit 1; entitlements still PASS", () => {
    const { results, exitCode } = evaluate(adhoc);
    expect(exitCode).toBe(1);
    expect(status(results, "APP2")).toBe("FAIL");
    expect(status(results, "SID3")).toBe("FAIL");
    expect(status(results, "TID1")).toBe("FAIL");
    expect(status(results, "GK1")).toBe("FAIL");
    expect(status(results, "GK2")).toBe("FAIL");
    expect(status(results, "ST1")).toBe("FAIL");
    expect(status(results, "ST2")).toBe("FAIL");
    expect(status(results, "ENT1")).toBe("PASS");
    expect(status(results, "ENT2")).toBe("PASS");
  });
});

describe("evaluate — --allow-adhoc, ad-hoc artifact", () => {
  const adhoc = signals({
    mode: "adhoc",
    app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
    sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
    spctlApp: { accepted: false, source: null },
    spctlDmg: { accepted: false, source: null },
    staplerApp: { stapled: false },
    staplerDmg: { stapled: false },
  });
  it("PASSes verify/runtime/entitlements; soft checks INFO; exit 0", () => {
    const { results, exitCode } = evaluate(adhoc);
    expect(exitCode).toBe(0);
    for (const id of ["APP1", "APP3", "SID2", "SID4", "ENT1", "ENT2"]) expect(status(results, id)).toBe("PASS");
    for (const id of ["GK1", "GK2", "ST1", "ST2", "APP4", "SID5", "TID1"]) expect(status(results, id)).toBe("INFO");
  });
});

describe("evaluate — sidecar + runtime + entitlement failures", () => {
  it("missing sidecar FAILs SID1 in release", () => {
    const r = evaluate(signals({ sidecar: { present: false, parsed: null, verifyExit: null, entitlementKeys: [] } }));
    expect(status(r.results, "SID1")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
  it("missing app runtime FAILs APP3 in --allow-adhoc too", () => {
    const r = evaluate(signals({ mode: "adhoc", app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: false }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app } }));
    expect(status(r.results, "APP3")).toBe("FAIL");
    expect(r.exitCode).toBe(1);
  });
  it("app entitlements missing a key FAILs ENT1 in both modes", () => {
    const r = evaluate(signals({ app: { parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: ["com.apple.security.cs.allow-jit"] } }));
    expect(status(r.results, "ENT1")).toBe("FAIL");
  });
  it("extra entitlement keys still PASS (subset semantics)", () => {
    const r = evaluate(signals({ app: { parsed: { adhoc: false, developerId: true, teamId: "AB12CD34EF", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: [...EXPECTED_ENT.app, "com.apple.security.network.client"] } }));
    expect(status(r.results, "ENT1")).toBe("PASS");
  });
});

describe("evaluate — Team ID equality (finding 2)", () => {
  it("FAILs TID1 when app and sidecar teams differ", () => {
    const r = evaluate(signals({ sidecar: { present: true, parsed: { adhoc: false, developerId: true, teamId: "ZZ99ZZ99ZZ", hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar } }));
    expect(status(r.results, "TID1")).toBe("FAIL");
    expect(status(r.results, "APP4")).toBe("PASS");
    expect(status(r.results, "SID5")).toBe("PASS");
  });
  it("PASSes TID1 when equal and no expected env", () => {
    expect(status(evaluate(signals()).results, "TID1")).toBe("PASS");
  });
  it("PASSes TID1 when both match expected env", () => {
    expect(status(evaluate(signals({ expectedTeamId: "AB12CD34EF" })).results, "TID1")).toBe("PASS");
  });
  it("FAILs TID1 when expected env differs", () => {
    expect(status(evaluate(signals({ expectedTeamId: "WRONG12345" })).results, "TID1")).toBe("FAIL");
  });
});

describe("evaluate — DMG-only checks (findings 3, input)", () => {
  it("FAILs GK2 when DMG Gatekeeper rejects", () => {
    const r = evaluate(signals({ spctlDmg: { accepted: false, source: "no usable signature" } }));
    expect(status(r.results, "GK2")).toBe("FAIL");
  });
  it("bare .app input → GK2 and ST2 INFO", () => {
    const r = evaluate(signals({ input: { dmgName: null, appName: "Astraler Skillbox.app" }, spctlDmg: null, staplerDmg: null }));
    expect(status(r.results, "GK2")).toBe("INFO");
    expect(status(r.results, "ST2")).toBe("INFO");
  });
});

describe("render", () => {
  it("groups by category and lists the missing remediations", () => {
    const { results, missing } = evaluate(signals({
      app: { parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.app },
      sidecar: { present: true, parsed: { adhoc: true, developerId: false, teamId: null, hardenedRuntime: true }, verifyExit: 0, entitlementKeys: EXPECTED_ENT.sidecar },
      spctlApp: { accepted: false, source: null }, spctlDmg: { accepted: false, source: null },
      staplerApp: { stapled: false }, staplerDmg: { stapled: false },
    }));
    const out = render(results, missing);
    expect(out).toMatch(/App signature/);
    expect(out).toMatch(/Sidecar \(core\/skillbox-core\)/);
    expect(out).toMatch(/Missing for a customer-ready release:/);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-verify.test.mjs`
Expected: FAIL — `Failed to resolve import "./release-mac-verify.lib.mjs"`.

- [ ] **Step 3: Write the evaluator module**

Create `apps/desktop/scripts/release-mac-verify.lib.mjs`:

```js
/**
 * Pure evaluator + renderer for the macOS release artifact verifier (Slice 3B2B).
 * NO process / filesystem / env access — callers pass parsed signals.
 *
 * @typedef {{ id:string, category:string, status:"PASS"|"FAIL"|"WARN"|"INFO", message:string, remediation?:string }} CheckResult
 */

const SIDECAR_REL = "Contents/Resources/core/skillbox-core";

function shortKey(k) {
  return k.replace(/^com\.apple\.security\.cs\./, "").replace(/^com\.apple\.security\./, "");
}
function missingKeys(expected, actual) {
  const set = new Set(actual ?? []);
  return (expected ?? []).filter((k) => !set.has(k));
}

/** @param {any} s signals @returns {{results:CheckResult[], missing:string[], exitCode:number}} */
export function evaluate(s) {
  const release = s.mode !== "adhoc";
  const soft = release ? "FAIL" : "INFO"; // notarization / stapling / team-id gaps
  const results = [];
  const push = (r) => results.push(r);

  push({
    id: "S1",
    category: "input",
    status: "PASS",
    message: s.input.dmgName
      ? `resolved DMG: ${s.input.dmgName} (top-level app: ${s.input.appName})`
      : `resolved app: ${s.input.appName}`,
  });

  const a = s.app.parsed;

  push(
    s.app.verifyExit === 0
      ? { id: "APP1", category: "app", status: "PASS", message: "codesign --verify --deep --strict" }
      : { id: "APP1", category: "app", status: "FAIL", message: "codesign --verify --deep --strict failed", remediation: "App signature invalid on disk; rebuild/sign via package:mac" }
  );

  if (a.developerId && !a.adhoc)
    push({ id: "APP2", category: "app", status: "PASS", message: "Developer ID Application signature" });
  else if (!release && a.adhoc)
    push({ id: "APP2", category: "app", status: "INFO", message: "ad-hoc signature (dry-run)" });
  else
    push({ id: "APP2", category: "app", status: "FAIL", message: a.adhoc ? "signature is ad-hoc, expected Developer ID Application" : "no Developer ID Application signature", remediation: "App and sidecar must be signed with a Developer ID Application identity (not ad-hoc)" });

  push(
    a.hardenedRuntime
      ? { id: "APP3", category: "app", status: "PASS", message: "hardened runtime enabled" }
      : { id: "APP3", category: "app", status: "FAIL", message: "hardened runtime not enabled", remediation: "Enable hardened runtime (mac.hardenedRuntime: true)" }
  );

  push(
    a.teamId
      ? { id: "APP4", category: "app", status: "PASS", message: "app TeamIdentifier present" }
      : { id: "APP4", category: "app", status: soft, message: release ? "app has no TeamIdentifier" : "app has no TeamIdentifier (ad-hoc)", ...(release ? { remediation: "Sign with a Developer ID identity that carries a Team ID" } : {}) }
  );

  {
    const miss = missingKeys(s.expectedEntitlements.app, s.app.entitlementKeys);
    push(
      miss.length === 0
        ? { id: "ENT1", category: "app", status: "PASS", message: `entitlements include ${s.expectedEntitlements.app.map(shortKey).join(", ")}` }
        : { id: "ENT1", category: "app", status: "FAIL", message: `entitlements missing ${miss.map(shortKey).join(", ")}`, remediation: "App must embed every key from build/entitlements.mac.plist" }
    );
  }

  push(
    s.sidecar.present
      ? { id: "SID1", category: "sidecar", status: "PASS", message: "present" }
      : { id: "SID1", category: "sidecar", status: "FAIL", message: `sidecar missing at ${SIDECAR_REL}`, remediation: "Bundle the sidecar (mac.binaries / extraResources)" }
  );

  if (s.sidecar.present) {
    const d = s.sidecar.parsed ?? { adhoc: false, developerId: false, teamId: null, hardenedRuntime: false };

    push(
      s.sidecar.verifyExit === 0
        ? { id: "SID2", category: "sidecar", status: "PASS", message: "codesign --verify --strict" }
        : { id: "SID2", category: "sidecar", status: "FAIL", message: "codesign --verify --strict failed", remediation: "Sidecar signature invalid; ensure mac.binaries reaches it" }
    );

    if (d.developerId && !d.adhoc)
      push({ id: "SID3", category: "sidecar", status: "PASS", message: "Developer ID Application signature" });
    else if (!release && d.adhoc)
      push({ id: "SID3", category: "sidecar", status: "INFO", message: "ad-hoc signature (dry-run)" });
    else
      push({ id: "SID3", category: "sidecar", status: "FAIL", message: d.adhoc ? "signature is ad-hoc, expected Developer ID Application" : "no Developer ID Application signature", remediation: "App and sidecar must be signed with a Developer ID Application identity (not ad-hoc)" });

    push(
      d.hardenedRuntime
        ? { id: "SID4", category: "sidecar", status: "PASS", message: "hardened runtime enabled" }
        : { id: "SID4", category: "sidecar", status: "FAIL", message: "hardened runtime not enabled", remediation: "Sidecar must be signed with hardened runtime" }
    );

    push(
      d.teamId
        ? { id: "SID5", category: "sidecar", status: "PASS", message: "sidecar TeamIdentifier present" }
        : { id: "SID5", category: "sidecar", status: soft, message: release ? "sidecar has no TeamIdentifier" : "sidecar has no TeamIdentifier (ad-hoc)", ...(release ? { remediation: "Sign the sidecar with a Developer ID identity that carries a Team ID" } : {}) }
    );

    const miss = missingKeys(s.expectedEntitlements.sidecar, s.sidecar.entitlementKeys);
    push(
      miss.length === 0
        ? { id: "ENT2", category: "sidecar", status: "PASS", message: `entitlements include ${s.expectedEntitlements.sidecar.map(shortKey).join(", ")}` }
        : { id: "ENT2", category: "sidecar", status: "FAIL", message: `entitlements missing ${miss.map(shortKey).join(", ")}`, remediation: "Sidecar must embed every key from build/entitlements.mac.inherit.plist" }
    );
  }

  // TID1 — app and sidecar Team ID present + equal (+ match expected env if set)
  {
    const at = a.teamId;
    const st = s.sidecar.present ? s.sidecar.parsed?.teamId ?? null : null;
    let status, message, remediation;
    if (!at || !st) {
      status = soft;
      message = release
        ? "app/sidecar TeamIdentifier not both present; cannot confirm a single team"
        : "no TeamIdentifier (ad-hoc); team equality not applicable";
      if (release) remediation = "App and sidecar must share one Team ID";
    } else if (at !== st) {
      status = "FAIL";
      message = `app TeamIdentifier (${at}) != sidecar (${st})`;
      remediation = "App and sidecar must be signed by the same Team ID";
    } else if (s.expectedTeamId && at !== s.expectedTeamId) {
      status = "FAIL";
      message = `TeamIdentifier ${at} != expected ${s.expectedTeamId}`;
      remediation = `Sign with the expected Team ID (${s.expectedTeamId})`;
    } else {
      status = "PASS";
      message = s.expectedTeamId ? `app and sidecar Team ID ${at} (matches expected)` : `app and sidecar share Team ID ${at}`;
    }
    push({ id: "TID1", category: "identity", status, message, ...(remediation ? { remediation } : {}) });
  }

  // GK1 — app Gatekeeper (spctl -t exec)
  if (s.spctlApp.accepted && /Notarized Developer ID/.test(s.spctlApp.source ?? ""))
    push({ id: "GK1", category: "gatekeeper", status: "PASS", message: "spctl -t exec accepted (Notarized Developer ID)" });
  else
    push({ id: "GK1", category: "gatekeeper", status: soft, message: release ? `spctl -t exec (app) ${s.spctlApp.accepted ? "accepted but not notarized" : "rejected"}` : "spctl -t exec (app) not notarized (dry-run)", ...(release ? { remediation: "Artifact must be notarized (spctl: source=Notarized Developer ID)" } : {}) });

  // GK2 — DMG Gatekeeper (spctl -t open), only when a DMG was supplied
  if (s.spctlDmg) {
    if (s.spctlDmg.accepted && /Notarized Developer ID/.test(s.spctlDmg.source ?? ""))
      push({ id: "GK2", category: "gatekeeper", status: "PASS", message: "spctl -t open accepted (Notarized Developer ID)" });
    else
      push({ id: "GK2", category: "gatekeeper", status: soft, message: release ? `spctl -t open (dmg) ${s.spctlDmg.accepted ? "accepted but not notarized" : "rejected"}` : "spctl -t open (dmg) not notarized (dry-run)", ...(release ? { remediation: "DMG container must pass Gatekeeper (notarized)" } : {}) });
  } else {
    push({ id: "GK2", category: "gatekeeper", status: "INFO", message: "DMG Gatekeeper not checked (no .dmg input)" });
  }

  // ST1 — app stapled
  push(
    s.staplerApp.stapled
      ? { id: "ST1", category: "staple", status: "PASS", message: "app stapled" }
      : { id: "ST1", category: "staple", status: soft, message: release ? "app has no stapled ticket" : "app has no stapled ticket (dry-run)", ...(release ? { remediation: "Staple the notarization ticket to the app" } : {}) }
  );

  // ST2 — dmg stapled (only when a DMG was supplied)
  if (s.staplerDmg)
    push(
      s.staplerDmg.stapled
        ? { id: "ST2", category: "staple", status: "PASS", message: "dmg stapled" }
        : { id: "ST2", category: "staple", status: soft, message: release ? "dmg has no stapled ticket" : "dmg has no stapled ticket (dry-run)", ...(release ? { remediation: "Staple the notarization ticket to the dmg" } : {}) }
    );
  else push({ id: "ST2", category: "staple", status: "INFO", message: "DMG stapling not checked (no .dmg input)" });

  const fails = results.filter((r) => r.status === "FAIL");
  const seen = new Set();
  const missing = [];
  for (const r of fails) {
    const key = r.remediation ?? r.message;
    if (!seen.has(key)) {
      seen.add(key);
      missing.push(key);
    }
  }
  return { results, missing, exitCode: fails.length > 0 ? 1 : 0 };
}

const CATEGORY_ORDER = ["input", "app", "sidecar", "identity", "gatekeeper", "staple"];
const CATEGORY_LABEL = {
  input: "Input",
  app: "App signature",
  sidecar: "Sidecar (core/skillbox-core)",
  identity: "Identity",
  gatekeeper: "Gatekeeper",
  staple: "Stapling",
};

/** @param {CheckResult[]} results @param {string[]} missing */
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
    lines.push("Missing for a customer-ready release:");
    for (const m of missing) lines.push(`  - ${m}`);
  }
  return lines.join("\n");
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-verify.test.mjs`
Expected: PASS (all parser + evaluator + render suites green).

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-verify.lib.mjs apps/desktop/scripts/release-mac-verify.test.mjs
git commit -m "feat(3b2b): pure evaluator + renderer for artifact verification (release + --allow-adhoc)"
```

---

## Task 3: Thin IO shell + npm script

**Files:**
- Create: `apps/desktop/scripts/release-mac-verify.mjs`
- Modify: `apps/desktop/package.json` (add `release:mac:verify` to `scripts`)

This module is not unit-tested (it spawns real tools and mounts DMGs); it is covered by the manual SMOKE line and the verification gates in Task 5.

- [ ] **Step 1: Write the IO shell**

Create `apps/desktop/scripts/release-mac-verify.mjs`:

```js
/**
 * macOS Release Artifact Verification Harness (Slice 3B2B).
 * Inspects a built .app/.dmg and reports whether it is a customer-ready notarized
 * release. Read-only except for a read-only DMG mount/detach. NEVER builds, signs,
 * notarizes, staples, calls the network, or mutates the keychain.
 *
 * Run from apps/desktop/:
 *   pnpm release:mac:verify [path]            # release mode (default)
 *   pnpm release:mac:verify --allow-adhoc [path]
 * Optional: SKILLBOX_EXPECTED_TEAM_ID=ABCDE12345 pins both app + sidecar to that team.
 */
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  parseCodesign,
  parseSpctl,
  parseStapler,
  parseEntitlementKeys,
  pickTopLevelApp,
  discoverDmg,
} from "./release-mac-verify.parse.mjs";
import { evaluate, render } from "./release-mac-verify.lib.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, ".."); // apps/desktop
const SIDECAR_REL = "Contents/Resources/core/skillbox-core";

const argv = process.argv.slice(2);
const mode = argv.includes("--allow-adhoc") ? "adhoc" : "release";
const pathArg = argv.find((x) => !x.startsWith("--")) ?? null;
const expectedTeamId = (process.env.SKILLBOX_EXPECTED_TEAM_ID || "").trim() || null;

/** Spawn read-only; capture stdout+stderr+exit. Never throws on non-zero. */
function run(cmd, args) {
  const r = spawnSync(cmd, args, { encoding: "utf8" });
  return { text: `${r.stdout ?? ""}\n${r.stderr ?? ""}`, code: typeof r.status === "number" ? r.status : 1 };
}

let mountPoint = null;
function detach() {
  if (!mountPoint) return;
  let r = run("/usr/bin/hdiutil", ["detach", mountPoint]);
  if (r.code !== 0) r = run("/usr/bin/hdiutil", ["detach", "-force", mountPoint]);
  try {
    rmSync(mountPoint, { recursive: true, force: true });
  } catch {
    /* mountpoint dir removed by detach */
  }
  if (r.code !== 0) console.error(`WARNING: failed to detach ${mountPoint}; detach it manually.`);
  mountPoint = null;
}

/** Print a single S1 FAIL and exit 1 (used before any artifact checks run). */
function failInput(message) {
  detach();
  console.log(render([{ id: "S1", category: "input", status: "FAIL", message }], [message]));
  process.exit(1);
}

function expectedKeysFrom(rel) {
  const p = path.join(desktop, rel);
  return existsSync(p) ? parseEntitlementKeys(readFileSync(p, "utf8")) : [];
}

function gatherSigning(target, deep) {
  const dvvv = run("/usr/bin/codesign", ["-dvvv", target]);
  const verifyArgs = deep
    ? ["--verify", "--deep", "--strict", "--verbose=2", target]
    : ["--verify", "--strict", "--verbose=2", target];
  const verify = run("/usr/bin/codesign", verifyArgs);
  const ent = run("/usr/bin/codesign", ["-d", "--entitlements", ":-", target]);
  return { parsed: parseCodesign(dvvv.text), verifyExit: verify.code, entitlementKeys: parseEntitlementKeys(ent.text) };
}

// --- Resolve input ---
let appPath = null;
let appName = null;
let dmgPath = null;
let dmgName = null;

if (pathArg) {
  const abs = path.resolve(process.cwd(), pathArg);
  if (!existsSync(abs)) failInput(`input path does not exist: ${pathArg}`);
  if (abs.endsWith(".app")) {
    appPath = abs;
    appName = path.basename(abs);
  } else if (abs.endsWith(".dmg")) {
    dmgPath = abs;
    dmgName = path.basename(abs);
  } else {
    failInput(`input is neither a .app nor a .dmg: ${pathArg}`);
  }
} else {
  const dist = path.join(desktop, "dist");
  const d = discoverDmg(existsSync(dist) ? readdirSync(dist) : []);
  if (d.error) failInput(d.error);
  dmgPath = path.join(dist, d.dmg);
  dmgName = d.dmg;
}

try {
  if (dmgPath) {
    mountPoint = mkdtempSync(path.join(tmpdir(), "skillbox-verify-"));
    const att = run("/usr/bin/hdiutil", ["attach", "-readonly", "-nobrowse", "-mountpoint", mountPoint, dmgPath]);
    if (att.code !== 0) failInput(`failed to mount DMG read-only (hdiutil attach exit ${att.code})`);
    const pick = pickTopLevelApp(readdirSync(mountPoint)); // non-recursive: ignores nested helper apps
    if (pick.error) failInput(pick.error);
    appPath = path.join(mountPoint, pick.app);
    appName = pick.app;
  }

  const app = gatherSigning(appPath, true);
  const sidePath = path.join(appPath, SIDECAR_REL);
  const sidecar = existsSync(sidePath)
    ? { present: true, ...gatherSigning(sidePath, false) }
    : { present: false, parsed: null, verifyExit: null, entitlementKeys: [] };

  const spA = run("/usr/sbin/spctl", ["-a", "-vvv", "-t", "exec", appPath]);
  const stA = run("/usr/bin/xcrun", ["stapler", "validate", appPath]);
  let spctlDmg = null;
  let staplerDmg = null;
  if (dmgPath) {
    const spD = run("/usr/sbin/spctl", ["-a", "-vvv", "-t", "open", dmgPath]);
    const stD = run("/usr/bin/xcrun", ["stapler", "validate", dmgPath]);
    spctlDmg = parseSpctl(spD.text, spD.code);
    staplerDmg = parseStapler(stD.text, stD.code);
  }

  const { results, missing, exitCode } = evaluate({
    mode,
    expectedTeamId,
    expectedEntitlements: {
      app: expectedKeysFrom("build/entitlements.mac.plist"),
      sidecar: expectedKeysFrom("build/entitlements.mac.inherit.plist"),
    },
    input: { dmgName, appName },
    app,
    sidecar,
    spctlApp: parseSpctl(spA.text, spA.code),
    spctlDmg,
    staplerApp: parseStapler(stA.text, stA.code),
    staplerDmg,
  });

  console.log(render(results, missing));
  process.exitCode = exitCode;
} finally {
  detach();
}
```

- [ ] **Step 2: Add the npm script** — edit `apps/desktop/package.json`, in `"scripts"`, after the `"release:mac:check"` line add:

```json
    "release:mac:verify": "node scripts/release-mac-verify.mjs"
```

(Ensure the preceding line ends with a comma. Do not change any other script.)

- [ ] **Step 3: Smoke the shell wiring with a bogus path (no real artifact needed)**

Run: `cd apps/desktop && pnpm release:mac:verify /tmp/does-not-exist.app; echo "exit=$?"`
Expected: prints `Input` + a `FAIL  input path does not exist: …` line and `exit=1`.

- [ ] **Step 4: Smoke auto-discovery with an empty dist (no DMG present)**

Run: `cd apps/desktop && pnpm release:mac:verify; echo "exit=$?"`
Expected (when `dist/` has no `.dmg`): `FAIL  no .dmg found in apps/desktop/dist…` and `exit=1`. (If a `.dmg` already exists it will instead try to verify it — that is fine; the real artifact run is Task 5.)

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/scripts/release-mac-verify.mjs apps/desktop/package.json
git commit -m "feat(3b2b): IO shell for release:mac:verify (read-only DMG mount, tool spawning, exit wiring)"
```

---

## Task 4: Documentation (SMOKE.md, SCAFFOLD.md)

**Files:**
- Modify: `SMOKE.md`
- Modify: `SCAFFOLD.md`

- [ ] **Step 1: Correct the `spctl` assessment type in the 3B2 smoke** — in `SMOKE.md`, in the "Signed + Notarized Smoke (Slice 3B2 …)" code block, replace this line:

```sh
spctl -a -vvv -t open "$APP"                # expect: accepted, source=Notarized Developer ID
```

with:

```sh
spctl -a -vvv -t exec "$APP"                # app: expect accepted, source=Notarized Developer ID
spctl -a -vvv -t open "$DMG"                # dmg container: expect accepted, source=Notarized Developer ID
```

- [ ] **Step 2: Add the 3B2B smoke section** — in `SMOKE.md`, immediately after the "## Release Preflight (Slice 3B2A)" section (before the `## Notes` section), insert:

```markdown
## Release Artifact Verification (Slice 3B2B)

Post-build, read-only verifier. No Apple credentials required for `--allow-adhoc`. The only
side effect is a read-only DMG mount/detach; it never builds, signs, notarizes, staples, calls
the network, or mutates the keychain.

### Dry-run against the 3B1 ad-hoc bundle
- [ ] Build the ad-hoc bundle (see "Signed Packaging Dry-Run (Slice 3B1)") so a `.dmg` exists under `apps/desktop/dist/`.
- [ ] `(cd apps/desktop && pnpm release:mac:verify --allow-adhoc); echo "exit=$?"`
  Expected: `exit=0`. App + sidecar `codesign --verify` / hardened-runtime / **entitlements (ENT1/ENT2)** are PASS; Gatekeeper/stapling/Team-ID lines are INFO.
- [ ] Release mode against the same artifact: `(cd apps/desktop && pnpm release:mac:verify); echo "exit=$?"`
  Expected: `exit=1`, FAILing on Developer ID signature, Team-ID equality, Gatekeeper (app + dmg), and stapling. The "Missing for a customer-ready release:" list is non-empty.
- [ ] No leftover mount after either run: `hdiutil info | grep -i skillbox-verify || echo "clean"` → `clean`.

### Release mode against a real notarized DMG (Slice 3B2 — needs credentials)
- [ ] After a real `pnpm package:mac`, run `(cd apps/desktop && pnpm release:mac:verify dist/"Astraler Skillbox-0.1.0-arm64.dmg"); echo "exit=$?"` → `exit=0`, all checks PASS.
- [ ] (Optional) pin the team: `SKILLBOX_EXPECTED_TEAM_ID=<TEAMID> pnpm release:mac:verify …`.
```

- [ ] **Step 3: Add the post-build gate to SCAFFOLD.md** — in `SCAFFOLD.md`, immediately after the "### Release preflight (Slice 3B2A)" subsection (before the `---` that precedes "## Release Tag"), insert:

```markdown
### Release artifact verification (Slice 3B2B)
- `pnpm release:mac:verify [path]` — read-only **post-build** gate (the bookend to `release:mac:check`).
  Verifies a built `.app`/`.dmg` is customer-ready: Developer ID signature on the app **and** the
  nested sidecar, a single shared Team ID, hardened runtime, the expected entitlements, Gatekeeper
  acceptance of the app (`spctl -t exec`) and the DMG (`spctl -t open`), and a stapled ticket on both.
- Input: an explicit `.app`, an explicit `.dmg`, or (no arg) the single `apps/desktop/dist/*.dmg`
  (multiple → pass an explicit path). A `.dmg` is mounted **read-only**; the single top-level `.app`
  is verified (nested Electron helper apps are ignored), then unmounted.
- `--allow-adhoc` verifies the 3B1 ad-hoc dry-run bundle (signature/runtime/entitlements PASS;
  notarization/stapling/Team-ID reported INFO). `SKILLBOX_EXPECTED_TEAM_ID` optionally pins the team.
- It never builds, signs, notarizes, staples, calls Apple, or mutates the keychain. Run it AFTER
  `pnpm package:mac`. See SMOKE.md → "Release Artifact Verification (Slice 3B2B)".
```

- [ ] **Step 4: Commit**

```bash
git add SMOKE.md SCAFFOLD.md
git commit -m "docs(3b2b): document release:mac:verify gate; fix spctl assessment types in SMOKE"
```

---

## Task 5: Verification gates

No code changes — run every gate and confirm output. Fix and re-commit if any fails.

- [ ] **Step 1: Targeted unit tests**

Run: `cd apps/desktop && pnpm exec vitest run scripts/release-mac-verify.test.mjs`
Expected: all suites PASS.

- [ ] **Step 2: Full frontend test suite**

Run: `cd apps/desktop && pnpm test`
Expected: PASS, including the existing `release-mac-check.test.mjs` (no regressions).

- [ ] **Step 3: Typecheck + contract drift**

Run: `cd apps/desktop && pnpm typecheck && pnpm check:contracts-drift`
Expected: both succeed; contracts report **no drift** (proves no schema/contract change).

- [ ] **Step 4: Go test suite (proves no core change)**

Run: `cd core-go && go test ./...`
Expected: PASS.

- [ ] **Step 5: Production build**

Run: `cd apps/desktop && pnpm build`
Expected: `electron-vite build` succeeds.

- [ ] **Step 6: Build the 3B1 ad-hoc artifact (acceptance fixture)**

Run:
```bash
cd apps/desktop && pnpm build:core && pnpm build && \
  pnpm exec electron-builder --mac dmg -c.mac.identity=- -c.mac.notarize=false
```
Expected: produces `apps/desktop/dist/Astraler Skillbox-0.1.0-arm64.dmg` (ad-hoc signed). (macOS dev machine required.)

- [ ] **Step 7: `--allow-adhoc` passes on the ad-hoc artifact**

Run: `cd apps/desktop && pnpm release:mac:verify --allow-adhoc; echo "exit=$?"`
Expected: `exit=0`; ENT1/ENT2 PASS; Gatekeeper/stapling/Team-ID INFO.

- [ ] **Step 8: Release mode fails on the ad-hoc artifact**

Run: `cd apps/desktop && pnpm release:mac:verify; echo "exit=$?"`
Expected: `exit=1`; FAILs on Developer ID (APP2/SID3), TID1, GK1, GK2, ST1, ST2; ENT1/ENT2 still PASS.

- [ ] **Step 9: DMG mount cleanup**

Run: `hdiutil info | grep -i "skillbox-verify" || echo "clean"`
Expected: `clean` (no leaked read-only mount from Steps 7–8).

- [ ] **Step 10: No forbidden side effects (review)**

Confirm by reading `release-mac-verify.mjs`: it spawns only `codesign`, `spctl`, `xcrun stapler`, and `hdiutil attach/detach`; there is **no** `codesign -s`, `notarytool`, `stapler staple`, `electron-builder`, `security`, network call, or any write outside the temp mountpoint.

- [ ] **Step 11: Final working tree is clean**

Run: `git status --porcelain` (and remove the built `dist/` artifact if your checkout tracks artifact hygiene — `dist/` is gitignored, so this should already be clean of tracked changes).
Expected: no uncommitted tracked changes; all plan work is committed across Tasks 1–4.

---

## Out of Scope — MUST NOT do in this plan

- No product / renderer / Electron-main / `core-go` / schema / migration / JSON-RPC contract changes.
- No changes to `package:mac`, `package:mac:unsigned`, `release-mac-check.*`, `electron-builder.yml`, or the entitlements plists (the verifier only **reads** the plists).
- No real signing, notarization, stapling, network calls, or keychain mutation in the harness.
- No CI automation, auto-update, universal binary, Windows/Linux, or `.pkg`/Mac App Store work.

## Self-Review Notes (spec coverage)

- §2 input resolution → Task 3 (explicit `.app`/`.dmg`, auto-discover, zero/multiple) + Task 1 `discoverDmg`/`pickTopLevelApp` tests.
- §3 read-only mount + top-level app + reliable detach (force fallback) → Task 3 shell + Task 1 `pickTopLevelApp`.
- §4 architecture (parse / lib / shell split) → Tasks 1–3.
- §5 signals/commands (exit-code primary, stderr captured, `-t exec` app / `-t open` dmg, entitlements) → Task 3 `run()` + commands; Task 1 parsers.
- §6 check table (S1, APP1–4, ENT1, SID1–5, ENT2, TID1, GK1, GK2, ST1, ST2) → Task 2 `evaluate` + Task 2 tests.
- §7 modes (release vs `--allow-adhoc`; ENT FAIL-in-both) → Task 2 `soft` handling + tests.
- §8 Team ID equality + `SKILLBOX_EXPECTED_TEAM_ID` → Task 2 TID1 + tests; Task 3 env read.
- §11 every fixture/case → Task 1 + Task 2 tests.
- §10 docs → Task 4. §12 acceptance → Task 5 gates.
