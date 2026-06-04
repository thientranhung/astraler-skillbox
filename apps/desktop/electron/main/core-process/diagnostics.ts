export interface DiagnosticsOptions {
  appVersion: string;
  electronVersion: string;
  chromeVersion: string;
  nodeVersion: string;
  platform: string;
  arch: string;
  dbPath: string;
  homeDir: string;
  exportedAt: string;
  coreLogLines: string[];
}

export function buildDiagnosticsText(opts: DiagnosticsOptions): string {
  const redact = (s: string): string =>
    opts.homeDir ? s.replaceAll(opts.homeDir, "~") : s;

  const logSection =
    opts.coreLogLines.length === 0
      ? "(no output captured)"
      : opts.coreLogLines.map(redact).join("\n");

  return [
    "=== Astraler Skillbox Diagnostics ===",
    `Exported: ${opts.exportedAt}`,
    "",
    `App version: ${opts.appVersion}`,
    `Electron: ${opts.electronVersion}`,
    `Chrome: ${opts.chromeVersion}`,
    `Node: ${opts.nodeVersion}`,
    `Platform: ${opts.platform} ${opts.arch}`,
    "",
    `DB path: ${redact(opts.dbPath)}`,
    "",
    "=== Core Log Tail (last 100 lines) ===",
    logSection,
  ].join("\n");
}
