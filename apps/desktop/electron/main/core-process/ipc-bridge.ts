import { ipcMain, BrowserWindow } from "electron";
import { getGoClient } from "./manager.js";
import { ALLOWLIST } from "./method-allowlist.js";

export function registerIpcBridge(win: BrowserWindow): void {
  ipcMain.handle("core:invoke", async (_event, method: string, params: unknown) => {
    if (!ALLOWLIST.has(method)) {
      throw new Error(`method_not_allowed: ${method}`);
    }
    return getGoClient().call(method, params);
  });

  // Forward Go push notifications to renderer
  getGoClient().on("operation.progress", (params) => {
    if (!win.isDestroyed()) {
      win.webContents.send("core:event", "operation.progress", params);
    }
  });
}
