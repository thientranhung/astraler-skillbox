import { contextBridge, ipcRenderer } from "electron";

function toStructuredError(message: string): Error | Record<string, unknown> {
  const start = message.indexOf("{");
  const end = message.lastIndexOf("}");
  if (start === -1 || end === -1 || end <= start) return new Error(message);
  const candidate = message.slice(start, end + 1);
  try {
    const parsed = JSON.parse(candidate) as {
      code?: number;
      message?: string;
      data?: {
        code?: string;
        rpcCode?: number;
        userMessage?: string;
        technicalMessage?: string;
      };
    };
    if (parsed.data != null) {
      return {
        code: parsed.data.code,
        rpcCode: parsed.data.rpcCode,
        userMessage: parsed.data.userMessage ?? parsed.message,
        technicalMessage: parsed.data.technicalMessage,
        message: parsed.data.userMessage ?? parsed.message ?? candidate,
      };
    }
    return new Error(candidate);
  } catch {
    return new Error(message);
  }
}

contextBridge.exposeInMainWorld("core", {
  invoke: async (method: string, params: unknown): Promise<unknown> => {
    try {
      return await ipcRenderer.invoke("core:invoke", method, params);
    } catch (err) {
      if (err instanceof Error) {
        throw toStructuredError(err.message);
      }
      throw err;
    }
  },

  onEvent: (event: string, cb: (params: unknown) => void): (() => void) => {
    const handler = (_: Electron.IpcRendererEvent, method: string, params: unknown): void => {
      if (method === event) cb(params);
    };
    ipcRenderer.on("core:event", handler);
    return () => ipcRenderer.off("core:event", handler);
  },

  getStartupError: (): Promise<string | null> =>
    ipcRenderer.invoke("core:startup-error-get") as Promise<string | null>,
});
