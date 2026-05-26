// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { resetProviderPaths: vi.fn() },
}));

import { useResetProviderPaths } from "../use-reset-provider-paths.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockResetProviderPaths = methods.resetProviderPaths as ReturnType<typeof vi.fn>;

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

describe("useResetProviderPaths", () => {
  it("calls methods.resetProviderPaths with the provided request", async () => {
    mockResetProviderPaths.mockResolvedValue({ reset: true });
    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useResetProviderPaths(), { wrapper: Wrapper });

    const req = { providerKey: "claude_code", scope: "project" as const, purpose: "skills" as const };
    await act(async () => { result.current.mutate(req); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockResetProviderPaths).toHaveBeenCalledWith(req);
  });

  it("invalidates providers.list on success", async () => {
    mockResetProviderPaths.mockResolvedValue({ reset: true });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useResetProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude_code", scope: "project", purpose: "skills" });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["providers", "list"] });
  });

  it("surfaces mutation error on failure", async () => {
    mockResetProviderPaths.mockRejectedValue(new Error("provider not found"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useResetProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "unknown", scope: "project", purpose: "skills" });
    });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
