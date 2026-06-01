import React from "react";
import { useNavigate } from "@tanstack/react-router";
import { useDashboard } from "../features/dashboard/use-dashboard.js";
import { ErrorDisplay } from "../components/error-display.js";
import type { DashboardGetWarning } from "@contracts/index.js";

const HOST_STATUS_CLASS: Record<string, string> = {
  active: "bg-green-100 text-green-800",
  missing: "bg-red-100 text-red-800",
  error: "bg-red-100 text-red-800",
};

export function DashboardScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const { data, isPending, isError, error, refetch } = useDashboard();

  function navigateToAttention(warnings: DashboardGetWarning[]): void {
    if (warnings.some((warning) => warning.scopeType.startsWith("global_"))) {
      navigate({ to: "/global" });
      return;
    }
    if (warnings.some((warning) => warning.scopeType === "skill" || warning.scopeType === "skill_host_folder")) {
      navigate({ to: "/skills" });
      return;
    }
    navigate({ to: "/projects" });
  }

  if (isPending) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-6">
        <ErrorDisplay error={error} />
        <button
          onClick={() => void refetch()}
          className="mt-3 cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
        >
          Retry
        </button>
      </div>
    );
  }

  if (data?.activeHost == null) {
    return (
      <div className="p-6">
        <p className="text-sm text-zinc-500">No Skill Host Folder configured.</p>
        <button
          onClick={() => navigate({ to: "/setup" })}
          className="mt-3 cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
        >
          Go to Setup
        </button>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {data.activeHost.status === "missing" && (
        <div className="rounded border border-amber-200 bg-amber-50 p-3">
          <p className="text-sm font-medium text-amber-700">Skill Host Folder not found</p>
          <p className="mt-0.5 text-xs text-amber-600">
            The configured folder no longer exists on disk. Go to Settings to choose a new folder.
          </p>
          <button
            onClick={() => navigate({ to: "/settings" })}
            className="mt-2 cursor-pointer rounded border border-amber-300 bg-white px-3 py-1.5 text-xs text-amber-700 hover:bg-amber-50"
          >
            Go to Settings
          </button>
        </div>
      )}
      {/* Host block */}
      <section>
        <h2 className="text-base font-semibold text-zinc-900">Skill Host Folder</h2>
        <div className="mt-2 divide-y divide-zinc-100 rounded border border-zinc-200">
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Path</span>
            <span className="font-mono text-xs text-zinc-500">{data.activeHost.path}</span>
          </div>
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Status</span>
            <span className={`rounded px-2 py-0.5 text-xs font-medium ${HOST_STATUS_CLASS[data.activeHost.status] ?? "bg-zinc-100 text-zinc-600"}`}>
              {data.activeHost.status}
            </span>
          </div>
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Last Scan</span>
            <span className="text-sm text-zinc-500">
              {data.activeHost.lastScanAt ?? "Never"}
            </span>
          </div>
        </div>
      </section>

      {/* Summary */}
      <section>
        <h2 className="text-base font-semibold text-zinc-900">Summary</h2>
        <div className="mt-2 divide-y divide-zinc-100 rounded border border-zinc-200">
          <button
            type="button"
            onClick={() => navigate({ to: "/skills" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Skills</span>
            <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.skills}</span>
          </button>
          <button
            type="button"
            onClick={() => navigate({ to: "/projects" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Projects</span>
            <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.projects}</span>
          </button>
          {data.summary.warnings > 0 && (
            <button
              type="button"
              onClick={() => navigateToAttention(data.warnings)}
              className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
            >
              <span className="text-sm font-medium text-zinc-700">Attention needed</span>
              <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.warnings}</span>
            </button>
          )}
          <button
            type="button"
            onClick={() => navigate({ to: "/global" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Global Skills</span>
            <span className="text-xs text-blue-600 group-hover:text-blue-700 group-hover:underline">Open global view</span>
          </button>
          <button
            type="button"
            onClick={() => navigate({ to: "/plugins" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Global Plugins</span>
            <span className="text-xs text-blue-600 group-hover:text-blue-700 group-hover:underline">Open plugins view</span>
          </button>
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Updates</span>
            <span className="text-xs text-zinc-400">Not in this slice</span>
          </div>
        </div>
      </section>

      {/* Installs by mode */}
      <section>
        <h2 className="text-base font-semibold text-zinc-900">Installs by Mode</h2>
        <div className="mt-2 divide-y divide-zinc-100 rounded border border-zinc-200">
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Symlink</span>
            <span className="text-sm text-zinc-500">{data.installsByMode.symlink}</span>
          </div>
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Rsync-copy</span>
            <span className="text-sm text-zinc-500">{data.installsByMode.rsyncCopy}</span>
          </div>
          <div className="flex items-center justify-between px-4 py-3">
            <span className="text-sm font-medium text-zinc-700">Direct</span>
            <span className="text-sm text-zinc-500">{data.installsByMode.direct}</span>
          </div>
        </div>
      </section>

      {/* Zero-data CTA */}
      {data.summary.projects === 0 && (
        <div className="flex gap-2">
          <button
            onClick={() => navigate({ to: "/projects" })}
            className="cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Add Project
          </button>
          <button
            onClick={() => navigate({ to: "/skills" })}
            className="cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            View Skills
          </button>
        </div>
      )}

    </div>
  );
}
