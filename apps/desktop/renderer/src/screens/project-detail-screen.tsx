import React, { useState } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, RefreshCw, FolderOpen, TerminalSquare, Trash2, AlertCircle, PlusCircle, Copy, Check } from "lucide-react";
import { useProjectDetail } from "../features/projects/use-project-detail.js";
import { useScanProject } from "../features/projects/use-scan-project.js";
import { useOpenProjectFolder } from "../features/projects/use-open-project-folder.js";
import { useOpenProjectTerminal } from "../features/projects/use-open-project-terminal.js";
import { useRemoveProject } from "../features/projects/use-remove-project.js";
import { useRemoveSkill } from "../features/projects/use-remove-skill.js";
import { RemoveSkillDialog } from "../features/projects/remove-skill-dialog.js";
import { useProviderPluginList } from "../features/provider-plugins/use-provider-plugin-list.js";
import { useSetProviderPluginEnabled } from "../features/provider-plugins/use-set-provider-plugin-enabled.js";
import { useRemoveProviderPluginOverride } from "../features/provider-plugins/use-remove-provider-plugin-override.js";
import { ProjectStatusBadge } from "../features/projects/project-status-badge.js";
import { AddSkillWizard } from "../features/projects/add-skill-wizard.js";
import { useActiveHostSkills } from "../features/skills/use-active-host-skills.js";
import { ErrorDisplay } from "../components/error-display.js";
import { ProviderIcon } from "../components/provider-icon.js";
import type { ProjectGetEntry, ProjectGetProvider, PPLayerStatus, PPProjectEntry } from "@contracts/index.js";

const JSON_WRITE_PROVIDERS = new Set(["claude", "antigravity_cli", "codex"]);

const ENTRY_STATUS_CONFIG: Record<ProjectGetEntry["status"], { label: string; description: string; cls: string }> = {
  current: { label: "Linked to active host", description: "This project entry points to the active Skill Host copy.", cls: "bg-green-100 text-green-800" },
  old_host: { label: "Linked to old host", description: "The entry points to a previous Skill Host folder.", cls: "bg-yellow-100 text-yellow-700" },
  external_symlink: { label: "Points outside host", description: "The symlink target is outside the active Skill Host folder.", cls: "bg-yellow-100 text-yellow-700" },
  broken_symlink: { label: "Broken link", description: "The symlink target no longer exists.", cls: "bg-red-100 text-red-800" },
  missing: { label: "Missing from disk", description: "Skillbox has a record for this entry, but it was not found during the last scan.", cls: "bg-red-100 text-red-800" },
  error: { label: "Needs attention", description: "Skillbox could not classify this entry cleanly.", cls: "bg-red-100 text-red-800" },
};

const DETECTION_CLS: Record<ProjectGetProvider["detectionStatus"], string> = {
  detected: "bg-blue-100 text-blue-800",
  configured: "bg-blue-100 text-blue-800",
  missing: "bg-zinc-100 text-zinc-500",
  invalid_structure: "bg-yellow-100 text-yellow-700",
};

const PROVIDER_STATUS_CONFIG: Record<ProjectGetProvider["providerStatus"], { label: string; description: string; cls: string }> = {
  supported: { label: "Ready", description: "Skillbox can manage this provider with the current feature set.", cls: "bg-green-100 text-green-800" },
  experimental: { label: "Preview", description: "This provider is available, but behavior may still change.", cls: "bg-yellow-100 text-yellow-700" },
  unsupported: { label: "Not supported", description: "Skillbox can display this provider but cannot manage it yet.", cls: "bg-zinc-100 text-zinc-500" },
  disabled: { label: "Disabled", description: "This provider is currently turned off.", cls: "bg-zinc-100 text-zinc-400" },
};

