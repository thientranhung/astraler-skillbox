import type { AppClientError, PingResult } from "./types.js";

export { AppClientError };

async function invoke<T>(method: string, params: unknown): Promise<T> {
  try {
    return (await window.core.invoke(method, params)) as T;
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    throw { code: "client_error", message: msg } satisfies AppClientError;
  }
}

export const methods = {
  ping: (): Promise<PingResult> => invoke<PingResult>("ping", {}),
};
