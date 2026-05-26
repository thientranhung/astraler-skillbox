// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { scanProviderPluginsProject: vi.fn() },
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

import { useScanProviderPluginsProject } from "../use-scan-provider-plugins-project.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockScan = methods.scanProviderPluginsProject as ReturnType<typeof vi.fn>;
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

describe("useScanProviderPluginsProject", () => {
  it("sets operationId after successful RPC call", async () => {
    mockScan.mockResolvedValue({ operationId: 42 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanProviderPluginsProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(5); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(42);
    expect(mockScan).toHaveBeenCalledWith({ projectId: 5 });
  });

  it("shows loading toast and subscribes to progress", async () => {
    mockScan.mockResolvedValue({ operationId: 7 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useScanProviderPluginsProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.loading).toHaveBeenCalledWith("Scanning project plugin settings…");
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(7, expect.any(Function));
  });

  it("invalidates providerPlugins.list and projects.detail on terminal success", async () => {
    mockScan.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id: number, cb: (e: OperationProgressNotification) => void) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useScanProviderPluginsProject(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(3); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: 1, total: 1, message: null });
    });

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["providerPlugins", "list"] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 3] });
    expect(result.current.operationId).toBeNull();
    expect(toast.success).toHaveBeenCalledWith("Project plugin settings scanned", expect.objectContaining({ id: "mock-toast-id" }));
  });
});
