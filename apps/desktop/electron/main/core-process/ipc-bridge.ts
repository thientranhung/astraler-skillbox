import { ipcMain, BrowserWindow, dialog } from "electron";
import { getGoClient } from "./manager.js";
import { ALLOWLIST } from "./method-allowlist.js";

let notifUnsub: (() => void) | null = null;

export function registerIpcBridge(win: BrowserWindow): void {
  // Remove existing handler before re-registering to avoid Electron throwing
  // on duplicate channel registration across window recreations.
  ipcMain.removeHandler("core:invoke");
  ipcMain.handle("core:invoke", async (event, method: string, params: unknown) => {
    if (!ALLOWLIST.has(method)) {
      throw new Error(`method_not_allowed: ${method}`);
    }

    // Electron-native dialog — handled in main process, not forwarded to Go.
    if (method === "dialog.openHostFolder") {
      const parentWin = BrowserWindow.fromWebContents(event.sender);
      const opts: Electron.OpenDialogOptions = {
        properties: ["openDirectory"],
        title: "Choose Skill Host Folder",
      };
      const result = parentWin
        ? await dialog.showOpenDialog(parentWin, opts)
        : await dialog.showOpenDialog(opts);
      const path =
        result.canceled || result.filePaths.length === 0
          ? null
          : result.filePaths[0];
      return { path };
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
