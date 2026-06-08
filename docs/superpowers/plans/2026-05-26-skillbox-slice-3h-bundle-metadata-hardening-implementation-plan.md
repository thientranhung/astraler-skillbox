# Slice 3H — Bundle Metadata Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Treat commit steps as PM checkpoints only; do not run `git add` or `git commit` unless explicitly instructed by the PM.

**Goal:** Make macOS release artifacts more customer-safe without Apple credentials by pinning deterministic bundle/release metadata (`copyright`, `artifactName`) in `electron-builder.yml` and gating those values with new FAIL-level checks (D6/D7) in `release:mac:check`.

**Architecture:** Add two top-level keys to `electron-builder.yml`. Add a pure `checkBundleMetadata(config)` evaluator to `release-mac-check.lib.mjs` (sibling to the existing `checkConfig` D1–D5 checks) and wire it into `evaluate`. Update the in-repo test fixture so existing `evaluate` tests stay green now that two more FAIL-capable checks exist. Update the three live docs (RELEASE/SMOKE/SCAFFOLD) that hard-code the old space-containing DMG basename. No script logic changes — downstream DMG selection is already name-agnostic (snapshot-diff / auto-discovery), so this is verified, not modified.

**Tech Stack:** electron-builder (YAML config), Node ESM scripts (`.mjs`), Vitest, js-yaml.

---

## Decisions locked by PM (do not deviate)

- Artifact template (exact literal stored in config): `astraler-skillbox-${version}-${arch}.${ext}`
- Copyright string (exact): `Copyright (c) 2026 Astraler`
- **No** real app icon in this slice (stays Electron default; becomes Slice 3I after artwork).
- **No** `extendInfo` keys (none are required by electron-builder for this slice).
- **No** `productName` / bundle-name change — keep `Astraler Skillbox` and `Astraler Skillbox.app`.
- **No** version-policy change — `package.json` stays `0.1.0`.

## Non-goals

- No icon generation/wiring, no signing/notarization/keychain/upload changes, no release-full path changes.
- Do **not** edit historical plans/specs under `docs/superpowers/plans/**` or `docs/superpowers/specs/**` — they are frozen records.
- Do **not** rewrite synthetic test fixtures in `release-mac-manifest.test.mjs`, `release-mac-dry-run.test.mjs`, `release-mac-verify.test.mjs` that use the old basename as arbitrary input strings — they remain valid (Task 4 confirms the suite stays green).

## Why downstream scripts stay compatible (context, not a task)

The new DMG basename (`astraler-skillbox-0.1.0-arm64.dmg`) flows through every consumer without code change because none of them hard-codes the name:

- `release:mac:dry-run` / `release:mac:full` select the produced DMG via `selectChangedDmg` (before/after `dist/*.dmg` snapshot diff). See `scripts/release-mac-full.lib.mjs` (re-exported by `release-mac-dry-run.lib.mjs:10`).
- `release:mac:verify` auto-discovers the single `dist/*.dmg` (or takes an explicit path).
- `release:mac:manifest` takes an explicit path argument.
- `release:mac:dmg-smoke` auto-discovers the DMG and asserts the **`.app` bundle name** (`EXPECTED_BUNDLE = "Astraler Skillbox.app"` in `scripts/release-mac-dmg-smoke.lib.mjs:8`), which this slice does **not** change.

Only docs and synthetic test fixtures contain the literal old basename.

---

## Lead Review Checkpoint (BEFORE implementation)

- [ ] **Stop and get lead sign-off on this plan before writing any code.** Confirm with the lead:
  - The exact artifact template `astraler-skillbox-${version}-${arch}.${ext}` and copyright `Copyright (c) 2026 Astraler` are final.
  - D6 and D7 must be **FAIL** (not WARN), so the live `release:mac:check` exit code goes non-zero if the config drifts from the pinned values.
  - Implementing D6/D7 as FAIL-capable checks requires updating the shared `GOOD_CONFIG` fixture (Task 2, Step 1) so the existing `evaluate` exit-code tests stay green — confirm this fixture edit is acceptable.
  - Live docs `RELEASE.md`, `SMOKE.md`, `SCAFFOLD.md` will be rewritten to the new basename; historical plans/specs are intentionally left untouched.
