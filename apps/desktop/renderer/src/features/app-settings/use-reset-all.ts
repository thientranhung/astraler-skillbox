import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";

export function useResetAll() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    mutationFn: () => methods.resetAllData(),
    onSuccess: () => {
      queryClient.clear();
      void navigate({ to: "/setup", replace: true });
    },
    onError: (err) => {
      const msg = err instanceof Error ? err.message : String(err);
      toast.error(`Reset failed: ${msg}`);
    },
  });
}
