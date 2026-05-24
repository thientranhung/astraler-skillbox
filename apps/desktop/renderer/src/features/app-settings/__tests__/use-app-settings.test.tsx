// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { getSettings: vi.fn() },
}));

import { useAppSettings } from "../use-app-settings.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockGetSettings = methods.getSettings as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useAppSettings", () => {
  it("returns data from methods.getSettings on success", async () => {
    const fakeSettings = {
      activeSkillHostFolderId: 1,
      defaultInstallMode: "symlink",
      databaseVersion: 3,
      activeHost: { hostId: 1, path: "/tmp/host", skillsPath: "/tmp/host/.agents/skills", status: "active", lastScannedAt: null },
    };
    mockGetSettings.mockResolvedValue(fakeSettings);

    const { result } = renderHook(() => useAppSettings(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(fakeSettings);
    expect(mockGetSettings).toHaveBeenCalledOnce();
  });

  it("exposes error when getSettings rejects", async () => {
    mockGetSettings.mockRejectedValue(new Error("db failure"));

    const { result } = renderHook(() => useAppSettings(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
