/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for remove.skill JSON-RPC method. Removes one current symlink install from a project provider; runs asynchronously and reports progress via operation.progress notifications.
 */
export type RemoveSkillMethod = RemoveSkillRequest | RemoveSkillResponse;

/**
 * Params for remove.skill.
 */
export interface RemoveSkillRequest {
  /**
   * ID of the project the install belongs to
   */
  projectId: number;
  /**
   * ID of the installed-skill row to remove (from project.get entries)
   */
  installId: number;
}
/**
 * Immediate response — the removal runs asynchronously. Errors: validation_error (1001) project/install not found or not removable; conflict_error (1005) project busy or entry changed on disk; filesystem_error (1002) unlink failed.
 */
export interface RemoveSkillResponse {
  /**
   * ID of the created remove operation; use with operation.progress notifications and operation.cancel
   */
  operationId: number;
}
