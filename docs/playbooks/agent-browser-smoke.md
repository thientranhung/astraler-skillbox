# Agent Browser — Electron Smoke Automation

How to drive **the running dev app** for smoke tests / UI automation via the Chrome
DevTools Protocol (CDP). Self-contained — read it on demand, only when actually
running browser automation.

## Principle: connect, don't launch

`pnpm dev` already runs the app. The agent **attaches** to that instance — it never
launches a second app. `pnpm dev` exposes CDP on a fixed localhost port (default
`49222`, override `SKILLBOX_CDP_PORT`; gated on `ELECTRON_RENDERER_URL` in
`electron/main/index.ts` so packaged builds never open it). Port band reserved for
agent-browser: **49222–49250**.

## Workflow

```sh
curl -s http://127.0.0.1:49222/json/version   # confirm CDP is live (Browser field)
agent-browser connect 49222                     # attach to the running dev app
agent-browser --cdp 49222 snapshot -i           # then the normal agent-browser workflow
```

## Gotchas (learned the hard way)

- **Never `agent-browser open <url>` for app smoke tests.** It spawns its own headless
  Chrome on an ephemeral port that orphans when its daemon dies (we found one running
  4.5 days). Always `connect`.
- **`get url` fails on Electron** (`Target.createTarget: Not supported`). To read the
  current page URL, query the CDP endpoint instead: `curl -s http://127.0.0.1:49222/json`
  and read the `page` target's `url`.
- **Teardown cleanly.** Quit the dev app normally (closes the port). If killing from the
  shell, kill the **electron-vite watcher first** (else it respawns the app):
  `pkill -f "electron-vite.js dev"`, then the electron + go core. Stop a lingering
  automation daemon with `pkill -f agent-browser-darwin`.
- **Do NOT `pkill -f skillbox-core` when multiple dev instances run** — it matches every
  instance's Go core and crashes them all (onFatal → app.quit). Target the specific
  process tree (launch with `setsid` and kill the process group).
- **Audit the band anytime:** `lsof -nP -iTCP:49222-49250 -sTCP:LISTEN`.
