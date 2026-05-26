// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("@tanstack/react-router", () => ({
  useParams: vi.fn(),
  useNavigate: vi.fn(),
}));
vi.mock("../../features/projects/use-project-detail.js", () => ({
  useProjectDetail: vi.fn(),
}));
vi.mock("../../features/projects/use-scan-project.js", () => ({
  useScanProject: vi.fn(),
}));
vi.mock("../../features/projects/use-open-project-folder.js", () => ({
  useOpenProjectFolder: vi.fn(),
}));
vi.mock("../../features/projects/use-open-project-terminal.js", () => ({
  useOpenProjectTerminal: vi.fn(),
}));
vi.mock("../../features/projects/use-remove-project.js", () => ({
  useRemoveProject: vi.fn(),
}));
vi.mock("../../features/projects/use-remove-skill.js", () => ({
  useRemoveSkill: vi.fn(),
}));
vi.mock("../../features/skills/use-active-host-skills.js", () => ({
  useActiveHostSkills: vi.fn(),
}));

import { useParams, useNavigate } from "@tanstack/react-router";
import { ProjectDetailScreen } from "../project-detail-screen.js";
import { useProjectDetail } from "../../features/projects/use-project-detail.js";
import { useScanProject } from "../../features/projects/use-scan-project.js";
import { useOpenProjectFolder } from "../../features/projects/use-open-project-folder.js";
import { useOpenProjectTerminal } from "../../features/projects/use-open-project-terminal.js";
import { useRemoveProject } from "../../features/projects/use-remove-project.js";
import { useRemoveSkill } from "../../features/projects/use-remove-skill.js";
import { useActiveHostSkills } from "../../features/skills/use-active-host-skills.js";
import type { ProjectGetResponse } from "@contracts/index.js";

const mockUseParams = useParams as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;
const mockUseProjectDetail = useProjectDetail as ReturnType<typeof vi.fn>;
const mockUseScanProject = useScanProject as ReturnType<typeof vi.fn>;
const mockUseOpenProjectFolder = useOpenProjectFolder as ReturnType<typeof vi.fn>;
const mockUseOpenProjectTerminal = useOpenProjectTerminal as ReturnType<typeof vi.fn>;
const mockUseRemoveProject = useRemoveProject as ReturnType<typeof vi.fn>;
const mockUseRemoveSkill = useRemoveSkill as ReturnType<typeof vi.fn>;
const mockUseActiveHostSkills = useActiveHostSkills as ReturnType<typeof vi.fn>;

const projectDetail: ProjectGetResponse = {
  project: { id: 7, name: "demo", path: "/repo/demo", status: "active", lastScannedAt: null },
  providers: [
    {
      projectProviderId: 11,
      providerKey: "generic_agents",
      displayName: "Shared Agent Skills",
      providerStatus: "supported",
      detectionStatus: "detected",
      detectedPath: "/repo/demo/.agents",
      skillsPath: "/repo/demo/.agents/skills",
      entryCount: 1,
    },
    {
      projectProviderId: 12,
      providerKey: "claude",
      displayName: "Claude",
      providerStatus: "experimental",
      detectionStatus: "detected",
      detectedPath: "/repo/demo/.claude",
      skillsPath: "/repo/demo/.claude/skills",
      entryCount: 1,
    },
  ],
  entries: [
    {
      id: 101,
      projectProviderId: 11,
      providerKey: "generic_agents",
      name: "current-skill",
      mode: "symlink",
      status: "current",
      projectSkillPath: "/repo/demo/.agents/skills/current-skill",
      symlinkTargetPath: "/host/.agents/skills/current-skill",
      skillId: 201,
    },
    {
      id: 102,
      projectProviderId: 12,
      providerKey: "claude",
      name: "broken-skill",
      mode: "symlink",
      status: "broken_symlink",
      projectSkillPath: "/repo/demo/.claude/skills/broken-skill",
      symlinkTargetPath: "/missing/broken-skill",
      skillId: null,
    },
  ],
  warnings: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseParams.mockReturnValue({ projectId: "7" });
  mockUseNavigate.mockReturnValue(vi.fn());
  mockUseProjectDetail.mockReturnValue({ data: projectDetail, isPending: false, isError: false, error: null });
  mockUseScanProject.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
  mockUseOpenProjectFolder.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseOpenProjectTerminal.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseRemoveProject.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseRemoveSkill.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseActiveHostSkills.mockReturnValue({ skills: [] });
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
  });
});

afterEach(() => cleanup());

describe("ProjectDetailScreen UX clarity", () => {
  it("renders friendly provider and skill status labels", () => {
    render(<ProjectDetailScreen />);

    expect(screen.getByText("Ready")).toBeTruthy();
    expect(screen.getByText("Skillbox can manage this provider with the current feature set.")).toBeTruthy();
    expect(screen.getByText("Preview")).toBeTruthy();
    expect(screen.getByText("Linked to active host")).toBeTruthy();
    expect(screen.getByText("Broken link")).toBeTruthy();
  });

  it("filters skill entries by provider tab", () => {
    render(<ProjectDetailScreen />);

    expect(screen.getByText("current-skill")).toBeTruthy();
    expect(screen.getByText("broken-skill")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Claude 1/i }));

    expect(screen.queryByText("current-skill")).toBeNull();
    expect(screen.getByText("broken-skill")).toBeTruthy();
  });

  it("resets provider filter when refreshed data no longer contains the selected provider", () => {
    const { rerender } = render(<ProjectDetailScreen />);
    fireEvent.click(screen.getByRole("button", { name: /Claude 1/i }));
    expect(screen.queryByText("current-skill")).toBeNull();

    mockUseProjectDetail.mockReturnValue({
      data: {
        ...projectDetail,
        providers: [projectDetail.providers[0]],
        entries: [projectDetail.entries[0]],
      },
      isPending: false,
      isError: false,
      error: null,
    });

    rerender(<ProjectDetailScreen />);
    expect(screen.getByText("current-skill")).toBeTruthy();
    expect(screen.getByRole("button", { name: /All providers 1/i })).toBeTruthy();
  });

  it("shows full project skill paths directly and keeps copy actions", () => {
    render(<ProjectDetailScreen />);

    expect(screen.queryByRole("button", { name: /show full project skill path/i })).toBeNull();
    expect(screen.getAllByText("/repo/demo/.agents/skills/current-skill").length).toBeGreaterThan(0);

    fireEvent.click(screen.getAllByRole("button", { name: /copy project skill path/i })[0]);
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("/repo/demo/.agents/skills/current-skill");
  });

  it("shows project and target path detail lines below each skill entry", () => {
    render(<ProjectDetailScreen />);

    expect(screen.getAllByText("project:").length).toBe(projectDetail.entries.length);
    expect(screen.getAllByText("target:").length).toBe(projectDetail.entries.length);
    expect(screen.getAllByText("/repo/demo/.agents/skills/current-skill").length).toBeGreaterThan(1);
    expect(screen.getAllByText("/host/.agents/skills/current-skill").length).toBeGreaterThan(1);
  });

  it("does not mark copy as successful when clipboard is unavailable", () => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: undefined,
    });
    const timeoutSpy = vi.spyOn(window, "setTimeout");

    render(<ProjectDetailScreen />);
    fireEvent.click(screen.getAllByRole("button", { name: /copy project skill path/i })[0]);
    expect(screen.getAllByRole("button", { name: /copy project skill path/i }).length).toBeGreaterThan(0);
    expect(timeoutSpy).not.toHaveBeenCalled();
    timeoutSpy.mockRestore();
  });
});
