/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for skill.get JSON-RPC method. Returns read-only skill metadata plus project/provider installs.
 */
export type SkillGetMethod = SkillGetRequest | SkillGetResponse;

/**
 * Params for skill.get.
 */
export interface SkillGetRequest {
  /**
   * ID of the skill to fetch. Must be > 0.
   */
  skillId: number;
}
/**
 * Read-only skill detail. Errors: validation_error (1001) if skillId not found or <= 0.
 */
export interface SkillGetResponse {
  skill: SkillGetSkill;
  /**
   * One row per project/provider install. Removed projects excluded.
   */
  projects: SkillGetProjectInstall[];
}
/**
 * Skill metadata in the skill.get response.
 */
export interface SkillGetSkill {
  id: number;
  name: string;
  relativePath: string;
  absolutePath: string;
  status: 'available' | 'missing' | 'unreadable' | 'local_modified' | 'unknown';
  sourceLabel: string | null;
  hostPath: string;
  lastScannedAt: string | null;
}
/**
 * One project/provider install row referencing this skill.
 */
export interface SkillGetProjectInstall {
  projectId: number;
  projectName: string;
  projectProviderId: number;
  providerKey: string;
  providerDisplayName: string;
  mode: 'symlink' | 'rsync_copy' | 'direct';
  status:
    | 'current'
    | 'outdated'
    | 'missing'
    | 'broken_symlink'
    | 'old_host'
    | 'external_symlink'
    | 'conflict'
    | 'needs_sync'
    | 'error';
  projectSkillPath: string;
}
