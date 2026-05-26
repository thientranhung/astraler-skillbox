import React from "react";
import { RefreshCw, AlertTriangle, FolderOpen } from "lucide-react";
import { useGlobalList } from "../features/global-skills/use-global-list.js";
import { useScanGlobal } from "../features/global-skills/use-scan-global.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { methods } from "../lib/core-client/methods.js";
import type { GlobalListLocation, GlobalListEntry, GlobalListWarning } from "@contracts/index.js";

function statusBadgeClass(status: GlobalListLocation["status"]): string {
  switch (status) {
    case "active": return "bg-green-100 text-green-700";
    case "empty": return "bg-zinc-100 text-zinc-500";
    case "missing": return "bg-red-100 text-red-600";
    case "not_configured": return "bg-zinc-100 text-zinc-400";
    default: return "bg-yellow-100 text-yellow-700";
  }
}

function entryStatusBadgeClass(status: GlobalListEntry["status"]): string {
  switch (status) {
    case "current": return "bg-green-100 text-green-700";
    case "broken_symlink":
    case "missing": return "bg-red-100 text-red-600";
    default: return "bg-yellow-100 text-yellow-700";
  }
}

function WarningSeverityIcon({ severity }: { severity: GlobalListWarning["severity"] }) {
  const cls = severity === "error" || severity === "blocking" ? "text-red-500" : "text-yellow-500";
  return <AlertTriangle size={12} className={`mt-0.5 shrink-0 ${cls}`} />;
}

export function GlobalSkillsScreen(): React.JSX.Element {
  const { data, isPending, isError, error } = useGlobalList();
  const scanMutation = useScanGlobal();

  const isScanning = scanMutation.operationId != null;
  const locations = data?.locations ?? [];

  function handleOpenFolder(path: string): void {
    void methods.openPath(path);
  }

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <h2 className="text-sm font-semibold text-zinc-900">Global Skills</h2>
        <button
          onClick={() => scanMutation.mutate()}
          disabled={isScanning || scanMutation.isPending}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
          <RefreshCw size={13} className={isScanning ? "animate-spin" : ""} />
          {isScanning ? "Scanning…" : "Scan Global"}
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {isPending && (
          <div className="flex h-40 items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-700" />
          </div>
        )}

        {isError && (
          <div className="p-4">
            <ErrorDisplay error={error} />
          </div>
        )}

        {!isPending && !isError && locations.length === 0 && (
          <EmptyState
            message="No global skills found."
            description="Run Scan Global to detect skills in global provider locations."
          />
        )}

        {!isPending && !isError && locations.length > 0 && (
          <div className="divide-y divide-zinc-100">
            {locations.map((loc) => (
              <div key={loc.globalProviderLocationId} className="p-4">
                {/* Location header */}
                <div className="mb-2 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-zinc-900">{loc.providerDisplayName}</span>
                    <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${statusBadgeClass(loc.status)}`}>
                      {loc.status}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    {loc.lastScannedAt != null && (
                      <span className="text-xs text-zinc-400">
                        Scanned {new Date(loc.lastScannedAt).toLocaleString()}
                      </span>
                    )}
                    {(loc.skillsPath ?? loc.path) != null && (
                      <button
                        onClick={() => handleOpenFolder((loc.skillsPath ?? loc.path)!)}
                        className="flex items-center gap-1 rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50"
                      >
                        <FolderOpen size={12} />
                        Open Folder
                      </button>
                    )}
                  </div>
                </div>

                {loc.path != null && (
                  <p className="mb-2 font-mono text-xs text-zinc-400">{loc.skillsPath ?? loc.path}</p>
                )}

                {/* Location warnings */}
                {loc.warnings.length > 0 && (
                  <div className="mb-2 rounded border border-yellow-100 bg-yellow-50 px-3 py-2">
                    {loc.warnings.map((w, i) => (
                      <div key={i} className="flex items-start gap-1.5 text-xs text-yellow-800">
                        <WarningSeverityIcon severity={w.severity} />
                        <span>{w.message}</span>
                      </div>
                    ))}
                  </div>
                )}

                {/* Entries table */}
                {loc.entries.length > 0 && (
                  <table className="w-full text-left">
                    <thead className="border-b border-zinc-200 bg-zinc-50">
                      <tr>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Skill</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Mode</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Path</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500"></th>
                      </tr>
                    </thead>
                    <tbody>
                      {loc.entries.map((entry) => (
                        <tr key={entry.globalInstallId} className="border-b border-zinc-100">
                          <td className="px-3 py-1.5 text-xs text-zinc-900">{entry.skillName}</td>
                          <td className="px-3 py-1.5 text-xs text-zinc-500">{entry.mode}</td>
                          <td className="px-3 py-1.5">
                            <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${entryStatusBadgeClass(entry.status)}`}>
                              {entry.status}
                            </span>
                          </td>
                          <td className="px-3 py-1.5 font-mono text-xs text-zinc-400">{entry.globalSkillPath}</td>
                          <td className="px-3 py-1.5">
                            <button
                              onClick={() => handleOpenFolder(entry.globalSkillPath)}
                              className="flex items-center gap-1 rounded border border-zinc-200 px-2 py-0.5 text-xs text-zinc-500 hover:bg-zinc-50"
                            >
                              <FolderOpen size={11} />
                              Open
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}

                {loc.entries.length === 0 && (
                  <p className="text-xs text-zinc-400">
                    {loc.status === "active" ? "No entries in this location." : "No global skills found."}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
