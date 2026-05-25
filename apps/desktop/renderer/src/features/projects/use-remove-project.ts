import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

interface RemoveProjectOptions {
  navigateAfter?: boolean;
}

export function useRemoveProject(options: RemoveProjectOptions = {}) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (projectId: number) => methods.removeProject({ projectId }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
      if (options.navigateAfter) {
        void navigate({ to: "/projects" });
      }
    },
    onError: () => {
      toast.error("Failed to remove project");
    },
  });
}
