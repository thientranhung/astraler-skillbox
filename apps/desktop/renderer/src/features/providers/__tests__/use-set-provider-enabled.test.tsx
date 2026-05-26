// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { setProviderEnabled: vi.fn() },
}));

import { useSetProviderEnabled } from "../use-set-provider-enabled.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockSetProviderEnabled = methods.setProviderEnabled as ReturnType<typeof vi.fn>;

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

describe("useSetProviderEnabled", () => {
  it("calls methods.setProviderEnabled with the provided request", async () => {
    mockSetProviderEnabled.mockResolvedValue({ updated: true });
    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useSetProviderEnabled(), { wrapper: Wrapper });

    const req = { providerKey: "claude", enabled: false };
    await act(async () => { result.current.mutate(req); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockSetProviderEnabled).toHaveBeenCalledWith(req);
  });

  it("invalidates providers.list on success", async () => {
    mockSetProviderEnabled.mockResolvedValue({ updated: true });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useSetProviderEnabled(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "generic_agents", enabled: true });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["providers", "list"] });
  });

  it("surfaces mutation error on failure", async () => {
    mockSetProviderEnabled.mockRejectedValue(new Error("validation_error: provider cannot be enabled"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useSetProviderEnabled(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "opencode", enabled: true });
    });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
