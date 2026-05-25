// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { scanProject: vi.fn() },
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

import { useScanProject } from "../use-scan-project.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockScanProject = methods.scanProject as ReturnType<typeof vi.fn>;
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
  mockSubscribeAllProgress.mockReturnValue(vi.fn());
  mockSubscribeOpProgress.mockReturnValue(vi.fn());
});

describe("useScanProject — normal flow", () => {
  it("sets operationId after successful scan RPC call", async () => {
    mockScanProject.mockResolvedValue({ operationId: 42 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(42);
  });

  it("subscribes to progress events and shows loading toast", async () => {
    mockScanProject.mockResolvedValue({ operationId: 7 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.loading).toHaveBeenCalledWith("Scanning project…");
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(7, expect.any(Function));
  });

  it("invalidates projects.detail and projects.list on terminal success event", async () => {
    mockScanProject.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    expect(progressCb).not.toBeNull();

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: 1, total: 1, message: null });
    });

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 3] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
    expect(toast.success).toHaveBeenCalledWith("Project scanned", expect.objectContaining({ id: "mock-toast-id" }));
  });

  it("shows error toast with message and invalidates on terminal failed event", async () => {
    mockScanProject.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "failed", phase: "done", processed: null, total: null, message: "disk error" });
    });

    expect(toast.error).toHaveBeenCalledWith(
      expect.stringContaining("disk error"),
      expect.objectContaining({ id: "mock-toast-id" }),
    );
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 3] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
  });

  it("dismisses toast and invalidates on cancelled event", async () => {
    mockScanProject.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "cancelled", phase: "done", processed: null, total: null, message: null });
    });

    expect(toast.dismiss).toHaveBeenCalledWith("mock-toast-id");
    expect(result.current.operationId).toBeNull();
  });
});

describe("useScanProject — race condition: terminal event arrives before response", () => {
  it("does not enter scanning state when terminal event was buffered", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "success", phase: "done", processed: 1, total: 1, message: null });
      return vi.fn();
    });

    mockScanProject.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBeNull();
    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(toast.success).toHaveBeenCalledWith("Project scanned");
  });

  it("ignores buffered events for a different operationId", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 0, status: "success", phase: "done", processed: 1, total: 1, message: null });
      return vi.fn();
    });

    mockScanProject.mockResolvedValue({ operationId: 50 });

    const { Wrapper } = makeWrapper();
    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(50);
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(50, expect.any(Function));
  });

  it("shows failed toast immediately when failed event is buffered", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 77, status: "failed", phase: "done", processed: null, total: null, message: "io error" });
      return vi.fn();
    });

    mockScanProject.mockResolvedValue({ operationId: 77 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(8); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBeNull();
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("io error"));
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 8] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
  });
});

describe("useScanProject — clearOperation", () => {
  it("clears operationId and unsubscribes", async () => {
    mockScanProject.mockResolvedValue({ operationId: 10 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(1); });
    await waitFor(() => expect(result.current.operationId).toBe(10));

    act(() => { result.current.clearOperation(); });

    expect(result.current.operationId).toBeNull();
  });
});
