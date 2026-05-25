// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { getDashboard: vi.fn() },
}));

import { useDashboard } from "../use-dashboard.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockGetDashboard = methods.getDashboard as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useDashboard", () => {
  it("returns data from methods.getDashboard on success", async () => {
    const fakeDashboard = {
      activeHost: {
        hostId: 1,
        path: "/tmp/host",
        skillsPath: "/tmp/host/.agents/skills",
        status: "active" as const,
        lastScanAt: null,
      },
      summary: { skills: 5, projects: 3, warnings: 1 },
      installsByMode: { symlink: 4, rsyncCopy: 1, direct: 0 },
      warningsBySeverity: { info: 1, warning: 0, error: 0, blocking: 0 },
      warnings: [
        {
          code: "skill_host_folder.missing_skills",
          message: "No skills found",
          severity: "info" as const,
          scopeType: "skill_host_folder" as const,
          scopeId: 1,
          actionKey: null,
        },
      ],
    };
    mockGetDashboard.mockResolvedValue(fakeDashboard);

    const { result } = renderHook(() => useDashboard(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(fakeDashboard);
    expect(mockGetDashboard).toHaveBeenCalledOnce();
  });

  it("exposes error when getDashboard rejects", async () => {
    mockGetDashboard.mockRejectedValue(new Error("db failure"));

    const { result } = renderHook(() => useDashboard(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
