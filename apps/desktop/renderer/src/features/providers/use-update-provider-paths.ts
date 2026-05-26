import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { ProviderUpdatePathsRequest } from "@contracts/index.js";

export function useUpdateProviderPaths() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProviderUpdatePathsRequest) => methods.updateProviderPaths(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providers.list() });
    },
  });
}
