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

export function useScanProviderPluginsGlobal() {
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
    mutationFn: async () => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.scanProviderPluginsGlobal();
        return { operationId: result.operationId, buffered };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, buffered }) => {
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success("Plugin settings scanned");
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Plugin scan failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
        return;
      }

      const toastId = toast.loading("Scanning plugin settings…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success("Plugin settings scanned", { id: toastId });
        } else if (event.status === "failed") {
          toast.error(
            `Plugin scan failed${event.message ? `: ${event.message}` : ""}`,
            { id: toastId },
          );
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(
            event.message ? `Scanning: ${event.message}` : "Scanning plugin settings…",
            { id: toastId },
          );
        }

        if (isTerminal(event.status)) {
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
