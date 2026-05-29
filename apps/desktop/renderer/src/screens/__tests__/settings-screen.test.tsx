// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import React from "react";

vi.mock("../../features/app-settings/use-app-settings.js", () => ({
  useAppSettings: vi.fn(),
}));
vi.mock("../../features/skill-host/use-choose-host.js", () => ({
  useChooseHost: vi.fn(),
}));
vi.mock("../../features/providers/use-provider-list.js", () => ({
  useProviderList: vi.fn(),
}));
vi.mock("../../lib/core-client/methods.js", () => ({
  methods: {
    openHostFolder: vi.fn(),
  },
}));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: vi.fn(),
}));
vi.mock("../../features/providers/provider-paths-editor.js", () => ({
  ProviderPathsEditor: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="paths-editor">
      <button onClick={onClose}>CloseEditor</button>
    </div>
  ),
}));
vi.mock("../../features/providers/use-reset-provider-paths.js", () => ({
  useResetProviderPaths: vi.fn(),
}));
vi.mock("../../features/app-settings/use-reset-all.js", () => ({
  useResetAll: vi.fn(),
}));

import { SettingsScreen } from "../settings-screen.js";
import { useAppSettings } from "../../features/app-settings/use-app-settings.js";
import { useChooseHost } from "../../features/skill-host/use-choose-host.js";
import { useProviderList } from "../../features/providers/use-provider-list.js";
import { useNavigate } from "@tanstack/react-router";
import { useResetProviderPaths } from "../../features/providers/use-reset-provider-paths.js";
import { useResetAll } from "../../features/app-settings/use-reset-all.js";

const mockUseAppSettings = useAppSettings as ReturnType<typeof vi.fn>;
const mockUseChooseHost = useChooseHost as ReturnType<typeof vi.fn>;
const mockUseProviderList = useProviderList as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;
const mockUseResetProviderPaths = useResetProviderPaths as ReturnType<typeof vi.fn>;
const mockUseResetAll = useResetAll as ReturnType<typeof vi.fn>;

const baseSettings = {
  activeSkillHostFolderId: 1,
  defaultInstallMode: "symlink",
  databaseVersion: 9,
  activeHost: {
    hostId: 1,
    path: "/tmp/my-host",
    skillsPath: "/tmp/my-host/.agents/skills",
    status: "active",
    lastScannedAt: null,
  },
};