- [ ] Only proceed to Task 1 after explicit approval.

---

## Task 1: Pin `copyright` and `artifactName` in electron-builder.yml

**Files:**
- Modify: `apps/desktop/electron-builder.yml:1-4` (insert two top-level keys after `productName`)

- [ ] **Step 1: Add the two top-level metadata keys**

The current head of the file is:

```yaml
appId: com.astraler.skillbox
productName: Astraler Skillbox
directories:
  output: dist
```

Change it to (insert `copyright` and `artifactName` as top-level keys, siblings of `productName`):

```yaml
appId: com.astraler.skillbox
productName: Astraler Skillbox
copyright: Copyright (c) 2026 Astraler
artifactName: astraler-skillbox-${version}-${arch}.${ext}
directories:
  output: dist
```

Leave the entire `mac:` block and everything else unchanged. The `${version}`/`${arch}`/`${ext}` tokens are electron-builder build-time placeholders stored verbatim in the config — do **not** expand them.

- [ ] **Step 2: Confirm the YAML still parses and the keys are present**

Run:

```bash
cd apps/desktop && grep -n "^copyright:\|^artifactName:" electron-builder.yml
```

Expected (exactly two matching lines; `grep -n` prefixes line numbers):

```
3:copyright: Copyright (c) 2026 Astraler
4:artifactName: astraler-skillbox-${version}-${arch}.${ext}
```

- [ ] **Step 3: Confirm `release:mac:check` still loads the config (no parse error)**

Run:

```bash
cd apps/desktop && pnpm release:mac:check; echo "exit=$?"
```

Expected: the command runs to completion and prints the report (it loads `electron-builder.yml` via js-yaml; a malformed YAML would throw). D6/D7 do not exist yet, so the "electron-builder config" section still shows only D1–D5. Exit code reflects the host's credential state (it may be non-zero off a signing machine — that is pre-existing and unrelated to this change).

- [ ] **Step 4: PM checkpoint**

Stop and summarize the diff for `apps/desktop/electron-builder.yml`. Do not stage or commit.

---

## Task 2: Add D6/D7 bundle-metadata checks to release:mac:check (TDD)

**Files:**
- Modify: `apps/desktop/scripts/release-mac-check.lib.mjs` (add `checkBundleMetadata`, wire into `evaluate`)
- Test: `apps/desktop/scripts/release-mac-check.test.mjs` (new `describe` block; extend `GOOD_CONFIG`)

- [ ] **Step 1: Update the shared `GOOD_CONFIG` fixture so existing `evaluate` tests stay green**

`GOOD_CONFIG` (currently at `release-mac-check.test.mjs:185-192`) only has a `mac` block. Once `evaluate` runs D6/D7, the fixture must also carry the pinned top-level values, or the `evaluate` exit-code tests (`release-mac-check.test.mjs:279`, `:289`) will break.

Change the fixture from:

```js
const GOOD_CONFIG = {
  mac: {
    hardenedRuntime: true,
    notarize: true,
    binaries: ["Contents/Resources/core/skillbox-core"],
    target: [{ target: "dmg", arch: ["arm64"] }],
  },
};
```

to (add the two top-level keys; **use double quotes**, never backticks, so `${version}` stays literal):

```js
const GOOD_CONFIG = {
  artifactName: "astraler-skillbox-${version}-${arch}.${ext}",
  copyright: "Copyright (c) 2026 Astraler",
  mac: {
    hardenedRuntime: true,
    notarize: true,
    binaries: ["Contents/Resources/core/skillbox-core"],
    target: [{ target: "dmg", arch: ["arm64"] }],
  },
};
```

- [ ] **Step 2: Add the `checkBundleMetadata` import and a failing test block**

Add `checkBundleMetadata` to the import list at the top of `release-mac-check.test.mjs` (the existing block at lines 2-14, alongside `checkConfig`):

```js
  checkConfig,
  checkBundleMetadata,
  checkSidecar,
```

Then add this new `describe` block immediately after the `checkConfig (D1–D5)` block (after `release-mac-check.test.mjs:219`):

