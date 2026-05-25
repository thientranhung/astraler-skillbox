// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { removeProject: vi.fn() },
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));

import { useRemoveProject } from "../use-remove-project.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockRemoveProject = methods.removeProject as ReturnType<typeof vi.fn>;

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

describe("useRemoveProject", () => {
  it("calls methods.removeProject with projectId", async () => {
    mockRemoveProject.mockResolvedValue({ removed: true });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useRemoveProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockRemoveProject).toHaveBeenCalledWith({ projectId: 5 });
  });

  it("invalidates projects.list on success", async () => {
    mockRemoveProject.mockResolvedValue({ removed: true });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
  });

  it("surfaces mutation error on failure", async () => {
    mockRemoveProject.mockRejectedValue(new Error("not found"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useRemoveProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(99); });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
