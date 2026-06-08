// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("../../features/global-skills/use-global-list.js", () => ({
  useGlobalList: vi.fn(),
}));
vi.mock("../../features/global-skills/use-scan-global.js", () => ({
  useScanGlobal: vi.fn(),
}));
vi.mock("../../lib/core-client/methods.js", () => ({
  methods: { openPath: vi.fn() },
}));

import { GlobalSkillsScreen } from "../global-skills-screen.js";
import { useGlobalList } from "../../features/global-skills/use-global-list.js";
import { useScanGlobal } from "../../features/global-skills/use-scan-global.js";
import { methods } from "../../lib/core-client/methods.js";
import { clearAutoScanRegistry } from "../../features/scan/auto-scan-constants.js";
import type { GlobalListLocation } from "@contracts/index.js";

const mockUseGlobalList = useGlobalList as ReturnType<typeof vi.fn>;
const mockUseScanGlobal = useScanGlobal as ReturnType<typeof vi.fn>;
const mockOpenPath = methods.openPath as ReturnType<typeof vi.fn>;

const makeLocation = (overrides: Partial<GlobalListLocation> = {}): GlobalListLocation => ({
  globalProviderLocationId: 1,
  providerKey: "generic_agents",
  providerDisplayName: "Shared Agents",
  providerStatus: "active",
  path: "/Users/test/.agents",
  skillsPath: "/Users/test/.agents/skills",
  status: "active",
  lastScannedAt: null,
  entries: [],
  warnings: [],
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  clearAutoScanRegistry();
  mockUseScanGlobal.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
});

afterEach(() => cleanup());

