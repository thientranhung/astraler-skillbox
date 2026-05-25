/**
 * Pure evaluator for the macOS release preflight (Slice 3B2A).
 * NO process / filesystem / env access here — callers pass plain facts.
 * Never emit any credential VALUE or file PATH; only variable names + state tokens.
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

/** @param {{identityNames:string[], env:Record<string,string|undefined>, fileProbes:any}} facts */
export function checkSigning({ identityNames, env, fileProbes }) {
  const hasKeychain = identityNames.length > 0;
  const cscLinkSet = isSet(env.CSC_LINK);
  const cscPwSet = isSet(env.CSC_KEY_PASSWORD);

  let pathB = false;
  let pathBProblem = null;
  let localPathMissing = false;
  if (cscLinkSet && cscPwSet) {
    const p = fileProbes.cscLink;
    if (p && p.isLocalPath) {
      if (p.exists && p.readable) pathB = true;
      else {
        pathBProblem = "CSC_LINK points to a local file that is missing or unreadable";
        localPathMissing = true;
      }
    } else {
      pathB = true; // URL/base64 form — presence is sufficient; never fetched/decoded
    }
  } else if (cscLinkSet && !cscPwSet) {
    pathBProblem = "CSC_LINK is set but CSC_KEY_PASSWORD is missing";
  } else if (!cscLinkSet && cscPwSet) {
    pathBProblem = "CSC_KEY_PASSWORD is set but CSC_LINK is missing";
  }

  // A broken local .p12 is an explicit configuration error that must FAIL even when a
  // keychain identity is present — the missing/unreadable file cannot be silently ignored.
  if (localPathMissing) {
    return {
      id: "B1",
      category: "signing",
      status: "FAIL",
      message: `no signing credential (${pathBProblem})`,
      remediation:
        "Signing credential: a Developer ID Application identity in the login keychain, OR CSC_LINK + CSC_KEY_PASSWORD",
    };
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
  // If APPLE_API_KEY is set at all, a missing/unreadable .p8 is a real problem to
  // surface — independent of whether the other Group 1 vars are present.
  const apiKeyBadPath = isSet(env.APPLE_API_KEY) && !apiKeyFileOk;
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
  if (apiKeyBadPath) {
    // Surface the bad .p8 first, regardless of which other Group 1 vars are set.
    // NEVER print the path — only the variable name and the generic problem.
    const stillMissing = g1Missing.filter((v) => v !== "APPLE_API_KEY");
    const more = stillMissing.length ? ` (also missing ${stillMissing.join(", ")})` : "";
    msg = `the APPLE_API_KEY .p8 file is missing or unreadable${more}`;
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

/** @param {Record<string, boolean | undefined>} tools */
export function checkTooling(tools) {
  const defs = [
    ["A2", "notarytool", "xcrun notarytool"],
    ["A3", "stapler", "xcrun stapler"],
    ["A4a", "codesign", "codesign"],
    ["A4b", "spctl", "spctl"],
    ["A4c", "plutil", "plutil"],
  ];
  return defs.map(([id, key, label]) =>
    tools[key] === true
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
