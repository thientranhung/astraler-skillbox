// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { openTerminal: vi.fn() },
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn() } }));

import { useOpenProjectTerminal } from "../use-open-project-terminal.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockOpenTerminal = methods.openTerminal as ReturnType<typeof vi.fn>;

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

describe("useOpenProjectTerminal", () => {
  it("calls methods.openTerminal with the project path", async () => {
    mockOpenTerminal.mockResolvedValue({ opened: true });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useOpenProjectTerminal(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/home/user/myproject"); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockOpenTerminal).toHaveBeenCalledWith("/home/user/myproject");
  });

  it("surfaces mutation error on failure", async () => {
    mockOpenTerminal.mockRejectedValue(new Error("terminal failed"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useOpenProjectTerminal(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate("/bad/path"); });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toBeInstanceOf(Error);
  });
});
