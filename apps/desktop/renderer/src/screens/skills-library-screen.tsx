import React, { useCallback } from "react";
import { RefreshCw, AlertTriangle } from "lucide-react";
import { useActiveHost } from "../features/skill-host/use-active-host.js";
import { useSkillsList } from "../features/skills-library/use-skills-list.js";
import { useScanHost } from "../features/skill-host/use-scan-host.js";
import { SkillRow } from "../features/skills-library/skill-row.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { OperationProgressToast } from "../components/operation-progress-toast.js";

export function SkillsLibraryScreen(): React.JSX.Element {
  const activeHost = useActiveHost();
  const { data, isPending, isError, error } = useSkillsList();
  const scanMutation = useScanHost();

  const handleScanComplete = useCallback(() => {
    if (activeHost != null) {
      scanMutation.handleScanComplete(activeHost.hostId);
    }
  }, [activeHost, scanMutation]);

  function handleScan(): void {
    if (activeHost == null) return;
    scanMutation.mutate(activeHost.hostId);
  }

  const isScanning = scanMutation.operationId != null;

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900">Skills Library</h2>
          {data?.hostPath != null && (
            <p className="mt-0.5 font-mono text-xs text-zinc-400">{data.hostPath}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {data?.lastScanAt != null && (
            <span className="text-xs text-zinc-400">
              Last scan: {new Date(data.lastScanAt).toLocaleString()}
            </span>
          )}
          <button
            onClick={handleScan}
            disabled={isScanning || scanMutation.isPending || activeHost == null}
            className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
          >
            <RefreshCw size={13} className={isScanning ? "animate-spin" : ""} />
            {isScanning ? "Scanning…" : "Scan"}
          </button>
        </div>
      </div>

      {/* Warnings */}
      {data?.warnings != null && data.warnings.length > 0 && (
        <div className="border-b border-yellow-100 bg-yellow-50 px-4 py-2">
          {data.warnings.map((w, i) => (
            <div key={i} className="flex items-start gap-1.5 text-xs text-yellow-800">
              <AlertTriangle size={12} className="mt-0.5 shrink-0" />
              <span>{w.message}</span>
            </div>
          ))}
        </div>
      )}

      {/* Totals */}
      {data != null && (
        <div className="flex gap-4 border-b border-zinc-100 px-4 py-2 text-xs text-zinc-500">
          <span>{data.totals.available} available</span>
          {data.totals.missing > 0 && (
            <span className="text-red-600">{data.totals.missing} missing</span>
          )}
          {data.totals.unreadable > 0 && (
            <span className="text-red-600">{data.totals.unreadable} unreadable</span>
          )}
          {data.totals.local_modified > 0 && (
            <span className="text-yellow-600">{data.totals.local_modified} modified</span>
          )}
          {data.totals.unknown > 0 && (
            <span className="text-zinc-400">{data.totals.unknown} unknown</span>
          )}
        </div>
      )}

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

        {!isPending && !isError && data?.skills.length === 0 && (
          <EmptyState
            message="No skills found"
            description="Scan the host folder to discover skills."
          />
        )}

        {!isPending && !isError && data != null && data.skills.length > 0 && (
          <table className="w-full text-left">
            <thead className="border-b border-zinc-200 bg-zinc-50">
              <tr>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Name</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Status</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Path</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Source</th>
              </tr>
            </thead>
            <tbody>
              {data.skills.map((skill) => (
                <SkillRow key={skill.id} skill={skill} />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Operation progress tracking */}
      {scanMutation.operationId != null && (
        <OperationProgressToast
          operationId={scanMutation.operationId}
          label="Scanning skills"
          onComplete={handleScanComplete}
        />
      )}
    </div>
  );
}
