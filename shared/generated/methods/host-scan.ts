/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for host.scan JSON-RPC method. Starts a long-running scan operation; progress arrives via operation.progress notifications.
 */
export type HostScanMethod = HostScanRequest | HostScanResponse;

/**
 * Params for host.scan.
 */
export interface HostScanRequest {
  /**
   * ID of the skill_host_folders row to scan
   */
  hostId: number;
}
/**
 * Immediate response — the scan runs asynchronously. Errors: validation_error (1001) hostId not found; conflict_error (1005) host already being scanned.
 */
export interface HostScanResponse {
  /**
   * ID of the created scan operation; use with operation.progress notifications and operation.cancel
   */
  operationId: number;
}
