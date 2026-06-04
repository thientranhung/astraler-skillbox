import React, { useRef, useEffect, useState, useMemo } from "react";
import { RefreshCw, FolderOpen } from "lucide-react";
import { useGlobalList } from "../features/global-skills/use-global-list.js";
import { useScanGlobal } from "../features/global-skills/use-scan-global.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { ProviderIcon } from "../components/provider-icon.js";
import { methods } from "../lib/core-client/methods.js";
import { displayPath } from "../lib/display-path.js";
import type { GlobalListLocation, GlobalListEntry } from "@contracts/index.js";
import { sessionAutoScanRegistry, isDataStale } from "../features/scan/auto-scan-constants.js";

function statusBadgeClass(status: GlobalListLocation["status"]): string {
  switch (status) {
    case "active": return "bg-green-100 text-green-700";
    case "empty": return "bg-zinc-100 text-zinc-500";
    case "missing": return "bg-red-100 text-red-600";
    case "not_configured": return "bg-zinc-100 text-zinc-400";
    case "no_global_skills": return "bg-zinc-100 text-zinc-400";
    case "disabled": return "bg-zinc-100 text-zinc-400";
    default: return "bg-yellow-100 text-yellow-700";
  }
}

function statusLabel(status: GlobalListLocation["status"]): string {
  switch (status) {
    case "no_global_skills": return "no global skills";
    case "not_configured": return "not configured";
    default: return status;
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

function ProviderTab({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      onClick={onClick}
      className={`-mb-px mr-0.5 rounded-t px-3 py-1.5 text-xs font-medium border-b-2 ${
        active
          ? "border-zinc-700 text-zinc-900"
          : "border-transparent text-zinc-500 hover:text-zinc-700"
      }`}
    >
      {label}
      <span
        className={`ml-1.5 rounded-full px-1.5 py-0.5 text-[10px] ${
          active ? "bg-zinc-200 text-zinc-700" : "bg-zinc-100 text-zinc-500"
        }`}
      >
        {count}
      </span>
    </button>
  );
}

export function GlobalSkillsScreen(): React.JSX.Element {
  const { data, isPending, isError, error } = useGlobalList();
  const scanMutation = useScanGlobal();
  const [activeProvider, setActiveProvider] = useState<string>("all");

  const isScanning = scanMutation.operationId != null;
  const locations = data?.locations ?? [];

  const autoScannedRef = useRef(false);
  const oldestScannedAt = (() => {
    const locs = data?.locations;
    if (!locs?.length) return null;
    if (locs.some((l) => l.lastScannedAt == null)) return null;
    return locs.reduce<string>((oldest, l) => (l.lastScannedAt! < oldest ? l.lastScannedAt! : oldest), locs[0].lastScannedAt!);
  })();

  useEffect(() => {
    if (data == null) return;
    if (scanMutation.isPending || scanMutation.operationId != null) return;
    const stale = data.locations.length === 0 || data.locations.some((l) => l.lastScannedAt == null) || isDataStale(oldestScannedAt);
    if (!stale) return;
    const key = "auto-scan:global";
    if (autoScannedRef.current || sessionAutoScanRegistry.has(key)) return;
    autoScannedRef.current = true;
    sessionAutoScanRegistry.add(key);
    scanMutation.mutate({ silent: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, oldestScannedAt]);

  const providerTabs = useMemo(() => {
    const seen = new Set<string>();
    const tabs: Array<{ key: string; displayName: string; count: number }> = [];
    for (const loc of locations) {
      if (!seen.has(loc.providerKey)) {
        seen.add(loc.providerKey);
        const count = locations
          .filter((l) => l.providerKey === loc.providerKey)
          .reduce((sum, l) => sum + l.entries.length, 0);
        tabs.push({ key: loc.providerKey, displayName: loc.providerDisplayName, count });
      }
    }
    return tabs;
  }, [locations]);

  const totalCount = locations.reduce((sum, l) => sum + l.entries.length, 0);

  const visibleLocations =
    activeProvider === "all"
      ? locations
      : locations.filter((l) => l.providerKey === activeProvider);

  function handleOpenFolder(path: string): void {
    void methods.openPath(path);
  }

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900">Global Skills</h2>
          <p className="mt-0.5 text-xs text-zinc-400">
            Read-only scan of global provider folders. Status badges show whether each location and skill entry is usable.
          </p>
        </div>
        <button
          onClick={() => scanMutation.mutate()}
          disabled={isScanning || scanMutation.isPending}
          className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <RefreshCw size={13} className={isScanning ? "animate-spin" : ""} />
          {isScanning ? "Scanning…" : "Scan Global"}
        </button>
      </div>

      {/* Provider tabs */}
      {!isPending && !isError && locations.length > 0 && (
        <div className="flex border-b border-zinc-200 px-4 pt-2">
          <ProviderTab
            label="All"
            count={totalCount}
            active={activeProvider === "all"}
            onClick={() => setActiveProvider("all")}
          />
          {providerTabs.map((tab) => (
            <ProviderTab
              key={tab.key}
              label={tab.displayName}
              count={tab.count}
              active={activeProvider === tab.key}
              onClick={() => setActiveProvider(tab.key)}
            />
          ))}
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

        {!isPending && !isError && locations.length === 0 && (
          <EmptyState
            message="No global skills found."
            description="Run Scan Global to detect skills in global provider locations."
          />
        )}

        {!isPending && !isError && locations.length > 0 && visibleLocations.length === 0 && (
          <EmptyState
            message="No skills for this provider."
            description="This provider has no global skills in the current scan."
          />
        )}

        {!isPending && !isError && visibleLocations.length > 0 && (
          <div className="divide-y divide-zinc-100">
            {visibleLocations.map((loc) => (
              <div key={loc.globalProviderLocationId} className="p-4">
                {/* Location header */}
                <div className="mb-2 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex items-center gap-1.5 text-sm font-medium text-zinc-900">
                      <ProviderIcon providerKey={loc.providerKey} />
                      {loc.providerDisplayName}
                    </span>
                    <span className={`rounded px-1.5 py-0.5 text-xs font-medium ${statusBadgeClass(loc.status)}`}>
                      {statusLabel(loc.status)}
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
                        className="flex cursor-pointer items-center gap-1 rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50"
                      >
                        <FolderOpen size={12} />
                        Open Folder
                      </button>
                    )}
                  </div>
                </div>

                {loc.path != null && (
                  <p className="mb-2 font-mono text-xs text-zinc-400">
                    {displayPath(loc.skillsPath ?? loc.path)}
                  </p>
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
                          <td className="px-3 py-1.5 font-mono text-xs text-zinc-400">
                            {displayPath(entry.globalSkillPath)}
                          </td>
                          <td className="px-3 py-1.5">
                            <button
                              onClick={() => handleOpenFolder(entry.globalSkillPath)}
                              className="flex cursor-pointer items-center gap-1 rounded border border-zinc-200 px-2 py-0.5 text-xs text-zinc-500 hover:bg-zinc-50"
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
