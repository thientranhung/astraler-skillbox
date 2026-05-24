// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { chooseHost: vi.fn(), getSettings: vi.fn() },
}));

import { useChooseHost } from "../use-choose-host.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockChooseHost = methods.chooseHost as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return {
    client,
    Wrapper: function Wrapper({ children }: { children: React.ReactNode }) {
      return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    },
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useChooseHost", () => {
  it("calls methods.chooseHost with the provided path", async () => {
    mockChooseHost.mockResolvedValue({ hostId: 5 });
    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useChooseHost(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate("/some/path");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockChooseHost).toHaveBeenCalledWith({ path: "/some/path" });
  });

  it("invalidates settings.app and skills.list on success", async () => {
    mockChooseHost.mockResolvedValue({ hostId: 7 });
    const { client, Wrapper } = makeWrapper();
    const invalidateSpy = vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useChooseHost(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate("/tmp/host");
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const keys = invalidateSpy.mock.calls.map((c) => JSON.stringify(c[0]));
    expect(keys).toContain(JSON.stringify({ queryKey: ["settings", "app"] }));
    expect(keys).toContain(JSON.stringify({ queryKey: ["skills", "list", 7] }));
  });
});
