// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { checkAppUpdate: vi.fn() },
}));

import { useCheckAppUpdate } from "../use-check-app-update.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockCheckAppUpdate = methods.checkAppUpdate as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useCheckAppUpdate", () => {
  it("starts idle", () => {
    mockCheckAppUpdate.mockResolvedValue({
      currentVersion: "0.1.0",
      latestVersion: null,
      updateAvailable: false,
      releaseUrl: null,
      error: null,
    });
    const { result } = renderHook(() => useCheckAppUpdate(), { wrapper: makeWrapper() });
    expect(result.current.status).toBe("idle");
    expect(result.current.isPending).toBe(false);
  });

  it("returns up-to-date when versions match", async () => {
    mockCheckAppUpdate.mockResolvedValue({
      currentVersion: "0.1.0",
      latestVersion: "0.1.0",
      updateAvailable: false,
      releaseUrl: "https://github.com/example/releases/tag/v0.1.0",
      error: null,
    });

    const { result } = renderHook(() => useCheckAppUpdate(), { wrapper: makeWrapper() });

    await act(async () => {
      result.current.check();
    });

    await waitFor(() => expect(result.current.status).toBe("up-to-date"));
    expect(result.current.updateAvailable).toBe(false);
    expect(result.current.currentVersion).toBe("0.1.0");
  });

  it("returns available when latestVersion differs", async () => {
    mockCheckAppUpdate.mockResolvedValue({
      currentVersion: "0.1.0",
      latestVersion: "1.2.3",
      updateAvailable: true,
      releaseUrl: "https://github.com/example/releases/tag/v1.2.3",
      error: null,
    });

    const { result } = renderHook(() => useCheckAppUpdate(), { wrapper: makeWrapper() });

    await act(async () => {
      result.current.check();
    });

    await waitFor(() => expect(result.current.status).toBe("available"));
    expect(result.current.latestVersion).toBe("1.2.3");
    expect(result.current.releaseUrl).toBe("https://github.com/example/releases/tag/v1.2.3");
  });

  it("returns disabled when network is off", async () => {
    mockCheckAppUpdate.mockResolvedValue({
      currentVersion: "0.1.0",
      latestVersion: null,
      updateAvailable: false,
      releaseUrl: null,
      error: "network_disabled",
    });

    const { result } = renderHook(() => useCheckAppUpdate(), { wrapper: makeWrapper() });

    await act(async () => {
      result.current.check();
    });

    await waitFor(() => expect(result.current.status).toBe("disabled"));
  });

  it("returns error on network failure", async () => {
    mockCheckAppUpdate.mockResolvedValue({
      currentVersion: "0.1.0",
      latestVersion: null,
      updateAvailable: false,
      releaseUrl: null,
      error: "network_error",
    });

    const { result } = renderHook(() => useCheckAppUpdate(), { wrapper: makeWrapper() });

    await act(async () => {
      result.current.check();
    });

    await waitFor(() => expect(result.current.status).toBe("error"));
  });
});
