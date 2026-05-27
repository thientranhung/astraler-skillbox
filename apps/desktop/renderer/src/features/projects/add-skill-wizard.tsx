import React, { useState, useEffect, useMemo } from "react";
import { X } from "lucide-react";
import type { ProjectGetProvider, SkillListSkill } from "@contracts/index.js";
import { ProviderIcon } from "../../components/provider-icon.js";
import { useInstallSkill } from "./use-install-skill.js";

interface AddSkillWizardProps {
  projectId: number;
  providers: ProjectGetProvider[];
  skills: SkillListSkill[];
  onClose: () => void;
}

export function AddSkillWizard({
  projectId,
  providers,
  skills,
  onClose,
}: AddSkillWizardProps): React.JSX.Element {
  const installSkill = useInstallSkill();

  const installableProviders = useMemo(
    () =>
      providers.filter(
        (p) =>
          (p.providerStatus === "supported" || p.providerStatus === "experimental") &&
          (p.detectionStatus === "detected" || p.detectionStatus === "configured"),
      ),
    [providers],
  );

  const availableSkills = skills.filter((s) => s.status === "available");

  const [selectedSkillIds, setSelectedSkillIds] = useState<Set<number>>(new Set());
  const [selectedProviderKey, setSelectedProviderKey] = useState<string>("");

  useEffect(() => {
    setSelectedProviderKey((prev) =>
      installableProviders.some((p) => p.providerKey === prev)
        ? prev
        : installableProviders.length === 1
          ? installableProviders[0].providerKey
          : "",
    );
  }, [installableProviders]);

  const canInstall =
    installableProviders.length > 0 &&
    selectedSkillIds.size > 0 &&
    selectedProviderKey !== "";

  function handleToggleSkill(id: number): void {
    setSelectedSkillIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  function handleInstall(): void {
    if (!canInstall) return;
    installSkill.mutate({
      projectId,
      providerKey: selectedProviderKey as "generic_agents" | "claude",
      skillIds: [...selectedSkillIds] as [number, ...number[]],
    });
    onClose();
  }

  return (
    <div className="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-zinc-900">Add Skills</h2>
        <button
          onClick={onClose}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600"
          title="Close"
        >
          <X size={14} />
        </button>
      </div>

      {installableProviders.length === 0 ? (
        <p className="mb-4 text-xs text-zinc-500">No provider is ready for install.</p>
      ) : (
        <div className="mb-4">
          <label className="mb-1 block text-xs font-medium text-zinc-700">Provider</label>
          <div className="flex flex-col gap-1">
            {installableProviders.map((p) => (
              <label key={p.providerKey} className="flex items-center gap-2 text-xs text-zinc-700">
                <input
                  type="radio"
                  name="provider"
                  value={p.providerKey}
                  checked={selectedProviderKey === p.providerKey}
                  onChange={() => setSelectedProviderKey(p.providerKey)}
                  className="accent-blue-600"
                />
                <span className="inline-flex items-center gap-1.5">
                  <ProviderIcon providerKey={p.providerKey} />
                  {p.displayName}
                </span>
              </label>
            ))}
          </div>
        </div>
      )}

      <div className="mb-4">
        <label className="mb-1 block text-xs font-medium text-zinc-700">Skills</label>
        {availableSkills.length === 0 ? (
          <p className="text-xs text-zinc-400">No available skills.</p>
        ) : (
          <div className="max-h-48 overflow-y-auto rounded border border-zinc-100 bg-zinc-50">
            {availableSkills.map((skill) => (
              <label
                key={skill.id}
                className="flex cursor-pointer items-center gap-2 border-b border-zinc-100 px-3 py-2 last:border-0 hover:bg-zinc-100"
              >
                <input
                  type="checkbox"
                  checked={selectedSkillIds.has(skill.id)}
                  onChange={() => handleToggleSkill(skill.id)}
                  className="accent-blue-600"
                />
                <span className="text-xs font-medium text-zinc-800">{skill.name}</span>
                <span className="ml-auto max-w-[65%] break-all text-right font-mono text-xs leading-snug text-zinc-400">{skill.relativePath}</span>
              </label>
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-2">
        <button
          onClick={onClose}
          className="rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
        >
          Cancel
        </button>
        <button
          onClick={handleInstall}
          disabled={!canInstall || installSkill.isPending}
          className="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {installSkill.isPending ? "Installing…" : "Install"}
        </button>
      </div>
    </div>
  );
}
