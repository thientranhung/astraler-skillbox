import React, { useRef, useState } from "react";
import { Link } from "@tanstack/react-router";
import { Pencil, RotateCcw } from "lucide-react";
import { useAppSettings } from "../features/app-settings/use-app-settings.js";
import { useResetAll } from "../features/app-settings/use-reset-all.js";
import { useProviderList } from "../features/providers/use-provider-list.js";
import { useResetProviderPaths } from "../features/providers/use-reset-provider-paths.js";
import { ProviderPathsEditor } from "../features/providers/provider-paths-editor.js";
import { ErrorDisplay } from "../components/error-display.js";
import { ProviderIcon } from "../components/provider-icon.js";
import { displayPath } from "../lib/display-path.js";
import type { ProviderListProvider } from "@contracts/index.js";

type ProviderPathScope = "project" | "global";
type ProviderPathPurpose = "detect" | "skills" | "config" | "commands";

const INSTALL_MODE_LABEL: Record<string, string> = {
  symlink: "Symlink",
  rsync_copy: "Copy (rsync)",
};

function hasOverride(provider: ProviderListProvider): boolean {
  return provider.candidates.some((c) => c.source === "override");
}

function slotHasOverride(
  provider: ProviderListProvider,
  scope: ProviderPathScope,
  purpose: ProviderPathPurpose,
): boolean {
  return provider.candidates.some(
    (c) => c.scope === scope && c.purpose === purpose && c.source === "override",
  );
}

