/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for app.checkUpdate RPC method. Checks GitHub Releases for a newer app version. Respects the network_settings gate (update_check_enabled). Returns error field on failure instead of RPC error.
 */
export type AppCheckUpdateMethod = AppCheckUpdateRequest | AppCheckUpdateResponse;

/**
 * No params required.
 */
export interface AppCheckUpdateRequest {}
export interface AppCheckUpdateResponse {
  /**
   * The version string embedded in the binary at build time.
   */
  currentVersion: string;
  /**
   * tag_name from the latest GitHub release, with the 'v' prefix stripped. Null when the check could not be performed.
   */
  latestVersion: string | null;
  /**
   * True when latestVersion differs from currentVersion.
   */
  updateAvailable: boolean;
  /**
   * html_url of the latest GitHub release. Null when the check failed.
   */
  releaseUrl: string | null;
  /**
   * Error code: 'network_disabled' | 'network_error' | 'no_releases' | 'http_error' | 'parse_error'. Null on success.
   */
  error: string | null;
}
