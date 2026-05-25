// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { listProjects: vi.fn() },
}));

import { useProjectsList } from "../use-projects-list.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockListProjects = methods.listProjects as ReturnType<typeof vi.fn>;

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

describe("useProjectsList", () => {
  it("calls project.list and returns data", async () => {
    const fakeProjects = { projects: [{ id: 1, name: "alpha", path: "/alpha", status: "active", providers: [], skillCount: 0, warningCount: 0, lastScannedAt: null }] };
    mockListProjects.mockResolvedValue(fakeProjects);

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useProjectsList(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockListProjects).toHaveBeenCalledOnce();
    expect(result.current.data).toEqual(fakeProjects);
  });

  it("uses queryKey projects.list", async () => {
    mockListProjects.mockResolvedValue({ projects: [] });
    const { client, Wrapper } = makeWrapper();

    renderHook(() => useProjectsList(), { wrapper: Wrapper });
    await waitFor(() => expect(mockListProjects).toHaveBeenCalled());

    const cached = client.getQueryData(["projects", "list"]);
    expect(cached).toBeDefined();
  });
});
