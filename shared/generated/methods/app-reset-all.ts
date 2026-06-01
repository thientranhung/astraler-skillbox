/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for app.resetAll JSON-RPC method. Truncates all user-data tables and resets settings to defaults; user-managed folders on disk are not deleted. Returns success; the renderer clears its query cache and navigates to /setup. Irreversible — all Skillbox metadata is lost.
 */
export type AppResetAllMethod = AppResetAllRequest | AppResetAllResponse;

/**
 * No params required for app.resetAll.
 */
export interface AppResetAllRequest {}
export interface AppResetAllResponse {
  /**
   * Always true on success. The renderer uses this as a completion signal: it clears the TanStack Query cache and navigates to /setup.
   */
  restarting: boolean;
}
