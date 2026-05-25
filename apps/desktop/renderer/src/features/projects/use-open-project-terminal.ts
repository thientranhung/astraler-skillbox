import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";

export function useOpenProjectTerminal() {
  return useMutation({
    mutationFn: (path: string) => methods.openTerminal(path),
    onError: () => {
      toast.error("Failed to open Terminal");
    },
  });
}
