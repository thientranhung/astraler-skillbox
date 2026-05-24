import { ipcMain, BrowserWindow } from "electron";
import { getGoClient } from "./manager.js";
import { ALLOWLIST } from "./method-allowlist.js";

let notifUnsub: (() => void) | null = null;

export function registerIpcBridge(win: BrowserWindow): void {
  // Remove existing handler before re-registering to avoid Electron throwing
  // on duplicate channel registration across window recreations.
  ipcMain.removeHandler("core:invoke");
  ipcMain.handle("core:invoke", async (_event, method: string, params: unknown) => {
    if (!ALLOWLIST.has(method)) {
      throw new Error(`method_not_allowed: ${method}`);
    }
    return getGoClient().call(method, params);
  });

  // Unsubscribe previous window's notification forwarder before subscribing
  // the new window, so notifications don't accumulate across window recreations.
  notifUnsub?.();
  notifUnsub = getGoClient().on("operation.progress", (params) => {
    if (!win.isDestroyed()) {
      win.webContents.send("core:event", "operation.progress", params);
    }
  });
}
