// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { runUpdateCheck: vi.fn() },
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}));

import { useRunUpdateCheck } from "../use-run-update-check.js";
import { methods } from "../../../lib/core-client/methods.js";
import { toast } from "sonner";

const mockRunUpdateCheck = methods.runUpdateCheck as ReturnType<typeof vi.fn>;
const mockToast = toast as unknown as {
  success: ReturnType<typeof vi.fn>;
  error: ReturnType<typeof vi.fn>;
  warning: ReturnType<typeof vi.fn>;
  info: ReturnType<typeof vi.fn>;
};

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useRunUpdateCheck", () => {
  it("starts idle with empty results", () => {
    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });
    expect(result.current.status).toBe("idle");
    expect(result.current.results).toHaveLength(0);
    expect(result.current.isRunning).toBe(false);
  });

  it("shows all_failed status and error toast when all plugins have errors", async () => {
    mockRunUpdateCheck.mockResolvedValue({
      status: "ok",
      plugins: [
        { pluginName: "p1", marketplaceName: "m", updateAvailable: null, error: "timeout" },
        { pluginName: "p2", marketplaceName: "m", updateAvailable: null, error: "git_ls_remote_failed" },
      ],
    });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("all_failed"));

    expect(mockToast.error).toHaveBeenCalledWith(expect.stringMatching(/failed for all 2/));
    expect(mockToast.success).not.toHaveBeenCalled();
  });

  it("shows ok status and success toast when all plugins are up to date", async () => {
    mockRunUpdateCheck.mockResolvedValue({
      status: "ok",
      plugins: [
        { pluginName: "p1", marketplaceName: "m", updateAvailable: false, error: "" },
      ],
    });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("ok"));

    expect(mockToast.success).toHaveBeenCalledWith("All plugins up to date");
  });

  it("shows ok status and update count toast when updates are available", async () => {
    mockRunUpdateCheck.mockResolvedValue({
      status: "ok",
      plugins: [
        { pluginName: "p1", marketplaceName: "m", updateAvailable: true, error: "" },
        { pluginName: "p2", marketplaceName: "m", updateAvailable: false, error: "" },
      ],
    });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("ok"));

    expect(mockToast.success).toHaveBeenCalledWith("1 update available");
  });

  it("shows partial_error status and warning toast when some plugins fail", async () => {
    mockRunUpdateCheck.mockResolvedValue({
      status: "ok",
      plugins: [
        { pluginName: "p1", marketplaceName: "m", updateAvailable: false, error: "" },
        { pluginName: "p2", marketplaceName: "m", updateAvailable: null, error: "timeout" },
      ],
    });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("partial_error"));

    expect(mockToast.warning).toHaveBeenCalledWith(
      expect.stringMatching(/1 plugin could not be checked/),
    );
  });

  it("shows git_not_found status when git is unavailable", async () => {
    mockRunUpdateCheck.mockResolvedValue({ status: "git_not_found", plugins: [] });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("git_not_found"));

    expect(mockToast.error).toHaveBeenCalledWith(expect.stringMatching(/git is required/));
  });

  it("zero-checkable: shows info toast instead of misleading 'All plugins up to date'", async () => {
    // FB-002 edge case: no plugins with update sources -> plugins=[] from Go.
    mockRunUpdateCheck.mockResolvedValue({ status: "ok", plugins: [] });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("ok"));

    expect(mockToast.info).toHaveBeenCalledWith(
      expect.stringMatching(/No plugins to check/),
    );
    expect(mockToast.success).not.toHaveBeenCalled();
  });

  it("shows error status on RPC failure", async () => {
    mockRunUpdateCheck.mockRejectedValue(new Error("core_unavailable"));

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("error"));

    expect(mockToast.error).toHaveBeenCalledWith("Update check failed");
  });

  it("rate-limit prevents re-trigger within 10s", async () => {
    mockRunUpdateCheck.mockResolvedValue({ status: "ok", plugins: [] });

    const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

    await act(async () => { result.current.run(); });
    await waitFor(() => expect(result.current.status).toBe("ok"));

    expect(result.current.isRateLimited()).toBe(true);
    const callsBefore = mockRunUpdateCheck.mock.calls.length;

    await act(async () => { result.current.run(); });
    // should not make another call
    expect(mockRunUpdateCheck.mock.calls.length).toBe(callsBefore);
    expect(result.current.status).toBe("ok");
  });

  it("re-enables in place after RATE_LIMIT_MS following no-update-sources terminal response (TC-PLUGIN-005 regression)", async () => {
    // Regression guard: the Check Updates button must re-enable after the rate-limit window
    // without the user navigating away. Previously isRateLimited was ref-based and never
    // triggered a re-render after the timer expired, leaving the button permanently disabled.
    //
    // We spy on the 10 000ms setTimeout to capture its callback so we can fire it
    // synchronously inside act(), avoiding fake-timer / waitFor polling conflicts.
    let capturedExpiryCb: (() => void) | null = null;
    const origSetTimeout = globalThis.setTimeout;
    vi.spyOn(globalThis, "setTimeout").mockImplementation((fn: any, ms?: number, ...rest: any[]) => {
      if (ms === 10_000 && typeof fn === "function") {
        capturedExpiryCb = () => fn(...rest);
        return 999_999 as unknown as ReturnType<typeof setTimeout>;
      }
      return origSetTimeout(fn, ms, ...rest) as ReturnType<typeof setTimeout>;
    });

    try {
      mockRunUpdateCheck.mockResolvedValue({ status: "ok", plugins: [] });
      const { result } = renderHook(() => useRunUpdateCheck(), { wrapper: makeWrapper() });

      await act(async () => { result.current.run(); });
      await waitFor(() => expect(result.current.status).toBe("ok"));

      // After terminal response, button must be rate-limited (disabled).
      expect(result.current.isRateLimited()).toBe(true);
      expect(capturedExpiryCb).not.toBeNull();

      // Simulate timer expiry - triggers setRateLimited(false), re-render, and button re-enable.
      await act(async () => { capturedExpiryCb!(); });

      // Rate limit expired - button re-enables in place, no navigation needed.
      expect(result.current.isRateLimited()).toBe(false);
    } finally {
      vi.restoreAllMocks();
    }
  });
});
