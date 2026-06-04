import { ipcMain, BrowserWindow, dialog, shell, clipboard, app } from "electron";
import { execFile } from "child_process";
import os from "os";
import path from "path";
import fs from "fs/promises";
import { getGoClient, getCoreLogs } from "./manager.js";
import { ALLOWLIST } from "./method-allowlist.js";
import { buildDiagnosticsText } from "./diagnostics.js";

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

    // Collect diagnostics snapshot for export or copy.
    if (method === "dialog.exportDiagnostics" || method === "dialog.copyDiagnostics") {
      const homeDir = os.homedir();
      const dbPath =
        process.env["SKILLBOX_DB_PATH"] ??
        path.join(app.getPath("userData"), "skillbox.db");
      const text = buildDiagnosticsText({
        appVersion: app.getVersion(),
        electronVersion: process.versions.electron ?? "unknown",
        chromeVersion: process.versions.chrome ?? "unknown",
        nodeVersion: process.versions.node ?? "unknown",
        platform: process.platform,
        arch: process.arch,
        dbPath,
        homeDir,
        exportedAt: new Date().toISOString(),
        coreLogLines: getCoreLogs(),
      });

      if (method === "dialog.copyDiagnostics") {
        clipboard.writeText(text);
        return { copied: true };
      }

      // dialog.exportDiagnostics: show save dialog
      const parentWin = BrowserWindow.fromWebContents(event.sender);
      const saveResult = parentWin
        ? await dialog.showSaveDialog(parentWin, {
            title: "Export Diagnostics",
            defaultPath: `skillbox-diagnostics-${Date.now()}.txt`,
            filters: [{ name: "Text Files", extensions: ["txt"] }],
          })
        : await dialog.showSaveDialog({
            title: "Export Diagnostics",
            defaultPath: `skillbox-diagnostics-${Date.now()}.txt`,
            filters: [{ name: "Text Files", extensions: ["txt"] }],
          });

      if (saveResult.canceled || !saveResult.filePath) {
        return { saved: false, filePath: null };
      }
      await fs.writeFile(saveResult.filePath, text, "utf-8");
      return { saved: true, filePath: saveResult.filePath };
    }

    const result = await getGoClient().call(method, params);
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