```js
describe("checkBundleMetadata (D6/D7)", () => {
  const ids = (rows) => Object.fromEntries(rows.map((r) => [r.id, r.status]));

  it("passes the pinned artifactName + copyright", () => {
    expect(ids(checkBundleMetadata(GOOD_CONFIG))).toEqual({ D6: "PASS", D7: "PASS" });
  });

  it("fails D6 when artifactName is missing", () => {
    const cfg = { ...GOOD_CONFIG, artifactName: undefined };
    expect(ids(checkBundleMetadata(cfg)).D6).toBe("FAIL");
  });

  it("fails D6 when artifactName contains whitespace", () => {
    const cfg = { ...GOOD_CONFIG, artifactName: "Astraler Skillbox-${version}-${arch}.${ext}" };
    expect(ids(checkBundleMetadata(cfg)).D6).toBe("FAIL");
  });

  it("fails D6 when artifactName is not the exact pinned template", () => {
    const cfg = { ...GOOD_CONFIG, artifactName: "skillbox-${version}.${ext}" };
    expect(ids(checkBundleMetadata(cfg)).D6).toBe("FAIL");
  });

  it("fails D7 when copyright is missing", () => {
    const cfg = { ...GOOD_CONFIG, copyright: undefined };
    expect(ids(checkBundleMetadata(cfg)).D7).toBe("FAIL");
  });

  it("fails D7 when copyright is not the exact pinned string", () => {
    const cfg = { ...GOOD_CONFIG, copyright: "Copyright 2026 Astraler" };
    expect(ids(checkBundleMetadata(cfg)).D7).toBe("FAIL");
  });
});
```

- [ ] **Step 3: Run the new tests to verify they fail**

Run:

```bash
cd apps/desktop && pnpm test -- release-mac-check.test.mjs
```

Expected: FAIL — `checkBundleMetadata` is not exported yet (Vitest will report an ESM missing-export/import error).

- [ ] **Step 4: Implement `checkBundleMetadata` in the lib**

In `release-mac-check.lib.mjs`, add this function immediately after `checkConfig` (after the closing brace at `release-mac-check.lib.mjs:216`). **Use double-quoted constants** so the placeholder tokens stay literal:

```js
const EXPECTED_ARTIFACT_NAME = "astraler-skillbox-${version}-${arch}.${ext}";
const EXPECTED_COPYRIGHT = "Copyright (c) 2026 Astraler";

/** @param {any} config */
export function checkBundleMetadata(config) {
  const out = [];
  const artifactName = config && config.artifactName;
  const copyright = config && config.copyright;

  if (!isSet(artifactName)) {
    out.push({ id: "D6", category: "config", status: "FAIL", message: "artifactName is not set", remediation: `Set artifactName: ${EXPECTED_ARTIFACT_NAME} in electron-builder.yml` });
  } else if (/\s/.test(artifactName)) {
    out.push({ id: "D6", category: "config", status: "FAIL", message: `artifactName contains whitespace (got ${JSON.stringify(artifactName)})`, remediation: `Set artifactName: ${EXPECTED_ARTIFACT_NAME} (no spaces) in electron-builder.yml` });
  } else if (artifactName !== EXPECTED_ARTIFACT_NAME) {
    out.push({ id: "D6", category: "config", status: "FAIL", message: `artifactName is ${JSON.stringify(artifactName)}, expected ${EXPECTED_ARTIFACT_NAME}`, remediation: `Set artifactName: ${EXPECTED_ARTIFACT_NAME} in electron-builder.yml` });
  } else {
    out.push({ id: "D6", category: "config", status: "PASS", message: `artifactName: ${EXPECTED_ARTIFACT_NAME}` });
  }

  if (!isSet(copyright)) {
    out.push({ id: "D7", category: "config", status: "FAIL", message: "copyright is not set", remediation: `Set copyright: ${EXPECTED_COPYRIGHT} in electron-builder.yml` });
  } else if (copyright !== EXPECTED_COPYRIGHT) {
    out.push({ id: "D7", category: "config", status: "FAIL", message: `copyright is ${JSON.stringify(copyright)}, expected ${JSON.stringify(EXPECTED_COPYRIGHT)}`, remediation: `Set copyright to exactly "${EXPECTED_COPYRIGHT}" in electron-builder.yml` });
  } else {
    out.push({ id: "D7", category: "config", status: "PASS", message: `copyright: ${EXPECTED_COPYRIGHT}` });
  }

  return out;
}
```

