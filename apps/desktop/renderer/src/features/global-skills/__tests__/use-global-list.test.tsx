// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: {
    listGlobal: vi.fn(),
  },
}));

import { useGlobalList } from "../use-global-list.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockListGlobal = methods.listGlobal as ReturnType<typeof vi.fn>;

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

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useGlobalList", () => {
  it("fetches global locations via methods.listGlobal", async () => {
    const fakeData = { locations: [] };
    mockListGlobal.mockResolvedValue(fakeData);

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useGlobalList(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(methods.listGlobal).toHaveBeenCalled();
    expect(result.current.data).toEqual(fakeData);
  });
});
