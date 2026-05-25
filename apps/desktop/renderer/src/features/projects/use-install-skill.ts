import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { InstallSkillRequest, OperationProgressNotification } from "@contracts/index.js";

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

export function useInstallSkill() {
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
    mutationFn: async (req: InstallSkillRequest) => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.installSkill(req);
        return { operationId: result.operationId, projectId: req.projectId, buffered };
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
          toast.success("Skills installed");
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Skill install failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
        return;
      }

      const toastId = toast.loading("Installing skills…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success("Skills installed", { id: toastId });
        } else if (event.status === "failed") {
          toast.error(
            `Skill install failed${event.message ? `: ${event.message}` : ""}`,
            { id: toastId },
          );
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Installing: ${event.message}` : "Installing skills…", {
            id: toastId,
          });
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
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
