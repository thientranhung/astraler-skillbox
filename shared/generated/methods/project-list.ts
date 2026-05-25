/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for project.list JSON-RPC method. Returns all projects with provider badges and summary counts.
 */
export type ProjectListMethod = ProjectListRequest | ProjectListResponse;

/**
 * No params needed for project.list.
 */
export interface ProjectListRequest {}
/**
 * All projects view model.
 */
export interface ProjectListResponse {
  /**
   * All projects, sorted by name
   */
  projects: ProjectListItem[];
}
/**
 * Single project row in the projects list
 */
export interface ProjectListItem {
  /**
   * projects row ID
   */
  id: number;
  /**
   * Project display name
   */
  name: string;
  /**
   * Normalised absolute path
   */
  path: string;
  /**
   * Project status
   */
  status: 'active' | 'missing' | 'unreadable';
  /**
   * Detected providers for this project
   */
  providers: ProjectListProviderSummary[];
  /**
   * Number of observed skill entries across all providers
   */
  skillCount: number;
  /**
   * Number of active warnings (project + project_provider + install scopes)
   */
  warningCount: number;
  /**
   * ISO 8601 timestamp of the most recent completed scan, or null
   */
  lastScannedAt: string | null;
}
/**
 * Brief provider info for the projects list view
 */
export interface ProjectListProviderSummary {
  /**
   * Provider definition key (e.g. generic_agents)
   */
  key: string;
  /**
   * Human-readable provider name
   */
  displayName: string;
  /**
   * Provider definition status
   */
  providerStatus: 'supported' | 'experimental' | 'unsupported' | 'disabled';
  /**
   * Detection status for this provider in the project
   */
  detectionStatus: 'detected' | 'configured' | 'missing' | 'invalid_structure';
}
