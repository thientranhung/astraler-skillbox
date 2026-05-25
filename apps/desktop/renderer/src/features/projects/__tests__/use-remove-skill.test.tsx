// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import type { OperationProgressNotification } from "@contracts/index.js";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { removeSkill: vi.fn() },
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

import { useRemoveSkill } from "../use-remove-skill.js";
import { methods } from "../../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../../lib/core-client/progress.js";
import { toast } from "sonner";

const mockRemoveSkill = methods.removeSkill as ReturnType<typeof vi.fn>;
const mockSubscribeOpProgress = subscribeOperationProgress as ReturnType<typeof vi.fn>;
const mockSubscribeAllProgress = subscribeAllProgress as ReturnType<typeof vi.fn>;

const REQ = { projectId: 5, installId: 88 };

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

describe("useRemoveSkill — subscribed terminal path", () => {
  it("invalidates detail + list and shows success toast on terminal success", async () => {
    mockRemoveSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "success", phase: "done", processed: null, total: null, message: null, metadata: { skillName: "documentation-writer", providerKey: "generic_agents", alreadyAbsent: false } });
    });

    expect(toast.success).toHaveBeenCalledWith("Removed documentation-writer", expect.objectContaining({ id: "mock-toast-id" }));
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
  });

  it("shows error toast and invalidates on terminal failed event", async () => {
    mockRemoveSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "failed", phase: "done", processed: null, total: null, message: "permission denied", metadata: null });
    });

    expect(toast.error).toHaveBeenCalledWith(
      expect.stringContaining("permission denied"),
      expect.objectContaining({ id: "mock-toast-id" }),
    );
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
    expect(result.current.operationId).toBeNull();
  });

  it("does NOT invalidate on intermediate running/queued events", async () => {
    mockRemoveSkill.mockResolvedValue({ operationId: 7 });
    let progressCb: ((e: OperationProgressNotification) => void) | null = null;
    mockSubscribeOpProgress.mockImplementation((_id, cb) => {
      progressCb = cb;
      return vi.fn();
    });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.operationId).toBe(7));

    await act(async () => {
      progressCb!({ operationId: 7, status: "queued", phase: "validating", processed: null, total: null, message: null });
      progressCb!({ operationId: 7, status: "running", phase: "removing_symlink", processed: null, total: null, message: "removing_symlink" });
    });

    expect(client.invalidateQueries).not.toHaveBeenCalled();
    expect(result.current.operationId).toBe(7);
  });
});

describe("useRemoveSkill — buffered terminal path (event arrives before response)", () => {
  it("shows success toast immediately when terminal success was buffered", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "success", phase: "done", processed: null, total: null, message: null, metadata: { skillName: "adr-helper", providerKey: "generic_agents", alreadyAbsent: true } });
      return vi.fn();
    });
    mockRemoveSkill.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalledWith("Removed adr-helper");
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
  });

  it("shows error toast immediately when terminal failure was buffered", async () => {
    mockSubscribeAllProgress.mockImplementation((cb: (p: OperationProgressNotification) => void) => {
      cb({ operationId: 99, status: "failed", phase: "done", processed: null, total: null, message: "entry changed on disk", metadata: null });
      return vi.fn();
    });
    mockRemoveSkill.mockResolvedValue({ operationId: 99 });

    const { client, Wrapper } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRemoveSkill(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate(REQ); });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockSubscribeOpProgress).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining("entry changed on disk"));
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "detail", 5] });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["projects", "list"] });
  });
});
