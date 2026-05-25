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
