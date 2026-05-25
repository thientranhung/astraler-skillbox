/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for dashboard.get JSON-RPC method. Query used to populate the Dashboard screen with host status, skill/project summary, install breakdown, and warnings.
 */
export type DashboardGetMethod = DashboardGetRequest | DashboardGetResponse;

/**
 * Params for dashboard.get. Empty — no params needed.
 */
export interface DashboardGetRequest {}
/**
 * Dashboard data snapshot. Errors: database_error (1004) if DB unavailable.
 */
export interface DashboardGetResponse {
  /**
   * Populated active host detail, or null when no active host or host row missing
   */
  activeHost: DashboardGetActiveHost | null;
  summary: DashboardGetSummary;
  installsByMode: DashboardGetInstallsByMode;
  warningsBySeverity: DashboardGetWarningsBySeverity;
  /**
   * List of active warnings to display on the dashboard
   */
  warnings: DashboardGetWarning[];
}
/**
 * Active Skill Host Folder details, present when an active host is configured
 */
export interface DashboardGetActiveHost {
  hostId: number;
  path: string;
  skillsPath: string;
  status: 'active' | 'missing' | 'unreadable' | 'unwritable' | 'invalid_structure' | 'empty' | 'inactive';
  lastScanAt: string | null;
}
/**
 * High-level counts for skills, projects, and warnings
 */
export interface DashboardGetSummary {
  skills: number;
  projects: number;
  warnings: number;
}
/**
 * Active installs grouped by install mode
 */
export interface DashboardGetInstallsByMode {
  symlink: number;
  rsyncCopy: number;
  direct: number;
}
/**
 * Active warnings grouped by severity
 */
export interface DashboardGetWarningsBySeverity {
  info: number;
  warning: number;
  error: number;
  blocking: number;
}
/**
 * A single warning entry surfaced on the dashboard
 */
export interface DashboardGetWarning {
  code: string;
  message: string;
  severity: 'info' | 'warning' | 'error' | 'blocking';
  scopeType: 'app' | 'skill_host_folder' | 'skill' | 'project' | 'project_provider' | 'install';
  scopeId: number | null;
  actionKey: string | null;
}
