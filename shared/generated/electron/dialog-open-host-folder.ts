/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for dialog.openHostFolder. ELECTRON-HANDLED ONLY — this method is intercepted by Electron main and never forwarded to the Go core. It opens the native macOS/Windows folder picker.
 */
export type DialogOpenHostFolderMethod = DialogOpenHostFolderRequest | DialogOpenHostFolderResponse;

/**
 * No params needed for the dialog.
 */
export interface DialogOpenHostFolderRequest {}
/**
 * The selected folder path, or null if the user dismissed the dialog.
 */
export interface DialogOpenHostFolderResponse {
  /**
   * Absolute path chosen by the user, or null if cancelled
   */
  path: string | null;
}
