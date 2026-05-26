/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for provider.resetPaths command. Removes user path override for a provider (scope+purpose) slot, restoring built-in defaults.
 */
export type ProviderResetPathsMethod = ProviderResetPathsRequest | ProviderResetPathsResponse;

/**
 * Params for provider.resetPaths.
 */
export interface ProviderResetPathsRequest {
  /**
   * Stable provider key
   */
  providerKey: string;
  /**
   * Scope of the slot to reset
   */
  scope: 'project' | 'global';
  /**
   * Purpose of the slot to reset
   */
  purpose: 'detect' | 'skills' | 'config' | 'commands';
}
/**
 * Result of provider.resetPaths. Errors: validation_error (1001), database_error (1004).
 */
export interface ProviderResetPathsResponse {
  /**
   * True if an override existed and was removed; false if no override was stored
   */
  reset: boolean;
}
