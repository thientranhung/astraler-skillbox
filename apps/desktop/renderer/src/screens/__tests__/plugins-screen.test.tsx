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
vi.mock("../../features/update-check/use-run-update-check.js", () => ({
  useRunUpdateCheck: vi.fn(),
}));

import { PluginsScreen } from "../plugins-screen.js";
import { useProviderPluginList } from "../../features/provider-plugins/use-provider-plugin-list.js";
import { useScanProviderPluginsGlobal } from "../../features/provider-plugins/use-scan-provider-plugins-global.js";
import { useSetProviderPluginEnabled } from "../../features/provider-plugins/use-set-provider-plugin-enabled.js";
import { useRunUpdateCheck } from "../../features/update-check/use-run-update-check.js";
import { clearAutoScanRegistry } from "../../features/scan/auto-scan-constants.js";
import type { PPGlobalView } from "@contracts/index.js";

const mockUseList = useProviderPluginList as ReturnType<typeof vi.fn>;
const mockUseScan = useScanProviderPluginsGlobal as ReturnType<typeof vi.fn>;
const mockUseSetEnabled = useSetProviderPluginEnabled as ReturnType<typeof vi.fn>;
const mockUseRunUpdateCheck = useRunUpdateCheck as ReturnType<typeof vi.fn>;

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
  mockUseRunUpdateCheck.mockReturnValue({ run: vi.fn(), isRunning: false, isRateLimited: () => false, status: "idle", results: [] });
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

  it("falls back to legacy global view when globals array is empty", () => {
    const global = makeGlobal({
      userLayerStatus: "ok",
      plugins: [{ pluginName: "legacy-plugin", marketplaceName: "local", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("legacy-plugin")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Claude/ }).textContent).toContain("1");
    expect(screen.queryByRole("button", { name: /^All/ })).toBeNull();
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
    fireEvent.click(screen.getByRole("button", { name: /Codex/ }));
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

  it("shows Version column header and value when at least one plugin has a version", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [
        { pluginName: "versioned-plugin", marketplaceName: "npm", status: "enabled", version: "1.2.3" },
        { pluginName: "no-version-plugin", marketplaceName: "npm", status: "enabled", version: null },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("Version")).toBeTruthy();
    expect(screen.getByText("1.2.3")).toBeTruthy();
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(1);
  });

  it("hides Version column when all plugins have null version", () => {
    const global = makeGlobal({
      providerKey: "codex",
      userLayerStatus: "ok",
      plugins: [
        { pluginName: "codex-plugin", marketplaceName: "openai", status: "enabled", version: null },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.queryByText("Version")).toBeNull();
  });

  it("shows 'unknown' version literal when plugin has version: 'unknown'", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [
        { pluginName: "some-plugin", marketplaceName: "official", status: "enabled", version: "unknown" },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("unknown")).toBeTruthy();
  });

  it("shows per-plugin error badge when update check timed out", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "slow-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    mockUseRunUpdateCheck.mockReturnValue({
      run: vi.fn(),
      isRunning: false,
      isRateLimited: () => false,
      status: "all_failed",
      results: [{ pluginName: "slow-plugin", marketplaceName: "npm", updateAvailable: null, error: "timeout" }],
    });
    render(<PluginsScreen />);
    expect(screen.getByTitle("Update check: timeout")).toBeTruthy();
    expect(screen.getByText(/Check timed out/)).toBeTruthy();
  });

  it("shows per-plugin network error badge", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "net-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    mockUseRunUpdateCheck.mockReturnValue({
      run: vi.fn(),
      isRunning: false,
      isRateLimited: () => false,
      status: "all_failed",
      results: [{ pluginName: "net-plugin", marketplaceName: "npm", updateAvailable: null, error: "git_ls_remote_failed" }],
    });
    render(<PluginsScreen />);
    expect(screen.getByText(/Network error/)).toBeTruthy();
  });

  it("shows '—' defensively when version field is undefined (legacy DB)", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [
        // version field absent (undefined) simulates pre-migration DB entry
        { pluginName: "legacy-plugin", marketplaceName: "mkt", status: "enabled" },
        { pluginName: "newer-plugin", marketplaceName: "mkt", status: "enabled", version: "2.0.0" },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    // Version column should show since newer-plugin has a non-null version
    expect(screen.getByText("Version")).toBeTruthy();
    // legacy-plugin (undefined version) should show "—"
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("2.0.0")).toBeTruthy();
  });

  // UX-005: provider tabs
  it("shows per-provider tabs without an All aggregate when globals are present", () => {
    const sharedAgents = makeGlobal({
      providerKey: "generic_agents",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "shared-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    const claude = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "p1", marketplaceName: "npm", status: "enabled" }],
    });
    const codex = makeGlobal({
      providerKey: "codex",
      userLayerStatus: "ok",
      plugins: [
        { pluginName: "p2", marketplaceName: "npm", status: "enabled" },
        { pluginName: "p3", marketplaceName: "npm", status: "enabled" },
      ],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [claude, codex, sharedAgents], global: claude, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.queryByRole("button", { name: /^All/ })).toBeNull();
    expect(screen.getByRole("button", { name: /Shared Agents/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Claude/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Codex/ })).toBeTruthy();
    expect(screen.getByText("shared-plugin")).toBeTruthy();
    expect(screen.queryByText("p1")).toBeNull();
    expect(screen.queryByText("p2")).toBeNull();
  });

  it("clicking a provider tab shows only that provider's plugins", () => {
    const claude = makeGlobal({
      providerKey: "claude",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "claude-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    const codex = makeGlobal({
      providerKey: "codex",
      userLayerStatus: "ok",
      plugins: [{ pluginName: "codex-plugin", marketplaceName: "npm", status: "enabled" }],
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [claude, codex], global: claude, projects: [] } });
    render(<PluginsScreen />);
    // First provider is selected by default; no aggregate view is shown.
    expect(screen.getByText("claude-plugin")).toBeTruthy();
    expect(screen.queryByText("codex-plugin")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /Codex/ }));
    expect(screen.queryByText("claude-plugin")).toBeNull();
    expect(screen.getByText("codex-plugin")).toBeTruthy();
  });

  it("provider tab count shows 0 when provider has no plugins", () => {
    const claude = makeGlobal({ providerKey: "claude", userLayerStatus: "ok", plugins: [] });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [claude], global: claude, projects: [] } });
    render(<PluginsScreen />);
    const claudeTab = screen.getByRole("button", { name: /Claude/ });
    expect(claudeTab.textContent).toMatch(/0/);
    expect(screen.queryByRole("button", { name: /^All/ })).toBeNull();
  });

  // UX-006: userLayerPath with trailing slash is trimmed
  it("trims trailing slash from displayed userLayerPath", () => {
    const global = makeGlobal({
      providerKey: "claude",
      userLayerPath: "/Users/test/.claude/settings.json/",
      userLayerStatus: "ok",
    });
    mockUseList.mockReturnValue({ isPending: false, isError: false, data: { globals: [global], global, projects: [] } });
    render(<PluginsScreen />);
    expect(screen.getByText("/Users/test/.claude/settings.json")).toBeTruthy();
    expect(screen.queryByText("/Users/test/.claude/settings.json/")).toBeNull();
  });
});
