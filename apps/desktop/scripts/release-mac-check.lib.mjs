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
