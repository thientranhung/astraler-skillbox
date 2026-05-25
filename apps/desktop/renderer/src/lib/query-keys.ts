export const queryKeys = {
  settings: {
    app: () => ["settings", "app"] as const,
  },
  skills: {
    list: (hostId: number) => ["skills", "list", hostId] as const,
  },
  projects: {
    list: () => ["projects", "list"] as const,
    detail: (projectId: number) => ["projects", "detail", projectId] as const,
  },
};
