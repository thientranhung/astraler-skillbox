/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for provider.setEnabled JSON-RPC method. Persists the user's enabled/disabled preference for a provider.
 */
export type ProviderSetEnabledMethod = ProviderSetEnabledRequest | ProviderSetEnabledResponse;

/**
 * Params for provider.setEnabled.
 */
export interface ProviderSetEnabledRequest {
  /**
   * Stable provider key (e.g. generic_agents, claude)
   */
  providerKey: string;
  /**
   * Whether to enable or disable the provider
   */
  enabled: boolean;
}
/**
 * Confirmation that the provider enablement preference was saved. Errors: validation_error (1001) if provider unknown or enabled=true for a canToggle=false provider; database_error (1004) if DB unavailable.
 */
export interface ProviderSetEnabledResponse {
  /**
   * Always true; indicates the preference was persisted successfully
   */
  updated: boolean;
}
