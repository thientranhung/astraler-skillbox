/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for skill.list JSON-RPC method. Returns Skills Library view model for a given host.
 */
export type SkillListMethod = SkillListRequest | SkillListResponse;

/**
 * Params for skill.list.
 */
export interface SkillListRequest {
  /**
   * ID of the skill_host_folders row to list skills for
   */
  hostId: number;
}
/**
 * Skills Library view model. Errors: validation_error (1001) hostId not found.
 */
export interface SkillListResponse {
  /**
   * Absolute path of the skill host folder
   */
  hostPath: string;
  /**
   * All skills known to this host, sorted by name
   */
  skills: SkillListSkill[];
  totals: SkillListTotals;
  /**
   * ISO 8601 timestamp of most recent completed scan, or null
   */
  lastScanAt: string | null;
  /**
   * Active warnings for this host or its skills
   */
  warnings: SkillListWarning[];
}
/**
 * Skill item within the skill.list response
 */
export interface SkillListSkill {
  id: number;
  name: string;
  relativePath: string;
  status: 'available' | 'missing' | 'unreadable' | 'local_modified' | 'unknown';
  sourceLabel: string | null;
  lastScannedAt: string | null;
  projectsUsingCount: number;
}
/**
 * Counts per status for the skills list
 */
export interface SkillListTotals {
  available: number;
  missing: number;
  unreadable: number;
  local_modified: number;
  unknown: number;
}
/**
 * Warning attached to the skill host or a specific skill
 */
export interface SkillListWarning {
  code: string;
  message: string;
  scopeRef: string | null;
}
