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
vi.mock("../../features/provider-plugins/use-provider-plugin-list.js", () => ({
  useProviderPluginList: vi.fn(),
}));
vi.mock("../../features/provider-plugins/use-set-provider-plugin-enabled.js", () => ({
  useSetProviderPluginEnabled: vi.fn(),
}));
vi.mock("../../features/provider-plugins/use-remove-provider-plugin-override.js", () => ({
  useRemoveProviderPluginOverride: vi.fn(),
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
import { useProviderPluginList } from "../../features/provider-plugins/use-provider-plugin-list.js";
import { useSetProviderPluginEnabled } from "../../features/provider-plugins/use-set-provider-plugin-enabled.js";
import { useRemoveProviderPluginOverride } from "../../features/provider-plugins/use-remove-provider-plugin-override.js";
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
const mockUseProviderPluginList = useProviderPluginList as ReturnType<typeof vi.fn>;
const mockUseSetProviderPluginEnabled = useSetProviderPluginEnabled as ReturnType<typeof vi.fn>;
const mockUseRemoveProviderPluginOverride = useRemoveProviderPluginOverride as ReturnType<typeof vi.fn>;

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
  mockUseProviderPluginList.mockReturnValue({ isPending: false, isError: false, data: null });
  mockUseSetProviderPluginEnabled.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
  mockUseRemoveProviderPluginOverride.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
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

  it("shows project-relative skill paths directly and keeps copy actions for absolute paths", () => {
    render(<ProjectDetailScreen />);

    expect(screen.queryByRole("button", { name: /show full project skill path/i })).toBeNull();
    expect(screen.getByText(".agents/skills/current-skill")).toBeTruthy();
    expect(screen.queryByText("/repo/demo/.agents/skills/current-skill")).toBeNull();

    fireEvent.click(screen.getAllByRole("button", { name: /copy project skill path/i })[0]);
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("/repo/demo/.agents/skills/current-skill");
  });

  it("shows symlink target paths in the main row", () => {
    render(<ProjectDetailScreen />);

    expect(screen.queryByText("project:")).toBeNull();
    expect(screen.queryByText("target:")).toBeNull();
    expect(screen.getByText("/host/.agents/skills/current-skill")).toBeTruthy();
  });

  it("does not render a separate Scan Plugins button", () => {
    render(<ProjectDetailScreen />);
    expect(screen.queryByRole("button", { name: /scan plugins/i })).toBeNull();
    expect(screen.getByText("Provider Plugins")).toBeTruthy();
  });

  it("disables plugin toggle while the unified scan is in flight (F1)", () => {
    mockUseScanProject.mockReturnValue({ mutate: vi.fn(), isPending: true, operationId: 1 });
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "p", marketplaceName: "m", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    const projectBtn = screen.getByRole("button", { name: "enabled" });
    expect(projectBtn.hasAttribute("disabled")).toBe(true);
  });

  it("shows 'No plugin data' when provider plugin list returns null data", () => {
    mockUseProviderPluginList.mockReturnValue({ isPending: false, isError: false, data: null, error: null });
    render(<ProjectDetailScreen />);
    expect(screen.getByText(/No plugin data/i)).toBeTruthy();
  });

  it("shows loading state when provider plugin list is pending", () => {
    mockUseProviderPluginList.mockReturnValue({ isPending: true, isError: false, data: null, error: null });
    render(<ProjectDetailScreen />);
    expect(screen.getByText(/Loading plugin data/i)).toBeTruthy();
    expect(screen.queryByText(/No plugin data/i)).toBeNull();
  });

  it("shows error state when provider plugin list fails", () => {
    const err = new Error("providerPlugin.list failed");
    mockUseProviderPluginList.mockReturnValue({ isPending: false, isError: true, data: null, error: err });
    render(<ProjectDetailScreen />);
    expect(screen.queryByText(/No plugin data/i)).toBeNull();
    expect(screen.queryByText(/Loading plugin data/i)).toBeNull();
  });

  it("shows layer statuses and plugins when provider plugin data is available", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false,
      isError: false,
      data: {
        globals: [],
        global: {
          providerKey: "claude",
          userLayerPath: "/Users/test/.claude/settings.json",
          userLayerStatus: null,
          lastScannedAt: null,
          scanWarnings: [],
          plugins: [],
          marketplaces: [],
          managedOutOfScope: false,
        },
        projects: [{
          projectId: 7,
          providerKey: "claude",
          layerStatuses: [
            { layer: "user", scanStatus: "ok", filePath: "/Users/test/.claude/settings.json", lastScannedAt: null, scanWarnings: [] },
            { layer: "project", scanStatus: "missing", filePath: "/repo/demo/.claude/settings.json", lastScannedAt: null, scanWarnings: [] },
          ],
          plugins: [{ pluginName: "dev-tools", marketplaceName: "npm", effectiveStatus: "enabled", provenanceLayer: "user", layerBreakdown: [] }],
          marketplaces: [],
          managedOutOfScope: false,
        }],
      },
    });
    render(<ProjectDetailScreen />);
    expect(screen.getAllByText("user").length).toBeGreaterThan(0);
    expect(screen.getByText("not configured")).toBeTruthy();
    expect(screen.getByText("dev-tools")).toBeTruthy();
    expect(screen.getByText("enabled")).toBeTruthy();
  });

  it("shows missing layer as 'not configured', not error language", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false,
      isError: false,
      data: {
        globals: [],
        global: {
          providerKey: "claude",
          userLayerPath: "/Users/test/.claude/settings.json",
          userLayerStatus: null,
          lastScannedAt: null,
          scanWarnings: [],
          plugins: [],
          marketplaces: [],
          managedOutOfScope: false,
        },
        projects: [{
          projectId: 7,
          providerKey: "claude",
          layerStatuses: [
            { layer: "project", scanStatus: "missing", filePath: "/repo/demo/.claude/settings.json", lastScannedAt: null, scanWarnings: [] },
          ],
          plugins: [],
          marketplaces: [],
          managedOutOfScope: false,
        }],
      },
    });
    render(<ProjectDetailScreen />);
    expect(screen.getByText("not configured")).toBeTruthy();
    expect(screen.queryByText(/error/i)).toBeNull();
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

  function makeLayerBreakdown(projectDecl: string | null, userDecl: string | null) {
    return [
      { layer: "project", scanStatus: "ok", declaration: projectDecl },
      { layer: "user", scanStatus: "ok", declaration: userDecl },
    ];
  }

  function makeProjectPluginData(providerKey: string, plugins: Array<{ pluginName: string; marketplaceName: string; effectiveStatus: string; provenanceLayer: string | null; layerBreakdown?: Array<{ layer: string; scanStatus: string; declaration: string | null }> }>) {
    return {
      globals: [],
      global: {
        providerKey: "claude",
        userLayerPath: "/Users/test/.claude/settings.json",
        userLayerStatus: null,
        lastScannedAt: null,
        scanWarnings: [],
        plugins: [],
        marketplaces: [],
        managedOutOfScope: false,
      },
      projects: [{
        projectId: 7,
        providerKey,
        layerStatuses: [
          { layer: "project", scanStatus: "ok", filePath: `/repo/demo/.${providerKey}/settings.json`, lastScannedAt: null, scanWarnings: [] },
        ],
        plugins: plugins.map((p) => ({ ...p, layerBreakdown: p.layerBreakdown ?? [] })),
        marketplaces: [],
        managedOutOfScope: false,
      }],
    };
  }

  it("shows Project and User columns for claude project plugins", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "plugin-a", marketplaceName: "npm", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", null) },
        { pluginName: "plugin-b", marketplaceName: "npm", effectiveStatus: "disabled", provenanceLayer: "user", layerBreakdown: makeLayerBreakdown(null, "disabled") },
      ]),
    });
    render(<ProjectDetailScreen />);
    expect(screen.getByRole("columnheader", { name: "Project" })).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "User" })).toBeTruthy();
  });

  it("shows Project and User columns for antigravity_cli project plugins", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("antigravity_cli", [
        { pluginName: "ag-plugin", marketplaceName: "market", effectiveStatus: "disabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("disabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    expect(screen.getByRole("columnheader", { name: "Project" })).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "User" })).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "disabled" }).length).toBeGreaterThanOrEqual(1);
  });

  it("shows toggle buttons for codex project plugins (TOML write support added)", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("codex", [
        { pluginName: "codex-plugin", marketplaceName: "openai", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    expect(screen.getByRole("button", { name: "enabled" })).toBeTruthy();
  });

  it("clicking project-layer enabled button calls setEnabled with enabled=false (cycle to disabled)", () => {
    const mutateFn = vi.fn();
    mockUseSetProviderPluginEnabled.mockReturnValue({ mutate: mutateFn, operationId: null, isPending: false });
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "my-plugin", marketplaceName: "my-market", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    fireEvent.click(screen.getByRole("button", { name: "enabled" }));
    expect(mutateFn).toHaveBeenCalledWith({
      providerKey: "claude",
      pluginName: "my-plugin",
      marketplaceName: "my-market",
      layer: "project",
      projectId: 7,
      enabled: false,
    });
  });

  it("clicking project-layer not-set button calls setEnabled with enabled=true", () => {
    const mutateFn = vi.fn();
    mockUseSetProviderPluginEnabled.mockReturnValue({ mutate: mutateFn, operationId: null, isPending: false });
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "my-plugin", marketplaceName: "my-market", effectiveStatus: "enabled", provenanceLayer: "user", layerBreakdown: makeLayerBreakdown(null, "enabled") },
      ]),
    });
    render(<ProjectDetailScreen />);
    fireEvent.click(screen.getByRole("button", { name: "—" }));
    expect(mutateFn).toHaveBeenCalledWith({
      providerKey: "claude",
      pluginName: "my-plugin",
      marketplaceName: "my-market",
      layer: "project",
      projectId: 7,
      enabled: true,
    });
  });

  it("clicking project-layer disabled button calls removeOverride (cycle back to not-set)", () => {
    const removeOverrideFn = vi.fn();
    mockUseRemoveProviderPluginOverride.mockReturnValue({ mutate: removeOverrideFn, operationId: null, isPending: false });
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "my-plugin", marketplaceName: "my-market", effectiveStatus: "disabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("disabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    // Project column is rendered before user column; click the first "disabled" button (project)
    const disabledBtns = screen.getAllByRole("button", { name: "disabled" });
    fireEvent.click(disabledBtns[0]);
    expect(removeOverrideFn).toHaveBeenCalledWith({
      providerKey: "claude",
      pluginName: "my-plugin",
      marketplaceName: "my-market",
      layer: "project",
      projectId: 7,
    });
  });

  it("local override shows 'overridden' text in both Project and User columns", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "local-plugin", marketplaceName: "npm", effectiveStatus: "enabled", provenanceLayer: "local", layerBreakdown: [] },
      ]),
    });
    render(<ProjectDetailScreen />);
    expect(screen.queryByRole("button", { name: "enabled" })).toBeNull();
    expect(screen.getAllByText("overridden").length).toBeGreaterThanOrEqual(2);
  });

  it("user column is dimmed when project layer has a value", () => {
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "p", marketplaceName: "m", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", "enabled") },
      ]),
    });
    render(<ProjectDetailScreen />);
    // Both project and user columns show "enabled" buttons; user wrapper div gets opacity-40
    const allEnabledBtns = screen.getAllByRole("button", { name: "enabled" });
    // There should be 2: project column + user column
    expect(allEnabledBtns.length).toBe(2);
    // One of them is wrapped in a div with opacity-40 (the user column)
    const dimmedBtns = allEnabledBtns.filter((b) => b.closest("div")?.className.includes("opacity-40"));
    expect(dimmedBtns.length).toBe(1);
  });

  it("toggle button is disabled when an operation is in flight", () => {
    mockUseSetProviderPluginEnabled.mockReturnValue({ mutate: vi.fn(), operationId: 5, isPending: false });
    mockUseProviderPluginList.mockReturnValue({
      isPending: false, isError: false,
      data: makeProjectPluginData("claude", [
        { pluginName: "p", marketplaceName: "m", effectiveStatus: "enabled", provenanceLayer: "project", layerBreakdown: makeLayerBreakdown("enabled", null) },
      ]),
    });
    render(<ProjectDetailScreen />);
    const btns = screen.getAllByRole("button", { name: "enabled" });
    expect(btns.some((b) => b.hasAttribute("disabled"))).toBe(true);
  });
});
