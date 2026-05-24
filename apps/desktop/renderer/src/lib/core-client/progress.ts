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

// Subscribes to ALL operation.progress events without operationId filtering.
// Used to buffer events that may arrive before an operationId is known.
export function subscribeAllProgress(
  onProgress: (p: OperationProgressNotification) => void,
): () => void {
  return window.core.onEvent("operation.progress", (params) => {
    onProgress(params as OperationProgressNotification);
  });
}
