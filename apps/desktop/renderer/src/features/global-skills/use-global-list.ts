import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useGlobalList() {
  return useQuery({
    queryKey: queryKeys.global.list(),
    queryFn: () => methods.listGlobal(),
  });
}
