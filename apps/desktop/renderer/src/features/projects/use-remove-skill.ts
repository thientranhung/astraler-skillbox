import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { RemoveSkillRequest, OperationProgressNotification } from "@contracts/index.js";

interface RemoveMetadata {
  skillName?: string;
  providerKey?: string;
  alreadyAbsent?: boolean;
}

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

function extractMeta(event: OperationProgressNotification): RemoveMetadata | null {
  if (event.metadata == null || typeof event.metadata !== "object") return null;
  return event.metadata as RemoveMetadata;
}

function successMessage(meta: RemoveMetadata | null): string {
  if (meta?.skillName != null) return `Removed ${meta.skillName}`;
  return "Skill removed";
}

function failedMessage(rawMessage: string | null): string {
  return rawMessage ? `Remove failed: ${rawMessage}` : "Remove failed";
}

export function useRemoveSkill() {
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
    mutationFn: async (req: RemoveSkillRequest) => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.removeSkill(req);
        return { operationId: result.operationId, projectId: req.projectId, buffered };
      } finally {
        tempUnsub();
      }
    },

    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      toast.error(failedMessage(message));
    },

    onSuccess: ({ operationId: opId, projectId, buffered }) => {
      const invalidate = () => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
      };

      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success(successMessage(extractMeta(terminalInBuffer)));
        } else if (terminalInBuffer.status === "failed") {
          toast.error(failedMessage(terminalInBuffer.message));
        }
        invalidate();
        return;
      }

      const toastId = toast.loading("Removing skill…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success(successMessage(extractMeta(event)), { id: toastId });
        } else if (event.status === "failed") {
          toast.error(failedMessage(event.message), { id: toastId });
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Removing: ${event.message}` : "Removing skill…", {
            id: toastId,
          });
        }

        if (isTerminal(event.status)) {
          invalidate();
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
