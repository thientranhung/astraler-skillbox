import React from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, RefreshCw, AlertTriangle, AlertCircle, Info } from "lucide-react";
import { useProjectDetail } from "../features/projects/use-project-detail.js";
import { useScanProject } from "../features/projects/use-scan-project.js";
import { ProjectStatusBadge } from "../features/projects/project-status-badge.js";
import { ErrorDisplay } from "../components/error-display.js";
import type { ProjectGetEntry, ProjectGetWarning, ProjectGetProvider } from "@contracts/index.js";

const ENTRY_STATUS_CONFIG: Record<ProjectGetEntry["status"], { label: string; cls: string }> = {
  current: { label: "Current", cls: "bg-green-100 text-green-800" },
  old_host: { label: "Old Host", cls: "bg-yellow-100 text-yellow-700" },
  external_symlink: { label: "External", cls: "bg-yellow-100 text-yellow-700" },
  broken_symlink: { label: "Broken", cls: "bg-red-100 text-red-800" },
  missing: { label: "Missing", cls: "bg-red-100 text-red-800" },
  error: { label: "Error", cls: "bg-red-100 text-red-800" },
};

const WARNING_CONFIG: Record<
  ProjectGetWarning["severity"],
  { cls: string; Icon: React.ElementType }
> = {
  info: { cls: "border-blue-200 bg-blue-50 text-blue-800", Icon: Info },
  warning: { cls: "border-yellow-200 bg-yellow-50 text-yellow-800", Icon: AlertTriangle },
  error: { cls: "border-red-200 bg-red-50 text-red-800", Icon: AlertCircle },
};

const DETECTION_CLS: Record<ProjectGetProvider["detectionStatus"], string> = {
  detected: "bg-blue-100 text-blue-800",
  missing: "bg-zinc-100 text-zinc-500",
  invalid_structure: "bg-yellow-100 text-yellow-700",
};

const PROVIDER_STATUS_CLS: Record<ProjectGetProvider["providerStatus"], string> = {
  supported: "bg-green-100 text-green-800",
  experimental: "bg-yellow-100 text-yellow-700",
  unsupported: "bg-zinc-100 text-zinc-500",
  disabled: "bg-zinc-100 text-zinc-400",
};

