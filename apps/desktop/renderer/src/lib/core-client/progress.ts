import type { OperationProgressNotification } from "@contracts/notifications/operation-progress.js";

export function subscribeOperationProgress(
  operationId: number,
  onProgress: (p: OperationProgressNotification) => void,
): () => void {
  return window.core.onEvent("operation.progress", (params) => {
    const p = params as OperationProgressNotification;
    if (p.operationId === operationId) {
      onProgress(p);
    }
  });
}
