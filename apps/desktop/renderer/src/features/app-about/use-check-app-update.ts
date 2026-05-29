import { useEffect } from "react";
import { useMutation } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import type { AppCheckUpdateResponse } from "@contracts/index.js";

export type CheckAppUpdateStatus =
  | "idle"
  | "checking"
  | "up-to-date"
  | "available"
  | "error";

export interface CheckAppUpdateState {
  isPending: boolean;
  status: CheckAppUpdateStatus;
  currentVersion: string | null;
  latestVersion: string | null;
  updateAvailable: boolean;
  releaseUrl: string | null;
  check: () => void;
}

function deriveStatus(data: AppCheckUpdateResponse | undefined): CheckAppUpdateStatus {
  if (!data) return "idle";
  if (data.error != null) return "error";
  return data.updateAvailable ? "available" : "up-to-date";
}

export function useCheckAppUpdate(): CheckAppUpdateState {
  const mutation = useMutation({
    mutationFn: () => methods.checkAppUpdate(),
  });

  // Auto-check when About screen mounts — no opt-in needed.
  useEffect(() => {
    mutation.mutate();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const data = mutation.data;
  const status: CheckAppUpdateStatus = mutation.isPending
    ? "checking"
    : deriveStatus(data);

  return {
    isPending: mutation.isPending,
    status,
    currentVersion: data?.currentVersion ?? import.meta.env.VITE_APP_VERSION ?? null,
    latestVersion: data?.latestVersion ?? null,
    updateAvailable: data?.updateAvailable ?? false,
    releaseUrl: data?.releaseUrl ?? null,
    check: mutation.mutate,
  };
}
