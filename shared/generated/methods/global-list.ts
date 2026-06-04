/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for global.list JSON-RPC method. Returns persisted global provider locations with entries and warnings.
 */
export type GlobalListMethod = GlobalListRequest | GlobalListResponse;

/**
 * Params for global.list (no params required).
 */
export interface GlobalListRequest {}
/**
 * Response containing all global provider locations.
 */
export interface GlobalListResponse {
  locations: GlobalListLocation[];
}
export interface GlobalListLocation {
  globalProviderLocationId: number;
  providerKey: string;
  providerDisplayName: string;
  providerStatus: string;
  path: string | null;
  skillsPath: string | null;
  status:
    | 'active'
    | 'not_configured'
    | 'missing'
    | 'unreadable'
    | 'invalid_structure'
    | 'empty'
    | 'disabled'
    | 'no_global_skills';
  lastScannedAt: string | null;
  entries: GlobalListEntry[];
  warnings: GlobalListWarning[];
}
export interface GlobalListEntry {
  globalInstallId: number;
  skillName: string;
  skillId: number | null;
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
  globalSkillPath: string;
  sourceSkillPath: string | null;
  symlinkTargetPath: string | null;
}
export interface GlobalListWarning {
  code: string;
  severity: 'info' | 'warning' | 'error' | 'blocking';
  scopeType: 'global_provider_location' | 'global_install';
  scopeId: number | null;
  actionKey: string | null;
  message: string;
}
