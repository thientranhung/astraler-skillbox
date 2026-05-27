import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { OperationProgressNotification, ProviderPluginSetEnabledRequest } from "@contracts/index.js";

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

export function useSetProviderPluginEnabled() {
  const queryClient = useQueryClient();
  const [operationId, setOperationId] = useState<number | null>(null);
  const unsubRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    return () => {
      unsubRef.current?.();
      unsubRef.current = null;
    };
  }, []);

  const mutation = useMutation({
    mutationFn: async (req: ProviderPluginSetEnabledRequest) => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.setProviderPluginEnabled(req);
        return { operationId: result.operationId, buffered, req };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, buffered, req }) => {
      const action = req.enabled ? "enabled" : "disabled";
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success(`Plugin ${action}`);
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Plugin update failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
        if (req.projectId != null) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(req.projectId) });
        }
        return;
      }

      const toastId = toast.loading(`Updating plugin…`);

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success(`Plugin ${action}`, { id: toastId });
        } else if (event.status === "failed") {
          toast.error(
            `Plugin update failed${event.message ? `: ${event.message}` : ""}`,
            { id: toastId },
          );
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
          if (req.projectId != null) {
            void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(req.projectId) });
          }
          setOperationId(null);
          unsub();
          unsubRef.current = null;
        }
      });

      unsubRef.current = unsub;
      setOperationId(opId);
    },

    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Plugin update failed: ${msg}`);
    },
  });

  const clearOperation = useCallback(() => {
    unsubRef.current?.();
    unsubRef.current = null;
    setOperationId(null);
  }, []);

  return { ...mutation, operationId, clearOperation };
}
