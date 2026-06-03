import { app, BrowserWindow, dialog, ipcMain } from "electron";
import path from "path";
import { spawnGoCore, onFatal } from "./core-process/manager.js";
import { registerIpcBridge } from "./core-process/ipc-bridge.js";

// electron-vite (v5) sets ELECTRON_RENDERER_URL in dev — NOT VITE_DEV_SERVER_URL.
// Gating on it lets the renderer load the HMR dev server in `pnpm dev` while
// packaged builds (where it's unset) fall back to the built index.html.
const ELECTRON_RENDERER_URL = process.env["ELECTRON_RENDERER_URL"];

// Phase 1 does not store credentials, cookies, or tokens. On macOS, Chromium's
// default profile storage can still request a Keychain "Safe Storage" item at
// startup, which is scary and unnecessary for this product phase. Keep Keychain
// disabled by default; phase 2 credential work can opt back in explicitly.
if (process.platform === "darwin" && process.env["SKILLBOX_ENABLE_KEYCHAIN"] !== "1") {
  app.commandLine.appendSwitch("use-mock-keychain");
}

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

// Stores the pre-ready startup error (Go failed before server.ready).
// Set before createWindow() so the renderer can query it via core:startup-error-get.
let preReadyStartupError: string | null = null;

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

  // Dev CSP: permit Vite React Refresh inline preamble (unsafe-inline + unsafe-eval)
  // and HMR WebSocket (ws://localhost, http://localhost). Only active when
  // ELECTRON_RENDERER_URL is set — electron-vite sets it in `pnpm dev`, never in
  // packaged builds. Prod CSP is strict; no relaxation reaches packaged users.
  const csp = ELECTRON_RENDERER_URL
    ? "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws://localhost:* http://localhost:*"
    : "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'";

  win.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        "Content-Security-Policy": [csp],
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

  // The renderer queries this via preload.getStartupError() (ipcRenderer.invoke).
  // Registered once; returns null for normal launches, or the error message when
  // Go failed before server.ready.
  ipcMain.handle("core:startup-error-get", () => preReadyStartupError);

  onFatal((message) => {
    // Mid-run fatal: Go crashed after server.ready. The window is open; send the
    // error as a core:event so the renderer can navigate to the startup-error screen.
    // pre-ready failures never reach this handler (manager.ts no longer calls fatal()
    // from timeout/spawn-error paths).
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send("core:event", "startup.error", { message });
    } else {
      dialog.showErrorBox("Skillbox Core Error", message);
      app.quit();
    }
  });

  try {
    await spawnGoCore();
  } catch (err) {
    // Go failed before server.ready (invalid/corrupt DB, dirty migration, etc.).
    // Store the error so the renderer can retrieve it via getStartupError(), then
    // create the window. The renderer queries the error on mount via IPC handler —
    // no one-shot event, no race.
    preReadyStartupError = (err as Error).message;
    mainWindow = createWindow();
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
