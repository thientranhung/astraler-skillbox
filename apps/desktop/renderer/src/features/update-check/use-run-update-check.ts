import { useCallback, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import type { UpdateCheckPluginResult } from "@contracts/index.js";

const RATE_LIMIT_MS = 10_000; // Larry-4: minimum re-trigger interval

export type UpdateCheckStatus = "idle" | "running" | "ok" | "disabled" | "git_not_found" | "error";

export function useRunUpdateCheck() {
  const [status, setStatus] = useState<UpdateCheckStatus>("idle");
  const [results, setResults] = useState<UpdateCheckPluginResult[]>([]);
  const lastRunRef = useRef<number>(0);

  const mutation = useMutation({
    mutationFn: async () => {
      const now = Date.now();
      // Larry-4: rate-limit — prevent re-trigger within 10s of last run.
      if (now - lastRunRef.current < RATE_LIMIT_MS) {
        return null;
      }
      return methods.runUpdateCheck();
    },
    onMutate: () => {
      setStatus("running");
    },
    onSuccess: (data) => {
      if (data === null) return; // rate-limited, no state change
      lastRunRef.current = Date.now();
      if (data.status === "disabled") {
        setStatus("disabled");
        toast.info("Update check is disabled. Enable it in Settings → Network.");
      } else if (data.status === "git_not_found") {
        setStatus("git_not_found");
        toast.error("git is required for update checks. Please install git.");
      } else if (data.status === "ok") {
        setStatus("ok");
        setResults(data.plugins ?? []);
        const updateCount = (data.plugins ?? []).filter((p) => p.updateAvailable === true).length;
        if (updateCount > 0) {
          toast.success(`${updateCount} update${updateCount > 1 ? "s" : ""} available`);
        } else {
          toast.success("All plugins up to date");
        }
      } else {
        setStatus("error");
        toast.error("Update check failed");
      }
    },
    onError: () => {
      setStatus("error");
      toast.error("Update check failed");
    },
  });

  const isRateLimited = useCallback(() => {
    return Date.now() - lastRunRef.current < RATE_LIMIT_MS;
  }, []);

  return {
    run: mutation.mutate,
    isRunning: mutation.isPending,
    isRateLimited,
    status,
    results,
  };
}