function EntryStatusBadge({ status }: { status: ProjectGetEntry["status"] }): React.JSX.Element {
  const cfg = ENTRY_STATUS_CONFIG[status] ?? ENTRY_STATUS_CONFIG.error;
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${cfg.cls}`}
      title={`${cfg.description} Raw status: ${status}`}
    >
      {cfg.label}
    </span>
  );
}

function ProviderRow({ provider }: { provider: ProjectGetProvider }): React.JSX.Element {
  const providerStatus = PROVIDER_STATUS_CONFIG[provider.providerStatus] ?? PROVIDER_STATUS_CONFIG.unsupported;
  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">
        <span className="inline-flex items-center gap-1.5">
          <ProviderIcon providerKey={provider.providerKey} />
          {provider.displayName}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <span className={`inline-flex items-center rounded px-1.5 py-0.5 font-medium ${DETECTION_CLS[provider.detectionStatus] ?? DETECTION_CLS.missing}`}>
          {provider.detectionStatus.replace("_", " ")}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <div className="flex flex-col gap-0.5">
          <span
            className={`inline-flex w-fit items-center rounded px-1.5 py-0.5 font-medium ${providerStatus.cls}`}
            title={`Raw status: ${provider.providerStatus}`}
          >
            {providerStatus.label}
          </span>
          <span className="text-[11px] leading-tight text-zinc-400">{providerStatus.description}</span>
        </div>
      </td>
      <td className="max-w-md break-all px-3 py-1.5 font-mono text-xs leading-snug text-zinc-400">
        {provider.detectedPath ?? "—"}
      </td>
      <td className="px-3 py-1.5 text-xs text-zinc-500">{provider.entryCount}</td>
    </tr>
  );
}

function PathCell({
  value,
  displayValue,
  label,
}: {
  value: string | null;
  displayValue?: string | null;
  label: string;
}): React.JSX.Element {
  const [copied, setCopied] = useState(false);

  if (value == null || value === "") {
    return <span className="text-zinc-400">—</span>;
  }

  async function copyPath(): Promise<void> {
    if (value == null) return;
    if (navigator.clipboard == null) return;
    await navigator.clipboard.writeText(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  return (
    <div className="min-w-0">
      <div className="flex min-w-0 items-start gap-1">
        <span className="break-all font-mono text-xs leading-snug text-zinc-500">
          {displayValue ?? value}
        </span>
        <button
          type="button"
          onClick={() => void copyPath()}
          aria-label={`Copy ${label}`}
          title={`Copy ${label}`}
          className="shrink-0 rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
        >
          {copied ? <Check size={12} /> : <Copy size={12} />}
        </button>
      </div>
    </div>
  );
}

function isRemovable(entry: ProjectGetEntry): boolean {
  return entry.mode === "symlink" && entry.status === "current";
}

function projectRelativePath(path: string, projectRoot: string): string {
  const normalizedRoot = projectRoot.endsWith("/") ? projectRoot.slice(0, -1) : projectRoot;
  const prefix = `${normalizedRoot}/`;
  if (path.startsWith(prefix)) {
    return path.slice(prefix.length);
  }
  return path;
}

function EntryRow({
  entry,
  projectRoot,
  providerDisplayName,
  onRemove,
}: {
  entry: ProjectGetEntry;
  projectRoot: string;
  providerDisplayName: string;
  onRemove: (entry: ProjectGetEntry) => void;
}): React.JSX.Element {
  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-1.5 text-xs text-zinc-500">
        <span className="inline-flex items-center gap-1.5">
          <ProviderIcon providerKey={entry.providerKey} />
          {providerDisplayName}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{entry.name}</td>
      <td className="px-3 py-1.5 text-xs">
        <span className="inline-flex items-center rounded bg-zinc-100 px-1.5 py-0.5 font-medium text-zinc-600">
          {entry.mode}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <EntryStatusBadge status={entry.status} />
      </td>
      <td className="max-w-sm px-3 py-1.5 text-xs">
        <PathCell
          value={entry.projectSkillPath}
          displayValue={projectRelativePath(entry.projectSkillPath, projectRoot)}
          label="project skill path"
        />
      </td>
      <td className="max-w-sm px-3 py-1.5 text-xs">
        <PathCell value={entry.symlinkTargetPath} label="symlink target path" />
      </td>
      <td className="px-3 py-1.5 text-xs text-zinc-400">{entry.skillId ?? "—"}</td>
      <td className="px-3 py-1.5 text-xs">
        <button
          onClick={() => onRemove(entry)}
          disabled={!isRemovable(entry)}
          title={isRemovable(entry) ? "Remove skill from project" : "Only current symlink installs can be removed in this slice"}
          className="rounded border border-zinc-300 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Remove
        </button>
      </td>
    </tr>
  );
}

const LAYER_SCAN_LABEL: Record<PPLayerStatus["scanStatus"], string> = {
  ok: "ok",
  missing: "not configured",
  unreadable: "unreadable",
  malformed: "malformed",
  too_large: "too large",
  symlink: "symlink (skipped)",
  path_escape: "path escape (skipped)",
};

function layerScanClass(status: PPLayerStatus["scanStatus"]): string {
  switch (status) {
    case "ok": return "bg-green-100 text-green-700";
    case "missing": return "bg-zinc-100 text-zinc-400";
    default: return "bg-yellow-100 text-yellow-700";
  }
}

function effectiveStatusClass(status: PPProjectEntry["effectiveStatus"]): string {
  switch (status) {
    case "enabled": return "bg-green-100 text-green-700";
    case "disabled": return "bg-zinc-100 text-zinc-500";
    case "absent": return "bg-zinc-100 text-zinc-400";
    default: return "bg-yellow-100 text-yellow-700";
  }
}

type ProjectLayerState = "enabled" | "disabled" | "not-set";

function getLayerDeclaration(
  layerBreakdown: Array<{ layer: string; scanStatus: string; declaration: string | null }>,
  layer: string,
): string | null {
  const entry = layerBreakdown.find((lb) => lb.layer === layer);
  return entry?.declaration ?? null;
}

function projectLayerState(
  layerBreakdown: Array<{ layer: string; scanStatus: string; declaration: string | null }>,
): ProjectLayerState {
  const decl = getLayerDeclaration(layerBreakdown, "project");
  if (decl === "enabled") return "enabled";
  if (decl === "disabled") return "disabled";
  return "not-set";
}

function projectStateBadgeClass(state: ProjectLayerState): string {
  switch (state) {
    case "enabled": return "bg-green-100 text-green-700";
    case "disabled": return "bg-zinc-100 text-zinc-500";
    case "not-set": return "";
  }
}

function projectStateLabel(state: ProjectLayerState): string {
  switch (state) {
    case "enabled": return "enabled";
    case "disabled": return "disabled";
    case "not-set": return "—";
  }
}

function ProjectPluginSection({ projectId, scanInFlight }: { projectId: number; scanInFlight: boolean }): React.JSX.Element {
  const { data, isPending, isError, error } = useProviderPluginList();
  const setEnabledMutation = useSetProviderPluginEnabled();
  const removeOverrideMutation = useRemoveProviderPluginOverride();
  const isTogglingPlugin = setEnabledMutation.isPending || setEnabledMutation.operationId != null;
  const isRemovingOverride = removeOverrideMutation.isPending || removeOverrideMutation.operationId != null;
  const isOperationInFlight = isTogglingPlugin || isRemovingOverride || scanInFlight;

  function handleToggleProjectPlugin(providerKey: string, pluginName: string, marketplaceName: string, enabled: boolean): void {
    setEnabledMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "project", projectId, enabled });
  }

  function handleToggleUserPlugin(providerKey: string, pluginName: string, marketplaceName: string, enabled: boolean): void {
    setEnabledMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "user", projectId: 0, enabled });
  }

  function handleRemoveProjectOverride(providerKey: string, pluginName: string, marketplaceName: string): void {
    removeOverrideMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "project", projectId });
  }

  const projectViews = data?.projects.filter((p) => p.projectId === projectId) ?? [];

  return (
    <div>
      <div className="mb-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
          Provider Plugins
        </h3>
      </div>

      {isPending && (
        <p className="text-xs text-zinc-400">Loading plugin data…</p>
      )}

      {isError && (
        <ErrorDisplay error={error} />
      )}

      {!isPending && !isError && projectViews.length === 0 && (
        <p className="text-xs text-zinc-400">No plugin data. Run a scan to populate.</p>
      )}

      {!isPending && !isError && projectViews.length > 0 && (
        <div className="flex flex-col gap-5">
          {projectViews.map((projectView) => (
            <div key={projectView.providerKey} className="flex flex-col gap-3">
              <div className="flex items-center gap-1.5 text-xs font-medium text-zinc-700">
                <ProviderIcon providerKey={projectView.providerKey} />
                <span>{projectView.providerKey === "codex" ? "Codex" : projectView.providerKey === "claude" ? "Claude" : projectView.providerKey}</span>
              </div>
          {/* Layer statuses */}
          <div className="overflow-x-auto rounded border border-zinc-200">
            <table className="w-full text-left">
              <thead className="border-b border-zinc-200 bg-zinc-50">
                <tr>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Layer</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Status</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">File</th>
                  <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Last Scanned</th>
                </tr>
              </thead>
              <tbody>
                {projectView.layerStatuses.map((ls) => (
                  <tr key={ls.layer} className="border-b border-zinc-100">
                    <td className="px-3 py-1.5 text-xs font-medium text-zinc-700">{ls.layer}</td>
                    <td className="px-3 py-1.5 text-xs">
                      <span className={`rounded px-1.5 py-0.5 font-medium ${layerScanClass(ls.scanStatus)}`}>
                        {LAYER_SCAN_LABEL[ls.scanStatus] ?? ls.scanStatus}
                      </span>
                    </td>
                    <td className="max-w-xs break-all px-3 py-1.5 font-mono text-xs text-zinc-400">
                      {ls.filePath}
                      {ls.scanWarnings.length > 0 && ls.scanStatus !== "missing" && (
                        <ul className="mt-0.5 list-none">
                          {ls.scanWarnings.map((w, i) => (
                            <li key={i} className="text-zinc-500">{w}</li>
                          ))}
                        </ul>
                      )}
                    </td>
                    <td className="px-3 py-1.5 text-xs text-zinc-400">
                      {ls.lastScannedAt != null ? new Date(ls.lastScannedAt).toLocaleString() : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Plugin layer table */}
          {projectView.plugins.length > 0 && (() => {
            const canToggle = JSON_WRITE_PROVIDERS.has(projectView.providerKey);
            return (
            <div className="overflow-x-auto rounded border border-zinc-200">
              <table className="w-full text-left">
                <thead className="border-b border-zinc-200 bg-zinc-50">
                  <tr>
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Plugin</th>
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Marketplace</th>
                    {canToggle && (
                      <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Project</th>
                    )}
                    {canToggle && (
                      <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">User</th>
                    )}
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Effective</th>
                  </tr>
                </thead>
                <tbody>
                  {projectView.plugins.map((p, i) => {
                    const isLocalOverride = p.provenanceLayer === "local";
                    const projState = projectLayerState(p.layerBreakdown);
                    const userDecl = getLayerDeclaration(p.layerBreakdown, "user");
                    const isUserEnabled = userDecl === "enabled";
                    const projectHasValue = projState !== "not-set";
                    return (
                      <tr key={i} className="border-b border-zinc-100 hover:bg-zinc-50">
                        <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{p.pluginName}</td>
                        <td className="px-3 py-1.5 text-xs text-zinc-500">{p.marketplaceName || "—"}</td>

                        {/* Project column — 3-state cycle */}
                        {canToggle && (
                          <td className="px-3 py-1.5 text-xs">
                            {isLocalOverride ? (
                              <span className="text-xs text-zinc-400 opacity-40">overridden</span>
                            ) : (
                              <button
                                onClick={() => {
                                  if (projState === "not-set") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, true);
                                  } else if (projState === "enabled") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, false);
                                  } else {
                                    handleRemoveProjectOverride(projectView.providerKey, p.pluginName, p.marketplaceName);
                                  }
                                }}
                                disabled={isOperationInFlight}
                                title={
                                  projState === "not-set"
                                    ? "Click to enable at project level"
                                    : projState === "enabled"
                                      ? "Click to disable at project level"
                                      : "Click to clear project override"
                                }
                                className={`rounded px-1.5 py-0.5 font-medium disabled:cursor-not-allowed disabled:opacity-40 ${
                                  projState === "not-set"
                                    ? "text-zinc-400 hover:bg-zinc-100"
                                    : projectStateBadgeClass(projState) + " hover:opacity-80"
                                }`}
                              >
                                {projectStateLabel(projState)}
                              </button>
                            )}
                          </td>
                        )}

                        {/* User column — 2-state toggle */}
                        {canToggle && (
                          <td className="px-3 py-1.5 text-xs">
                            {isLocalOverride ? (
                              <span className="text-xs text-zinc-400 opacity-40">overridden</span>
                            ) : (
                              <div className={projectHasValue ? "opacity-40" : ""}>
                                <button
                                  onClick={() => handleToggleUserPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, !isUserEnabled)}
                                  disabled={isOperationInFlight}
                                  title={
                                    projectHasValue
                                      ? "Project layer overrides this setting"
                                      : isUserEnabled
                                        ? "Disable globally"
                                        : "Enable globally"
                                  }
                                  className={`rounded px-1.5 py-0.5 font-medium hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40 ${
                                    isUserEnabled
                                      ? "bg-green-100 text-green-700"
                                      : "bg-zinc-100 text-zinc-500"
                                  }`}
                                >
                                  {isUserEnabled ? "enabled" : "disabled"}
                                </button>
                              </div>
                            )}
                          </td>
                        )}

                        {/* Effective column — read-only */}
                        <td className="px-3 py-1.5 text-xs">
                          <span className={`rounded px-1.5 py-0.5 font-medium ${effectiveStatusClass(p.effectiveStatus)}`}>
                            {p.effectiveStatus}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            );
          })()}

          {projectView.plugins.length === 0 && (
            <p className="text-xs text-zinc-400">No plugins found across layers.</p>
          )}

          {projectView.managedOutOfScope && (
            <p className="text-xs text-zinc-400">
              Some settings in this project are managed outside Skillbox.
            </p>
          )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function ProjectDetailScreen(): React.JSX.Element {
  const { projectId: projectIdStr = "" } = useParams({ strict: false });
  const validId: number | null =
    /^\d+$/.test(projectIdStr) && Number(projectIdStr) > 0 ? Number(projectIdStr) : null;
  const navigate = useNavigate();
  const { data, isPending, isError, error } = useProjectDetail(validId);
  const scan = useScanProject();
  const isScanning = scan.operationId != null || scan.isPending;
  const openFolder = useOpenProjectFolder();
  const openTerminal = useOpenProjectTerminal();
  const remove = useRemoveProject({ navigateAfter: true });
  const removeSkill = useRemoveSkill();
  const [removeTarget, setRemoveTarget] = useState<ProjectGetEntry | null>(null);
  const activeHostSkills = useActiveHostSkills();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [selectedProviderId, setSelectedProviderId] = useState<"all" | number>("all");

  const providerDisplayNameFor = (entry: ProjectGetEntry): string => {
    const match = data?.providers.find((p) => p.projectProviderId === entry.projectProviderId);
    return match?.displayName ?? entry.providerKey;
  };

  const filteredEntries =
    data == null || selectedProviderId === "all"
      ? data?.entries ?? []
      : data.entries.filter((entry) => entry.projectProviderId === selectedProviderId);

  React.useEffect(() => {
    if (selectedProviderId === "all" || data == null) return;
    if (!data.providers.some((provider) => provider.projectProviderId === selectedProviderId)) {
      setSelectedProviderId("all");
    }
  }, [data, selectedProviderId, validId]);

  function confirmRemoveSkill(): void {
    if (removeTarget == null || validId == null) return;
    removeSkill.mutate({ projectId: validId, installId: removeTarget.id });
    setRemoveTarget(null);
  }

  function handleRemove(): void {
    if (window.confirm("Remove this project from Skillbox? Files on disk will not be deleted.")) {
      remove.mutate(validId!);
    }
  }

  return (
    <div className="flex flex-1 flex-col">
      {/* Header */}
      <div className="flex flex-col gap-3 border-b border-zinc-200 px-4 py-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 flex-wrap items-start gap-3">
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
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <span className="truncate text-sm font-semibold text-zinc-900">{data.project.name}</span>
                <span className="break-all font-mono text-xs leading-snug text-zinc-400">
                  {data.project.path}
                </span>
              </div>
              <ProjectStatusBadge status={data.project.status} />
            </>
          )}
        </div>
        {data != null && (
          <div className="flex shrink-0 items-center gap-2">
            {data.project.lastScannedAt != null && (
              <span className="hidden text-xs text-zinc-400 sm:block">
                Scanned {new Date(data.project.lastScannedAt).toLocaleString()}
              </span>
            )}
            <button
              onClick={() => scan.mutate(validId!)}
              disabled={isScanning}
              title="Scan project"
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
            >
              <RefreshCw size={12} className={isScanning ? "animate-spin" : ""} />
              {isScanning ? "Scanning…" : "Scan"}
            </button>
            <button
              onClick={() => openFolder.mutate(data.project.path)}
              disabled={openFolder.isPending}
              title="Open folder"
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
            >
              <FolderOpen size={12} />
              Open Folder
            </button>
            <button
              onClick={() => openTerminal.mutate(data.project.path)}
              disabled={openTerminal.isPending}
              title="Open terminal"
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
            >
              <TerminalSquare size={12} />
              Terminal
            </button>
            <button
              onClick={() => setWizardOpen(true)}
              title="Add skill to project"
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50"
            >
              <PlusCircle size={12} />
              Add Skill
            </button>
            <button
              onClick={handleRemove}
              disabled={remove.isPending}
              title="Remove project"
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-500 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
            >
              <Trash2 size={12} />
              Remove
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

            {/* Provider Plugins */}
            <ProjectPluginSection projectId={validId} scanInFlight={isScanning} />

            {/* Skill Entries */}
            <div>
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-500">
                Skill Entries
                {data.entries.length > 0 && (
                  <span className="ml-1 font-normal normal-case text-zinc-400">
                    ({filteredEntries.length} of {data.entries.length})
                  </span>
                )}
              </h3>
              {data.providers.length > 0 && (
                <div className="mb-2 flex flex-wrap gap-1">
                  <button
                    type="button"
                    onClick={() => setSelectedProviderId("all")}
                    className={`rounded border px-2 py-1 text-xs font-medium ${
                      selectedProviderId === "all"
                        ? "border-zinc-700 bg-zinc-900 text-white"
                        : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
                    }`}
                  >
                    All providers
                    <span className="ml-1 opacity-70">{data.entries.length}</span>
                  </button>
                  {data.providers.map((provider) => (
                    <button
                      key={provider.projectProviderId}
                      type="button"
                      onClick={() => setSelectedProviderId(provider.projectProviderId)}
                      className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium ${
                        selectedProviderId === provider.projectProviderId
                          ? "border-zinc-700 bg-zinc-900 text-white"
                          : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
                      }`}
                    >
                      <ProviderIcon providerKey={provider.providerKey} />
                      {provider.displayName}
                      <span className="opacity-70">{provider.entryCount}</span>
                    </button>
                  ))}
                </div>
              )}
              {data.entries.length === 0 ? (
                <p className="text-xs text-zinc-400">
                  No skill entries observed. Run a scan to populate entries.
                </p>
              ) : filteredEntries.length === 0 ? (
                <p className="text-xs text-zinc-400">
                  No skill entries for this provider.
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
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredEntries.map((entry) => (
                        <EntryRow
                          key={entry.id}
                          entry={entry}
                          projectRoot={data.project.path}
                          providerDisplayName={providerDisplayNameFor(entry)}
                          onRemove={setRemoveTarget}
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {removeTarget != null && (
        <RemoveSkillDialog
          skillName={removeTarget.name}
          providerDisplayName={providerDisplayNameFor(removeTarget)}
          path={removeTarget.projectSkillPath}
          isPending={removeSkill.isPending}
          onConfirm={confirmRemoveSkill}
          onCancel={() => setRemoveTarget(null)}
        />
      )}

      {wizardOpen && validId != null && data != null && (
        <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/30">
          <div className="w-full max-w-lg rounded-lg border border-zinc-200 bg-white shadow-xl">
            <AddSkillWizard
              projectId={validId}
              providers={data.providers}
              skills={activeHostSkills.skills}
              onClose={() => setWizardOpen(false)}
            />
          </div>
        </div>
      )}
    </div>
  );
}
