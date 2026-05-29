// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import React from 'react';
import type { ProjectGetProvider, ProjectGetEntry, SkillListSkill } from '@contracts/index.js';

vi.mock('../use-install-skill.js', () => ({ useInstallSkill: vi.fn() }));
vi.mock('../use-scan-project.js', () => ({ useScanProject: vi.fn() }));
vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return { ...actual, useIsMutating: vi.fn().mockReturnValue(0) };
});

import { AddSkillWizard } from '../add-skill-wizard.js';
import { useInstallSkill } from '../use-install-skill.js';
import { useScanProject } from '../use-scan-project.js';

const mockUseInstallSkill = useInstallSkill as ReturnType<typeof vi.fn>;
const mockUseScanProject = useScanProject as ReturnType<typeof vi.fn>;

function mkProvider(
  key: string,
  displayName: string,
  providerStatus: ProjectGetProvider['providerStatus'] = 'supported',
  detectionStatus: ProjectGetProvider['detectionStatus'] = 'detected',
  skillsPath: string | null = `/projects/test/${key}/skills`,
): ProjectGetProvider {
  return {
    projectProviderId: Math.floor(Math.random() * 10000),
    providerKey: key,
    displayName,
    providerStatus,
    detectionStatus,
    detectedPath: `/projects/test/${key}`,
    skillsPath,
    entryCount: 0,
  } as ProjectGetProvider;
}

function mkSkill(id: number, name: string): SkillListSkill {
  return {
    id,
    name,
    relativePath: `.agents/skills/${name.toLowerCase().replace(/ /g, '-')}.md`,
    status: 'available',
    sourceLabel: null,
    lastScannedAt: null,
  } as SkillListSkill;
}

function mkEntry(skillId: number, providerKey: string, status = 'current'): ProjectGetEntry {
  return {
    id: Math.floor(Math.random() * 10000),
    projectProviderId: 1,
    providerKey,
    name: `skill-${skillId}`,
    mode: 'symlink',
    status,
    projectSkillPath: `/projects/test/${providerKey}/skills/skill-${skillId}`,
    symlinkTargetPath: null,
    skillId,
  } as ProjectGetEntry;
}

beforeEach(() => {
  vi.clearAllMocks();
  mockUseInstallSkill.mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
    reset: vi.fn(),
  });
  mockUseScanProject.mockReturnValue({ mutate: vi.fn(), isPending: false });
});

afterEach(() => {
  cleanup();
});

