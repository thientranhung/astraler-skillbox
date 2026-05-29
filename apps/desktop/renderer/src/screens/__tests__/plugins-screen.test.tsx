// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";

vi.mock("../../features/provider-plugins/use-provider-plugin-list.js", () => ({
  useProviderPluginList: vi.fn(),
}));
vi.mock("../../features/provider-plugins/use-scan-provider-plugins-global.js", () => ({
  useScanProviderPluginsGlobal: vi.fn(),
}));
vi.mock("../../features/provider-plugins/use-set-provider-plugin-enabled.js", () => ({
  useSetProviderPluginEnabled: vi.fn(),
}));

import { PluginsScreen } from "../plugins-screen.js";
import { useProviderPluginList } from "../../features/provider-plugins/use-provider-plugin-list.js";
import { useScanProviderPluginsGlobal } from "../../features/provider-plugins/use-scan-provider-plugins-global.js";
import { useSetProviderPluginEnabled } from "../../features/provider-plugins/use-set-provider-plugin-enabled.js";
import { clearAutoScanRegistry } from "../../features/scan/auto-scan-constants.js";
import type { PPGlobalView } from "@contracts/index.js";

const mockUseList = useProviderPluginList as ReturnType<typeof vi.fn>;
const mockUseScan = useScanProviderPluginsGlobal as ReturnType<typeof vi.fn>;
const mockUseSetEnabled = useSetProviderPluginEnabled as ReturnType<typeof vi.fn>;

function makeGlobal(overrides: Partial<PPGlobalView> = {}): PPGlobalView {
  return {
    providerKey: "claude",
    userLayerPath: "/Users/test/.claude/settings.json",
    userLayerStatus: null,
    lastScannedAt: null,
    scanWarnings: [],
    plugins: [],
    marketplaces: [],
    managedOutOfScope: false,
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  clearAutoScanRegistry();
  mockUseScan.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
  mockUseSetEnabled.mockReturnValue({ mutate: vi.fn(), operationId: null, isPending: false });
});

afterEach(() => cleanup());

describe("PluginsScreen", () => {
  it("shows loading spinner while pending", () => {
    mockUseList.mockReturnValue({ isPending: true, isError: false, data: null });
    render(<PluginsScreen />);
    expect(document.querySelector(".animate-spin")).toBeTruthy();
  });

  it("shows empty state when data is null", () => {
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: null });
    render(<PluginsScreen />);
    expect(screen.getByText(/No plugin data/i)).toBeTruthy();
  });

  it("shows Scan Global button", () => {
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: null });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: /scan global/i })).toBeTruthy();
  });

  it("Scan Global button calls mutate", () => {
    const mockMutate = vi.fn();
    mockUseScan.mockReturnValue({ mutate: mockMutate, operationId: null, isPending: false });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: null });
    render(<PluginsScreen />);
    fireEvent.click(screen.getByRole("button", { name: /scan global/i }));
    expect(mockMutate).toHaveBeenCalled();
  });

  it("shows never scanned status when userLayerStatus is null", () => {
    const global = makeGlobal({ userLayerStatus: null });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("never scanned")).toBeTruthy();
  });

  it("shows 'not configured' for missing status — not error language", () => {
    const global = makeGlobal({ userLayerStatus: "missing" });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("not configured")).toBeTruthy();
    expect(screen.queryByText(/error/i)).toBeNull();
  });

  it("shows plugins table when plugins are present", () => {
    const global = makeGlobal({
      userLayerStatus: "ok",
      plugins: [{ pluginName: "my-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("my-plugin")).toBeTruthy();
    expect(screen.getByText("enabled")).toBeTruthy();
    expect(screen.getByText("npm")).toBeTruthy();
  });

  it("does not show Marketplaces section (removed in UI polish batch)", () => {
    const global = makeGlobal({
      userLayerStatus: "ok",
      marketplaces: [{ marketplaceName: "my-marketplace", sourceType: "npm", sourceSummary: "registry.npmjs.org" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.queryByText("my-marketplace")).toBeNull();
    expect(screen.queryByText("Marketplaces")).toBeNull();
  });

  it("shows scan notes for ok status with warnings", () => {
    const global = makeGlobal({
      userLayerStatus: "ok",
      scanWarnings: ["Truncated entry at line 42"],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("Truncated entry at line 42")).toBeTruthy();
    expect(screen.getByText("Scan notes")).toBeTruthy();
  });

  it("does not show scan notes section for missing status", () => {
    const global = makeGlobal({
      userLayerStatus: "missing",
      scanWarnings: ["some warning"],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.queryByText("Scan notes")).toBeNull();
  });

  it("shows managedOutOfScope note concisely", () => {
    const global = makeGlobal({ userLayerStatus: "ok", managedOutOfScope: true });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText(/managed outside Skillbox/i)).toBeTruthy();
  });

  it("renders multiple provider global views", () => {
    const claude = makeGlobal({ providerKey: "claude", userLayerPath: "/Users/test/.claude/settings.json" });
    const codex = makeGlobal({
      providerKey: "codex",
      userLayerPath: "/Users/test/.codex/config.toml",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "github", marketplaceName: "openai-curated", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [claude, codex], global: claude, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getAllByText("Claude").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Codex").length).toBeGreaterThan(0);
    expect(screen.getByText("/Users/test/.codex/config.toml")).toBeTruthy();
    expect(screen.getByText("github")).toBeTruthy();
  });

  it("shows scanning state when operationId is set", () => {
    mockUseScan.mockReturnValue({ mutate: vi.fn(), operationId: 5, isPending: false });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: null });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: /scanning/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /scanning/i }).hasAttribute("disabled")).toBe(true);
  });

  it("shows Enable/Disable buttons for claude provider when status is ok", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [
        { pluginName: "plugin-a", marketplaceName: "npm", status: "enabled" },
        { pluginName: "plugin-b", marketplaceName: "npm", status: "disabled" },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: "Disable" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Enable" })).toBeTruthy();
  });

  it("shows Enable/Disable buttons for antigravity_cli provider when status is ok", () => {
    const global = makeGlobal({
      providerKey: "antigravity_cli",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "ag-plugin", marketplaceName: "market", status: "disabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: "Enable" })).toBeTruthy();
  });

  it("shows toggle buttons for codex provider (TOML write support added)", () => {
    const global = makeGlobal({
      providerKey: "codex",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "codex-plugin", marketplaceName: "openai", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: "Disable" })).toBeTruthy();
  });

  it("toggle button calls setEnabled mutation with correct args", () => {
    const mutateFn = vi.fn();
    mockUseSetEnabled.mockReturnValue({ mutate: mutateFn, operationId: null, isPending: false });
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "my-plugin", marketplaceName: "my-market", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    fireEvent.click(screen.getByRole("button", { name: "Disable" }));
    expect(mutateFn).toHaveBeenCalledWith({
      providerKey: "claude",
      pluginName: "my-plugin",
      marketplaceName: "my-market",
      layer: "user",
      enabled: false,
    });
  });

  it("toggle button is disabled when a plugin operation is in flight", () => {
    mockUseSetEnabled.mockReturnValue({ mutate: vi.fn(), operationId: 7, isPending: false });
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "p", marketplaceName: "m", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByRole("button", { name: "Disable" }).hasAttribute("disabled")).toBe(true);
  });
});
