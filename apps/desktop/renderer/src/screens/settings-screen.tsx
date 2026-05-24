import React from "react";
import { useNavigate } from "@tanstack/react-router";
import { FolderOpen } from "lucide-react";
import { useAppSettings } from "../features/app-settings/use-app-settings.js";
import { useChooseHost } from "../features/skill-host/use-choose-host.js";
import { methods } from "../lib/core-client/methods.js";
import { ErrorDisplay } from "../components/error-display.js";

const INSTALL_MODE_LABEL: Record<string, string> = {
  symlink: "Symlink",
  rsync_copy: "Copy (rsync)",
};

export function SettingsScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const { data: settings, isPending, isError, error } = useAppSettings();
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
    <div className="p-6">
      <h2 className="text-base font-semibold text-zinc-900">Settings</h2>

      <div className="mt-6 max-w-lg divide-y divide-zinc-100 rounded border border-zinc-200">
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
  );
}
