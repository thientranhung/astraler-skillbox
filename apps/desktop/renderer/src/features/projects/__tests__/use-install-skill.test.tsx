// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { installSkill: vi.fn() },
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

import { useInstallSkill } from "../use-install-skill.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockInstallSkill = methods.installSkill as ReturnType<typeof vi.fn>;
const mockSubscribeOpProgress = subscribeOperationProgress as ReturnType<typeof vi.fn>;
const mockSubscribeAllProgress = subscribeAllProgress as ReturnType<typeof vi.fn>;

const REQ = { projectId: 5, providerKey: "generic_agents" as const, skillIds: [1, 2] as [number, ...number[]] };

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

describe("useInstallSkill — normal flow", () => {
  it("sets operationId after successful RPC call", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 42 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(42);
  });

  it("shows loading toast and subscribes to progress events", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 7 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.loading).toHaveBeenCalledWith("Installing skills…");
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(7, expect.any(Function));
  });

  it("invalidates projects.detail on terminal success event", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: 2, total: 2, message: null, metadata: { created: 2, requested: 2, failed: 0 } });
    });

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
    expect(toast.success).toHaveBeenCalledWith("Skills installed (2)", expect.objectContaining({ id: "mock-toast-id" }));
  });

  it("shows error toast and invalidates on terminal failed event", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "failed", phase: "done", processed: null, total: null, message: "disk error", metadata: { created: 1, requested: 2, failed: 1 } });
    });

    expect(toast.error).toHaveBeenCalledWith(
      expect.stringContaining("disk error"),
      expect.objectContaining({ id: "mock-toast-id" }),
    );
    expect(toast.error).toHaveBeenCalledWith(
      expect.stringContaining("1/2 installed"),
      expect.objectContaining({ id: "mock-toast-id" }),
    );
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
  });

  it("dismisses toast on cancelled event", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "cancelled", phase: "done", processed: null, total: null, message: null });
    });

    expect(toast.dismiss).toHaveBeenCalledWith("mock-toast-id");
    expect(result.current.operationId).toBeNull();
  });
});

describe("useInstallSkill — race condition: terminal event arrives before response", () => {
  it("shows success toast immediately when terminal event was buffered", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "success", phase: "done", processed: 2, total: 2, message: null });
      return vi.fn();
    });

    mockInstallSkill.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBeNull();
    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(toast.success).toHaveBeenCalledWith("Skills installed");
  });
});

describe("useInstallSkill — clearOperation", () => {
  it("clears operationId and unsubscribes", async () => {
    mockInstallSkill.mockResolvedValue({ operationId: 10 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useInstallSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(10));

    act(() => { result.current.clearOperation(); });

    expect(result.current.operationId).toBeNull();
  });
});
