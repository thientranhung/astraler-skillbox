/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Notification sent by Go core immediately after startup. Electron main waits for this before forwarding any renderer requests. Must arrive within 10 seconds or Electron shows a blocking startup error.
 */
export interface ServerReadyNotification {
  /**
   * Skillbox core version string
   */
  version: string;
  /**
   * PID of the Go core process
   */
  pid: number;
  /**
   * List of registered RPC method names (used for feature detection)
   */
  capabilities: string[];
}
