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
