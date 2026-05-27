// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { removeProviderPluginOverride: vi.fn() },
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

import { useRemoveProviderPluginOverride } from "../use-remove-provider-plugin-override.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockRemove = methods.removeProviderPluginOverride as ReturnType<typeof vi.fn>;
const mockSubscribeOpProgress = subscribeOperationProgress as ReturnType<typeof vi.fn>;
const mockSubscribeAllProgress = subscribeAllProgress as ReturnType<typeof vi.fn>;

const reqFixture = {
  providerKey: "claude",
  pluginName: "my-plugin",
  marketplaceName: "npm",
  layer: "project" as const,
  projectId: 7,
};

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

describe("useRemoveProviderPluginOverride", () => {
  it("sets operationId after successful RPC call", async () => {
    mockRemove.mockResolvedValue({ operationId: 42 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useRemoveProviderPluginOverride(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(reqFixture); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBe(42);
  });

  it("shows loading toast and subscribes to progress", async () => {
    mockRemove.mockResolvedValue({ operationId: 7 });
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useRemoveProviderPluginOverride(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(reqFixture); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.loading).toHaveBeenCalledWith("Removing plugin override…");
    expect(mockSubscribeOpProgress).toHaveBeenCalledWith(7, expect.any(Function));
  });

  it("invalidates providerPlugins.list and project detail on terminal success event", async () => {
    mockRemove.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id: number, cb: (e: OperationProgressNotification) => void) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveProviderPluginOverride(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(reqFixture); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: 1, total: 1, message: null });
    });

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["providerPlugins", "list"] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: expect.arrayContaining(["projects"]) });
    expect(result.current.operationId).toBeNull();
    expect(toast.success).toHaveBeenCalledWith("Plugin override removed", expect.objectContaining({ id: "mock-toast-id" }));
  });

  it("does not enter operation state when terminal event was buffered before subscribe", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "success", phase: "done", processed: 1, total: 1, message: null });
      return vi.fn();
    });
    mockRemove.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveProviderPluginOverride(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(reqFixture); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.operationId).toBeNull();
    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["providerPlugins", "list"] });
    expect(toast.success).toHaveBeenCalledWith("Plugin override removed");
  });

  it("shows error toast when mutation RPC call fails", async () => {
    mockRemove.mockRejectedValue(new Error("rpc failed"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useRemoveProviderPluginOverride(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(reqFixture); });
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("rpc failed"));
  });
});
