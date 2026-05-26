export const queryKeys = {
  settings: {
    app: () => ["settings", "app"] as const,
  },
  skills: {
    list: (hostId: number) => ["skills", "list", hostId] as const,
    detail: (skillId: number) => ["skills", "detail", skillId] as const,
  },
  projects: {
    list: () => ["projects", "list"] as const,
    detail: (projectId: number) => ["projects", "detail", projectId] as const,
  },
  dashboard: {
    root: () => ["dashboard"] as const,
  },
  global: {
    list: () => ["global", "list"] as const,
  },
  providers: {
    list: () => ["providers", "list"] as const,
  },
  providerPlugins: {
    list: () => ["providerPlugins", "list"] as const,
  },
};
