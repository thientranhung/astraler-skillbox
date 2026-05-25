import React from "react";
import { RefreshCw, AlertTriangle } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import type { ProjectListItem } from "@contracts/index.js";
import { useScanProject } from "./use-scan-project.js";
import { ProjectStatusBadge } from "./project-status-badge.js";
import { ProjectProviderBadge } from "./project-provider-badge.js";

interface ProjectRowProps {
  project: ProjectListItem;
}

export function ProjectRow({ project }: ProjectRowProps): React.JSX.Element {
  const scan = useScanProject();
  const isScanning = scan.operationId != null || scan.isPending;
  const navigate = useNavigate();

  function handleRowClick(): void {
    void navigate({ to: "/projects/$projectId", params: { projectId: String(project.id) } });
  }

  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-2">
        <button
          onClick={handleRowClick}
          className="block w-full text-left"
        >
          <div className="text-sm font-medium text-zinc-900 hover:underline">{project.name}</div>
          <div className="font-mono text-xs text-zinc-400">{project.path}</div>
        </button>
      </td>
      <td className="px-3 py-2">
        <ProjectStatusBadge status={project.status} />
      </td>
      <td className="px-3 py-2">
        <div className="flex flex-wrap gap-1">
          {project.providers.length === 0 ? (
            <span className="text-xs text-zinc-400">—</span>
          ) : (
            project.providers.map((p) => <ProjectProviderBadge key={p.key} provider={p} />)
          )}
        </div>
      </td>
      <td className="px-3 py-2 text-sm text-zinc-500">{project.skillCount}</td>
      <td className="px-3 py-2">
        {project.warningCount > 0 ? (
          <span className="flex items-center gap-1 text-xs text-yellow-700">
            <AlertTriangle size={12} />
            {project.warningCount}
          </span>
        ) : (
          <span className="text-xs text-zinc-400">—</span>
        )}
      </td>
      <td className="px-3 py-2 text-xs text-zinc-400">
        {project.lastScannedAt != null
          ? new Date(project.lastScannedAt).toLocaleString()
          : "—"}
      </td>
      <td className="px-3 py-2">
        <button
          onClick={() => scan.mutate(project.id)}
          disabled={isScanning}
          title="Scan project"
          className="flex items-center gap-1 rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50 disabled:opacity-50"
        >
          <RefreshCw size={12} className={isScanning ? "animate-spin" : ""} />
          {isScanning ? "Scanning…" : "Scan"}
        </button>
      </td>
    </tr>
  );
}
