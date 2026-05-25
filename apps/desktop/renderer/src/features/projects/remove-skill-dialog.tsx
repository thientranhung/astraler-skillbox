import React from "react";

interface RemoveSkillDialogProps {
  skillName: string;
  providerDisplayName: string;
  path: string;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function RemoveSkillDialog({
  skillName,
  providerDisplayName,
  path,
  isPending,
  onConfirm,
  onCancel,
}: RemoveSkillDialogProps): React.JSX.Element {
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/30">
      <div className="w-full max-w-lg rounded-lg border border-zinc-200 bg-white p-5 shadow-xl">
        <h2 className="mb-3 text-sm font-semibold text-zinc-900">Remove skill from project</h2>

        <div className="mb-3 text-xs text-zinc-700">
          <div className="mb-1">
            Remove <span className="font-medium text-zinc-900">{skillName}</span>
          </div>
          <div>
            from <span className="font-medium text-zinc-900">{providerDisplayName}</span>
          </div>
        </div>

        <div className="mb-3 text-xs text-zinc-700">
          This deletes the symlink at:
          <div className="mt-1 break-all rounded bg-zinc-50 px-2 py-1 font-mono text-[11px] text-zinc-600">
            {path}
          </div>
        </div>

        <p className="mb-4 text-xs text-zinc-500">
          The skill in your Skill Host Folder is not affected.
        </p>

        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            disabled={isPending}
            className="rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isPending}
            className="rounded border border-red-300 bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 disabled:opacity-50"
          >
            Remove
          </button>
        </div>
      </div>
    </div>
  );
}
