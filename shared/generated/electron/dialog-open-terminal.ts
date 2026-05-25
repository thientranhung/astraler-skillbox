/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for dialog.openTerminal. ELECTRON-HANDLED ONLY — opens Terminal at a folder on macOS. Never forwarded to the Go core.
 */
export type DialogOpenTerminalMethod = DialogOpenTerminalRequest | DialogOpenTerminalResponse;

/**
 * Params for dialog.openTerminal.
 */
export interface DialogOpenTerminalRequest {
  /**
   * Absolute path of the folder to open in Terminal
   */
  path: string;
}
/**
 * Result of dialog.openTerminal. Errors: unknown_error if Terminal cannot be launched.
 */
export interface DialogOpenTerminalResponse {
  /**
   * Always true on success
   */
  opened: true;
}
