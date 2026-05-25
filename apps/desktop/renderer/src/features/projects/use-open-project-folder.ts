import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";

export function useOpenProjectFolder() {
  return useMutation({
    mutationFn: (path: string) => methods.openPath(path),
    onError: () => {
      toast.error("Failed to open project folder");
    },
  });
}
