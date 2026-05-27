/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for providerPlugin.setEnabled JSON-RPC method. Enables or disables a provider plugin at the specified layer by writing to the provider's settings file, then rescans the layer. Only layer=user is supported in this slice.
 */
export type ProviderPluginSetEnabledMethod = ProviderPluginSetEnabledRequest | ProviderPluginSetEnabledResponse;

/**
 * Params for providerPlugin.setEnabled.
 */
export interface ProviderPluginSetEnabledRequest {
  /**
   * Provider key (e.g. claude, antigravity_cli). Codex returns validation_error.
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
   * Settings layer to write. user writes to the global user settings; project writes to the project-local settings.
   */
  layer: 'user' | 'project';
  /**
   * Required when layer=project. The project ID whose settings file will be written.
   */
  projectId?: number;
  /**
   * Whether to enable (true) or disable (false) the plugin
   */
  enabled: boolean;
}
/**
 * Immediate response — the write and rescan run asynchronously. Errors: validation_error (1001) for unknown/unsupported provider or layer; filesystem_error (1002) if the settings file cannot be written; conflict_error (1005) if a scan or write is already running.
 */
export interface ProviderPluginSetEnabledResponse {
  /**
   * ID of the created write+rescan operation
   */
  operationId: number;
}
