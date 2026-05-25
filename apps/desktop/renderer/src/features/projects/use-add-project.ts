import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useAddProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (path: string) => methods.addProject({ path }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
    },
  });
}
