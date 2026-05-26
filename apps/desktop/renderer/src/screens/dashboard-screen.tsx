import React from "react";
import { useNavigate } from "@tanstack/react-router";
import { useDashboard } from "../features/dashboard/use-dashboard.js";
import { ErrorDisplay } from "../components/error-display.js";
import type { DashboardGetWarning } from "@contracts/index.js";

const WARNING_SEVERITY_CLASS: Record<DashboardGetWarning["severity"], string> = {
  info: "bg-blue-100 text-blue-700",
  warning: "bg-yellow-100 text-yellow-700",
  error: "bg-red-100 text-red-700",
  blocking: "bg-red-100 text-red-800",
};

function scopeLabel(scopeType: DashboardGetWarning["scopeType"]): string {
  switch (scopeType) {
    case "project": return "Project warning";
    case "skill_host_folder": return "Skill Host";
    case "project_provider": return "Project provider";
    case "install": return "Project skill";
    default: return scopeType.replaceAll("_", " ");
  }
}

export function DashboardScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const { data, isPending, isError, error, refetch } = useDashboard();
  const warningsRef = React.useRef<HTMLElement | null>(null);

  function scrollToWarnings(): void {
    warningsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function navigateForWarning(warning: DashboardGetWarning): void {
    if (warning.scopeType === "project" && warning.scopeId != null) {
      navigate({
        to: "/projects/$projectId",
        params: { projectId: String(warning.scopeId) },
      });
      return;
    }
    if (warning.scopeType === "skill" && warning.scopeId != null) {
      navigate({
        to: "/skills/$skillId",
        params: { skillId: String(warning.scopeId) },
      });
      return;
    }
    if (warning.scopeType === "skill_host_folder") {
      navigate({ to: "/skills" });
      return;
    }
    if (warning.scopeType === "project_provider" || warning.scopeType === "install") {
      navigate({ to: "/projects" });
      return;
    }
    scrollToWarnings();
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
          className="mt-3 rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
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
          className="mt-3 rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
        >
          Go to Setup
        </button>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
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
            <span className="rounded bg-zinc-100 px-2 py-0.5 text-xs text-zinc-600">
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
            className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Skills</span>
            <span className="text-sm text-zinc-500">{data.summary.skills}</span>
          </button>
          <button
            type="button"
            onClick={() => navigate({ to: "/projects" })}
            className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Projects</span>
            <span className="text-sm text-zinc-500">{data.summary.projects}</span>
          </button>
          <button
            type="button"
            onClick={scrollToWarnings}
            className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Warnings</span>
            <span className="text-sm text-zinc-500">{data.summary.warnings}</span>
          </button>
          <button
            type="button"
            onClick={() => navigate({ to: "/global" })}
            className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Global Skills</span>
            <span className="text-xs text-zinc-500">Open global view</span>
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
            className="rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Add Project
          </button>
          <button
            onClick={() => navigate({ to: "/skills" })}
            className="rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            View Skills
          </button>
        </div>
      )}

      {/* Warnings */}
      <section ref={warningsRef}>
        <h2 className="text-base font-semibold text-zinc-900">Warnings</h2>
        <p className="mt-1 text-xs text-zinc-400">
          Warnings point to places Skillbox could not read, classify, or validate during the last scan.
        </p>
        <div className="mt-2">
          {data.warnings.length === 0 ? (
            <p className="text-sm text-zinc-500">No active warnings</p>
          ) : (
            <div className="divide-y divide-zinc-100 rounded border border-zinc-200">
              {data.warnings.map((w) => (
                <div key={`${w.scopeType}-${String(w.scopeId)}-${w.code}`} className="flex items-start justify-between gap-3 px-4 py-3">
                  <div>
                    <div className="mb-1 flex flex-wrap items-center gap-1.5">
                      <span className={`rounded px-1.5 py-0.5 text-xs font-medium uppercase ${WARNING_SEVERITY_CLASS[w.severity] ?? WARNING_SEVERITY_CLASS.warning}`}>
                        {w.severity}
                      </span>
                      <span className="text-xs font-medium text-zinc-500">{scopeLabel(w.scopeType)}</span>
                      <span className="font-mono text-[11px] text-zinc-400">{w.code}</span>
                    </div>
                    <p className="text-sm text-zinc-700">{w.message}</p>
                  </div>
                  {(w.scopeId != null || w.scopeType === "skill_host_folder" || w.scopeType === "project_provider" || w.scopeType === "install") && (
                    <button
                      type="button"
                      onClick={() => navigateForWarning(w)}
                      className="shrink-0 rounded border border-zinc-200 px-2 py-1 text-xs font-medium text-zinc-600 hover:bg-zinc-50"
                    >
                      Open
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
