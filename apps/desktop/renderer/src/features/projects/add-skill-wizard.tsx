import React, { useState, useEffect, useMemo } from 'react';
import { useIsMutating } from '@tanstack/react-query';
import { X } from 'lucide-react';
import type { ProjectGetProvider, ProjectGetEntry, SkillListSkill } from '@contracts/index.js';
import { ProviderIcon } from '../../components/provider-icon.js';
import { useInstallSkill } from './use-install-skill.js';
import { useScanProject } from './use-scan-project.js';

interface AddSkillWizardProps {
  projectId: number;
  providers: ProjectGetProvider[];
  skills: SkillListSkill[];
  entries: ProjectGetEntry[];
  onClose: () => void;
}

// broken_symlink is kept here: Go install_skill uses LstatExists (os.Lstat) which sees
// broken symlinks as existing paths and returns conflict_error — user must remove manually.
const ACTIVE_DISABLE_STATUSES = ['current', 'old_host', 'external_symlink', 'broken_symlink'] as const;

// TODO: keep in sync with install.skill contract providerKey enum.
// When Phase 2 adds a new install target, update here and the contract.
const INSTALLABLE_PROVIDER_KEYS = new Set<string>(['generic_agents', 'claude']);

function buildInstalledMap(entries: ProjectGetEntry[]): Map<string, Set<number>> {
  const map = new Map<string, Set<number>>();
  for (const entry of entries) {
    if (entry.skillId == null) continue;
    if (!(ACTIVE_DISABLE_STATUSES as readonly string[]).includes(entry.status)) continue;
    let set = map.get(entry.providerKey);
    if (set == null) {
      set = new Set<number>();
      map.set(entry.providerKey, set);
    }
    set.add(entry.skillId);
  }
  return map;
}

