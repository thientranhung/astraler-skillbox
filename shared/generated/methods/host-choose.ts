/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for host.choose JSON-RPC method. Caller passes absolute path from Electron native dialog; Go initialises .agents/skills if needed and persists the host.
 */
export type HostChooseMethod = HostChooseRequest | HostChooseResponse;

/**
 * Params for host.choose. Path must be absolute; obtained from Electron showOpenDialog before calling.
 */
export interface HostChooseRequest {
  /**
   * Absolute filesystem path of the chosen Skill Host Folder
   */
  path: string;
}
/**
 * Result of host.choose. Errors: validation_error (1001) if path invalid; filesystem_error (1002) if .agents/skills cannot be created.
 */
export interface HostChooseResponse {
  /**
   * Persisted skill_host_folders row ID
   */
  hostId: number;
  /**
   * Absolute path (normalised)
   */
  path: string;
  /**
   * <path>/.agents/skills absolute path
   */
  skillsPath: string;
  /**
   * true if .agents/skills directory was created by this call
   */
  initialized: boolean;
  /**
   * Computed status after choose
   */
  status: 'active' | 'missing' | 'unreadable' | 'unwritable' | 'invalid_structure' | 'empty' | 'inactive';
}
