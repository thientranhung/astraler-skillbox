// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { addProject: vi.fn() },
}));

import { useAddProject } from "../use-add-project.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockAddProject = methods.addProject as ReturnType<typeof vi.fn>;

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

describe("useAddProject", () => {
  it("calls methods.addProject with the provided path", async () => {
    mockAddProject.mockResolvedValue({ projectId: 1, name: "myproject", path: "/home/user/myproject", status: "active" });
    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useAddProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/home/user/myproject"); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockAddProject).toHaveBeenCalledWith({ path: "/home/user/myproject" });
  });

  it("invalidates projects.list on success", async () => {
    mockAddProject.mockResolvedValue({ projectId: 2, name: "proj", path: "/proj", status: "active" });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useAddProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/proj"); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
  });

  it("surfaces mutation error on failure", async () => {
    mockAddProject.mockRejectedValue(new Error("path not found"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useAddProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/bad/path"); });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
