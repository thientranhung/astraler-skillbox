/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for providerPlugin.scanProject JSON-RPC method. Starts a read-only scan of the Claude project and local settings layers for a given project.
 */
export type ProviderPluginScanProjectMethod = ProviderPluginScanProjectRequest | ProviderPluginScanProjectResponse;

/**
 * Params for providerPlugin.scanProject.
 */
export interface ProviderPluginScanProjectRequest {
  /**
   * ID of the project to scan plugins for
   */
  projectId: number;
}
/**
 * Immediate response — the scan runs asynchronously. Errors: invalid_params if projectId <= 0, conflict_error (1005) if a scan is already running for this project.
 */
export interface ProviderPluginScanProjectResponse {
  /**
   * ID of the created scan operation
   */
  operationId: number;
}
