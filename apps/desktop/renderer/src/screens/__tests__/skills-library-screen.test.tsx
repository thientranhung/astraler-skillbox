// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("../../features/skill-host/use-active-host.js", () => ({
  useActiveHost: vi.fn(),
}));
vi.mock("../../features/skills-library/use-skills-list.js", () => ({
  useSkillsList: vi.fn(),
}));
vi.mock("../../features/skill-host/use-scan-host.js", () => ({
  useScanHost: vi.fn(),
}));
vi.mock("../../lib/core-client/methods.js", () => ({
  methods: {
    openPath: vi.fn(),
    openTerminal: vi.fn(),
  },
}));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: vi.fn(),
}));

import { SkillsLibraryScreen } from "../skills-library-screen.js";
import { useSkillsList } from "../../features/skills-library/use-skills-list.js";
import { useActiveHost } from "../../features/skill-host/use-active-host.js";
import { useScanHost } from "../../features/skill-host/use-scan-host.js";
import { useNavigate } from "@tanstack/react-router";
import { methods } from "../../lib/core-client/methods.js";

const mockOpenPath = methods.openPath as ReturnType<typeof vi.fn>;
const mockOpenTerminal = methods.openTerminal as ReturnType<typeof vi.fn>;

const mockUseSkillsList = useSkillsList as ReturnType<typeof vi.fn>;
const mockUseActiveHost = useActiveHost as ReturnType<typeof vi.fn>;
const mockUseScanHost = useScanHost as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;

const makeSkill = (overrides = {}) => ({
  id: 1,
  name: "my-skill",
  relativePath: ".agents/skills/my-skill",
  status: "available" as const,
  sourceLabel: null,
  lastScannedAt: null,
  projectsUsingCount: 2,
  ...overrides,
});

const baseData = {
  hostPath: "/tmp/host",
  skills: [
    makeSkill({ id: 1, name: "alpha-skill", status: "available" as const, projectsUsingCount: 3 }),
    makeSkill({ id: 2, name: "beta-skill", status: "missing" as const, projectsUsingCount: 0 }),
  ],
  totals: { available: 1, missing: 1, unreadable: 0, local_modified: 0, unknown: 0 },
  lastScanAt: null,
  warnings: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseActiveHost.mockReturnValue({ hostId: 1, path: "/tmp/host", skillsPath: "/tmp/host/.agents/skills", status: "active", lastScannedAt: null });
  mockUseScanHost.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
  mockUseNavigate.mockReturnValue(vi.fn());
});

afterEach(() => cleanup());

describe("SkillsLibraryScreen", () => {
  it("renders Projects column header", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    expect(screen.getByText("Projects")).toBeTruthy();
  });

  it("renders projectsUsingCount in rows", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    expect(screen.getAllByText("3").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("0").length).toBeGreaterThanOrEqual(1);
  });

  it("search filters skills by name", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    const searchInput = screen.getByPlaceholderText("Search skills…");
    fireEvent.change(searchInput, { target: { value: "alpha" } });

    expect(screen.queryByText("alpha-skill")).toBeTruthy();
    expect(screen.queryByText("beta-skill")).toBeNull();
  });

  it("status filter narrows by status", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    const select = screen.getByRole("combobox");
    fireEvent.change(select, { target: { value: "missing" } });

    expect(screen.queryByText("alpha-skill")).toBeNull();
    expect(screen.queryByText("beta-skill")).toBeTruthy();
  });

  it("Open Folder button calls methods.openPath with hostPath", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    fireEvent.click(screen.getByRole("button", { name: /open folder/i }));
    expect(mockOpenPath).toHaveBeenCalledWith("/tmp/host");
  });

  it("Terminal button opens terminal at the host folder", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    fireEvent.click(screen.getByRole("button", { name: /terminal/i }));
    expect(mockOpenTerminal).toHaveBeenCalledWith("/tmp/host");
  });

  it("shows provider tabs for the current host provider view", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    expect(screen.getByRole("button", { name: /All skills 2/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Shared Agent Skills 2/i })).toBeTruthy();
  });

  it("navigates to /skills/$skillId on row click", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    const rows = screen.getAllByRole("row");
    // rows[0] = header, rows[1] = first data row
    fireEvent.click(rows[1]);
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/skills/$skillId",
      params: { skillId: "1" },
    });
  });
});
