export const STALE_MS = 10 * 60 * 1000;

// Tracks auto-scan triggers for the current app session.
// Keyed by "auto-scan:<target>:<id>" to prevent re-triggering on re-mount.
// Module-scope: persists for the lifetime of the renderer process.
export const sessionAutoScanRegistry = new Set<string>();

export function clearAutoScanRegistry(): void {
  sessionAutoScanRegistry.clear();
}

export function isDataStale(lastScannedAt: string | null | undefined): boolean {
  if (lastScannedAt == null) return true;
  return Date.now() - new Date(lastScannedAt).getTime() > STALE_MS;
}
