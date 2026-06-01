// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { resetAllData: vi.fn() },
}));

const mockNavigate = vi.fn();
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
}));

import { useResetAll } from "../use-reset-all.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockResetAllData = methods.resetAllData as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useResetAll", () => {
  it("clears query cache and navigates to /setup on success", async () => {
    mockResetAllData.mockResolvedValue({ restarting: true });

    const { result } = renderHook(() => useResetAll(), { wrapper: makeWrapper() });

    await act(async () => {
      await result.current.mutateAsync();
    });

    expect(mockResetAllData).toHaveBeenCalledOnce();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockNavigate).toHaveBeenCalledWith({ to: "/setup", replace: true });
  });

  it("exposes error when resetAllData rejects", async () => {
    mockResetAllData.mockRejectedValue(new Error("disk full"));

    const { result } = renderHook(() => useResetAll(), { wrapper: makeWrapper() });

    await act(async () => {
      try {
        await result.current.mutateAsync();
      } catch {
        // expected
      }
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
    expect(mockNavigate).not.toHaveBeenCalled();
  });
});
