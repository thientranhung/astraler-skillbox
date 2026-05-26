import React, { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { FolderOpen, Pencil, RotateCcw } from "lucide-react";
import { useAppSettings } from "../features/app-settings/use-app-settings.js";
import { useChooseHost } from "../features/skill-host/use-choose-host.js";
import { useProviderList } from "../features/providers/use-provider-list.js";
import { useResetProviderPaths } from "../features/providers/use-reset-provider-paths.js";
import { ProviderPathsEditor } from "../features/providers/provider-paths-editor.js";
import { methods } from "../lib/core-client/methods.js";
import { ErrorDisplay } from "../components/error-display.js";
import { ProviderIcon } from "../components/provider-icon.js";
import type { ProviderListProvider } from "@contracts/index.js";

const INSTALL_MODE_LABEL: Record<string, string> = {
  symlink: "Symlink",
  rsync_copy: "Copy (rsync)",
};

const PROVIDER_STATUS_CONFIG: Record<
  ProviderListProvider["status"],
  { label: string; className: string }
> = {
  supported: { label: "Supported", className: "bg-green-100 text-green-700" },
  experimental: { label: "Experimental", className: "bg-yellow-100 text-yellow-700" },
  unsupported: { label: "Unsupported", className: "bg-zinc-100 text-zinc-500" },
  disabled: { label: "Disabled", className: "bg-zinc-100 text-zinc-400" },
};

function ProviderStatusBadge({
  status,
}: {
  status: ProviderListProvider["status"];
}): React.JSX.Element {
  const cfg = PROVIDER_STATUS_CONFIG[status] ?? PROVIDER_STATUS_CONFIG.unsupported;
  return (
    <span className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium ${cfg.className}`}>
      {cfg.label}
    </span>
  );
}

function hasOverride(provider: ProviderListProvider): boolean {
  return provider.candidates.some((c) => c.source === "override");
}

function slotHasOverride(
  provider: ProviderListProvider,
  scope: "project" | "global",
  purpose: "detect" | "skills",
): boolean {
  return provider.candidates.some(
    (c) => c.scope === scope && c.purpose === purpose && c.source === "override",
  );
}

function candidatePathsWithSource(
  provider: ProviderListProvider,
  scope: "project" | "global",
  purpose: "detect" | "skills",
): { paths: string[]; source: string } {
  const cands = provider.candidates.filter((c) => c.scope === scope && c.purpose === purpose);
  const sorted = [...cands].sort((a, b) => b.priority - a.priority);
  const source = sorted.some((c) => c.source === "override") ? "override" : "builtin";
  return { paths: sorted.map((c) => c.relativePath), source };
}

function SlotCell({
  data,
  hasSlotOverride,
  onEdit,
  onReset,
  isResetting,
}: {
  data: { paths: string[]; source: string };
  hasSlotOverride: boolean;
  onEdit: () => void;
  onReset: () => void;
  isResetting: boolean;
}): React.JSX.Element {
  const pathText =
    data.paths.length === 0 ? (
      <span className="text-zinc-400">—</span>
    ) : (
      <span className={`font-mono text-xs ${data.source === "override" ? "text-blue-600" : "text-zinc-600"}`}>
        {data.paths.join(", ")}
      </span>
    );
  return (
    <div className="flex items-center gap-1">
      {pathText}
      <button
        onClick={onEdit}
        className="shrink-0 rounded p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
        aria-label="Edit paths"
        title="Edit paths"
      >
        <Pencil size={11} />
      </button>
      {hasSlotOverride && (
        <button
          onClick={onReset}
          disabled={isResetting}
          className="shrink-0 rounded p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-red-500 disabled:opacity-50"
          aria-label="Reset to default"
          title="Reset to default"
        >
          <RotateCcw size={11} />
        </button>
      )}
    </div>
  );
}

function ProviderRow({ provider }: { provider: ProviderListProvider }): React.JSX.Element {
  const [editSlot, setEditSlot] = useState<{
    scope: "project" | "global";
    purpose: "detect" | "skills" | "config" | "commands";
    paths: string[];
  } | null>(null);
  const resetMutation = useResetProviderPaths();

  const projectDetect = candidatePathsWithSource(provider, "project", "detect");
  const projectSkills = candidatePathsWithSource(provider, "project", "skills");
  const globalSkills = provider.hasGlobalLevel ? candidatePathsWithSource(provider, "global", "skills") : null;

  function handleResetSlot(scope: "project" | "global", purpose: "detect" | "skills"): void {
    resetMutation.mutate({ providerKey: provider.key, scope, purpose });
  }

  return (
    <>
      <tr className={!provider.isAvailable ? "opacity-50" : ""}>
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <ProviderIcon providerKey={provider.key} iconKey={provider.iconKey} />
            <span className="font-medium text-zinc-800">{provider.displayName}</span>
            {hasOverride(provider) && (
              <span className="inline-flex items-center rounded bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-600">
                Override
              </span>
            )}
          </div>
        </td>
        <td className="px-3 py-2 font-mono text-zinc-500">{provider.key}</td>
        <td className="px-3 py-2">
          <ProviderStatusBadge status={provider.status} />
        </td>
        <td className="px-3 py-2">
          <SlotCell
            data={projectDetect}
            hasSlotOverride={slotHasOverride(provider, "project", "detect")}
            onEdit={() => setEditSlot({ scope: "project", purpose: "detect", paths: projectDetect.paths })}
            onReset={() => handleResetSlot("project", "detect")}
            isResetting={resetMutation.isPending}
          />
        </td>
        <td className="px-3 py-2">
          <SlotCell
            data={projectSkills}
            hasSlotOverride={slotHasOverride(provider, "project", "skills")}
            onEdit={() => setEditSlot({ scope: "project", purpose: "skills", paths: projectSkills.paths })}
            onReset={() => handleResetSlot("project", "skills")}
            isResetting={resetMutation.isPending}
          />
        </td>
        <td className="px-3 py-2">
          {globalSkills != null ? (
            <SlotCell
              data={globalSkills}
              hasSlotOverride={slotHasOverride(provider, "global", "skills")}
              onEdit={() => setEditSlot({ scope: "global", purpose: "skills", paths: globalSkills.paths })}
              onReset={() => handleResetSlot("global", "skills")}
              isResetting={resetMutation.isPending}
            />
          ) : (
            <span className="text-zinc-300">—</span>
          )}
        </td>
      </tr>
      {editSlot != null && (
        <ProviderPathsEditor
          providerKey={provider.key}
          scope={editSlot.scope}
          purpose={editSlot.purpose}
          currentPaths={editSlot.paths}
          onClose={() => setEditSlot(null)}
        />
      )}
    </>
  );
}

export function SettingsScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const { data: settings, isPending, isError, error } = useAppSettings();
  const { data: providerData } = useProviderList();
  const chooseMutation = useChooseHost();

  async function handleChangeFolder(): Promise<void> {
    try {
      const result = await methods.openHostFolder();
      if (result.path != null) {
        chooseMutation.mutate(result.path, {
          onSuccess: () => {
            void navigate({ to: "/skills" });
          },
        });
      }
    } catch {
      // openHostFolder errors are not critical; dialog just closed
    }
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
      </div>
    );
  }

  return (
    <div className="p-6 space-y-8">
      <div>
        <h2 className="text-base font-semibold text-zinc-900">Settings</h2>

        <div className="mt-4 max-w-lg divide-y divide-zinc-100 rounded border border-zinc-200">
          <div className="flex items-center justify-between px-4 py-3">
            <div>
              <div className="text-sm font-medium text-zinc-700">Skill Host Folder</div>
              <div className="mt-0.5 font-mono text-xs text-zinc-500">
                {settings?.activeHost?.path ?? "Not configured"}
              </div>
            </div>
            <button
              onClick={handleChangeFolder}
              disabled={chooseMutation.isPending}
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50 disabled:opacity-50"
            >
              <FolderOpen size={13} />
              Change
            </button>
          </div>

          <div className="flex items-center justify-between px-4 py-3">
            <div className="text-sm font-medium text-zinc-700">Default Install Mode</div>
            <div className="text-sm text-zinc-500">
              {INSTALL_MODE_LABEL[settings?.defaultInstallMode ?? ""] ?? settings?.defaultInstallMode ?? "—"}
            </div>
          </div>

          <div className="flex items-center justify-between px-4 py-3">
            <div className="text-sm font-medium text-zinc-700">Database Version</div>
            <div className="text-sm text-zinc-500">{settings?.databaseVersion ?? "—"}</div>
          </div>
        </div>

        {chooseMutation.error != null && (
          <div className="mt-4 max-w-lg">
            <ErrorDisplay error={chooseMutation.error} />
          </div>
        )}
      </div>

      <div>
        <h3 className="text-sm font-semibold text-zinc-800">Providers</h3>
        <p className="mt-0.5 text-xs text-zinc-500">
          Override paths are configuration metadata only — scan and install behavior integration is a later slice.
        </p>

        <div className="mt-3 overflow-x-auto rounded border border-zinc-200">
          <table className="min-w-full text-xs">
            <thead>
              <tr className="border-b border-zinc-100 bg-zinc-50 text-left text-zinc-500">
                <th className="px-3 py-2 font-medium">Provider</th>
                <th className="px-3 py-2 font-medium">Key</th>
                <th className="px-3 py-2 font-medium">Status</th>
                <th className="px-3 py-2 font-medium">Project detect</th>
                <th className="px-3 py-2 font-medium">Project skills</th>
                <th className="px-3 py-2 font-medium">Global skills</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {(providerData?.providers ?? []).map((provider) => (
                <ProviderRow key={provider.key} provider={provider} />
              ))}
              {(providerData?.providers ?? []).length === 0 && (
                <tr>
                  <td colSpan={6} className="px-3 py-4 text-center text-zinc-400">
                    Loading providers…
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