- [ ] **Step 5: Wire `checkBundleMetadata` into `evaluate`**

In `release-mac-check.lib.mjs`, the `evaluate` results array currently reads (around `release-mac-check.lib.mjs:257-266`):

```js
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
```

Insert the new call immediately after the `checkConfig` line:

```js
    ...checkConfig(facts.config, facts.entitlements),
    ...checkBundleMetadata(facts.config),
    checkSidecar(facts.sidecar),
```

D6/D7 use `category: "config"`, so `render` lists them under the existing "electron-builder config" heading (no change to `CATEGORY_ORDER`/`CATEGORY_LABEL` needed).

- [ ] **Step 6: Run the tests to verify they pass**

Run:

```bash
cd apps/desktop && pnpm test -- release-mac-check.test.mjs
```

Expected: PASS — the new `checkBundleMetadata (D6/D7)` block is green AND the pre-existing `checkConfig (D1–D5)`, `evaluate` (exit 0 / failIds `["B1","C1"]`) tests stay green (because `GOOD_CONFIG` now carries the pinned values from Step 1).

- [ ] **Step 7: Confirm the live gate now reports D6/D7 PASS against the real config**

Run:

```bash
cd apps/desktop && pnpm release:mac:check
```

Expected: the "electron-builder config" section now includes `D6  ...  artifactName: astraler-skillbox-${version}-${arch}.${ext}` (PASS) and `D7  ...  copyright: Copyright (c) 2026 Astraler` (PASS), reading the values added in Task 1. (Overall exit code may still reflect host credential state — that is unrelated to D6/D7.)

- [ ] **Step 8: PM checkpoint**

Stop and summarize the diff for `release-mac-check.lib.mjs` and `release-mac-check.test.mjs`. Do not stage or commit.

---

## Task 3: Update live docs to the new artifact basename

The new DMG basename is `astraler-skillbox-<version>-arm64.dmg` (no spaces). Three live docs hard-code the old `Astraler Skillbox-<version>-arm64.dmg`. The safe transform is to replace the token `Astraler Skillbox-` (hyphen — only ever the DMG/manifest basename) with `astraler-skillbox-`. The `.app` bundle is referenced as `Astraler Skillbox.app` (dot, not hyphen) and must **stay unchanged**.

**Files:**
- Modify: `RELEASE.md` (lines 112, 113, 134)
- Modify: `SMOKE.md` (lines 272, 320, 356, 416, 458, 464, 479, 487, 494, 568, 571, 662, 688)
- Modify: `SCAFFOLD.md` (line 238)

- [ ] **Step 1: Rewrite the DMG basename in all three docs**

Apply this exact, scoped substitution in `RELEASE.md`, `SMOKE.md`, and `SCAFFOLD.md` only:

- Replace every `Astraler Skillbox-` with `astraler-skillbox-`
- Replace every `Astraler\ Skillbox-` (the backslash-escaped-space form, e.g. `RELEASE.md:134`) with `astraler-skillbox-`

Run (scoped to the three files; `-` after `Skillbox` ensures `.app` references are never touched):

```bash
cd <repo>
for f in RELEASE.md SMOKE.md SCAFFOLD.md; do
  perl -0pi -e 's/Astraler\\ Skillbox-/astraler-skillbox-/g; s/Astraler Skillbox-/astraler-skillbox-/g' "$f"
done
```

After this, paths like `ls "apps/desktop/dist/astraler-skillbox-0.1.0-arm64.dmg"` still work; the surrounding quotes are now redundant but harmless — leave them to minimize churn.

- [ ] **Step 2: Verify no old DMG basename remains and `.app` is untouched**

Run:

