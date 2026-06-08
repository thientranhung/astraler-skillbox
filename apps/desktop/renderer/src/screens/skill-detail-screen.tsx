import React from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, AlertCircle, FolderOpen } from "lucide-react";
import { useSkillDetail } from "../features/skills-library/use-skill-detail.js";
import { SkillStatusBadge } from "../features/skills-library/skill-status-badge.js";
import { ErrorDisplay } from "../components/error-display.js";
import { ProviderIcon } from "../components/provider-icon.js";
import { methods } from "../lib/core-client/methods.js";
import { providerDisplayName } from "../lib/provider-display.js";
import type { SkillGetProjectInstall } from "@contracts/index.js";

const MODE_CLS: Record<SkillGetProjectInstall["mode"], string> = {
  symlink: "bg-blue-100 text-blue-800",
  rsync_copy: "bg-purple-100 text-purple-800",
  direct: "bg-zinc-100 text-zinc-600",
};

const STATUS_CLS: Record<SkillGetProjectInstall["status"], string> = {
  current: "bg-green-100 text-green-800",
  outdated: "bg-yellow-100 text-yellow-700",
  missing: "bg-red-100 text-red-800",
  broken_symlink: "bg-red-100 text-red-800",
  old_host: "bg-yellow-100 text-yellow-700",
  external_symlink: "bg-yellow-100 text-yellow-700",
  conflict: "bg-red-100 text-red-800",
  needs_sync: "bg-orange-100 text-orange-700",
  error: "bg-red-100 text-red-800",
};

export function SkillDetailScreen(): React.JSX.Element {
  const { skillId: skillIdStr = "" } = useParams({ strict: false });
  const navigate = useNavigate();

  const validId: number | null =
    /^\d+$/.test(skillIdStr) && Number(skillIdStr) > 0 ? Number(skillIdStr) : null;

  const { data, isPending, isError, error } = useSkillDetail(validId);

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <button
            onClick={() => void navigate({ to: "/skills" })}
            className="flex shrink-0 items-center gap-1 text-xs text-zinc-500 hover:text-zinc-800"
          >
            <ArrowLeft size={13} />
            Skills Library
          </button>
          {data != null && (
            <>
              <span className="shrink-0 text-zinc-300">/</span>
              <span className="truncate text-sm font-semibold text-zinc-900">{data.skill.name}</span>
            </>
          )}
        </div>
        {data != null && (
          <button
            onClick={() => void methods.openPath(data.skill.hostPath)}
            className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50"
          >
            <FolderOpen size={12} />
            Open Host Folder
          </button>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {validId == null && (
          <div className="p-4">
            <div className="flex items-start gap-2 rounded border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-800">
              <AlertCircle size={13} className="mt-0.5 shrink-0" />
              Invalid skill ID: <span className="ml-1 font-mono">{skillIdStr}</span>
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
            {/* Metadata */}
            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                Skill Info
              </h3>
              <div className="flex flex-col gap-1.5 rounded border border-zinc-200 bg-zinc-50 px-4 py-3 text-xs">
                <div className="flex items-center gap-2">
                  <span className="w-28 shrink-0 text-zinc-400">Status</span>
                  <SkillStatusBadge status={data.skill.status} />
                </div>
                <div className="flex items-start gap-2">
                  <span className="w-28 shrink-0 text-zinc-400">Relative path</span>
                  <span className="break-all font-mono leading-snug text-zinc-600">{data.skill.relativePath}</span>
                </div>
                <div className="flex items-start gap-2">
                  <span className="w-28 shrink-0 text-zinc-400">Absolute path</span>
                  <span className="break-all font-mono leading-snug text-zinc-600">{data.skill.absolutePath}</span>
                </div>
                <div className="flex items-start gap-2">
                  <span className="w-28 shrink-0 text-zinc-400">Host folder</span>
                  <span className="break-all font-mono leading-snug text-zinc-600">{data.skill.hostPath}</span>
                </div>
                {data.skill.sourceLabel != null && (
                  <div className="flex items-center gap-2">
                    <span className="w-28 shrink-0 text-zinc-400">Source</span>
                    <span className="text-zinc-600">{data.skill.sourceLabel}</span>
                  </div>
                )}
                {data.skill.lastScannedAt != null && (
                  <div className="flex items-center gap-2">
                    <span className="w-28 shrink-0 text-zinc-400">Last scanned</span>
                    <span className="text-zinc-600">{new Date(data.skill.lastScannedAt).toLocaleString()}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Projects Using This Skill */}
            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                Projects Using This Skill
                {data.projects.length > 0 && (
                  <span className="ml-1 font-normal normal-case text-zinc-400">
                    ({data.projects.length})
                  </span>
                )}
              </h3>
              {data.projects.length === 0 ? (
                <p className="text-xs text-zinc-400">No projects use this skill.</p>
              ) : (
                <div className="overflow-x-auto rounded border border-zinc-200">
                  <table className="w-full text-left">
                    <thead className="border-b border-zinc-200 bg-zinc-50">
                      <tr>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Project</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Provider</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Mode</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Path</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.projects.map((p, i) => (
                        <tr key={i} className="border-b border-zinc-100 hover:bg-zinc-50">
                          <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{p.projectName}</td>
                          <td className="px-3 py-1.5 text-xs text-zinc-600">
                            <span className="inline-flex items-center gap-1.5">
                              <ProviderIcon providerKey={p.providerKey} />
                              {providerDisplayName(p.providerKey, p.providerDisplayName)}
                            </span>
                          </td>
                          <td className="px-3 py-1.5 text-xs">
                            <span className={`inline-flex items-center rounded px-1.5 py-0.5 font-medium ${MODE_CLS[p.mode] ?? MODE_CLS.direct}`}>
                              {p.mode}
                            </span>
                          </td>
                          <td className="px-3 py-1.5 text-xs">
                            <span className={`inline-flex items-center rounded px-1.5 py-0.5 font-medium ${STATUS_CLS[p.status] ?? STATUS_CLS.error}`}>
                              {p.status}
                            </span>
                          </td>
                          <td className="max-w-md break-all px-3 py-1.5 font-mono text-xs leading-snug text-zinc-400">
                            {p.projectSkillPath}
                          </td>
                        </tr>
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
