/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for dialog.openProjectFolder. ELECTRON-HANDLED ONLY — this method is intercepted by Electron main and never forwarded to the Go core. It opens the native macOS/Windows folder picker for selecting a project directory.
 */
export type DialogOpenProjectFolderMethod = DialogOpenProjectFolderRequest | DialogOpenProjectFolderResponse;

/**
 * No params needed for the dialog.
 */
export interface DialogOpenProjectFolderRequest {}
/**
 * The selected folder path, or null if the user dismissed the dialog.
 */
export interface DialogOpenProjectFolderResponse {
  /**
   * Absolute path chosen by the user, or null if cancelled
   */
  path: string | null;
}
