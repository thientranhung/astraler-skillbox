/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Error data payload carried in JSON-RPC error responses. App codes never fall in the reserved -32768..-32000 range.
 */
export interface AppError {
  /**
   * Machine-readable error category
   */
  code:
    | 'validation_error'
    | 'filesystem_error'
    | 'provider_error'
    | 'database_error'
    | 'auth_error'
    | 'network_error'
    | 'conflict_error'
    | 'operation_cancelled'
    | 'user_cancelled'
    | 'unknown_error';
  /**
   * Numeric JSON-RPC error code. validation_error=1001 filesystem_error=1002 provider_error=1003 database_error=1004 conflict_error=1005 user_cancelled=1006 operation_cancelled=1007 unknown_error=1099
   */
  rpcCode?: number;
  /**
   * Human-readable message safe to display in UI
   */
  userMessage: string;
  /**
   * Developer/log message with internal detail
   */
  technicalMessage: string;
  /**
   * Related operation ID if applicable
   */
  operationId?: number;
  /**
   * Affected entity reference (e.g. path, host ID string)
   */
  entityRef?: string;
}
