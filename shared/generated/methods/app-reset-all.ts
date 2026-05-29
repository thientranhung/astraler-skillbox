/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for app.resetAll JSON-RPC method. Deletes the SQLite database and signals the app to restart. Irreversible — all Skillbox data is lost.
 */
export type AppResetAllMethod = AppResetAllRequest | AppResetAllResponse;

/**
 * No params required for app.resetAll.
 */
export interface AppResetAllRequest {}
export interface AppResetAllResponse {
  /**
   * Always true on success. Electron main triggers app.relaunch() upon receiving this response.
   */
  restarting: boolean;
}
