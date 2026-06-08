export const SHARED_AGENTS_PROVIDER_KEY = "generic_agents";

export function providerDisplayName(providerKey: string, fallback: string): string {
  if (providerKey === SHARED_AGENTS_PROVIDER_KEY) return "Shared Agents";
  return fallback;
}

export function providerShortLabel(providerKey: string): string {
  switch (providerKey) {
    case SHARED_AGENTS_PROVIDER_KEY: return "Shared Agents";
    case "claude": return "Claude";
    case "codex": return "Codex";
    case "antigravity_cli": return "Antigravity";
    default: return providerKey;
  }
}

export function orderBySharedAgentsFirst<T extends { providerKey: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const aPriority = a.providerKey === SHARED_AGENTS_PROVIDER_KEY ? 0 : 1;
    const bPriority = b.providerKey === SHARED_AGENTS_PROVIDER_KEY ? 0 : 1;
    const priority = aPriority - bPriority;
    return priority !== 0 ? priority : a.providerKey.localeCompare(b.providerKey);
  });
}

export function orderBySharedAgentsKeyFirst<T extends { key: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    const aPriority = a.key === SHARED_AGENTS_PROVIDER_KEY ? 0 : 1;
    const bPriority = b.key === SHARED_AGENTS_PROVIDER_KEY ? 0 : 1;
    const priority = aPriority - bPriority;
    return priority !== 0 ? priority : a.key.localeCompare(b.key);
  });
}
