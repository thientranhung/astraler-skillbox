// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import React from "react";

vi.mock("../../features/dashboard/use-dashboard.js", () => ({
  useDashboard: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: vi.fn(),
}));

import { DashboardScreen } from "../dashboard-screen.js";
import { useDashboard } from "../../features/dashboard/use-dashboard.js";
import { useNavigate } from "@tanstack/react-router";

const mockUseDashboard = useDashboard as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;

const baseData = {
  activeHost: {
    hostId: 1,
    path: "/tmp/host",
    skillsPath: "/tmp/host/.agents/skills",
    status: "active",
    lastScanAt: "2024-01-01T00:00:00Z",
  },
  summary: { skills: 5, projects: 3, warnings: 2 },
  installsByMode: { symlink: 10, rsyncCopy: 2, direct: 1 },
  warningsBySeverity: { info: 1, warning: 1, error: 0, blocking: 0 },
  warnings: [
    { code: "missing_skill", message: "Skill not found", severity: "warning", scopeType: "global", scopeId: null, actionKey: null },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
});

describe("DashboardScreen", () => {
  it("shows spinner when loading", () => {
    mockUseDashboard.mockReturnValue({ isPending: true, isError: false, data: undefined, refetch: vi.fn() });

    const { container } = render(<DashboardScreen />);
    const spinner = container.querySelector(".animate-spin");
    expect(spinner).not.toBeNull();
  });

  it("shows error display and retry button when error", () => {
    const mockRefetch = vi.fn();
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: true,
      error: new Error("fail"),
      data: undefined,
      refetch: mockRefetch,
    });

    render(<DashboardScreen />);
    expect(screen.getByText("fail")).toBeTruthy();

    const retryBtn = screen.getByRole("button", { name: "Retry" });
    fireEvent.click(retryBtn);
    expect(mockRefetch).toHaveBeenCalledOnce();
  });

  it("shows no-host notice when activeHost is null", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, activeHost: null },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.getByText("No Skill Host Folder configured.")).toBeTruthy();

    const setupBtn = screen.getByRole("button", { name: "Go to Setup" });
    fireEvent.click(setupBtn);
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/setup" });
  });

  it("shows skills and projects counts when loaded", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    // Use the Summary section to scope count assertions
    const summarySection = screen.getByText("Summary").closest("section")!;
    expect(within(summarySection).getByText("5")).not.toBeNull(); // skills
    expect(within(summarySection).getByText("3")).not.toBeNull(); // projects
    expect(within(summarySection).getByText("2")).not.toBeNull(); // warnings
  });

  it("shows 'Not in this slice' placeholders for global skills and updates", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    // Two muted placeholders: "Global Skills" and "Updates" rows
    const placeholders = screen.getAllByText("Not in this slice");
    expect(placeholders.length).toBeGreaterThanOrEqual(2);
  });

  it("shows zero-data CTA when projects === 0", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, summary: { skills: 5, projects: 0, warnings: 0 } },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.getByRole("button", { name: "Add Project" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "View Skills" })).toBeTruthy();
  });

  it("shows 'No active warnings' when warnings list is empty", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, warnings: [] },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.getByText("No active warnings")).toBeTruthy();
  });

  it("navigates to project detail when clicking a project-scoped warning", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: {
        ...baseData,
        warnings: [
          { code: "install_broken", message: "Install broken", severity: "error", scopeType: "project", scopeId: 7, actionKey: null },
        ],
      },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    const warningBtn = screen.getByRole("button", { name: "error: Install broken" });
    fireEvent.click(warningBtn);
    expect(mockNavigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "7" },
    });
  });
});
