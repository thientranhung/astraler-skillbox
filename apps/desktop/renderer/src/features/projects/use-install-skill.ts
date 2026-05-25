import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { InstallSkillRequest, OperationProgressNotification } from "@contracts/index.js";

interface InstallMetadata {
  created?: number;
  failed?: number;
  requested?: number;
}

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

function extractMeta(event: OperationProgressNotification): InstallMetadata | null {
  if (event.metadata == null || typeof event.metadata !== "object") return null;
  return event.metadata as InstallMetadata;
}

function successMessage(meta: InstallMetadata | null): string {
  if (meta?.created != null) return `Skills installed (${meta.created})`;
  return "Skills installed";
}

function failedMessage(meta: InstallMetadata | null, rawMessage: string | null): string {
  const parts: string[] = [];
  if (meta?.created != null && meta?.requested != null) {
    parts.push(`${meta.created}/${meta.requested} installed`);
  }
  if (rawMessage) parts.push(rawMessage);
  return parts.length > 0 ? `Skill install failed: ${parts.join(". ")}` : "Skill install failed";
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
        const meta = extractMeta(terminalInBuffer);
        if (terminalInBuffer.status === "success") {
          toast.success(successMessage(meta));
        } else if (terminalInBuffer.status === "failed") {
          toast.error(failedMessage(meta, terminalInBuffer.message));
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
        return;
      }

      const toastId = toast.loading("Installing skills…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        const meta = extractMeta(event);
        if (event.status === "success") {
          toast.success(successMessage(meta), { id: toastId });
        } else if (event.status === "failed") {
          toast.error(failedMessage(meta, event.message), { id: toastId });
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Installing: ${event.message}` : "Installing skills…", {
            id: toastId,
          });
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
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
