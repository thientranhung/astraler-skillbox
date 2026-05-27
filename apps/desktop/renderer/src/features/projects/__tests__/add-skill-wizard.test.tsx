// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";
import type { ProjectGetProvider, SkillListSkill } from "@contracts/index.js";

vi.mock("../use-install-skill.js", () => ({
  useInstallSkill: vi.fn(),
}));

import { AddSkillWizard } from "../add-skill-wizard.js";
import { useInstallSkill } from "../use-install-skill.js";

const mockUseInstallSkill = useInstallSkill as ReturnType<typeof vi.fn>;

function makeProvider(overrides: Partial<ProjectGetProvider> = {}): ProjectGetProvider {
  return {
    projectProviderId: 1,
    providerKey: "generic_agents",
    displayName: "Generic Agents",
    providerStatus: "supported",
    detectionStatus: "detected",
    ...overrides,
  } as ProjectGetProvider;
}

function makeSkill(overrides: Partial<SkillListSkill> = {}): SkillListSkill {
  return {
    id: 1,
    name: "My Skill",
    relativePath: ".agents/skills/my-skill.md",
    status: "available",
    sourceLabel: null,
    lastScannedAt: null,
    ...overrides,
  } as SkillListSkill;
}

beforeEach(() => {
  vi.clearAllMocks();
  mockUseInstallSkill.mockReturnValue({ mutate: vi.fn(), isPending: false });
});

afterEach(() => {
  cleanup();
});

describe("AddSkillWizard", () => {
  it("zero installable providers: Install button disabled and reason text shown, mutate NOT called", () => {
    const mutate = vi.fn();
    mockUseInstallSkill.mockReturnValue({ mutate, isPending: false });

    const providers: ProjectGetProvider[] = [
      makeProvider({ detectionStatus: "missing" }),
      makeProvider({ providerStatus: "unsupported", detectionStatus: "detected" }),
    ];
    const skills: SkillListSkill[] = [makeSkill()];

    render(
      <AddSkillWizard
        projectId={1}
        providers={providers}
        skills={skills}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText(/no provider is ready for install/i)).toBeTruthy();

    const installButton = screen.getByRole("button", { name: /^install$/i });
    expect((installButton as HTMLButtonElement).disabled).toBe(true);

    fireEvent.click(installButton);
    expect(mutate).not.toHaveBeenCalled();
  });

  it("provider with detectionStatus configured is included as installable", () => {
    const mutate = vi.fn();
    mockUseInstallSkill.mockReturnValue({ mutate, isPending: false });

    const providers: ProjectGetProvider[] = [
      makeProvider({ providerKey: "generic_agents", detectionStatus: "configured" }),
    ];
    const skills: SkillListSkill[] = [makeSkill({ id: 5, name: "Skill C" })];

    render(
      <AddSkillWizard
        projectId={7}
        providers={providers}
        skills={skills}
        onClose={vi.fn()}
      />,
    );

    const radio = screen.getByRole("radio", { name: /Generic Agents/i });
    expect(radio).toBeTruthy();
    expect((radio as HTMLInputElement).checked).toBe(true);

    const checkbox = screen.getByRole("checkbox", { name: /Skill C/i });
    fireEvent.click(checkbox);

    const installButton = screen.getByRole("button", { name: /^install$/i });
    expect((installButton as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(installButton);
    expect(mutate).toHaveBeenCalledWith({ projectId: 7, providerKey: "generic_agents", skillIds: [5] });
  });

  it("one installable provider + skills selected: Install calls mutate with correct args", () => {
    const mutate = vi.fn();
    mockUseInstallSkill.mockReturnValue({ mutate, isPending: false });

    const providers: ProjectGetProvider[] = [
      makeProvider({ providerKey: "generic_agents", displayName: "Generic Agents" }),
    ];
    const skills: SkillListSkill[] = [
      makeSkill({ id: 10, name: "Skill A" }),
      makeSkill({ id: 20, name: "Skill B" }),
    ];

    render(
      <AddSkillWizard
        projectId={42}
        providers={providers}
        skills={skills}
        onClose={vi.fn()}
      />,
    );

    const radio = screen.getByRole("radio", { name: /Generic Agents/i });
    expect(radio).toBeTruthy();
    expect((radio as HTMLInputElement).checked).toBe(true);

    // Select skill A
    const checkboxA = screen.getByRole("checkbox", { name: /Skill A/i });
    fireEvent.click(checkboxA);

    const installButton = screen.getByRole("button", { name: /^install$/i });
    expect((installButton as HTMLButtonElement).disabled).toBe(false);

    fireEvent.click(installButton);

    expect(mutate).toHaveBeenCalledWith({
      projectId: 42,
      providerKey: "generic_agents",
      skillIds: [10],
    });
  });

  it("single installable provider: radio visible and pre-selected without user interaction", () => {
    render(
      <AddSkillWizard
        projectId={1}
        providers={[makeProvider({ providerKey: "generic_agents", displayName: "Generic Agents" })]}
        skills={[makeSkill({ id: 3, name: "Skill X" })]}
        onClose={vi.fn()}
      />,
    );

    const radio = screen.getByRole("radio", { name: /Generic Agents/i });
    expect(radio).toBeTruthy();
    expect((radio as HTMLInputElement).checked).toBe(true);

    // Install button enabled once a skill is selected — no provider interaction needed
    const installButton = screen.getByRole("button", { name: /^install$/i });
    expect((installButton as HTMLButtonElement).disabled).toBe(true);

    fireEvent.click(screen.getByRole("checkbox", { name: /Skill X/i }));
    expect((installButton as HTMLButtonElement).disabled).toBe(false);
  });
});
