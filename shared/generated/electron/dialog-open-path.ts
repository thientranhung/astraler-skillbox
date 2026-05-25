/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for dialog.openPath. ELECTRON-HANDLED ONLY — opens a folder in the native file manager (Finder on macOS). Never forwarded to the Go core.
 */
export type DialogOpenPathMethod = DialogOpenPathRequest | DialogOpenPathResponse;

/**
 * Params for dialog.openPath.
 */
export interface DialogOpenPathRequest {
  /**
   * Absolute path of the folder to reveal in Finder
   */
  path: string;
}
/**
 * Result of dialog.openPath. Errors: operation_error (1006) if Electron returns a non-empty error string.
 */
export interface DialogOpenPathResponse {
  /**
   * Always true on success
   */
  opened: true;
}
