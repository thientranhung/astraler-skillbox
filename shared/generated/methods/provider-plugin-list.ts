/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for providerPlugin.list JSON-RPC method. Returns persisted plugin visibility data for global (user layer) and all projects.
 */
export type ProviderPluginListMethod = ProviderPluginListRequest | ProviderPluginListResponse;

/**
 * Params for providerPlugin.list (no params required).
 */
export interface ProviderPluginListRequest {}
export interface ProviderPluginListResponse {
  global: PPGlobalView;
  projects: PPProjectView[];
}
export interface PPGlobalView {
  providerKey: string;
  userLayerPath: string;
  /**
   * null when never scanned
   */
  userLayerStatus: 'ok' | 'missing' | 'unreadable' | 'malformed' | 'too_large' | 'symlink' | 'path_escape' | null;
  /**
   * ISO-8601 timestamp or null
   */
  lastScannedAt: string | null;
  plugins: PPGlobalEntry[];
  marketplaces: PPMarketplace[];
  managedOutOfScope: boolean;
}
export interface PPGlobalEntry {
  pluginName: string;
  marketplaceName: string;
  status: 'enabled' | 'disabled';
}
export interface PPMarketplace {
  marketplaceName: string;
  sourceType: string;
  sourceSummary: string;
}
export interface PPProjectView {
  projectId: number;
  providerKey: string;
  layerStatuses: PPLayerStatus[];
  plugins: PPProjectEntry[];
  marketplaces: PPMarketplace[];
  managedOutOfScope: boolean;
}
export interface PPLayerStatus {
  layer: 'user' | 'project' | 'local';
  scanStatus: 'ok' | 'missing' | 'unreadable' | 'malformed' | 'too_large' | 'symlink' | 'path_escape';
  filePath: string;
  /**
   * ISO-8601 timestamp or null
   */
  lastScannedAt: string | null;
}
export interface PPProjectEntry {
  pluginName: string;
  marketplaceName: string;
  effectiveStatus: 'enabled' | 'disabled' | 'absent' | 'unknown';
  provenanceLayer: 'user' | 'project' | 'local' | null;
  layerBreakdown: PPLayerDetail[];
}
export interface PPLayerDetail {
  layer: 'user' | 'project' | 'local';
  scanStatus: 'ok' | 'missing' | 'unreadable' | 'malformed' | 'too_large' | 'symlink' | 'path_escape';
  declaration: 'enabled' | 'disabled' | null;
}
