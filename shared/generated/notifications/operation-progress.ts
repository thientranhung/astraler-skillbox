/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Server-push notification emitted during long-running operations (host.scan, etc.). UI re-fetches the view model when status reaches success/failed/cancelled.
 */
export interface OperationProgressNotification {
  /**
   * ID of the operation this progress update belongs to
   */
  operationId: number;
  /**
   * Current operation lifecycle state
   */
  status: 'queued' | 'running' | 'success' | 'failed' | 'cancelled' | 'partial';
  /**
   * Human-readable phase label, e.g. 'reading_host_folder', 'classifying_entries', 'done'
   */
  phase: string;
  /**
   * Number of items processed so far, or null if count is unknown
   */
  processed: number | null;
  /**
   * Total items to process, or null if count is unknown
   */
  total: number | null;
  /**
   * Optional human-readable status message for display in UI
   */
  message: string | null;
  /**
   * Optional result summary present only on terminal status events (success/failed/cancelled)
   */
  metadata?: {} | null;
}
