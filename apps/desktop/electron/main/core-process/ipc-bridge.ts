import { app, ipcMain, BrowserWindow, dialog, shell } from "electron";
import { execFile } from "child_process";
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

    // Electron-native dialogs — handled in main process, not forwarded to Go.
    if (method === "dialog.openHostFolder" || method === "dialog.openProjectFolder") {
      const parentWin = BrowserWindow.fromWebContents(event.sender);
      const title =
        method === "dialog.openProjectFolder"
          ? "Choose Project Folder"
          : "Choose Skill Host Folder";
      const opts: Electron.OpenDialogOptions = {
        properties: ["openDirectory"],
        title,
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

    // Open a folder in the native file manager (Finder on macOS).
    if (method === "dialog.openPath") {
      const { path } = params as { path: string };
      const errMsg = await shell.openPath(path);
      if (errMsg !== "") {
        // Throw a JSON-encoded envelope that the preload's toStructuredError
        // can parse, so the renderer gets AppClientError("unknown_error", ...)
        // instead of a generic client_error.
        throw new Error(
          JSON.stringify({
            code: -1,
            message: "Failed to open folder",
            data: {
              code: "unknown_error",
              rpcCode: 1099,
              userMessage: "Failed to open project folder",
              technicalMessage: `shell.openPath: ${errMsg}`,
            },
          }),
        );
      }
      return { opened: true };
    }

    // Open Terminal at the given folder (macOS). Argument-array launch prevents injection.
    if (method === "dialog.openTerminal") {
      const { path } = params as { path: string };
      await new Promise<void>((resolve, reject) => {
        execFile("open", ["-a", "Terminal", path], (err) => {
          if (err) reject(err);
          else resolve();
        });
      }).catch((err: Error) => {
        throw new Error(
          JSON.stringify({
            code: -1,
            message: "Failed to open Terminal",
            data: {
              code: "unknown_error",
              rpcCode: 1099,
              userMessage: "Failed to open Terminal",
              technicalMessage: `open -a Terminal: ${err.message}`,
            },
          }),
        );
      });
      return { opened: true };
    }

    const result = await getGoClient().call(method, params);

    // After Go deletes the DB and returns restarting:true, trigger a clean relaunch.
    if (method === "app.resetAll" && (result as { restarting?: boolean })?.restarting === true) {
      app.relaunch();
      app.exit(0);
    }

    return result;
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
