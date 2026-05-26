// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, within, cleanup } from "@testing-library/react";
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
    { code: "missing_global_location", message: "Location not found", severity: "warning", scopeType: "global_provider_location", scopeId: null, actionKey: null },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
});

afterEach(() => cleanup());

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
    expect(within(summarySection).getByText("2")).not.toBeNull(); // attention needed
  });

  it("renders active host status as a green badge", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    const activeBadge = screen.getByText("active");
    expect(activeBadge.className).toContain("bg-green-100");
    expect(activeBadge.className).toContain("text-green-800");
  });

  it("opens global view from summary and keeps updates disabled", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    fireEvent.click(screen.getByRole("button", { name: /Global Skills Open global view/i }));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/global" });
    expect(screen.getByText("Not in this slice")).toBeTruthy();
  });

  it("navigates to skills and projects from summary rows", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    fireEvent.click(screen.getByRole("button", { name: /^Skills 5$/i }));
    fireEvent.click(screen.getByRole("button", { name: /^Projects 3$/i }));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/skills" });
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/projects" });
  });

  it("uses pointer cursor for clickable summary rows", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.getByRole("button", { name: /^Skills 5$/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /^Projects 3$/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /^Attention needed 2$/i }).className).toContain("cursor-pointer");
  });

  it("does not render raw warning explanations on the dashboard", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.queryByText(/Warning means Skillbox found something unusual/i)).toBeNull();
    expect(screen.queryByText("Location not found")).toBeNull();
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

  it("hides attention row when there are no warnings", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, summary: { ...baseData.summary, warnings: 0 }, warnings: [] },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    expect(screen.queryByText("Attention needed")).toBeNull();
  });

  it("routes attention to global when global warnings exist", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    fireEvent.click(screen.getByRole("button", { name: /^Attention needed 2$/i }));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/global" });
  });

  it("routes attention to projects for project and install warnings", () => {
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
    fireEvent.click(screen.getByRole("button", { name: /^Attention needed 2$/i }));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/projects" });
  });

  it("routes attention to skills for skill host warnings", () => {
    const mockNavigate = vi.fn();
    mockUseNavigate.mockReturnValue(mockNavigate);
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: {
        ...baseData,
        warnings: [
          { code: "host_missing", message: "Host missing", severity: "error", scopeType: "skill_host_folder", scopeId: 1, actionKey: null },
        ],
      },
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    fireEvent.click(screen.getByRole("button", { name: /^Attention needed 2$/i }));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/skills" });
  });
});