export function AddSkillWizard({
  projectId,
  providers,
  skills,
  entries,
  onClose,
}: AddSkillWizardProps): React.JSX.Element {
  const installSkill = useInstallSkill();
  const scan = useScanProject();
  const isScanning = useIsMutating({ mutationKey: ['scan-project'] }) > 0;

  const [activeProviderKey, setActiveProviderKey] = useState<string>('');
  const [selectedSkillIds, setSelectedSkillIds] = useState<Set<number>>(new Set());

  const installableProviders = useMemo(
    () =>
      providers.filter(
        (p) =>
          INSTALLABLE_PROVIDER_KEYS.has(p.providerKey) &&
          (p.providerStatus === 'supported' || p.providerStatus === 'experimental') &&
          (p.detectionStatus === 'detected' || p.detectionStatus === 'configured'),
      ),
    [providers],
  );

  const installedMap = useMemo(() => buildInstalledMap(entries), [entries]);

  const activeProvider = installableProviders.find((p) => p.providerKey === activeProviderKey);
  const installedForActive = installedMap.get(activeProviderKey) ?? new Set<number>();
  const availableSkills = skills.filter((s) => s.status === 'available');
  const isInstalling = installSkill.isPending || installSkill.operationId != null;

  // Set default active tab when providers load
  useEffect(() => {
    setActiveProviderKey((prev) =>
      installableProviders.some((p) => p.providerKey === prev)
        ? prev
        : installableProviders[0]?.providerKey ?? '',
    );
  }, [installableProviders]);

  // Reset selection and clear install error when tab switches
  useEffect(() => {
    setSelectedSkillIds(new Set());
    installSkill.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeProviderKey]);

  function handleSwitchTab(key: string): void {
    if (key !== activeProviderKey) setActiveProviderKey(key);
  }

  function handleToggleSkill(id: number): void {
    setSelectedSkillIds((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }

  function handleInstall(): void {
    if (!activeProvider || selectedSkillIds.size === 0) return;
    installSkill.mutate(
      {
        projectId,
        providerKey: activeProviderKey as 'generic_agents' | 'claude',
        skillIds: [...selectedSkillIds] as [number, ...number[]],
      },
    );
  }

  // Empty state
  if (installableProviders.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-900">Add Skills</h2>
          <button
            onClick={onClose}
            className="cursor-pointer rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600"
            title="Close"
          >
            <X size={14} />
          </button>
        </div>
        <p className="mb-1 text-xs font-medium text-zinc-700">No provider is ready for install.</p>
        <p className="mb-4 text-xs text-zinc-500">
          Create the provider skills folder in this project, then scan again. For Shared Agent Skills, create .agents/skills.
        </p>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              scan.mutate(projectId);
              onClose();
            }}
            disabled={isScanning || scan.isPending}
            className="cursor-pointer rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isScanning || scan.isPending ? 'Scanning…' : 'Scan project'}
          </button>
          <button
            onClick={onClose}
            className="cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  // Tab layout
  return (
    <div className="flex min-h-0 flex-col rounded-lg border border-zinc-200 bg-white shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between p-4 pb-0">
        <h2 className="text-sm font-semibold text-zinc-900">Add Skills</h2>
        <button
          onClick={onClose}
          className="cursor-pointer rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600"
          title="Close"
        >
          <X size={14} />
        </button>
      </div>

      {/* Tab Strip */}
      <div className="flex flex-nowrap overflow-x-auto border-b border-zinc-200 px-4 pt-3">
        {installableProviders.map((p) => (
          <button
            key={p.providerKey}
            onClick={() => handleSwitchTab(p.providerKey)}
            title={p.skillsPath ?? ''}
            className={`mr-4 flex shrink-0 cursor-pointer items-center gap-1.5 pb-2 text-xs font-medium ${
              activeProviderKey === p.providerKey
                ? 'border-b-2 border-blue-600 text-zinc-900'
                : 'text-zinc-500 hover:text-zinc-700'
            }`}
          >
            <ProviderIcon providerKey={p.providerKey} />
            {p.displayName}
            {p.providerStatus === 'experimental' && (
              <span className="rounded bg-amber-100 px-1 py-0.5 text-xs font-medium text-amber-700">
                experimental
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab Body */}
      <div className="px-4 pb-2 pt-3">
        {availableSkills.length === 0 ? (
          <p className="text-xs text-zinc-400">No available skills.</p>
        ) : (
          <div className="max-h-48 overflow-y-auto rounded border border-zinc-100 bg-zinc-50">
            {availableSkills.map((skill) => {
              const isInstalled = installedForActive.has(skill.id);
              return (
                <label
                  key={skill.id}
                  className={`flex cursor-pointer items-center gap-2 border-b border-zinc-100 px-3 py-2 last:border-0 hover:bg-zinc-100 ${isInstalled ? 'opacity-50' : ''}`}
                >
                  <input
                    type="checkbox"
                    checked={selectedSkillIds.has(skill.id)}
                    disabled={isInstalled}
                    onChange={() => !isInstalled && handleToggleSkill(skill.id)}
                    className="accent-blue-600"
                  />
                  <span className="text-xs font-medium text-zinc-800">{skill.name}</span>
                  {isInstalled ? (
                    <span className="ml-auto rounded bg-zinc-200 px-1.5 py-0.5 text-xs text-zinc-500">
                      Installed
                    </span>
                  ) : (
                    <span className="ml-auto max-w-[65%] break-all text-right font-mono text-xs leading-snug text-zinc-400">
                      {skill.relativePath}
                    </span>
                  )}
                </label>
              );
            })}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="border-t border-zinc-100 px-4 py-3">
        {/* Hint */}
        {activeProvider?.skillsPath != null && (
          <p className="mb-2 truncate text-xs text-zinc-400" title={activeProvider.skillsPath}>
            Sẽ ghi vào: {activeProvider.skillsPath}
          </p>
        )}
        {/* Error row — covers both RPC-level errors and async operation failures (e.g. filesystem_error) */}
        {(installSkill.isError || installSkill.lastOperationError != null) && (
          <p className="mb-2 text-xs text-red-600">
            {installSkill.isError
              ? (installSkill.error instanceof Error
                  ? installSkill.error.message
                  : String(installSkill.error))
              : installSkill.lastOperationError}
          </p>
        )}
        {/* Button row */}
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={onClose}
            className="cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Cancel
          </button>
          <button
            onClick={handleInstall}
            disabled={selectedSkillIds.size === 0 || isInstalling}
            className="cursor-pointer rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isInstalling ? 'Installing…' : 'Install'}
          </button>
        </div>
      </div>
    </div>
  );
}
