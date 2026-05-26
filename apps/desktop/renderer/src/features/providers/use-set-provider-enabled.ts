import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { ProviderSetEnabledRequest } from "@contracts/index.js";

export function useSetProviderEnabled() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProviderSetEnabledRequest) => methods.setProviderEnabled(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providers.list() });
    },
  });
}
