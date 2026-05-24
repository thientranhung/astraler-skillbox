/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for operation.cancel JSON-RPC method. Sends cancel signal to a running operation.
 */
export type OperationCancelMethod = OperationCancelRequest | OperationCancelResponse;

/**
 * Params for operation.cancel.
 */
export interface OperationCancelRequest {
  /**
   * ID of the operation to cancel
   */
  operationId: number;
}
/**
 * Result of operation.cancel. Errors: validation_error (1001) if operationId does not exist.
 */
export interface OperationCancelResponse {
  /**
   * true if the cancel signal was sent to a running/queued operation; false if the operation had already finished
   */
  acknowledged: boolean;
}