function EntryStatusBadge({ status }: { status: ProjectGetEntry["status"] }): React.JSX.Element {
  const cfg = ENTRY_STATUS_CONFIG[status] ?? ENTRY_STATUS_CONFIG.error;
  return (
    <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${cfg.cls}`}>
      {cfg.label}
    </span>
  );
}

function WarningBanner({ warning }: { warning: ProjectGetWarning }): React.JSX.Element {
  const { cls, Icon } = WARNING_CONFIG[warning.severity] ?? WARNING_CONFIG.warning;
  return (
    <div className={`flex items-start gap-2 rounded border px-3 py-2 text-xs ${cls}`}>
      <Icon size={13} className="mt-0.5 shrink-0" />
      <div>
        <span className="font-medium">{warning.code}</span>
        {warning.scopeRef != null && (
          <span className="ml-1 opacity-70">({warning.scopeRef})</span>
        )}
        {" — "}
        {warning.message}
      </div>
    </div>
  );
}

function ProviderRow({ provider }: { provider: ProjectGetProvider }): React.JSX.Element {
  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{provider.displayName}</td>
      <td className="px-3 py-1.5 text-xs">
        <span className={`inline-flex items-center rounded px-1.5 py-0.5 font-medium ${DETECTION_CLS[provider.detectionStatus] ?? DETECTION_CLS.missing}`}>
          {provider.detectionStatus.replace("_", " ")}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <span className={`inline-flex items-center rounded px-1.5 py-0.5 font-medium ${PROVIDER_STATUS_CLS[provider.providerStatus] ?? PROVIDER_STATUS_CLS.unsupported}`}>
          {provider.providerStatus}
        </span>
      </td>
      <td className="max-w-xs truncate px-3 py-1.5 font-mono text-xs text-zinc-400" title={provider.detectedPath ?? undefined}>
        {provider.detectedPath ?? "—"}
      </td>
      <td className="px-3 py-1.5 text-xs text-zinc-500">{provider.entryCount}</td>
    </tr>
  );
}

function EntryRow({ entry }: { entry: ProjectGetEntry }): React.JSX.Element {
  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-1.5 text-xs text-zinc-500">{entry.providerKey}</td>
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{entry.name}</td>
      <td className="px-3 py-1.5 text-xs">
        <span className="inline-flex items-center rounded bg-zinc-100 px-1.5 py-0.5 font-medium text-zinc-600">
          {entry.mode}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <EntryStatusBadge status={entry.status} />
      </td>
      <td className="max-w-xs truncate px-3 py-1.5 font-mono text-xs text-zinc-400" title={entry.projectSkillPath}>
        {entry.projectSkillPath}
      </td>
      <td className="max-w-xs truncate px-3 py-1.5 font-mono text-xs text-zinc-400" title={entry.symlinkTargetPath ?? undefined}>
        {entry.symlinkTargetPath ?? "—"}
      </td>
      <td className="px-3 py-1.5 text-xs text-zinc-400">{entry.skillId ?? "—"}</td>
    </tr>
  );
}

export function ProjectDetailScreen(): React.JSX.Element {
  const { projectId: projectIdStr } = useParams({ from: "/projects/$projectId" });
  const validId: number | null =
    /^\d+$/.test(projectIdStr) && Number(projectIdStr) > 0 ? Number(projectIdStr) : null;
  const navigate = useNavigate();
  const { data, isPending, isError, error } = useProjectDetail(validId);
  const scan = useScanProject();
  const isScanning = scan.operationId != null || scan.isPending;

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <button
            onClick={() => void navigate({ to: "/projects" })}
            className="flex shrink-0 items-center gap-1 text-xs text-zinc-500 hover:text-zinc-800"
          >
            <ArrowLeft size={13} />
            Projects
          </button>
          {data != null && (
            <>
              <span className="shrink-0 text-zinc-300">/</span>
              <div className="flex min-w-0 items-center gap-2">
                <span className="truncate text-sm font-semibold text-zinc-900">{data.project.name}</span>
                <span className="hidden truncate font-mono text-xs text-zinc-400 md:block" title={data.project.path}>
                  {data.project.path}
                </span>
              </div>
              <ProjectStatusBadge status={data.project.status} />
            </>
          )}
        </div>
        {data != null && (
          <div className="flex shrink-0 items-center gap-3">
            {data.project.lastScannedAt != null && (
              <span className="hidden text-xs text-zinc-400 sm:block">
                Scanned {new Date(data.project.lastScannedAt).toLocaleString()}
              </span>
            )}
            <button
              onClick={() => scan.mutate(validId!)}
              disabled={isScanning}
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
            >
              <RefreshCw size={12} className={isScanning ? "animate-spin" : ""} />
              {isScanning ? "Scanning…" : "Scan"}
            </button>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {validId == null && (
          <div className="p-4">
            <div className="flex items-start gap-2 rounded border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-800">
              <AlertCircle size={13} className="mt-0.5 shrink-0" />
              Invalid project ID: <span className="ml-1 font-mono">{projectIdStr}</span>
            </div>
          </div>
        )}

        {validId != null && isPending && (
          <div className="flex h-40 items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
          </div>
        )}

        {validId != null && isError && (
          <div className="p-4">
            <ErrorDisplay error={error} />
          </div>
        )}

        {validId != null && !isPending && !isError && data != null && (
          <div className="flex flex-col gap-4 p-4">
            {/* Warnings */}
            {data.warnings.length > 0 && (
              <div className="flex flex-col gap-1.5">
                {data.warnings.map((w, i) => (
                  <WarningBanner key={i} warning={w} />
                ))}
              </div>
            )}

            {/* Providers */}
            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                Providers
              </h3>
              {data.providers.length === 0 ? (
                <p className="text-xs text-zinc-400">No providers detected.</p>
              ) : (
                <div className="overflow-x-auto rounded border border-zinc-200">
                  <table className="w-full text-left">
                    <thead className="border-b border-zinc-200 bg-zinc-50">
                      <tr>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Provider</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Detection</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Detected Path</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Entries</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.providers.map((p) => (
                        <ProviderRow key={p.projectProviderId} provider={p} />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Skill Entries */}
            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                Skill Entries
                {data.entries.length > 0 && (
                  <span className="ml-1 font-normal normal-case text-zinc-400">
                    ({data.entries.length})
                  </span>
                )}
              </h3>
              {data.entries.length === 0 ? (
                <p className="text-xs text-zinc-400">
                  No skill entries observed. Run a scan to populate entries.
                </p>
              ) : (
                <div className="overflow-x-auto rounded border border-zinc-200">
                  <table className="w-full text-left">
                    <thead className="border-b border-zinc-200 bg-zinc-50">
                      <tr>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Provider</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Name</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Mode</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Project Skill Path</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Symlink Target</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Skill ID</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.entries.map((entry) => (
                        <EntryRow key={entry.id} entry={entry} />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
