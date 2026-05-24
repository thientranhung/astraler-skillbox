import { createInterface } from "readline";
import type { ChildProcess } from "child_process";

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (err: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

const DEFAULT_TIMEOUT_MS = 30_000;

export class JsonRpcStdioClient {
  private nextId = 1;
  private pending = new Map<number, PendingRequest>();
  private subscribers = new Map<string, Set<(params: unknown) => void>>();
  private dead = false;

  constructor(private child: ChildProcess) {
    const rl = createInterface({ input: child.stdout! });
    rl.on("line", (line) => this.handleLine(line));

    child.stderr?.on("data", (chunk: Buffer) => {
      process.stderr.write(`[core] ${chunk}`);
    });

    child.on("exit", () => this.shutdown("child_exited"));
    child.on("error", (err) => {
      process.stderr.write(`[core] child error: ${err.message}\n`);
      this.shutdown("child_error");
    });
  }

  call<T>(method: string, params: unknown, opts?: { timeoutMs?: number }): Promise<T> {
    if (this.dead) {
      return Promise.reject(new Error("core_unavailable"));
    }

    const id = this.nextId++;
    const timeoutMs = opts?.timeoutMs ?? DEFAULT_TIMEOUT_MS;

    return new Promise<T>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`rpc_timeout: ${method}`));
      }, timeoutMs);

      this.pending.set(id, {
        resolve: resolve as (v: unknown) => void,
        reject,
        timer,
      });

      const msg = JSON.stringify({ jsonrpc: "2.0", id, method, params }) + "\n";
      this.child.stdin!.write(msg);
    });
  }

  on(method: string, handler: (params: unknown) => void): () => void {
    let set = this.subscribers.get(method);
    if (!set) {
      set = new Set();
      this.subscribers.set(method, set);
    }
    set.add(handler);
    return () => set!.delete(handler);
  }

  shutdown(reason: string): void {
    if (this.dead) return;
    this.dead = true;
    process.stderr.write(`[core] client shutdown: ${reason}\n`);

    for (const [, pending] of this.pending) {
      clearTimeout(pending.timer);
      pending.reject(new Error("core_unavailable"));
    }
    this.pending.clear();
  }

  private handleLine(line: string): void {
    if (!line.trim()) return;

    let msg: Record<string, unknown>;
    try {
      msg = JSON.parse(line) as Record<string, unknown>;
    } catch {
      process.stderr.write(`[core] non-JSON line: ${line}\n`);
      return;
    }

    if ("id" in msg && msg.id !== null && msg.id !== undefined) {
      // Response to a call
      const id = msg.id as number;
      const pending = this.pending.get(id);
      if (!pending) {
        process.stderr.write(`[core] orphan response id=${id}\n`);
        return;
      }
      this.pending.delete(id);
      clearTimeout(pending.timer);

      if ("error" in msg) {
        pending.reject(new Error(JSON.stringify(msg.error)));
      } else {
        pending.resolve(msg.result);
      }
    } else if ("method" in msg) {
      // Server-push notification (no id)
      const method = msg.method as string;
      const params = msg.params;
      const set = this.subscribers.get(method);
      if (set) {
        for (const handler of set) {
          handler(params);
        }
      }
    } else {
      process.stderr.write(`[core] unexpected message: ${line}\n`);
    }
  }
}
