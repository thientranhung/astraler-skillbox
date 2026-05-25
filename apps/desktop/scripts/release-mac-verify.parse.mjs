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
