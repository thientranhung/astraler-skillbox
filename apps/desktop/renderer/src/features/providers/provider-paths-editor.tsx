import React, { useState } from "react";
import { X } from "lucide-react";
import { useUpdateProviderPaths } from "./use-update-provider-paths.js";

interface Props {
  providerKey: string;
  scope: "project" | "global";
  purpose: "detect" | "skills" | "config" | "commands";
  currentPaths: string[];
  onClose: () => void;
}

export function ProviderPathsEditor({ providerKey, scope, purpose, currentPaths, onClose }: Props): React.JSX.Element {
  const [rawPaths, setRawPaths] = useState(currentPaths.join("\n"));
  const mutation = useUpdateProviderPaths();

  function handleSave() {
    const parsed = rawPaths.split("\n").map((p) => p.trim()).filter(Boolean);
    const paths = (parsed.length > 0 ? parsed : currentPaths) as [string, ...string[]];
    mutation.mutate(
      { providerKey, scope, purpose, paths },
      { onSuccess: onClose },
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30">
      <div className="w-full max-w-sm rounded border border-zinc-200 bg-white shadow-lg">
        <div className="flex items-center justify-between border-b border-zinc-100 px-4 py-3">
          <div>
            <div className="text-sm font-semibold text-zinc-800">Edit paths</div>
            <div className="mt-0.5 text-xs text-zinc-500">
              <span className="font-mono">{providerKey}</span>
              {" · "}
              <span>{scope}</span>
              {" · "}
              <span>{purpose}</span>
            </div>
          </div>
          <button onClick={onClose} className="rounded p-1 text-zinc-400 hover:bg-zinc-100">
            <X size={14} />
          </button>
        </div>
        <div className="px-4 py-3">
          <label className="mb-1 block text-xs font-medium text-zinc-600">
            Paths (one per line)
          </label>
          <textarea
            className="w-full rounded border border-zinc-200 px-2 py-1.5 font-mono text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400"
            rows={4}
            value={rawPaths}
            onChange={(e) => setRawPaths(e.target.value)}
          />
          <p className="mt-2 text-xs text-zinc-400">
            {scope === "project"
              ? "Project paths must be relative (e.g. .agents/skills). Saving updates the effective path candidates for future scans and installs."
              : "Global paths must start with / or ~/. Saving updates the effective path candidates for future global scans."}
          </p>
          {mutation.isError && mutation.error != null && (
            <p className="mt-1 text-xs text-red-500">{String(mutation.error)}</p>
          )}
        </div>
        <div className="flex justify-end gap-2 border-t border-zinc-100 px-4 py-3">
          <button
            onClick={onClose}
            className="rounded border border-zinc-200 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={mutation.isPending}
            className="rounded bg-zinc-800 px-3 py-1.5 text-xs text-white hover:bg-zinc-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