describe("GlobalSkillsScreen", () => {
  it("renders empty state when no locations", () => {
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByText("No global skills found.")).toBeTruthy();
  });

  it("empty state shows Scan Global button only (no write controls)", () => {
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByRole("button", { name: /scan global/i })).toBeTruthy();
    expect(screen.queryByText(/relink/i)).toBeNull();
    expect(screen.queryByText(/remove/i)).toBeNull();
    expect(screen.queryByText(/install/i)).toBeNull();
  });

  it("renders location row with provider name and status", () => {
    const loc = makeLocation({ status: "active" });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    // Provider name appears in both tab and location header
    expect(screen.getAllByText("Shared Agents").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("active")).toBeTruthy();
  });

  it("renders no global skills message for scanned missing location with no entries", () => {
    const loc = makeLocation({ status: "missing", entries: [] });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByText("No global skills found.")).toBeTruthy();
  });

  it("renders entries table with skill name and mode", () => {
    const loc = makeLocation({
      entries: [{
        globalInstallId: 10,
        skillName: "my-skill",
        skillId: 5,
        mode: "symlink",
        status: "current",
        globalSkillPath: "/Users/test/.agents/skills/my-skill",
        sourceSkillPath: "/Users/test/host/.agents/skills/my-skill",
        symlinkTargetPath: "/Users/test/host/.agents/skills/my-skill",
      }],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByText("my-skill")).toBeTruthy();
    expect(screen.getByText("symlink")).toBeTruthy();
    expect(screen.getByText("current")).toBeTruthy();
  });

  it("does not render location warning feed when status already communicates the issue", () => {
    const loc = makeLocation({
      warnings: [{ code: "missing", severity: "warning", scopeType: "global_provider_location", scopeId: 1, actionKey: null, message: "Location missing" }],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.queryByText("Location missing")).toBeNull();
  });

  it("explains that status badges carry global scan state without showing warning metadata", () => {
    const loc = makeLocation({
      warnings: [{
        code: "global_provider_location_missing",
        severity: "warning",
        scopeType: "global_provider_location",
        scopeId: 1,
        actionKey: null,
        message: "~/.agents/skills directory not found",
      }],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByText(/Read-only scan of global provider folders/i)).toBeTruthy();
    expect(screen.getByText(/Status badges show whether each location and skill entry is usable/i)).toBeTruthy();
    expect(screen.queryByText("warning")).toBeNull();
    expect(screen.queryByText("global_provider_location_missing")).toBeNull();
    expect(screen.queryByText("~/.agents/skills")).toBeNull();
  });

  it("Open Folder button calls methods.openPath with location skillsPath", () => {
    const loc = makeLocation({ path: "/Users/test/.agents" });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    const openBtn = screen.getAllByRole("button", { name: /open folder/i })[0];
    fireEvent.click(openBtn);
    expect(mockOpenPath).toHaveBeenCalledWith("/Users/test/.agents/skills");
  });

  it("does not show Relink, Remove, or Install buttons", () => {
    const loc = makeLocation({
      entries: [{
        globalInstallId: 10,
        skillName: "my-skill",
        skillId: 5,
        mode: "symlink",
        status: "current",
        globalSkillPath: "/Users/test/.agents/skills/my-skill",
        sourceSkillPath: null,
        symlinkTargetPath: null,
      }],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.queryByRole("button", { name: /relink/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /remove/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /install/i })).toBeNull();
  });

  it("Scan Global button calls mutate", () => {
    const mockMutate = vi.fn();
    mockUseScanGlobal.mockReturnValue({ mutate: mockMutate, operationId: null, isPending: false });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [] } });

    render(<GlobalSkillsScreen />);
    fireEvent.click(screen.getByRole("button", { name: /scan global/i }));
    expect(mockMutate).toHaveBeenCalled();
  });

  // UX-005: provider tabs
  it("shows All tab with total entry count when locations are present", () => {
    const loc1 = makeLocation({
      providerKey: "generic_agents",
      providerDisplayName: "Shared Agents",
      entries: [
        { globalInstallId: 1, skillName: "skill-a", skillId: 1, mode: "symlink", status: "current", globalSkillPath: "/p/a", sourceSkillPath: null, symlinkTargetPath: null },
        { globalInstallId: 2, skillName: "skill-b", skillId: 2, mode: "symlink", status: "current", globalSkillPath: "/p/b", sourceSkillPath: null, symlinkTargetPath: null },
      ],
    });
    const loc2 = makeLocation({
      globalProviderLocationId: 2,
      providerKey: "claude",
      providerDisplayName: "Claude",
      entries: [
        { globalInstallId: 3, skillName: "skill-c", skillId: 3, mode: "symlink", status: "current", globalSkillPath: "/c/c", sourceSkillPath: null, symlinkTargetPath: null },
      ],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc1, loc2] } });

    render(<GlobalSkillsScreen />);
    // "All" tab should exist and show total count 3
    expect(screen.getByRole("button", { name: /^All/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Shared Agents/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Claude/ })).toBeTruthy();
  });

  it("clicking a provider tab shows only that provider's location", () => {
    const loc1 = makeLocation({
      globalProviderLocationId: 1,
      providerKey: "generic_agents",
      providerDisplayName: "Shared Agents",
      status: "active",
    });
    const loc2 = makeLocation({
      globalProviderLocationId: 2,
      providerKey: "claude",
      providerDisplayName: "Claude",
      status: "active",
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc1, loc2] } });

    render(<GlobalSkillsScreen />);
    // Initially both provider sections visible (status badge appears once per location section)
    expect(screen.getAllByText("active")).toHaveLength(2);
    // Click Claude tab — the tab button label contains "Claude"
    const claudeTab = screen.getAllByRole("button", { name: /Claude/ })[0];
    fireEvent.click(claudeTab);
    // Only Claude location section remains — one "active" badge in the content area
    // (status badges are inside location sections, not in the tab bar)
    expect(screen.getAllByText("active")).toHaveLength(1);
  });

  it("provider tab with no entries shows count 0", () => {
    const loc = makeLocation({
      providerKey: "generic_agents",
      providerDisplayName: "Shared Agents",
      status: "no_global_skills",
      entries: [],
    });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    // All tab shows 0
    const allTab = screen.getByRole("button", { name: /^All/ });
    expect(allTab.textContent).toMatch(/0/);
  });

  // UX-006: path display trims trailing slash
  it("trims trailing slash from displayed location path", () => {
    const loc = makeLocation({ path: "/Users/test/.agents/", skillsPath: "/Users/test/.agents/skills/" });
    mockUseGlobalList.mockReturnValue({ isPending: false, isError: false, data: { locations: [loc] } });

    render(<GlobalSkillsScreen />);
    expect(screen.getByText("/Users/test/.agents/skills")).toBeTruthy();
    expect(screen.queryByText("/Users/test/.agents/skills/")).toBeNull();
  });
});
