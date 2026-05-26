import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useProviderList() {
  return useQuery({
    queryKey: queryKeys.providers.list(),
    queryFn: () => methods.listProviders(),
  });
}
