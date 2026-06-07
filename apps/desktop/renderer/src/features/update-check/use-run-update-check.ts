import { useCallback, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import type { UpdateCheckPluginResult } from "@contracts/index.js";

const RATE_LIMIT_MS = 10_000; // Larry-4: minimum re-trigger interval

export type UpdateCheckStatus = "idle" | "running" | "ok" | "partial_error" | "all_failed" | "git_not_found" | "error";

export function useRunUpdateCheck() {
  const [status, setStatus] = useState<UpdateCheckStatus>("idle");
  const [results, setResults] = useState<UpdateCheckPluginResult[]>([]);
  const lastRunRef = useRef<number>(0);

  const isRateLimited = useCallback(() => {
    return Date.now() - lastRunRef.current < RATE_LIMIT_MS;
  }, []);

  const mutation = useMutation({
    mutationFn: async () => {
      return methods.runUpdateCheck();
    },
    onMutate: () => {
      setStatus("running");
    },
    onSuccess: (data) => {
      lastRunRef.current = Date.now();
      if (data.status === "git_not_found") {
        setStatus("git_not_found");
        toast.error("git is required for update checks. Please install git.");
      } else if (data.status === "ok") {
        const plugins = data.plugins ?? [];
        setResults(plugins);
        const updateCount = plugins.filter((p) => p.updateAvailable === true).length;
        const errorCount = plugins.filter((p) => p.error != null && p.error !== "").length;
        if (errorCount > 0 && errorCount === plugins.length) {
          setStatus("all_failed");
          const s = errorCount === 1 ? "" : "s";
          toast.error(`Plugin update check failed for all ${errorCount} plugin${s}`);
        } else if (errorCount > 0) {
          setStatus("partial_error");
          const errS = errorCount === 1 ? "" : "s";
          if (updateCount > 0) {
            const updS = updateCount === 1 ? "" : "s";
            toast.warning(`${updateCount} update${updS} available; ${errorCount} plugin${errS} could not be checked`);
          } else {
            toast.warning(`All checked plugins up to date; ${errorCount} plugin${errS} could not be checked`);
          }
        } else {
          setStatus("ok");
          if (plugins.length === 0) {
            toast.info("No plugins to check - no update sources found");
          } else if (updateCount > 0) {
            toast.success(`${updateCount} update${updateCount > 1 ? "s" : ""} available`);
          } else {
            toast.success("All plugins up to date");
          }
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

  const run = useCallback(() => {
    // Larry-4: rate-limit - prevent re-trigger within 10s of last run.
    if (isRateLimited()) return;
    mutation.mutate();
  }, [isRateLimited, mutation]);

  return {
    run,
    isRunning: mutation.isPending,
    isRateLimited,
    status,
    results,
  };
}
