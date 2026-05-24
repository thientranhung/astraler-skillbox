import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";

export function useScanHost() {
  const queryClient = useQueryClient();
  const [operationId, setOperationId] = useState<number | null>(null);

  const mutation = useMutation({
    mutationFn: (hostId: number) => methods.scanHost({ hostId }),
    onSuccess: (data) => {
      setOperationId(data.operationId);
    },
  });

  const handleScanComplete = useCallback(
    (hostId: number) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.skills.list(hostId) });
      setOperationId(null);
    },
    [queryClient],
  );

  const clearOperation = useCallback(() => setOperationId(null), []);

  return { ...mutation, operationId, handleScanComplete, clearOperation };
}
