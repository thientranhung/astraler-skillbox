// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { listProviderPlugins: vi.fn() },
}));

import { useProviderPluginList } from "../use-provider-plugin-list.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockListProviderPlugins = methods.listProviderPlugins as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return {
    Wrapper: function Wrapper({ children }: { children: React.ReactNode }) {
      return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    },
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useProviderPluginList", () => {
  it("fetches plugin list via methods.listProviderPlugins", async () => {
    const fakeData = {
      global: {
        providerKey: "claude",
        userLayerPath: "/Users/test/.claude/settings.json",
        userLayerStatus: null,
        lastScannedAt: null,
        scanWarnings: [],
        plugins: [],
        marketplaces: [],
        managedOutOfScope: false,
      },
      projects: [],
    };
    mockListProviderPlugins.mockResolvedValue(fakeData);

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useProviderPluginList(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(methods.listProviderPlugins).toHaveBeenCalled();
    expect(result.current.data).toEqual(fakeData);
  });

  it("exposes error state on failure", async () => {
    mockListProviderPlugins.mockRejectedValue(new Error("rpc error"));

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useProviderPluginList(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
