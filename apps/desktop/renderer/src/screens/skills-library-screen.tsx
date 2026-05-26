import React, { useState } from "react";
import { RefreshCw, AlertTriangle, FolderOpen, Search, TerminalSquare } from "lucide-react";
import { useActiveHost } from "../features/skill-host/use-active-host.js";
import { useSkillsList } from "../features/skills-library/use-skills-list.js";
import { useScanHost } from "../features/skill-host/use-scan-host.js";
import { SkillRow } from "../features/skills-library/skill-row.js";
import { ErrorDisplay } from "../components/error-display.js";
import { EmptyState } from "../components/empty-state.js";
import { methods } from "../lib/core-client/methods.js";

type SkillStatus = "all" | "available" | "missing" | "unreadable" | "local_modified" | "unknown";
type ProviderView = "all" | "shared_agents";

export function SkillsLibraryScreen(): React.JSX.Element {
  const activeHost = useActiveHost();
  const { data, isPending, isError, error } = useSkillsList();
  const scanMutation = useScanHost();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<SkillStatus>("all");
  const [providerView, setProviderView] = useState<ProviderView>("all");

  function handleScan(): void {
    if (activeHost == null) return;
    scanMutation.mutate(activeHost.hostId);
  }

  function handleOpenHostFolder(): void {
    if (data?.hostPath == null) return;
    void methods.openPath(data.hostPath);
  }

  function handleOpenTerminal(): void {
    if (data?.hostPath == null) return;
    void methods.openTerminal(data.hostPath);
  }

  const isScanning = scanMutation.operationId != null;
  const skills = data?.skills ?? [];
  const sharedAgentSkills = skills.filter((skill) => skill.relativePath.startsWith(".agents/skills/"));
  const providerScopedSkills = providerView === "shared_agents" ? sharedAgentSkills : skills;

  const filteredSkills = providerScopedSkills.filter((skill) => {
    const matchesSearch = search.trim() === "" || skill.name.toLowerCase().includes(search.toLowerCase());
    const matchesStatus = statusFilter === "all" || skill.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

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
            onClick={handleOpenHostFolder}
            disabled={data?.hostPath == null}
            className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <FolderOpen size={13} />
            Open Folder
          </button>
          <button
            onClick={handleOpenTerminal}
            disabled={data?.hostPath == null}
            title="Open terminal at Skill Host Folder"
            className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <TerminalSquare size={13} />
            Terminal
          </button>
          <button
            onClick={handleScan}
            disabled={isScanning || scanMutation.isPending || activeHost == null}
            className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
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

      {/* Search + Filter bar */}
      {data != null && (
        <div className="flex flex-wrap items-center gap-2 border-b border-zinc-100 px-4 py-2">
          <div className="flex gap-1">
            <button
              type="button"
              onClick={() => setProviderView("all")}
              className={`cursor-pointer rounded border px-2 py-1 text-xs font-medium ${
                providerView === "all"
                  ? "border-zinc-700 bg-zinc-900 text-white"
                  : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
              }`}
            >
              All skills <span className="ml-1 opacity-70">{skills.length}</span>
            </button>
            <button
              type="button"
              onClick={() => setProviderView("shared_agents")}
              className={`cursor-pointer rounded border px-2 py-1 text-xs font-medium ${
                providerView === "shared_agents"
                  ? "border-zinc-700 bg-zinc-900 text-white"
                  : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
              }`}
            >
              Shared Agent Skills <span className="ml-1 opacity-70">{sharedAgentSkills.length}</span>
            </button>
          </div>
          <span className="text-xs text-zinc-400">
            Provider view is based on the current Skill Host scan.
          </span>
          <div className="relative flex-1 max-w-xs">
            <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-zinc-400" />
            <input
              type="text"
              placeholder="Search skills…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded border border-zinc-200 py-1.5 pl-7 pr-3 text-xs focus:border-zinc-400 focus:outline-none"
            />
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as SkillStatus)}
            className="rounded border border-zinc-200 py-1.5 pl-2 pr-6 text-xs focus:border-zinc-400 focus:outline-none"
          >
            <option value="all">All statuses</option>
            <option value="available">Available</option>
            <option value="missing">Missing</option>
            <option value="unreadable">Unreadable</option>
            <option value="local_modified">Modified</option>
            <option value="unknown">Unknown</option>
          </select>
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
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Projects</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Path</th>
                <th className="px-3 py-2 text-xs font-medium text-zinc-500">Source</th>
              </tr>
            </thead>
            <tbody>
              {filteredSkills.map((skill) => (
                <SkillRow key={skill.id} skill={skill} />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
