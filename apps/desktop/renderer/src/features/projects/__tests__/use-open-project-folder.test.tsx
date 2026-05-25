// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { openPath: vi.fn() },
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));

import { useOpenProjectFolder } from "../use-open-project-folder.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockOpenPath = methods.openPath as ReturnType<typeof vi.fn>;

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

describe("useOpenProjectFolder", () => {
  it("calls methods.openPath with the project path", async () => {
    mockOpenPath.mockResolvedValue({ opened: true });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useOpenProjectFolder(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/home/user/myproject"); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockOpenPath).toHaveBeenCalledWith("/home/user/myproject");
  });

  it("surfaces mutation error on failure", async () => {
    mockOpenPath.mockRejectedValue(new Error("path not found"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useOpenProjectFolder(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/bad/path"); });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
