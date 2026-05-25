/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for install.skill JSON-RPC method. Installs one or more skills from the Skill Host Folder into a project; runs asynchronously and reports progress via operation.progress notifications.
 */
export type InstallSkillMethod = InstallSkillRequest | InstallSkillResponse;

/**
 * Params for install.skill.
 */
export interface InstallSkillRequest {
  /**
   * ID of the project to install skills into
   */
  projectId: number;
  /**
   * Provider target for the installation
   */
  providerKey: 'generic_agents';
  /**
   * IDs of skills to install
   *
   * @minItems 1
   */
  skillIds: [number, ...number[]];
}
/**
 * Immediate response — the installation runs asynchronously. Errors: validation_error (1001) projectId or skillIds not found; conflict_error (1005) project already has an active operation.
 */
export interface InstallSkillResponse {
  /**
   * ID of the created install operation; use with operation.progress notifications and operation.cancel
   */
  operationId: number;
}
