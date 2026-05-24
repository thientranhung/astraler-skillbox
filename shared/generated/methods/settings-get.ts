/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for settings.get JSON-RPC method. App-boot query used to route /setup vs /skills and to populate the Settings screen.
 */
export type SettingsGetMethod = SettingsGetRequest | SettingsGetResponse;

/**
 * Params for settings.get. Empty — no params needed.
 */
export interface SettingsGetRequest {}
/**
 * Current app settings. Errors: database_error (1004) if DB unavailable.
 */
export interface SettingsGetResponse {
  /**
   * ID of the currently active skill_host_folders row, or null if none configured
   */
  activeSkillHostFolderId: number | null;
  /**
   * Default mechanism for installing skills into projects
   */
  defaultInstallMode: 'symlink' | 'rsync_copy';
  /**
   * Current applied migration version
   */
  databaseVersion: number;
  /**
   * Populated active host detail, or null when no active host or host row missing
   */
  activeHost: SettingsGetActiveHost | null;
}
/**
 * Active Skill Host Folder details, present when an active host is configured
 */
export interface SettingsGetActiveHost {
  hostId: number;
  path: string;
  skillsPath: string;
  status: 'active' | 'missing' | 'unreadable' | 'unwritable' | 'invalid_structure' | 'empty' | 'inactive';
  lastScannedAt: string | null;
}
