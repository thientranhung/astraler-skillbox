import { useEffect, useRef } from "react";
import { toast } from "sonner";
import { subscribeOperationProgress } from "../lib/core-client/progress.js";

interface OperationProgressToastProps {
  operationId: number;
  label: string;
  onComplete: () => void;
}

export function OperationProgressToast({
  operationId,
  label,
  onComplete,
}: OperationProgressToastProps): null {
  const onCompleteRef = useRef(onComplete);
  onCompleteRef.current = onComplete;

  useEffect(() => {
    const toastId = toast.loading(`${label}…`);

    const unsub = subscribeOperationProgress(operationId, (event) => {
      const terminal = event.status === "success" || event.status === "failed" || event.status === "cancelled";

      if (event.status === "success") {
        toast.success(label, { id: toastId });
      } else if (event.status === "failed") {
        toast.error(`${label} failed${event.message ? `: ${event.message}` : ""}`, { id: toastId });
      } else if (event.status === "cancelled") {
        toast.dismiss(toastId);
      } else {
        toast.loading(event.message ? `${label}: ${event.message}` : `${label}…`, { id: toastId });
      }

      if (terminal) {
        onCompleteRef.current();
        unsub();
      }
    });

    return () => {
      toast.dismiss(toastId);
      unsub();
    };
  }, [operationId, label]);

  return null;
}
