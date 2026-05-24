import { useAppSettings } from "../app-settings/use-app-settings.js";

export function useActiveHost() {
  const { data } = useAppSettings();
  return data?.activeHost ?? null;
}
