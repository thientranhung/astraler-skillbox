import { describe, it, expect, vi, beforeEach } from "vitest";
import { invoke, AppClientError } from "../client.js";

// Reset window.core before each test
beforeEach(() => {
  (globalThis as Record<string, unknown>).window = globalThis;
});

describe("AppClientError", () => {
  it("is an instance of Error", () => {
    const err = new AppClientError("validation_error", "Bad path", "path not found");
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(AppClientError);
  });

  it("exposes code, userMessage, technicalMessage", () => {
    const err = new AppClientError("database_error", "DB unavailable", "connection refused", 1004);
    expect(err.code).toBe("database_error");
    expect(err.userMessage).toBe("DB unavailable");
    expect(err.technicalMessage).toBe("connection refused");
    expect(err.rpcCode).toBe(1004);
  });

  it("message equals userMessage", () => {
    const err = new AppClientError("client_error", "Something failed", "detail");
    expect(err.message).toBe("Something failed");
  });
});

describe("invoke", () => {
  it("throws AppClientError when window.core is missing", async () => {
    // @ts-ignore
    delete window.core;
    await expect(invoke("ping", {})).rejects.toBeInstanceOf(AppClientError);
  });

  it("returns result on success", async () => {
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi.fn().mockResolvedValue({ pong: true, ts: "2026-01-01T00:00:00Z" }),
      onEvent: vi.fn(),
    };
    const result = await invoke<{ pong: boolean }>("ping", {});
    expect(result.pong).toBe(true);
  });

  it("maps JSON-RPC error data to AppClientError with correct fields", async () => {
    const rpcError = {
      code: 1001,
      message: "Host not found",
      data: {
        code: "validation_error",
        rpcCode: 1001,
        userMessage: "Host not found",
        technicalMessage: "path does not exist",
      },
    };
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi.fn().mockRejectedValue(new Error(JSON.stringify(rpcError))),
      onEvent: vi.fn(),
    };

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err: any = await invoke("host.choose", { path: "/bad" }).catch((e) => e);
    expect(err).toBeInstanceOf(AppClientError);
    expect(err.code).toBe("validation_error");
    expect(err.userMessage).toBe("Host not found");
    expect(err.technicalMessage).toBe("path does not exist");
    expect(err.rpcCode).toBe(1001);
  });

  it("maps Electron-prefixed JSON-RPC errors to AppClientError", async () => {
    const rpcError = {
      code: 1001,
      message: "Invalid host folder path",
      data: {
        code: "validation_error",
        rpcCode: 1001,
        userMessage: "Invalid host folder path",
        technicalMessage: "not_a_directory: path is not a directory",
      },
    };
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi
        .fn()
        .mockRejectedValue(
          new Error(`Error invoking remote method 'core:invoke': Error: ${JSON.stringify(rpcError)}`),
        ),
      onEvent: vi.fn(),
    };

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err: any = await invoke("host.choose", { path: "/etc/hosts" }).catch((e) => e);
    expect(err).toBeInstanceOf(AppClientError);
    expect(err.code).toBe("validation_error");
    expect(err.userMessage).toBe("Invalid host folder path");
    expect(err.technicalMessage).toBe("not_a_directory: path is not a directory");
    expect(err.rpcCode).toBe(1001);
  });

  it("maps structured preload errors to AppClientError", async () => {
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi.fn().mockRejectedValue({
        code: "validation_error",
        rpcCode: 1001,
        userMessage: "Invalid host folder path",
        technicalMessage: "not_a_directory: path is not a directory",
      }),
      onEvent: vi.fn(),
    };

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err: any = await invoke("host.choose", { path: "/etc/hosts" }).catch((e) => e);
    expect(err).toBeInstanceOf(AppClientError);
    expect(err.code).toBe("validation_error");
    expect(err.userMessage).toBe("Invalid host folder path");
    expect(err.technicalMessage).toBe("not_a_directory: path is not a directory");
    expect(err.rpcCode).toBe(1001);
  });

  it("falls back to generic AppClientError for non-JSON errors", async () => {
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi.fn().mockRejectedValue(new Error("connection refused")),
      onEvent: vi.fn(),
    };
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err: any = await invoke("ping", {}).catch((e) => e);
    expect(err).toBeInstanceOf(AppClientError);
    expect(err.code).toBe("client_error");
  });

  it("falls back for RPC error without data field", async () => {
    const rpcError = { code: -32601, message: "method not found" };
    (window as Window & typeof globalThis & { core: Window["core"] }).core = {
      invoke: vi.fn().mockRejectedValue(new Error(JSON.stringify(rpcError))),
      onEvent: vi.fn(),
    };
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err: any = await invoke("unknown", {}).catch((e) => e);
    expect(err).toBeInstanceOf(AppClientError);
    expect(err.code).toBe("client_error");
  });
});
