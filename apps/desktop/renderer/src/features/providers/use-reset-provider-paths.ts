import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { ProviderResetPathsRequest } from "@contracts/index.js";

export function useResetProviderPaths() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProviderResetPathsRequest) => methods.resetProviderPaths(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providers.list() });
    },
  });
}