const makeProvider = (overrides = {}) => ({
  key: "generic_agents",
  displayName: "Shared Agent Skills",
  providerType: "generic_agents",
  iconKey: "generic_agents",
  status: "supported" as const,
  isAvailable: true,
  isEnabled: true,
  canToggle: true,
  canCreateStructure: false,
  hasGlobalLevel: true,
  candidates: [
    { relativePath: ".agents", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: ".agents/skills", scope: "project" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: ".agents/settings.json", scope: "project" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: "~/.agents/skills", scope: "global" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: "~/.agents/settings.json", scope: "global" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
  ],
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
  mockUseChooseHost.mockReturnValue({ isPending: false, mutate: vi.fn(), error: null });
  mockUseResetProviderPaths.mockReturnValue({ mutate: vi.fn(), isPending: false });
  mockUseResetAll.mockReturnValue({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false, isError: false, isSuccess: false, error: null });
});

afterEach(() => cleanup());

describe("SettingsScreen", () => {
  it("shows spinner when settings loading", () => {
    mockUseAppSettings.mockReturnValue({ isPending: true, isError: false, data: undefined });
    mockUseProviderList.mockReturnValue({ data: undefined });

    const { container } = render(<SettingsScreen />);
    expect(container.querySelector(".animate-spin")).not.toBeNull();
  });

  it("shows error display when settings error", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: true, error: new Error("db failed"), data: undefined });
    mockUseProviderList.mockReturnValue({ data: undefined });

    render(<SettingsScreen />);
    expect(screen.queryByText(/db failed/i)).not.toBeNull();
  });

  it("shows skill host folder path when configured", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: undefined });

    render(<SettingsScreen />);
    expect(screen.getByText("/tmp/my-host")).not.toBeNull();
  });

  it("renders Providers section heading", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: undefined });

    render(<SettingsScreen />);
    expect(screen.getByText("Providers")).not.toBeNull();
  });

  it("renders provider table with a provider row", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: { providers: [makeProvider()] },
    });

    render(<SettingsScreen />);
    expect(screen.getByText("Shared Agent Skills")).not.toBeNull();
    expect(screen.getByText("generic_agents")).not.toBeNull();
  });

  it("shows 'Provider detection path' column header", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });

    render(<SettingsScreen />);
    expect(screen.getByText("Provider detection path")).not.toBeNull();
  });

  it("does not show Status or Enabled column headers", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });

    render(<SettingsScreen />);
    expect(screen.queryByText("Status")).toBeNull();
    expect(screen.queryByText("Enabled")).toBeNull();
  });

  it("renders unsupported provider row without opacity-50", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [makeProvider({ key: "codex", displayName: "Codex", status: "unsupported", isAvailable: false, hasGlobalLevel: false, candidates: [] })],
      },
    });

    const { container } = render(<SettingsScreen />);
    const rows = container.querySelectorAll("tbody tr");
    for (const row of rows) {
      expect(row.className).not.toContain("opacity-50");
    }
  });

  it("shows project detect and skills paths from candidates", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: { providers: [makeProvider()] },
    });

    render(<SettingsScreen />);
    expect(screen.getByText(".agents")).not.toBeNull();
    expect(screen.getByText(".agents/skills")).not.toBeNull();
  });

  it("shows project and global config paths from candidates", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [
          makeProvider({
            key: "codex",
            displayName: "Codex",
            iconKey: "codex",
            candidates: [
              { relativePath: ".codex", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: ".codex/skills", scope: "project" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: ".codex/config.toml", scope: "project" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: "~/.codex/config.toml", scope: "global" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
            ],
          }),
        ],
      },
    });

    render(<SettingsScreen />);
    expect(screen.getByText(".codex/config.toml")).not.toBeNull();
    expect(screen.getByText("~/.codex/config.toml")).not.toBeNull();
  });

  it("shows edit controls for empty optional global skills slots", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [
          makeProvider({
            key: "antigravity_cli",
            displayName: "Antigravity CLI",
            iconKey: "antigravity",
            candidates: [
              { relativePath: ".antigravity-cli", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: ".antigravity-cli/skills", scope: "project" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: ".gemini/antigravity-cli/settings.json", scope: "project" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
              { relativePath: "~/.gemini/antigravity-cli/settings.json", scope: "global" as const, purpose: "config" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
            ],
          }),
        ],
      },
    });

    render(<SettingsScreen />);
    expect(screen.getByText("Not set")).not.toBeNull();
    expect(screen.getAllByRole("button", { name: /edit paths/i })).toHaveLength(5);
  });

  it("shows global skills path for providers with hasGlobalLevel", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: { providers: [makeProvider()] },
    });

    render(<SettingsScreen />);
    expect(screen.getByText("~/.agents/skills")).not.toBeNull();
  });

  it("shows loading placeholder when providers not yet loaded", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: undefined });

    render(<SettingsScreen />);
    expect(screen.getByText(/loading providers/i)).not.toBeNull();
  });

  it("shows Override badge when any candidate has source=override", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [makeProvider({
          candidates: [
            { relativePath: ".custom", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "override" as const, verificationStatus: "assumed" as const },
          ],
        })],
      },
    });
    render(<SettingsScreen />);
    expect(screen.getByText("Override")).not.toBeNull();
  });

  it("shows Edit button for each provider row", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });
    render(<SettingsScreen />);
    expect(screen.getAllByRole("button", { name: /edit/i }).length).toBeGreaterThan(0);
  });

  it("opens editor dialog when Edit clicked", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });
    render(<SettingsScreen />);
    fireEvent.click(screen.getAllByRole("button", { name: /edit/i })[0]);
    expect(screen.getByTestId("paths-editor")).not.toBeNull();
  });

  it("shows Reset button for provider with override candidates", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [makeProvider({
          candidates: [
            { relativePath: ".custom", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "override" as const, verificationStatus: "assumed" as const },
          ],
        })],
      },
    });
    render(<SettingsScreen />);
    expect(screen.getByRole("button", { name: /reset to default/i })).not.toBeNull();
  });

});
