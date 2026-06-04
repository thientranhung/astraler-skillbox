import { spawn, type ChildProcess } from "child_process";
import { app } from "electron";
import { JsonRpcStdioClient } from "./json-rpc-client.js";
import { resolveCoreSpawn } from "./core-go-path.js";

const READY_TIMEOUT_MS = 10_000;
const MAX_RESTARTS = 3;
const SIGKILL_DELAY_MS = 3_000;
const MAX_LOG_LINES = 100;

let goClient: JsonRpcStdioClient | null = null;
let activeChild: ChildProcess | null = null;
let restartCount = 0;
let intentionalShutdown = false;
let onFatalError: ((message: string) => void) | null = null;

const coreLogBuffer: string[] = [];

function pushLogChunk(chunk: string): void {
  const lines = chunk.split("\n");
  coreLogBuffer.push(...lines);
  if (coreLogBuffer.length > MAX_LOG_LINES) {
    coreLogBuffer.splice(0, coreLogBuffer.length - MAX_LOG_LINES);
  }
}

export function getCoreLogs(): string[] {
  return [...coreLogBuffer];
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
    const spec = resolveCoreSpawn({
      isPackaged: app.isPackaged,
      baseDir: __dirname,
      resourcesPath: process.resourcesPath,
    });
    process.stderr.write(
      `[manager] spawning Go core: ${spec.command} ${spec.args.join(" ")} (cwd=${spec.cwd})\n`
    );

    const child = spawn(spec.command, spec.args, {
      cwd: spec.cwd,
      stdio: ["pipe", "pipe", "pipe"],
    });
    activeChild = child;

    const client = new JsonRpcStdioClient(child);

    const stderrLines: string[] = [];
    child.stderr?.on("data", (chunk: Buffer) => {
      const text = chunk.toString();
      stderrLines.push(text);
      pushLogChunk(text);
    });

    const timer = setTimeout(() => {
      child.kill("SIGTERM");
      const goOutput = stderrLines.join("").trim();
      const detail = goOutput ? `\n\n${goOutput}` : "";
      // Pre-ready failure: only reject so the catch block in main() handles it.
      // Do NOT call fatal() here — fatal() is for mid-run crashes after server.ready.
      reject(new Error(`server.ready timeout${detail}`));
    }, READY_TIMEOUT_MS);

    const unsubscribe = client.on("server.ready", () => {
      clearTimeout(timer);
      unsubscribe();
      goClient = client;
      process.stderr.write("[manager] Go core ready\n");

      child.on("exit", (code) => {
        // Never restart when the shutdown was initiated intentionally.
        if (intentionalShutdown) return;
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
      // Pre-ready failure: only reject so the catch block in main() handles it.
      reject(err);
    });
  });
}

export function shutdownGoCore(): void {
  // Set flag before sending signal so the exit handler never triggers a restart.
  intentionalShutdown = true;

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
