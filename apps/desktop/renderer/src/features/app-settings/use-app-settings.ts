import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useAppSettings() {
  return useQuery({
    queryKey: queryKeys.settings.app(),
    queryFn: () => methods.getSettings(),
  });
}