```bash
cd <repo>
echo "--- should be EMPTY (no old dmg basename) ---"
grep -n "Astraler Skillbox-\|Astraler\\\\ Skillbox-" RELEASE.md SMOKE.md SCAFFOLD.md || echo "clean"
echo "--- .app references must still be present/intact ---"
grep -n "Astraler Skillbox\.app" RELEASE.md SMOKE.md SCAFFOLD.md
echo "--- new basename present ---"
grep -n "astraler-skillbox-" RELEASE.md SMOKE.md SCAFFOLD.md | head
```

Expected: first grep prints `clean`; second grep still shows the `Astraler Skillbox.app` lines unchanged; third grep shows the new `astraler-skillbox-` basenames.

- [ ] **Step 3: PM checkpoint**

Stop and summarize the doc diff. Do not stage or commit.

---

## Task 4: Verify full suite + downstream compatibility (no code change)

This task changes no files; it confirms the slice did not regress any release gate.

- [ ] **Step 1: Run the full frontend/script test suite**

Run:

```bash
cd apps/desktop && pnpm test
```

Expected: PASS. The synthetic fixtures in `release-mac-manifest.test.mjs`, `release-mac-dry-run.test.mjs`, and `release-mac-verify.test.mjs` that still use the old basename are arbitrary input strings and remain valid — they must stay green without edits.

- [ ] **Step 2: Run typecheck**

Run:

```bash
cd apps/desktop && pnpm typecheck
```

Expected: PASS (no type changes were introduced).

- [ ] **Step 3: Confirm DMG selection is name-agnostic (static review, no run required)**

Confirm by reading that no consumer hard-codes the DMG name:

```bash
cd apps/desktop
grep -n "selectChangedDmg" scripts/release-mac-full.lib.mjs scripts/release-mac-dry-run.lib.mjs
grep -n "EXPECTED_BUNDLE" scripts/release-mac-dmg-smoke.lib.mjs
```

Expected: `selectChangedDmg` drives dry-run/full selection (snapshot diff), and `dmg-smoke` keys on `EXPECTED_BUNDLE = "Astraler Skillbox.app"` (the `.app` name, unchanged by this slice). Therefore the renamed DMG flows through verify → manifest → checksum and the smoke checks without code change.

- [ ] **Step 4 (OPTIONAL, macOS only): End-to-end dry-run against the renamed artifact**

Only if on a macOS dev machine and an end-to-end confirmation is wanted (this packages an ad-hoc DMG; no credentials needed):

```bash
cd apps/desktop && pnpm release:mac:dry-run; echo "exit=$?"
```

Expected: `exit=0`; the produced artifact is `dist/astraler-skillbox-0.1.0-arm64.dmg`, selected via snapshot diff, then verified (`--allow-adhoc`), manifested, and checksummed. Confirms the new basename round-trips through the whole no-credential pipeline.

- [ ] **Step 5: No commit** — this task is verification only. If any step fails, stop and report to the lead rather than patching around it.

---

## Self-Review (completed during plan authoring)

- **Spec coverage:** electron-builder.yml `artifactName` + `copyright` → Task 1. D6 (clean artifactName, no whitespace, exact template) + D7 (exact copyright), both FAIL → Task 2. `release-mac-check` tests → Task 2 (incl. mandatory `GOOD_CONFIG` fixture update). Live docs with old basename → Task 3. Downstream compatibility (dry-run/manifest/full/verify/dmg-smoke) → Task 4 (verified, not modified). Non-goals (icon, extendInfo, productName, version, credentials) explicitly excluded.
- **Placeholder scan:** none — every code/edit step shows the exact content and exact verification command with expected output.
- **Type/name consistency:** `checkBundleMetadata` is defined (Task 2 Step 4), exported, imported in the test (Step 2), and wired into `evaluate` (Step 5) with consistent ids `D6`/`D7` and `category: "config"`. The `GOOD_CONFIG` extension (Step 1) is what keeps the pre-existing `evaluate` tests (`exit 0`, failIds `["B1","C1"]`) green — called out as a hard dependency in the Lead Review Checkpoint.
- **Backtick hazard:** flagged explicitly — the artifact template contains `${...}`; both the lib constants and the test fixtures must use double quotes, never template literals.
