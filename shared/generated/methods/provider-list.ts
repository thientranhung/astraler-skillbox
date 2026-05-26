/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Contract for provider.list JSON-RPC method. Returns the read-only provider registry: all built-in providers with their metadata and path candidates.
 */
export type ProviderListMethod = ProviderListRequest | ProviderListResponse;

/**
 * Params for provider.list. Empty — no params needed.
 */
export interface ProviderListRequest {}
/**
 * List of all registered providers in the registry. Errors: database_error (1004) if DB unavailable.
 */
export interface ProviderListResponse {
  /**
   * All built-in providers ordered by registry insertion order
   */
  providers: ProviderListProvider[];
}
/**
 * Full provider registry entry for one provider.
 */
export interface ProviderListProvider {
  /**
   * Stable provider key (e.g. generic_agents, claude, codex)
   */
  key: string;
  /**
   * Human-readable provider name
   */
  displayName: string;
  /**
   * Provider type identifier
   */
  providerType: string;
  /**
   * Icon key used to look up the provider icon SVG, or null if not set
   */
  iconKey: string | null;
  /**
   * Skillbox support level for this provider
   */
  status: 'supported' | 'experimental' | 'unsupported' | 'disabled';
  /**
   * Whether this provider is enabled. Derived as true for supported/experimental built-ins until override storage is available.
   */
  enabled: boolean;
  /**
   * Whether Skillbox can create the provider directory structure in a project
   */
  canCreateStructure: boolean;
  /**
   * Whether this provider has a global-level skills location
   */
  hasGlobalLevel: boolean;
  /**
   * All path candidates for this provider, ordered by priority descending
   */
  candidates: ProviderListPathCandidate[];
}
/**
 * A single path candidate for a provider.
 */
export interface ProviderListPathCandidate {
  /**
   * Path relative to project root (project scope) or home dir (global scope). Global paths use ~ prefix.
   */
  relativePath: string;
  /**
   * Whether this candidate applies to a project or the user's global environment
   */
  scope: 'project' | 'global';
  /**
   * Role of this path: detect=presence check, skills=skills directory, config=config dir, commands=commands dir
   */
  purpose: 'detect' | 'skills' | 'config' | 'commands';
  /**
   * Ordering priority; higher values take precedence
   */
  priority: number;
  /**
   * Whether this candidate is a built-in default, user override, or custom provider path
   */
  source: 'builtin' | 'override' | 'custom';
  /**
   * Confidence level: verified=confirmed from docs, assumed=convention-based, experimental=uncertain
   */
  verificationStatus: 'verified' | 'assumed' | 'experimental';
}
