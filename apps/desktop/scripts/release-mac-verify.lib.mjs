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
    // --allow-adhoc: an ad-hoc signature is an accepted signature class (spec §6: PASS when ad-hoc OR Developer ID).
    push({ id: "APP2", category: "app", status: "PASS", message: "ad-hoc signature accepted (dry-run)" });
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
    const expApp = s.expectedEntitlements.app ?? [];
    if (expApp.length === 0)
      // An empty expected set would make the subset check vacuously pass and silently weaken
      // the gate, so treat it as a hard failure (the shell also refuses to run — see Task 3).
      push({ id: "ENT1", category: "app", status: "FAIL", message: "expected app entitlements unavailable (build/entitlements.mac.plist missing or empty)", remediation: "Restore build/entitlements.mac.plist with the expected entitlement keys" });
    else {
      const miss = missingKeys(expApp, s.app.entitlementKeys);
      push(
        miss.length === 0
          ? { id: "ENT1", category: "app", status: "PASS", message: `entitlements include ${expApp.map(shortKey).join(", ")}` }
          : { id: "ENT1", category: "app", status: "FAIL", message: `entitlements missing ${miss.map(shortKey).join(", ")}`, remediation: "App must embed every key from build/entitlements.mac.plist" }
      );
    }
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
      // --allow-adhoc: ad-hoc is an accepted signature class (spec §6).
      push({ id: "SID3", category: "sidecar", status: "PASS", message: "ad-hoc signature accepted (dry-run)" });
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

    const expSide = s.expectedEntitlements.sidecar ?? [];
    if (expSide.length === 0)
      push({ id: "ENT2", category: "sidecar", status: "FAIL", message: "expected sidecar entitlements unavailable (build/entitlements.mac.inherit.plist missing or empty)", remediation: "Restore build/entitlements.mac.inherit.plist with the expected entitlement keys" });
    else {
      const miss = missingKeys(expSide, s.sidecar.entitlementKeys);
      push(
        miss.length === 0
          ? { id: "ENT2", category: "sidecar", status: "PASS", message: `entitlements include ${expSide.map(shortKey).join(", ")}` }
          : { id: "ENT2", category: "sidecar", status: "FAIL", message: `entitlements missing ${miss.map(shortKey).join(", ")}`, remediation: "Sidecar must embed every key from build/entitlements.mac.inherit.plist" }
      );
    }
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
