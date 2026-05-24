import React, { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { FolderOpen, ArrowRight } from "lucide-react";
import { methods } from "../lib/core-client/methods.js";
import { useChooseHost } from "../features/skill-host/use-choose-host.js";
import { ErrorDisplay } from "../components/error-display.js";

export function SetupScreen(): React.JSX.Element {
  const navigate = useNavigate();
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [pickError, setPickError] = useState<unknown>(null);

  const chooseMutation = useChooseHost();

  async function handlePickFolder(): Promise<void> {
    setPickError(null);
    try {
      const result = await methods.openHostFolder();
      if (result.path != null) {
        setSelectedPath(result.path);
      }
    } catch (err) {
      setPickError(err);
    }
  }

  async function handleSetActive(): Promise<void> {
    if (selectedPath == null) return;
    chooseMutation.mutate(selectedPath, {
      onSuccess: () => {
        void navigate({ to: "/skills" });
      },
    });
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
          className="flex w-full items-center gap-2 rounded border border-zinc-300 bg-white px-4 py-2.5 text-sm text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100"
        >
          <FolderOpen size={16} />
          {selectedPath ?? "Choose Skill Host Folder…"}
        </button>

        {pickError != null && <ErrorDisplay error={pickError} />}
        {chooseMutation.error != null && <ErrorDisplay error={chooseMutation.error} />}

        {selectedPath != null && (
          <button
            onClick={handleSetActive}
            disabled={chooseMutation.isPending}
            className="flex w-full items-center justify-center gap-2 rounded bg-zinc-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50"
          >
            {chooseMutation.isPending ? "Setting up…" : "Set as Active Host"}
            {!chooseMutation.isPending && <ArrowRight size={15} />}
          </button>
        )}
      </div>
    </div>
  );
}
