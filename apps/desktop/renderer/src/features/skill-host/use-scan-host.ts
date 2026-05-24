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

export function useScanHost() {
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
    mutationFn: async (hostId: number) => {
      // Subscribe to ALL progress events BEFORE the RPC call so events emitted
      // during the round-trip (fast scan completing before response reaches
      // renderer) are captured in the buffer rather than silently dropped.
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.scanHost({ hostId });
        return { operationId: result.operationId, hostId, buffered };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, hostId, buffered }) => {
      // Check whether the terminal event already arrived during the round-trip.
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        // Scan completed before we could subscribe — handle immediately.
        if (terminalInBuffer.status === "success") {
          toast.success("Skills scanned");
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Scan failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.skills.list(hostId) });
        return;
      }

      // Not yet terminal — enter scanning state and subscribe for future events.
      const toastId = toast.loading("Scanning skills…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success("Skills scanned", { id: toastId });
        } else if (event.status === "failed") {
          toast.error(`Scan failed${event.message ? `: ${event.message}` : ""}`, { id: toastId });
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Scanning: ${event.message}` : "Scanning skills…", {
            id: toastId,
          });
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.skills.list(hostId) });
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
