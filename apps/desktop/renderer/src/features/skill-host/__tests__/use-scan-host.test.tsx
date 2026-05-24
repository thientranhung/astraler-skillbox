// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { scanHost: vi.fn(), getSettings: vi.fn() },
}));

vi.mock("../../../lib/core-client/progress.js", () => ({
  subscribeOperationProgress: vi.fn(),
  subscribeAllProgress: vi.fn(),
}));

vi.mock("sonner", () => ({
  toast: {
    loading: vi.fn().mockReturnValue("mock-toast-id"),
    success: vi.fn(),
    error: vi.fn(),
    dismiss: vi.fn(),
  },
}));

import { useScanHost } from "../use-scan-host.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockScanHost = methods.scanHost as ReturnType<typeof vi.fn>;
const mockSubscribeOpProgress = subscribeOperationProgress as ReturnType<typeof vi.fn>;
const mockSubscribeAllProgress = subscribeAllProgress as ReturnType<typeof vi.fn>;

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

beforeEach(() => {
  vi.clearAllMocks();
  // Default: no events buffered during the call
  mockSubscribeAllProgress.mockReturnValue(vi.fn());
  // Default: returns a no-op unsub
  mockSubscribeOpProgress.mockReturnValue(vi.fn());
});

describe("useScanHost — normal flow", () => {
  it("sets operationId after successful scan RPC call", async () => {
    mockScanHost.mockResolvedValue({ operationId: 42 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(42);
  });

  it("subscribes to progress events and shows loading toast", async () => {
    mockScanHost.mockResolvedValue({ operationId: 7 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.loading).toHaveBeenCalledWith("Scanning skills…");
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(7, expect.any(Function));
  });

  it("invalidates skills.list and clears operationId on terminal success event", async () => {
    mockScanHost.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    expect(progressCb).not.toBeNull();

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: 3, total: 3, message: null });
    });

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["skills", "list", 3] });
    expect(result.current.operationId).toBeNull();
    expect(toast.success).toHaveBeenCalledWith("Skills scanned", expect.objectContaining({ id: "mock-toast-id" }));
  });

  it("shows error toast and invalidates on terminal failed event", async () => {
    mockScanHost.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "failed", phase: "done", processed: null, total: null, message: "disk error" });
    });

    expect(toast.error).toHaveBeenCalledWith(
      expect.stringContaining("disk error"),
      expect.objectContaining({ id: "mock-toast-id" }),
    );
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["skills", "list", 3] });
    expect(result.current.operationId).toBeNull();
  });
});

describe("useScanHost — race condition: terminal event arrives before response", () => {
  it("does not enter scanning state when terminal event was buffered", async () => {
    // Simulates fast scan: the terminal event arrives during the RPC round-trip.
    // subscribeAllProgress fires the callback synchronously (before scanHost resolves).
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "success", phase: "done", processed: 2, total: 2, message: null });
      return vi.fn(); // unsub
    });

    mockScanHost.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // operationId must never be set — scan was already done
    expect(result.current.operationId).toBeNull();
    // subscribeOperationProgress must NOT be called — no ongoing scan to track
    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    // invalidation must still happen
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["skills", "list", 5] });
    // success toast must still show
    expect(toast.success).toHaveBeenCalledWith("Skills scanned");
  });

  it("ignores buffered events for a different operationId", async () => {
    // Stale event from a prior scan (different opId) should not affect the new scan.
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 0, status: "success", phase: "done", processed: 1, total: 1, message: null });
      return vi.fn();
    });

    mockScanHost.mockResolvedValue({ operationId: 50 });

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useScanHost(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // Own operationId is set — the stale buffered event was ignored
    expect(result.current.operationId).toBe(50);
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(50, expect.any(Function));
  });
});
