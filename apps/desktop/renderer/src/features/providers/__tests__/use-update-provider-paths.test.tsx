// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { updateProviderPaths: vi.fn() },
}));

import { useUpdateProviderPaths } from "../use-update-provider-paths.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockUpdateProviderPaths = methods.updateProviderPaths as ReturnType<typeof vi.fn>;

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

describe("useUpdateProviderPaths", () => {
  it("calls methods.updateProviderPaths with the provided request", async () => {
    mockUpdateProviderPaths.mockResolvedValue({ updated: true });
    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    const req = { providerKey: "claude_code", scope: "project" as const, purpose: "skills" as const, paths: [".agents/skills"] as [string, ...string[]] };
    await act(async () => { result.current.mutate(req); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockUpdateProviderPaths).toHaveBeenCalledWith(req);
  });

  it("invalidates providers.list on success", async () => {
    mockUpdateProviderPaths.mockResolvedValue({ updated: true });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude_code", scope: "project", purpose: "skills", paths: [".agents/skills"] });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["providers", "list"] });
  });

  it("surfaces mutation error on failure", async () => {
    mockUpdateProviderPaths.mockRejectedValue(new Error("invalid path"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude_code", scope: "project", purpose: "skills", paths: ["../bad"] });
    });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
