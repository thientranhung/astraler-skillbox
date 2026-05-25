import { app, BrowserWindow, dialog } from "electron";
import path from "path";
import { spawnGoCore, onFatal } from "./core-process/manager.js";
import { registerIpcBridge } from "./core-process/ipc-bridge.js";

const VITE_DEV_SERVER_URL = process.env["VITE_DEV_SERVER_URL"];

let mainWindow: BrowserWindow | null = null;

function createWindow(): BrowserWindow {
  const win = new BrowserWindow({
    width: 1200,
    height: 800,
    show: false,
    webPreferences: {
      preload: path.join(__dirname, "../preload/index.cjs"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  win.webContents.session.setPermissionRequestHandler((_webContents, _permission, callback) => {
    callback(false);
  });

  win.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        "Content-Security-Policy": [
          "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
        ],
      },
    });
  });

  if (VITE_DEV_SERVER_URL) {
    win.loadURL(VITE_DEV_SERVER_URL);
  } else {
    win.loadFile(path.join(__dirname, "../renderer/index.html"));
  }

  win.once("ready-to-show", () => win.show());
  return win;
}

async function main(): Promise<void> {
  await app.whenReady();

  onFatal((message) => {
    dialog.showErrorBox("Skillbox Core Error", message);
    app.quit();
  });

  try {
    await spawnGoCore();
  } catch (err) {
    dialog.showErrorBox(
      "Startup Error",
      `Failed to start Skillbox core: ${(err as Error).message}`
    );
    app.quit();
    return;
  }

  mainWindow = createWindow();
  registerIpcBridge(mainWindow);

  app.on("window-all-closed", () => {
    if (process.platform !== "darwin") app.quit();
  });

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      mainWindow = createWindow();
      if (mainWindow) registerIpcBridge(mainWindow);
    }
  });
}

main().catch((err) => {
  process.stderr.write(`[main] unhandled error: ${err}\n`);
  app.quit();
});
