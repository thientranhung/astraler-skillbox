/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for project.scan JSON-RPC method. Starts a long-running read-only project scan; progress arrives via operation.progress notifications.
 */
export type ProjectScanMethod = ProjectScanRequest | ProjectScanResponse;

/**
 * Params for project.scan.
 */
export interface ProjectScanRequest {
  /**
   * ID of the projects row to scan
   */
  projectId: number;
}
/**
 * Immediate response — the scan runs asynchronously. Errors: validation_error (1001) projectId not found; conflict_error (1005) project already being scanned.
 */
export interface ProjectScanResponse {
  /**
   * ID of the created scan operation; use with operation.progress notifications and operation.cancel
   */
  operationId: number;
}
