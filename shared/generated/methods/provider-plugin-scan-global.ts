/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for providerPlugin.scanGlobal JSON-RPC method. Starts a read-only scan of the Claude global user settings layer for plugin declarations.
 */
export type ProviderPluginScanGlobalMethod = ProviderPluginScanGlobalRequest | ProviderPluginScanGlobalResponse;

/**
 * Params for providerPlugin.scanGlobal (no params required).
 */
export interface ProviderPluginScanGlobalRequest {}
/**
 * Immediate response — the scan runs asynchronously. Errors: conflict_error (1005) if a scan is already running.
 */
export interface ProviderPluginScanGlobalResponse {
  /**
   * ID of the created scan operation
   */
  operationId: number;
}
