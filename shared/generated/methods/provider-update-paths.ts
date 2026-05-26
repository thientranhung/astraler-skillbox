/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for provider.updatePaths command. Stores user path overrides for a provider (scope+purpose) slot. Project-scope overrides replace the built-in path candidates used during project scan and install. Global-scope overrides replace the built-in path candidates used during global scan path resolution.
 */
export type ProviderUpdatePathsMethod = ProviderUpdatePathsRequest | ProviderUpdatePathsResponse;

/**
 * Params for provider.updatePaths.
 */
export interface ProviderUpdatePathsRequest {
  /**
   * Stable provider key (e.g. claude, generic_agents)
   */
  providerKey: string;
  /**
   * Whether override applies to project or global paths
   */
  scope: 'project' | 'global';
  /**
   * Role of the path slot being overridden
   */
  purpose: 'detect' | 'skills' | 'config' | 'commands';
  /**
   * Override paths. Project paths must be relative (no ..). Global paths must start with / or ~/.
   *
   * @minItems 1
   */
  paths: [string, ...string[]];
}
/**
 * Result of provider.updatePaths. Errors: validation_error (1001), database_error (1004).
 */
export interface ProviderUpdatePathsResponse {
  /**
   * True when the override was saved successfully
   */
  updated: boolean;
}
