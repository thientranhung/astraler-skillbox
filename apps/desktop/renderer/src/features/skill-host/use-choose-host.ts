import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useChooseHost() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (path: string) => methods.chooseHost({ path }),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.settings.app() });
      void queryClient.invalidateQueries({ queryKey: queryKeys.skills.list(data.hostId) });
    },
  });
}
