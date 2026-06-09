/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for providerPlugin.removeOverride JSON-RPC method. Removes a project-layer plugin override from the provider's settings file, then rescans the layers needed to resolve Project Detail effective state.
 */
export type ProviderPluginRemoveOverrideMethod =
  | ProviderPluginRemoveOverrideRequest
  | ProviderPluginRemoveOverrideResponse;

/**
 * Params for providerPlugin.removeOverride.
 */
export interface ProviderPluginRemoveOverrideRequest {
  /**
   * Provider key (e.g. claude, antigravity_cli, codex).
   */
  providerKey: string;
  /**
   * Plugin name (the part before @ in name@marketplace)
   */
  pluginName: string;
  /**
   * Marketplace name (the part after @ in name@marketplace)
   */
  marketplaceName: string;
  /**
   * Settings layer. Only project is supported.
   */
  layer: 'project';
  /**
   * Required. The project ID whose settings file will be modified.
   */
  projectId: number;
}
/**
 * Immediate response — the remove and rescan run asynchronously. Errors: validation_error (1001) for invalid params; filesystem_error (1002) if the settings file cannot be modified; conflict_error (1005) if an operation is already running.
 */
export interface ProviderPluginRemoveOverrideResponse {
  /**
   * ID of the created remove+rescan operation
   */
  operationId: number;
}
