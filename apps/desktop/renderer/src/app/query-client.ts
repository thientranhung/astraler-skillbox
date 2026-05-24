import { QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { AppClientError } from "../lib/core-client/client.js";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      gcTime: 5 * 60 * 1000,
      retry: (failureCount, error) => {
        const ae = error as AppClientError;
        if (ae?.code === "validation_error") return false;
        if (ae?.code === "conflict_error") return false;
        return failureCount < 1;
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      onError: (error) => {
        const ae = error as AppClientError;
        toast.error(ae?.userMessage ?? String(error));
      },
    },
  },
});
