// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("../../features/skills-library/use-skill-detail.js", () => ({
  useSkillDetail: vi.fn(),
}));
vi.mock("../../lib/core-client/methods.js", () => ({
  methods: {
    openPath: vi.fn(),
  },
}));
vi.mock("@tanstack/react-router", () => ({
  useParams: vi.fn(),
  useNavigate: vi.fn(),
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
}));

import { SkillDetailScreen } from "../skill-detail-screen.js";
import { useSkillDetail } from "../../features/skills-library/use-skill-detail.js";
import { useParams, useNavigate } from "@tanstack/react-router";
import { methods } from "../../lib/core-client/methods.js";

const mockOpenPath = methods.openPath as ReturnType<typeof vi.fn>;

const mockUseSkillDetail = useSkillDetail as ReturnType<typeof vi.fn>;
const mockUseParams = useParams as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;

const fakeSkill = {
  id: 1,
  name: "my-skill",
  relativePath: ".agents/skills/my-skill",
  absolutePath: "/host/.agents/skills/my-skill",
  status: "available" as const,
  sourceLabel: null,
  hostPath: "/host",
  lastScannedAt: "2026-05-26T10:00:00Z",
};

const fakeProject = {
  projectId: 10,
  projectName: "proj-alpha",
  projectProviderId: 5,
  providerKey: "generic_agents",
  providerDisplayName: "Shared Agent Skills (.agents)",
  mode: "symlink" as const,
  status: "current" as const,
  projectSkillPath: "/proj-alpha/.agents/skills/my-skill",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
});

afterEach(() => cleanup());

describe("SkillDetailScreen", () => {
  it("shows invalid skill ID error for non-numeric param", () => {
    mockUseParams.mockReturnValue({ skillId: "abc" });
    mockUseSkillDetail.mockReturnValue({ isPending: false, isError: false, data: undefined });

    render(<SkillDetailScreen />);
    expect(screen.getByText(/Invalid skill ID/i)).toBeTruthy();
    expect(mockUseSkillDetail).not.toHaveBeenCalledWith(expect.any(Number));
  });

  it("shows invalid skill ID error for id <= 0", () => {
    mockUseParams.mockReturnValue({ skillId: "0" });
    mockUseSkillDetail.mockReturnValue({ isPending: false, isError: false, data: undefined });

    render(<SkillDetailScreen />);
    expect(screen.getByText(/Invalid skill ID/i)).toBeTruthy();
  });

  it("renders skill metadata when loaded", () => {
    mockUseParams.mockReturnValue({ skillId: "1" });
    mockUseSkillDetail.mockReturnValue({
      isPending: false,
      isError: false,
      data: { skill: fakeSkill, projects: [fakeProject] },
    });

    render(<SkillDetailScreen />);
    expect(screen.getByText("my-skill")).toBeTruthy();
    expect(screen.getByText("proj-alpha")).toBeTruthy();
    expect(screen.getByText("Shared Agent Skills (.agents)")).toBeTruthy();
  });

  it("renders empty state when no projects use the skill", () => {
    mockUseParams.mockReturnValue({ skillId: "1" });
    mockUseSkillDetail.mockReturnValue({
      isPending: false,
      isError: false,
      data: { skill: fakeSkill, projects: [] },
    });

    render(<SkillDetailScreen />);
    expect(screen.getByText("No projects use this skill.")).toBeTruthy();
  });

  it("Open Host Folder button calls methods.openPath with skill.hostPath", () => {
    mockUseParams.mockReturnValue({ skillId: "1" });
    mockUseSkillDetail.mockReturnValue({
      isPending: false,
      isError: false,
      data: { skill: fakeSkill, projects: [] },
    });

    render(<SkillDetailScreen />);
    fireEvent.click(screen.getByRole("button", { name: /open host folder/i }));
    expect(mockOpenPath).toHaveBeenCalledWith("/host");
  });

  it("has no write-action controls", () => {
    mockUseParams.mockReturnValue({ skillId: "1" });
    mockUseSkillDetail.mockReturnValue({
      isPending: false,
      isError: false,
      data: { skill: fakeSkill, projects: [fakeProject] },
    });

    render(<SkillDetailScreen />);
    expect(screen.queryByRole("button", { name: /install/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /remove/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /update/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /switch/i })).toBeNull();
  });
});
