/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Skill entity as returned in Skills Library view model
 */
export interface SkillViewItem {
  /**
   * Skill row ID (integer auto-increment)
   */
  id: number;
  /**
   * Derived from folder name under .agents/skills/
   */
  name: string;
  /**
   * Path relative to skillsPath of the host folder
   */
  relativePath: string;
  /**
   * Computed status from last scan
   */
  status: 'available' | 'missing' | 'unreadable' | 'local_modified' | 'unknown';
  /**
   * Human-readable source label (e.g. 'GitHub: org/repo') or null if no source
   */
  sourceLabel: string | null;
  /**
   * ISO 8601 timestamp of last scan, or null if never scanned
   */
  lastScannedAt: string | null;
}
