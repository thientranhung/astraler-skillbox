import { useQuery } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useDashboard() {
  return useQuery({
    queryKey: queryKeys.dashboard.root(),
    queryFn: () => methods.getDashboard(),
  });
}
