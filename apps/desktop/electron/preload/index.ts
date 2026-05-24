import { contextBridge, ipcRenderer } from "electron";

contextBridge.exposeInMainWorld("core", {
  invoke: (method: string, params: unknown): Promise<unknown> =>
    ipcRenderer.invoke("core:invoke", method, params),

  onEvent: (event: string, cb: (params: unknown) => void): (() => void) => {
    const handler = (_: Electron.IpcRendererEvent, method: string, params: unknown): void => {
      if (method === event) cb(params);
    };
    ipcRenderer.on("core:event", handler);
    return () => ipcRenderer.off("core:event", handler);
  },
});
