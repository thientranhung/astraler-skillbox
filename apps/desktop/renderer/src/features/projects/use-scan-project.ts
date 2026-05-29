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

type ScanProjectArgs = number | { projectId: number; silent?: boolean };

function normalizeArgs(args: ScanProjectArgs): { projectId: number; silent: boolean } {
  if (typeof args === "number") return { projectId: args, silent: false };
  return { projectId: args.projectId, silent: args.silent ?? false };
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
    mutationFn: async (args: ScanProjectArgs) => {
      const { projectId, silent } = normalizeArgs(args);
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.scanProject({ projectId });
        return { operationId: result.operationId, projectId, buffered, silent };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, projectId, buffered, silent }) => {
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (!silent) {
          if (terminalInBuffer.status === "success") {
            toast.success("Project scanned");
          } else if (terminalInBuffer.status === "failed") {
            toast.error(
              `Project scan failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
            );
          }
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

      const toastId = silent ? undefined : toast.loading("Scanning project…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (!silent) {
          if (event.status === "success") {
            toast.success("Project scanned", { id: toastId });
          } else if (event.status === "failed") {
            toast.error(
              `Project scan failed${event.message ? `: ${event.message}` : ""}`,
              { id: toastId },
            );
          } else if (event.status === "cancelled") {
            if (toastId != null) toast.dismiss(toastId);
          } else {
            toast.loading(event.message ? `Scanning: ${event.message}` : "Scanning project…", {
              id: toastId,
            });
          }
        } else if (event.status === "failed") {
          toast.error(
            `Project scan failed${event.message ? `: ${event.message}` : ""}`,
          );
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
