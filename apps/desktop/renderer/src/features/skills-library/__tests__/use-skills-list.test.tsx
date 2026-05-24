// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: {
    getSettings: vi.fn(),
    listSkills: vi.fn(),
  },
}));

import { useSkillsList } from "../use-skills-list.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockGetSettings = methods.getSettings as ReturnType<typeof vi.fn>;
const mockListSkills = methods.listSkills as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useSkillsList", () => {
  it("does not call listSkills when no active host", async () => {
    mockGetSettings.mockResolvedValue({
      activeSkillHostFolderId: null,
      defaultInstallMode: "symlink",
      databaseVersion: 1,
      activeHost: null,
    });

    const { result } = renderHook(() => useSkillsList(), { wrapper: makeWrapper() });

    // Wait briefly; query should stay pending/idle since enabled=false
    await waitFor(() => expect(result.current.isPending).toBe(true));
    expect(mockListSkills).not.toHaveBeenCalled();
  });

  it("calls listSkills with the active host's hostId", async () => {
    mockGetSettings.mockResolvedValue({
      activeSkillHostFolderId: 3,
      defaultInstallMode: "symlink",
      databaseVersion: 1,
      activeHost: { hostId: 3, path: "/h", skillsPath: "/h/.agents/skills", status: "active", lastScannedAt: null },
    });

    const fakeList = {
      hostPath: "/h",
      skills: [],
      totals: { available: 0, missing: 0, unreadable: 0, local_modified: 0, unknown: 0 },
      lastScanAt: null,
      warnings: [],
    };
    mockListSkills.mockResolvedValue(fakeList);

    const { result } = renderHook(() => useSkillsList(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockListSkills).toHaveBeenCalledWith({ hostId: 3 });
    expect(result.current.data).toEqual(fakeList);
  });
});
