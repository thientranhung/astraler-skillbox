/* AUTO-GENERATED — do not edit by hand.
 * Source: shared/api-contracts/
 * Regenerate: (cd apps/desktop && pnpm generate:contracts)
 */

/**
 * Warning attached to a host, skill, or other entity
 */
export interface WarningItem {
  /**
   * Machine-readable warning code
   */
  code: string;
  /**
   * Human-readable warning message
   */
  message: string;
  /**
   * Optional reference to the affected entity (e.g. skill name)
   */
  scopeRef: string | null;
}
