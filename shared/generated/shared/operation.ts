/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Minimal operation reference returned by long-running commands
 */
export interface OperationRef {
  /**
   * Unique integer ID for the operation
   */
  operationId: number;
  /**
   * Current lifecycle state of the operation
   */
  status: 'queued' | 'running' | 'success' | 'failed' | 'cancelled' | 'partial';
}
