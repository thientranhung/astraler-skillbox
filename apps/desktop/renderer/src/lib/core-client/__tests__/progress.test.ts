import { describe, it, expect, vi, beforeEach } from "vitest";
import { subscribeOperationProgress } from "../progress.js";

beforeEach(() => {
  (globalThis as Record<string, unknown>).window = globalThis;
  (window as Window & typeof globalThis & { core: Window["core"] }).core = {
    invoke: vi.fn(),
    onEvent: vi.fn().mockReturnValue(() => {}),
  };
});

describe("subscribeOperationProgress", () => {
  it("subscribes to operation.progress event", () => {
    const onProgress = vi.fn();
    subscribeOperationProgress(42, onProgress);
    expect(window.core.onEvent).toHaveBeenCalledWith(
      "operation.progress",
      expect.any(Function)
    );
  });

  it("fires callback for matching operationId", () => {
    let capturedCb: ((params: unknown) => void) | null = null;
    (window.core.onEvent as ReturnType<typeof vi.fn>).mockImplementation(
      (_event: string, cb: (p: unknown) => void) => {
        capturedCb = cb;
        return () => {};
      }
    );

    const onProgress = vi.fn();
    subscribeOperationProgress(42, onProgress);

    capturedCb!({
      operationId: 42,
      status: "running",
      phase: "reading_host_folder",
      processed: 0,
      total: null,
      message: null,
    });
    expect(onProgress).toHaveBeenCalledTimes(1);
  });

  it("does NOT fire callback for different operationId", () => {
    let capturedCb: ((params: unknown) => void) | null = null;
    (window.core.onEvent as ReturnType<typeof vi.fn>).mockImplementation(
      (_event: string, cb: (p: unknown) => void) => {
        capturedCb = cb;
        return () => {};
      }
    );

    const onProgress = vi.fn();
    subscribeOperationProgress(42, onProgress);

    capturedCb!({
      operationId: 99,
      status: "running",
      phase: "test",
      processed: 1,
      total: 10,
      message: null,
    });
    expect(onProgress).not.toHaveBeenCalled();
  });

  it("returns the unsub function from onEvent", () => {
    const mockUnsub = vi.fn();
    (window.core.onEvent as ReturnType<typeof vi.fn>).mockReturnValue(mockUnsub);

    const unsub = subscribeOperationProgress(42, vi.fn());
    unsub();
    expect(mockUnsub).toHaveBeenCalledTimes(1);
  });
});
