import { useMutation } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";

export function useResetAll() {
  return useMutation({
    mutationFn: () => methods.resetAllData(),
  });
}
