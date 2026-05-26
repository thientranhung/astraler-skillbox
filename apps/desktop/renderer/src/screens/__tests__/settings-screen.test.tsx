// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
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

import { SettingsScreen } from "../settings-screen.js";
import { useAppSettings } from "../../features/app-settings/use-app-settings.js";
import { useChooseHost } from "../../features/skill-host/use-choose-host.js";
import { useProviderList } from "../../features/providers/use-provider-list.js";
import { useNavigate } from "@tanstack/react-router";

const mockUseAppSettings = useAppSettings as ReturnType<typeof vi.fn>;
const mockUseChooseHost = useChooseHost as ReturnType<typeof vi.fn>;
const mockUseProviderList = useProviderList as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;

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
  canCreateStructure: false,
  hasGlobalLevel: true,
  candidates: [
    { relativePath: ".agents", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: ".agents/skills", scope: "project" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
    { relativePath: "~/.agents/skills", scope: "global" as const, purpose: "skills" as const, priority: 10, source: "builtin" as const, verificationStatus: "assumed" as const },
  ],
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
  mockUseNavigate.mockReturnValue(vi.fn());
  mockUseChooseHost.mockReturnValue({ isPending: false, mutate: vi.fn(), error: null });
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

  it("shows Supported badge for supported provider", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [makeProvider({ status: "supported" })],
      },
    });

    render(<SettingsScreen />);
    expect(screen.getByText("Supported")).not.toBeNull();
  });

  it("shows Unsupported badge and dims row for unsupported provider", () => {
    mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
    mockUseProviderList.mockReturnValue({
      data: {
        providers: [makeProvider({ key: "opencode", displayName: "OpenCode", status: "unsupported", isAvailable: false, hasGlobalLevel: false, candidates: [] })],
      },
    });

    render(<SettingsScreen />);
    expect(screen.getByText("Unsupported")).not.toBeNull();
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
});
