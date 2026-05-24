import React, { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { FolderOpen } from "lucide-react";
import { methods } from "../lib/core-client/methods.js";
import { useChooseHost } from "../features/skill-host/use-choose-host.js";
import { ErrorDisplay } from "../components/error-display.js";

export function SetupScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const [dialogError, setDialogError] = useState<unknown>(null);
  const chooseMutation = useChooseHost();

  async function handlePickFolder(): Promise<void> {
    setDialogError(null);
    try {
      const result = await methods.openHostFolder();
      if (result.path == null) return; // user cancelled — no-op
      chooseMutation.mutate(result.path, {
        onSuccess: () => void navigate({ to: "/skills" }),
      });
    } catch (err) {
      setDialogError(err);
    }
  }

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-6 p-8">
      <div className="w-full max-w-md">
        <h1 className="text-lg font-semibold text-zinc-900">Welcome to Astraler Skillbox</h1>
        <p className="mt-1 text-sm text-zinc-500">
          Choose the folder that contains your agent skills to get started.
        </p>
      </div>

      <div className="w-full max-w-md space-y-3">
        <button
          onClick={handlePickFolder}
          disabled={chooseMutation.isPending}
          className="flex w-full items-center gap-2 rounded border border-zinc-300 bg-white px-4 py-2.5 text-sm text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100 disabled:opacity-50"
        >
          <FolderOpen size={16} />
          {chooseMutation.isPending ? "Setting up…" : "Choose Skill Host Folder…"}
        </button>

        {dialogError != null && <ErrorDisplay error={dialogError} />}
        {chooseMutation.error != null && <ErrorDisplay error={chooseMutation.error} />}
      </div>
    </div>
  );
}
