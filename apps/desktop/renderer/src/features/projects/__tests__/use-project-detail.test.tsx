// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { getProject: vi.fn() },
}));

import { useProjectDetail } from "../use-project-detail.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockGetProject = methods.getProject as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return {
    client,
    Wrapper: function Wrapper({ children }: { children: React.ReactNode }) {
      return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    },
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useProjectDetail", () => {
  it("fetches project detail when projectId is provided", async () => {
    const fakeDetail = { project: { id: 5, name: "beta", path: "/beta", status: "active", lastScannedAt: null }, providers: [], entries: [], warnings: [] };
    mockGetProject.mockResolvedValue(fakeDetail);

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useProjectDetail(5), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetProject).toHaveBeenCalledWith({ projectId: 5 });
    expect(result.current.data).toEqual(fakeDetail);
  });

  it("is disabled when projectId is null", () => {
    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useProjectDetail(null), { wrapper: Wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockGetProject).not.toHaveBeenCalled();
  });

  it("uses queryKey projects.detail(projectId)", async () => {
    const fakeDetail = { project: { id: 3, name: "gamma", path: "/gamma", status: "active", lastScannedAt: null }, providers: [], entries: [], warnings: [] };
    mockGetProject.mockResolvedValue(fakeDetail);

    const { client, Wrapper } = makeWrapper();
    renderHook(() => useProjectDetail(3), { wrapper: Wrapper });

    await waitFor(() => expect(mockGetProject).toHaveBeenCalled());
    const cached = client.getQueryData(["projects", "detail", 3]);
    expect(cached).toBeDefined();
  });
});
