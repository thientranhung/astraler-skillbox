import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useProviderPluginList() {
  return useQuery({
    queryKey: queryKeys.providerPlugins.list(),
    queryFn: () => methods.listProviderPlugins(),
  });
}
