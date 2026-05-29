import { app, BrowserWindow, dialog } from "electron";
import path from "path";
import { spawnGoCore, onFatal } from "./core-process/manager.js";
import { registerIpcBridge } from "./core-process/ipc-bridge.js";

// electron-vite (v5) sets ELECTRON_RENDERER_URL in dev — NOT VITE_DEV_SERVER_URL.
// Gating on it lets the renderer load the HMR dev server in `pnpm dev` while
// packaged builds (where it's unset) fall back to the built index.html.
const ELECTRON_RENDERER_URL = process.env["ELECTRON_RENDERER_URL"];

// Dev-only: expose the Chrome DevTools Protocol on a fixed localhost port so
// browser-automation agents (agent-browser) can `connect` to THIS running dev
// instance instead of launching a second app. Gated on ELECTRON_RENDERER_URL
// (set by electron-vite only in dev) so packaged builds never open a debugging
// port. Override the port via SKILLBOX_CDP_PORT. Must run before
// app.whenReady(). Band reserved for agent-browser: 49222-49250.
if (ELECTRON_RENDERER_URL) {
  app.commandLine.appendSwitch("remote-debugging-port", process.env["SKILLBOX_CDP_PORT"] ?? "49222");
}

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

  if (ELECTRON_RENDERER_URL) {
    win.loadURL(ELECTRON_RENDERER_URL);
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
