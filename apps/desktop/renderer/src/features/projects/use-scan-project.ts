import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { OperationProgressNotification } from "@contracts/index.js";

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

export function useScanProject() {
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
    mutationKey: ['scan-project'] as const,
    mutationFn: async (projectId: number) => {
      // Subscribe to ALL progress events BEFORE the RPC call so events emitted
      // during the round-trip are captured in the buffer rather than dropped.
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.scanProject({ projectId });
        return { operationId: result.operationId, projectId, buffered };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, projectId, buffered }) => {
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success("Project scanned");
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Project scan failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
        void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
        return;
      }

      const toastId = toast.loading("Scanning project…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success("Project scanned", { id: toastId });
        } else if (event.status === "failed") {
          toast.error(
            `Project scan failed${event.message ? `: ${event.message}` : ""}`,
            { id: toastId },
          );
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Scanning: ${event.message}` : "Scanning project…", {
            id: toastId,
          });
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
          void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
          setOperationId(null);
          unsub();
          unsubRef.current = null;
        }
      });

      unsubRef.current = unsub;
      setOperationId(opId);
    },
  });

  const clearOperation = useCallback(() => {
    unsubRef.current?.();
    unsubRef.current = null;
    setOperationId(null);
  }, []);

  return { ...mutation, operationId, clearOperation };
}
