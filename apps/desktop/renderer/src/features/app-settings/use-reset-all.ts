import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
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
  });
}
