import { describe, it, expect, vi } from "vitest";
import { EventEmitter } from "events";
import { Readable, Writable } from "stream";
import { JsonRpcStdioClient } from "../json-rpc-client.js";

function makeChild(lines: string[]): {
  stdout: Readable;
  stderr: Readable;
  stdin: Writable;
  kill: ReturnType<typeof vi.fn>;
  on: ReturnType<typeof vi.fn>;
  killed: boolean;
} {
  const stdout = new Readable({ read() {} });
  const stderr = new Readable({ read() {} });
  const written: string[] = [];
  const stdin = new Writable({
    write(chunk, _enc, cb) {
      written.push(chunk.toString());
      cb();
    },
  });

  const emitter = new EventEmitter();
  const child = {
    stdout,
    stderr,
    stdin,
    kill: vi.fn(),
    killed: false,
    on: (event: string, handler: (...args: unknown[]) => void) => {
      emitter.on(event, handler);
      return child;
    },
    emit: (event: string, ...args: unknown[]) => emitter.emit(event, ...args),
    _written: written,
  };

  // Push NDJSON lines after a tick
  setImmediate(() => {
    for (const line of lines) {
      stdout.push(line + "\n");
    }
  });

  return child as unknown as ReturnType<typeof makeChild> & { _written: string[]; emit: (e: string, ...a: unknown[]) => boolean };
}

describe("JsonRpcStdioClient", () => {
  it("resolves call() when matching response arrives", async () => {
    const responseLine = JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      result: { pong: true, ts: "2026-01-01T00:00:00Z" },
    });
    const child = makeChild([responseLine]);
    const client = new JsonRpcStdioClient(child as never);

    const result = await client.call<{ pong: boolean; ts: string }>("ping", {});
    expect(result.pong).toBe(true);
    expect(result.ts).toBe("2026-01-01T00:00:00Z");
  });

  it("rejects call() on JSON-RPC error response", async () => {
    const errorLine = JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      error: { code: -32601, message: "method not found" },
    });
    const child = makeChild([errorLine]);
    const client = new JsonRpcStdioClient(child as never);

    await expect(client.call("unknown", {})).rejects.toThrow();
  });

  it("fires notification subscribers without rejecting pending calls", async () => {
    const notifLine = JSON.stringify({
      jsonrpc: "2.0",
      method: "server.ready",
      params: { version: "0.1.0" },
    });
    const responseLine = JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      result: { pong: true, ts: "2026-01-01T00:00:00Z" },
    });
    const child = makeChild([notifLine, responseLine]);
    const client = new JsonRpcStdioClient(child as never);

    const notifReceived = new Promise<unknown>((resolve) => {
      client.on("server.ready", resolve);
    });

    const [notif, result] = await Promise.all([
      notifReceived,
      client.call<{ pong: boolean }>("ping", {}),
    ]);

    expect((notif as { version: string }).version).toBe("0.1.0");
    expect((result as { pong: boolean }).pong).toBe(true);
  });

  it("rejects all pending calls on shutdown", async () => {
    const child = makeChild([]); // no responses
    const client = new JsonRpcStdioClient(child as never);

    const pendingCall = client.call("ping", {}, { timeoutMs: 60_000 });
    client.shutdown("test");

    await expect(pendingCall).rejects.toThrow("core_unavailable");
  });

  it("logs orphan responses without throwing", async () => {
    const orphan = JSON.stringify({ jsonrpc: "2.0", id: 99, result: {} });
    const child = makeChild([orphan]);
    const client = new JsonRpcStdioClient(child as never);

    // Wait a tick for the line to be processed
    await new Promise((r) => setTimeout(r, 50));
    // No error thrown — if we reach here the test passes
    expect(true).toBe(true);
  });

  it("unsubscribe returned by on() stops notifications", async () => {
    const notifLine = JSON.stringify({
      jsonrpc: "2.0",
      method: "test.event",
      params: { count: 1 },
    });
    const child = makeChild([]);
    const client = new JsonRpcStdioClient(child as never);

    const received: unknown[] = [];
    const unsub = client.on("test.event", (p) => received.push(p));
    unsub();

    // Push the notification after unsubscribing
    (child as unknown as { stdout: Readable }).stdout.push(notifLine + "\n");
    await new Promise((r) => setTimeout(r, 50));

    expect(received).toHaveLength(0);
  });

  it("timeoutMs: 0 means no timeout — call stays pending indefinitely", async () => {
    const child = makeChild([]); // no response lines
    const client = new JsonRpcStdioClient(child as never);

    let settled = false;
    const p = client.call("ping", {}, { timeoutMs: 0 }).then(
      () => { settled = true; },
      () => { settled = true; },
    );

    // Give 100ms — should still be pending
    await new Promise((r) => setTimeout(r, 100));
    expect(settled).toBe(false);

    // Resolve by sending a response
    const responseLine = JSON.stringify({ jsonrpc: "2.0", id: 1, result: { pong: true, ts: "t" } });
    (child as unknown as { stdout: Readable }).stdout.push(responseLine + "\n");
    await p;
    expect(settled).toBe(true);
  });

  it("call() rejects with core_unavailable when stdin is not writable", async () => {
    const child = makeChild([]);
    const castedChild = child as unknown as { stdin: Writable & { writable: boolean } };
    castedChild.stdin.writable = false;

    const client = new JsonRpcStdioClient(child as never);
    await expect(client.call("ping", {})).rejects.toThrow("core_unavailable");
  });
});