function candidatePathsWithSource(
  provider: ProviderListProvider,
  scope: ProviderPathScope,
  purpose: ProviderPathPurpose,
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

function OptionalSlotCell({
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
  if (data.paths.length === 0 && !hasSlotOverride) {
    return (
      <div className="flex items-center gap-1">
        <span className="text-zinc-400">Not set</span>
        <button
          onClick={onEdit}
          className="shrink-0 rounded p-0.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
          aria-label="Edit paths"
          title="Edit paths"
        >
          <Pencil size={11} />
        </button>
      </div>
    );
  }
  return (
    <SlotCell
      data={data}
      hasSlotOverride={hasSlotOverride}
      onEdit={onEdit}
      onReset={onReset}
      isResetting={isResetting}
    />
  );
}

function ProviderRow({ provider }: { provider: ProviderListProvider }): React.JSX.Element {
  const [editSlot, setEditSlot] = useState<{
    scope: ProviderPathScope;
    purpose: ProviderPathPurpose;
    paths: string[];
  } | null>(null);
  const resetMutation = useResetProviderPaths();

  const projectDetect = candidatePathsWithSource(provider, "project", "detect");
  const projectSkills = candidatePathsWithSource(provider, "project", "skills");
  const projectConfig = candidatePathsWithSource(provider, "project", "config");
  const globalConfig = provider.hasGlobalLevel ? candidatePathsWithSource(provider, "global", "config") : null;
  const globalSkills = provider.hasGlobalLevel ? candidatePathsWithSource(provider, "global", "skills") : null;

  function handleResetSlot(scope: ProviderPathScope, purpose: ProviderPathPurpose): void {
    resetMutation.mutate({ providerKey: provider.key, scope, purpose });
  }

  return (
    <>
      <tr>
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
          <SlotCell
            data={projectDetect}
            hasSlotOverride={slotHasOverride(provider, "project", "detect")}
            onEdit={() => setEditSlot({ scope: "project", purpose: "detect", paths: projectDetect.paths })}
            onReset={() => handleResetSlot("project", "detect")}
            isResetting={resetMutation.isPending}
          />
        </td>
        <td className="px-3 py-2">
          <OptionalSlotCell
            data={globalConfig ?? { paths: [], source: "builtin" }}
            hasSlotOverride={slotHasOverride(provider, "global", "config")}
            onEdit={() => setEditSlot({ scope: "global", purpose: "config", paths: globalConfig?.paths ?? [] })}
            onReset={() => handleResetSlot("global", "config")}
            isResetting={resetMutation.isPending}
          />
        </td>
        <td className="px-3 py-2">
          <OptionalSlotCell
            data={projectConfig}
            hasSlotOverride={slotHasOverride(provider, "project", "config")}
            onEdit={() => setEditSlot({ scope: "project", purpose: "config", paths: projectConfig.paths })}
            onReset={() => handleResetSlot("project", "config")}
            isResetting={resetMutation.isPending}
          />
        </td>
        <td className="px-3 py-2">
          <OptionalSlotCell
            data={globalSkills ?? { paths: [], source: "builtin" }}
            hasSlotOverride={slotHasOverride(provider, "global", "skills")}
            onEdit={() => setEditSlot({ scope: "global", purpose: "skills", paths: globalSkills?.paths ?? [] })}
            onReset={() => handleResetSlot("global", "skills")}
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

type ResetStep = "idle" | "confirm1" | "confirm2" | "resetting";

function DangerZone(): React.JSX.Element {
  const [step, setStep] = useState<ResetStep>("idle");
  const [confirmInput, setConfirmInput] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const resetMutation = useResetAll();

  function openConfirm1(): void {
    setStep("confirm1");
    setConfirmInput("");
  }

  function toStep2(): void {
    setStep("confirm2");
    setConfirmInput("");
    setTimeout(() => inputRef.current?.focus(), 50);
  }

  function cancel(): void {
    setStep("idle");
    setConfirmInput("");
  }

  function handleReset(): void {
    if (confirmInput !== "RESET") return;
    setStep("resetting");
    resetMutation.mutate(undefined, {
      onError: () => setStep("confirm2"),
    });
  }

  return (
    <div className="rounded border border-red-200 bg-red-50 p-4">
      <h3 className="text-sm font-semibold text-red-700">Danger Zone</h3>
      <p className="mt-1 text-xs text-red-600">
        Xóa toàn bộ dữ liệu Skillbox (projects, skills, settings). Ứng dụng sẽ
        chuyển về màn hình cài đặt. Hành động này không thể hoàn tác.
      </p>

      {step === "idle" && (
        <button
          onClick={openConfirm1}
          className="mt-3 rounded border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
        >
          Reset All Data
        </button>
      )}

      {step === "confirm1" && (
        <div className="mt-3 space-y-2">
          <p className="text-xs font-medium text-red-700">
            Xóa toàn bộ dữ liệu? Hành động này không thể hoàn tác.
          </p>
          <div className="flex gap-2">
            <button
              onClick={cancel}
              className="rounded border border-zinc-300 bg-white px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
            >
              Cancel
            </button>
            <button
              onClick={toStep2}
              className="rounded border border-red-400 bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700"
            >
              Tiếp tục
            </button>
          </div>
        </div>
      )}

      {(step === "confirm2" || step === "resetting") && (
        <div className="mt-3 space-y-2">
          <p className="text-xs font-medium text-red-700">
            Gõ <span className="font-mono font-bold">RESET</span> để xác nhận.
          </p>
          <input
            ref={inputRef}
            type="text"
            value={confirmInput}
            onChange={(e) => setConfirmInput(e.target.value)}
            placeholder="RESET"
            disabled={step === "resetting"}
            className="w-40 rounded border border-zinc-300 px-2 py-1 text-xs font-mono disabled:opacity-50"
          />
          <div className="flex gap-2">
            <button
              onClick={cancel}
              disabled={step === "resetting"}
              className="rounded border border-zinc-300 bg-white px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={handleReset}
              disabled={confirmInput !== "RESET" || step === "resetting"}
              className="rounded border border-red-400 bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              {step === "resetting" ? "Đang xóa…" : "Xác nhận Reset"}
            </button>
          </div>
          {resetMutation.isError && (
            <p className="text-xs text-red-600">
              Lỗi: {String(resetMutation.error)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export function SettingsScreen(): React.JSX.Element {
  const { data: settings, isPending, isError, error } = useAppSettings();
  const { data: providerData } = useProviderList();

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
          <div className="px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium text-zinc-700">Skill Host Folder</div>
              <Link
                to="/skills"
                className="text-xs text-zinc-500 hover:text-zinc-800 hover:underline"
              >
                Manage in Host Skills →
              </Link>
            </div>
            <div className="mt-0.5 font-mono text-xs text-zinc-500">
              {settings?.activeHost?.path != null ? displayPath(settings.activeHost.path) : "Not configured"}
            </div>
          </div>
          {settings?.activeHost?.status === "missing" && (
            <div className="px-4 py-2 text-xs text-red-600 bg-red-50 border-t border-red-100">
              Folder not found on disk. Go to Host Skills to change it.
            </div>
          )}

          <div className="flex items-center justify-between px-4 py-3">
            <div className="text-sm font-medium text-zinc-700">Default Install Mode</div>
            <div className="text-sm text-zinc-500">
              {INSTALL_MODE_LABEL[settings?.defaultInstallMode ?? ""] ?? settings?.defaultInstallMode ?? "—"}
            </div>
          </div>
        </div>
      </div>

      <div>
        <h3 className="text-sm font-semibold text-zinc-800">Providers</h3>
        <p className="mt-0.5 text-xs text-zinc-500">
          Providers are activated by folder presence on disk. Overrides replace the built-in path candidates. Reset a slot to return to built-in defaults.
        </p>

        <div className="mt-3 overflow-x-auto rounded border border-zinc-200">
          <table className="min-w-full text-xs">
            <thead>
              <tr className="border-b border-zinc-100 bg-zinc-50 text-left text-zinc-500">
                <th className="px-3 py-2 font-medium">Provider</th>
                <th className="px-3 py-2 font-medium">Key</th>
                <th className="px-3 py-2 font-medium">Provider detection path</th>
                <th className="px-3 py-2 font-medium">Global config</th>
                <th className="px-3 py-2 font-medium">Project config</th>
                <th className="px-3 py-2 font-medium">Global skills</th>
                <th className="px-3 py-2 font-medium">Project skills</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {(providerData?.providers ?? []).map((provider) => (
                <ProviderRow key={provider.key} provider={provider} />
              ))}
              {(providerData?.providers ?? []).length === 0 && (
                <tr>
                  <td colSpan={7} className="px-3 py-4 text-center text-zinc-400">
                    Loading providers…
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <DangerZone />
    </div>
  );
}
