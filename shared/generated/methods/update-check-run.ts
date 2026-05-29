/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for updateCheck.run JSON-RPC method. Manually triggers plugin update check (ADR-0001 Phase 1). Requires network.update_check.enabled=true; returns status='disabled' otherwise.
 */
export type UpdateCheckRunMethod = UpdateCheckRunRequest | UpdateCheckRunResponse;

/**
 * No params required for updateCheck.run.
 */
export interface UpdateCheckRunRequest {}
export interface UpdateCheckRunResponse {
  /**
   * 'disabled' when setting is off (default). 'git_not_found' when git CLI unavailable. 'ok' on success.
   */
  status: 'ok' | 'disabled' | 'git_not_found' | 'error';
  /**
   * Per-plugin results. Empty when status != 'ok'.
   */
  plugins: UpdateCheckPluginResult[];
}
export interface UpdateCheckPluginResult {
  providerKey: string;
  pluginName: string;
  marketplaceName: string;
  /**
   * true=update available; false=up-to-date; null=unknown (no gitCommitSha or ref_not_found).
   */
  updateAvailable?: boolean | null;
  /**
   * Latest tag resolved remotely. null when not available.
   */
  latestVersion?: string | null;
  /**
   * ISO-8601 timestamp of this check.
   */
  lastCheckedAt?: string | null;
  /**
   * Non-empty on per-plugin error (non_https_scheme_rejected, git_not_found, timeout, ref_not_found, host_backoff, etc.).
   */
  error?: string;
}
