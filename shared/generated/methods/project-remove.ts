/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for project.remove JSON-RPC method. Soft-removes a project by setting its status to removed; files on disk are never touched.
 */
export type ProjectRemoveMethod = ProjectRemoveRequest | ProjectRemoveResponse;

/**
 * Params for project.remove.
 */
export interface ProjectRemoveRequest {
  /**
   * ID of the project to remove
   */
  projectId: number;
}
/**
 * Result of project.remove. Errors: validation_error (1001) if the project does not exist or is already removed.
 */
export interface ProjectRemoveResponse {
  /**
   * Always true on success
   */
  removed: true;
}
