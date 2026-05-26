/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for global.scan JSON-RPC method. Starts a read-only global skills scan; progress arrives via operation.progress notifications.
 */
export type GlobalScanMethod = GlobalScanRequest | GlobalScanResponse;

/**
 * Params for global.scan (no params required).
 */
export interface GlobalScanRequest {}
/**
 * Immediate response — the scan runs asynchronously. Errors: conflict_error (1005) if a scan is already running.
 */
export interface GlobalScanResponse {
  /**
   * ID of the created scan operation
   */
  operationId: number;
}
