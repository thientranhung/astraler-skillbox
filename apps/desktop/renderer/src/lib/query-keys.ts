export const queryKeys = {
  settings: {
    app: () => ["settings", "app"] as const,
  },
  skills: {
    list: (hostId: number) => ["skills", "list", hostId] as const,
  },
};
