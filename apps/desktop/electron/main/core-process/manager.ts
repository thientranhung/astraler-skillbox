import { spawn, type ChildProcess } from "child_process";
import path from "path";
import { app } from "electron";
import { JsonRpcStdioClient } from "./json-rpc-client.js";

const READY_TIMEOUT_MS = 10_000;
const MAX_RESTARTS = 3;
const SIGKILL_DELAY_MS = 3_000;

let goClient: JsonRpcStdioClient | null = null;
let activeChild: ChildProcess | null = null;
let restartCount = 0;
let onFatalError: ((message: string) => void) | null = null;

function coreGoPath(): string {
  return path.resolve(__dirname, "../../../../../core-go");
}

export function getGoClient(): JsonRpcStdioClient {
  if (!goClient) throw new Error("Go client not initialized");
  return goClient;
}

export function onFatal(handler: (message: string) => void): void {
  onFatalError = handler;
}

function fatal(message: string): void {
  process.stderr.write(`[manager] FATAL: ${message}\n`);
  onFatalError?.(message);
}

export function spawnGoCore(): Promise<JsonRpcStdioClient> {
  return new Promise((resolve, reject) => {
    const cwd = coreGoPath();
    process.stderr.write(`[manager] spawning Go core from ${cwd}\n`);

    const child = spawn("go", ["run", "./cmd/skillbox-core"], {
      cwd,
      stdio: ["pipe", "pipe", "pipe"],
    });
    activeChild = child;

    const client = new JsonRpcStdioClient(child);
    const timer = setTimeout(() => {
      child.kill("SIGTERM");
      reject(new Error("server.ready timeout"));
      fatal("Go core did not send server.ready within 10s");
    }, READY_TIMEOUT_MS);

    const unsubscribe = client.on("server.ready", () => {
      clearTimeout(timer);
      unsubscribe();
      goClient = client;
      process.stderr.write("[manager] Go core ready\n");

      child.on("exit", (code) => {
        if (code !== 0 && restartCount < MAX_RESTARTS) {
          restartCount++;
          process.stderr.write(
            `[manager] Go core exited (code=${code}), restart ${restartCount}/${MAX_RESTARTS}\n`
          );
          spawnGoCore().catch(() => fatal("Go core failed to restart"));
        } else if (code !== 0) {
          fatal("Go core crashed too many times; giving up");
        }
      });

      resolve(client);
    });

    child.on("error", (err) => {
      clearTimeout(timer);
      reject(err);
      fatal(`Failed to spawn Go core: ${err.message}`);
    });
  });
}

export function shutdownGoCore(): void {
  goClient?.shutdown("app_quit");
  goClient = null;

  const child = activeChild;
  activeChild = null;
  if (child && !child.killed) {
    child.kill("SIGTERM");
    const killer = setTimeout(() => {
      if (!child.killed) child.kill("SIGKILL");
    }, SIGKILL_DELAY_MS);
    child.on("exit", () => clearTimeout(killer));
  }
}

app.on("before-quit", shutdownGoCore);