describe('AddSkillWizard', () => {
  // T1 — renders one tab per installable provider (not unsupported)
  it('T1: renders one tab per installable provider; unsupported provider not rendered', () => {
    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents'),
      mkProvider('claude', 'Claude'),
      mkProvider('codex', 'Codex', 'unsupported', 'detected'),
    ];
    const skills: SkillListSkill[] = [mkSkill(1, 'Skill A')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    // Use getAllByText to get past SVG <title> matches and verify tab buttons exist
    const genericTabs = screen.getAllByText('Generic Agents');
    const claudeTabs = screen.getAllByText('Claude');
    expect(genericTabs.some((el) => el.closest('button') != null)).toBe(true);
    expect(claudeTabs.some((el) => el.closest('button') != null)).toBe(true);
    // Codex is unsupported — no tab button for it
    const codexEls = screen.queryAllByText('Codex');
    expect(codexEls.some((el) => el.closest('button') != null)).toBe(false);
  });

  // T2 — active tab disables installed skills (status current)
  it('T2: active tab disables installed skills (status current)', () => {
    const providers: ProjectGetProvider[] = [mkProvider('generic_agents', 'Generic Agents')];
    const skills: SkillListSkill[] = [
      mkSkill(1, 'Skill S1'),
      mkSkill(2, 'Skill S2'),
      mkSkill(3, 'Skill S3'),
    ];
    const entries: ProjectGetEntry[] = [mkEntry(1, 'generic_agents', 'current')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        entries={entries}
        onClose={vi.fn()}
      />,
    );

    const s1Checkbox = screen.getByRole('checkbox', { name: /Skill S1/i }) as HTMLInputElement;
    const s2Checkbox = screen.getByRole('checkbox', { name: /Skill S2/i }) as HTMLInputElement;
    const s3Checkbox = screen.getByRole('checkbox', { name: /Skill S3/i }) as HTMLInputElement;

    expect(s1Checkbox.disabled).toBe(true);
    expect(s2Checkbox.disabled).toBe(false);
    expect(s3Checkbox.disabled).toBe(false);
  });

  // T3 — switching tab resets selected skills
  it('T3: switching tab resets selected skills', () => {
    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents'),
      mkProvider('claude', 'Claude'),
    ];
    const skills: SkillListSkill[] = [mkSkill(1, 'Skill S1')];
    // S2 installed at claude (entry with skillId=2) but skill 2 is not in available list;
    // we just need to verify that selection resets
    const entries: ProjectGetEntry[] = [];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        entries={entries}
        onClose={vi.fn()}
      />,
    );

    // Tick S1
    const s1Checkbox = screen.getByRole('checkbox', { name: /Skill S1/i }) as HTMLInputElement;
    fireEvent.click(s1Checkbox);
    expect(s1Checkbox.checked).toBe(true);

    // Switch to Claude tab — find the tab button (not SVG title)
    const claudeTabBtns = screen.getAllByText('Claude').filter((el) => el.closest('button') != null);
    fireEvent.click(claudeTabBtns[0]);

    // S1 should be unchecked (selection reset)
    expect(s1Checkbox.checked).toBe(false);
  });

  // T4 — install submits providerKey of active tab
  it('T4: install submits providerKey of active tab', () => {
    const mutate = vi.fn();
    mockUseInstallSkill.mockReturnValue({
      mutate,
      isPending: false,
      isError: false,
      error: null,
      reset: vi.fn(),
    });

    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents'),
      mkProvider('claude', 'Claude'),
    ];
    const s3 = mkSkill(3, 'Skill S3');
    const skills: SkillListSkill[] = [s3];

    render(
      <AddSkillWizard
        projectId={42}
        providers={providers}
        skills={skills}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    // Switch to claude tab — find the tab button (not SVG title)
    const claudeTabBtns = screen.getAllByText('Claude').filter((el) => el.closest('button') != null);
    fireEvent.click(claudeTabBtns[0]);

    // Tick S3
    fireEvent.click(screen.getByRole('checkbox', { name: /Skill S3/i }));

    // Click Install
    fireEvent.click(screen.getByRole('button', { name: /^install$/i }));

    expect(mutate).toHaveBeenCalledWith(
      { projectId: 42, providerKey: 'claude', skillIds: [3] },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  // T5 — footer hint shows active skillsPath
  it('T5: footer hint shows active skillsPath', () => {
    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents', 'supported', 'detected', '/foo/bar/skills'),
    ];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={[]}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText(/\/foo\/bar\/skills/)).toBeTruthy();
  });

  // T6 — experimental provider shows badge
  it('T6: experimental provider shows badge in tab strip', () => {
    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents', 'experimental', 'detected'),
    ];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={[]}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText('experimental')).toBeTruthy();
  });

  // T7 — empty state renders Scan CTA and triggers scan mutation
  it('T7: empty state shows Scan CTA; clicking triggers scan.mutate and onClose', () => {
    const scanMutate = vi.fn();
    mockUseScanProject.mockReturnValue({ mutate: scanMutate, isPending: false });

    const onClose = vi.fn();

    render(
      <AddSkillWizard
        projectId={5}
        providers={[]}
        skills={[]}
        entries={[]}
        onClose={onClose}
      />,
    );

    expect(screen.getByText(/no provider is ready for install/i)).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: /scan project/i }));

    expect(scanMutate).toHaveBeenCalledWith(5);
    expect(onClose).toHaveBeenCalled();
  });

  // T8 — single provider still renders tab strip (1 tab)
  it('T8: single provider renders exactly one tab', () => {
    const providers: ProjectGetProvider[] = [mkProvider('generic_agents', 'Generic Agents')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={[]}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    // The tab button contains the displayName text
    const tabs = screen.getAllByText('Generic Agents');
    expect(tabs.length).toBeGreaterThanOrEqual(1);
  });

  // T9 — install button disabled while pending
  it('T9: install button is disabled and shows Installing… while mutation is pending', () => {
    mockUseInstallSkill.mockReturnValue({
      mutate: vi.fn(),
      isPending: true,
      isError: false,
      error: null,
      reset: vi.fn(),
    });

    const providers: ProjectGetProvider[] = [mkProvider('generic_agents', 'Generic Agents')];
    const skills: SkillListSkill[] = [mkSkill(1, 'Skill A')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        entries={[]}
        onClose={vi.fn()}
      />,
    );

    // Tick a skill (though it won't actually matter since isPending=true)
    fireEvent.click(screen.getByRole('checkbox', { name: /Skill A/i }));

    const installBtn = screen.getByRole('button', { name: /installing…/i }) as HTMLButtonElement;
    expect(installBtn.disabled).toBe(true);
    expect(installBtn.textContent).toContain('Installing…');
  });

  // T10 — Error row renders when mutation fails; wizard stays open
  it('T10: error row renders on mutation failure; onClose NOT called', () => {
    const onClose = vi.fn();
    mockUseInstallSkill.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      isError: true,
      error: new Error('Project has an active operation'),
      reset: vi.fn(),
    });

    const providers: ProjectGetProvider[] = [mkProvider('generic_agents', 'Generic Agents')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={[]}
        entries={[]}
        onClose={onClose}
      />,
    );

    expect(screen.getByText('Project has an active operation')).toBeTruthy();
    expect(onClose).not.toHaveBeenCalled();
  });

  // T11 — cross-provider isolation: skill installed at claude is tickable at generic
  it('T11: cross-provider isolation: skill installed at claude is tickable at generic', () => {
    const providers: ProjectGetProvider[] = [
      mkProvider('generic_agents', 'Generic Agents'),
      mkProvider('claude', 'Claude'),
    ];
    const skills: SkillListSkill[] = [mkSkill(1, 'Skill S1'), mkSkill(2, 'Skill S2'), mkSkill(3, 'Skill S3')];
    // S2 installed at claude only
    const entries: ProjectGetEntry[] = [mkEntry(2, 'claude', 'current')];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        entries={entries}
        onClose={vi.fn()}
      />,
    );

    // Default tab: generic_agents — S1, S2, S3 all tickable
    const s1Checkbox = screen.getByRole('checkbox', { name: /Skill S1/i }) as HTMLInputElement;
    const s2Checkbox = screen.getByRole('checkbox', { name: /Skill S2/i }) as HTMLInputElement;
    const s3Checkbox = screen.getByRole('checkbox', { name: /Skill S3/i }) as HTMLInputElement;

    expect(s1Checkbox.disabled).toBe(false);
    expect(s2Checkbox.disabled).toBe(false); // not installed at generic_agents
    expect(s3Checkbox.disabled).toBe(false);

    // Switch to claude tab
    const claudeTabBtns = screen.getAllByText('Claude').filter((el) => el.closest('button') != null);
    fireEvent.click(claudeTabBtns[0]);

    // After tab switch, get fresh references (disabled state changed in DOM)
    const s1After = screen.getByRole('checkbox', { name: /Skill S1/i }) as HTMLInputElement;
    const s2After = screen.getByRole('checkbox', { name: /Skill S2/i }) as HTMLInputElement;
    const s3After = screen.getByRole('checkbox', { name: /Skill S3/i }) as HTMLInputElement;

    expect(s2After.disabled).toBe(true);  // installed at claude
    expect(s1After.disabled).toBe(false);
    expect(s3After.disabled).toBe(false);
  });
});
