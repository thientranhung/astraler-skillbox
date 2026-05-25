/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for project.get JSON-RPC method. Returns full project detail: providers, observed entries, and warnings.
 */
export type ProjectGetMethod = ProjectGetRequest | ProjectGetResponse;

/**
 * Params for project.get.
 */
export interface ProjectGetRequest {
  /**
   * ID of the projects row to retrieve
   */
  projectId: number;
}
/**
 * Full project detail view model. Errors: validation_error (1001) if projectId not found.
 */
export interface ProjectGetResponse {
  project: ProjectGetProject;
  /**
   * Detected providers for this project
   */
  providers: ProjectGetProvider[];
  /**
   * All observed skill entries, grouped by provider in the UI
   */
  entries: ProjectGetEntry[];
  /**
   * Active warnings across project, provider, and install scopes
   */
  warnings: ProjectGetWarning[];
}
/**
 * Core project fields in the detail view
 */
export interface ProjectGetProject {
  id: number;
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
   * ISO 8601 timestamp of the most recent completed scan, or null
   */
  lastScannedAt: string | null;
}
/**
 * Provider detail for the project detail view
 */
export interface ProjectGetProvider {
  /**
   * project_providers row ID
   */
  projectProviderId: number;
  /**
   * Provider definition key (e.g. generic_agents)
   */
  providerKey: string;
  /**
   * Human-readable provider name
   */
  displayName: string;
  /**
   * Provider definition status
   */
  providerStatus: 'supported' | 'experimental' | 'unsupported' | 'disabled';
  /**
   * Detection status for this provider
   */
  detectionStatus: 'detected' | 'missing' | 'invalid_structure';
  /**
   * Absolute path where the provider was detected (e.g. <root>/.agents)
   */
  detectedPath: string | null;
  /**
   * Absolute path of the provider's skills directory (e.g. <root>/.agents/skills)
   */
  skillsPath: string | null;
  /**
   * Number of observed entries under this provider
   */
  entryCount: number;
}
/**
 * A single observed install entry (installs-as-observed-entries)
 */
export interface ProjectGetEntry {
  /**
   * installs row ID
   */
  id: number;
  /**
   * project_providers row ID this entry belongs to
   */
  projectProviderId: number;
  /**
   * Provider key for grouping in the UI
   */
  providerKey: string;
  /**
   * Skill name as observed on disk (skill_name)
   */
  name: string;
  /**
   * Filesystem mechanism: symlink or plain directory
   */
  mode: 'symlink' | 'direct';
  /**
   * Observed status of the entry
   */
  status: 'current' | 'old_host' | 'external_symlink' | 'broken_symlink' | 'missing' | 'error';
  /**
   * Absolute path of this entry within the project's skills directory
   */
  projectSkillPath: string;
  /**
   * Raw symlink target path (set for all symlinks including broken/external), or null for direct entries
   */
  symlinkTargetPath: string | null;
  /**
   * skills row ID if an exact host-relative identity match was found, otherwise null
   */
  skillId: number | null;
}
/**
 * Active warning for this project (project, project_provider, or install scope)
 */
export interface ProjectGetWarning {
  /**
   * Machine-readable warning code
   */
  code: string;
  /**
   * Warning severity level
   */
  severity: 'info' | 'warning' | 'error';
  /**
   * Human-readable warning message
   */
  message: string;
  /**
   * Scope type of the warning
   */
  scopeType: 'project' | 'project_provider' | 'install';
  /**
   * Optional reference to the affected entity (e.g. entry name)
   */
  scopeRef: string | null;
  /**
   * Read-only action key for the UI (rescan or open_folder only in 2A)
   */
  actionKey: string | null;
}
