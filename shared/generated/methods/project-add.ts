/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for project.add JSON-RPC method. Persists a project folder; idempotent by normalized absolute path.
 */
export type ProjectAddMethod = ProjectAddRequest | ProjectAddResponse;

/**
 * Params for project.add. Path must be absolute; obtained from dialog.openProjectFolder before calling.
 */
export interface ProjectAddRequest {
  /**
   * Absolute filesystem path of the project folder chosen by the user
   */
  path: string;
}
/**
 * Result of project.add. Errors: validation_error (1001) if path is not absolute, does not exist, or is not a directory; database_error (1004) on persistence failure.
 */
export interface ProjectAddResponse {
  /**
   * Persisted projects row ID
   */
  projectId: number;
  /**
   * Project display name derived from the folder name
   */
  name: string;
  /**
   * Normalised absolute path as stored
   */
  path: string;
  /**
   * Computed project status
   */
  status: 'active' | 'missing' | 'unreadable';
}
