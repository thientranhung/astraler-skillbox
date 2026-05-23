# Transport Decision Brainstorm: Electron Main ↔ Go Core

## Context

Astraler Skillbox splits into three runtime processes:

```
Electron renderer (React UI)
  ↕  contextBridge / ipcRenderer
Electron main process
  ↕  [TRANSPORT — this decision]
Go core runtime (sidecar)
```

This doc debates only the **Electron main ↔ Go core** boundary.
Renderer ↔ Electron main is settled: Electron IPC + contextBridge.

The transport must support:
- Request/response (commands and queries)
- Server-initiated notifications (operation progress)
- Reliable framing (no partial reads)
- Cross-platform (macOS first, Windows later)
- Single Go binary, no separate server process

---

## Option A: stdio JSON-RPC 2.0

### How it works

Electron main spawns Go binary with `child_process.spawn`.
Go reads JSON-RPC requests from `os.Stdin`, writes JSON-RPC responses and
notifications to `os.Stdout`.

Framing: NDJSON (one JSON object per line, newline-terminated).
Go's `encoding/json` escapes all literal newlines inside strings, so line
splitting on `\n` is unambiguous.

```
Electron main ──stdin──▶  Go core (reads line by line)
Electron main ◀─stdout── Go core (writes newline-terminated JSON)
```

JSON-RPC 2.0 message types:
- Request: has `id`, expects response.
- Notification: no `id`, no response. Go uses this for `operation.progress`.
- Response: has `id` matching request.

### For

- **Zero infrastructure.** No port, no socket file, no address discovery.
  Process stdin/stdout are set up by the OS when Electron spawns Go.
- **Lifecycle coupling.** When Electron main kills the child, transport is gone.
  No zombie Go process writing to a dead socket.
- **Security by default.** No network listener. Nothing on localhost:PORT for
  other processes to reach. The pipe is OS-private to the parent/child.
- **Battle-tested in Electron.** gopls, tsserver, rust-analyzer, pyright —
  all run this way inside VS Code (which is Electron). The pattern is known.
- **Testable in isolation.** Pipe JSON lines to Go's stdin via shell or test
  harness, read stdout. No Electron needed to test Go core.
- **Bidirectional out of the box.** Go can push notifications upstream
  (progress, warnings) without UI polling.

### Against

- **stdout is a protocol boundary.** Any `fmt.Println`, `log.Println`, or
  library that writes to stdout corrupts the JSON stream and is invisible until
  runtime. This requires team discipline: **all Go output except protocol
  messages goes to stderr or a log file.** Third-party libraries that log to
  stdout (rare but possible) become landmines.
- **No interactive debugging.** You cannot curl or Postman the Go core.
  Debugging requires either: (a) a test harness that pipes JSON to stdin, or
  (b) a separate HTTP debug server behind a build flag.
- **NDJSON framing is slightly informal.** LSP servers use
  `Content-Length: N\r\n\r\n` header framing for robustness. NDJSON is simpler
  but relies on Go's json encoder being correct about escaping. It is — but
  worth documenting explicitly.
- **One consumer.** Only Electron main reads Go's stdout. A future CLI tool or
  test harness has to re-implement the same reader. Not a Phase 1 problem.
- **Windows line endings.** Windows default mode for pipes is text mode, which
  can translate `\n` to `\r\n`. Go binary must open stdin/stdout in binary mode
  (`os.Stdin` in Go is binary by default, but worth verifying on Windows).

### Failure modes

| Failure | Consequence | Mitigation |
|---|---|---|
| Go writes non-JSON to stdout | JSON parse error in Electron, stream may desync | Strict no-stdout rule, test with `go vet`-style check |
| Go crashes mid-message | Partial JSON line, parse error | Electron detects child exit via `close` event; restart or show error |
| Electron kills Go while mid-write | Pipe broken, Go gets SIGPIPE | Go should handle SIGPIPE or check write errors; not fatal |
| Message too large (large JSON payload) | Stream still works; NDJSON has no size limit | Not a risk for command/response payloads in Skillbox |
| Multiple concurrent requests before response | JSON-RPC id field matches requests to responses | Standard JSON-RPC 2.0 multiplexing; handled by library |

### Dev ergonomics

```bash
# Test Go core without Electron
echo '{"jsonrpc":"2.0","id":1,"method":"getDashboard","params":{}}' | ./core-go

# Watch Go notifications
echo '{"jsonrpc":"2.0","id":1,"method":"scanProject","params":{"project_id":1}}' \
  | ./core-go | while read line; do echo "$line" | jq .; done
```

In Electron dev mode: Electron main forwards child stderr to `console.error`.
Go uses `log/slog` with stderr output. Dev gets Go logs in Electron devtools.

Hot reload: `air` watches Go files, rebuilds, Electron main detects child exit
and restarts new binary.

### Packaging/security

- Go binary bundled as `extraResources` in electron-builder.
- Electron main resolves binary path via `process.resourcesPath` in production,
  `path.join(__dirname, '../dist/core-go')` in dev.
- No firewall rules. No port in app entitlements.
- macOS: binary must be signed + Hardened Runtime. Electron-builder handles
  this alongside the app bundle signing. One signing pass covers both.
- Windows: binary codesigning with same cert as Electron app. No extra step.

---

## Option B: Local HTTP (REST or JSON-RPC over HTTP)

### How it works

Go listens on a random localhost port. Writes port to stdout (first line) or
a temp file. Electron main reads port, uses `fetch` or `axios` for requests.
For progress: Server-Sent Events (SSE) or WebSocket on a second endpoint.

```
Electron main ──HTTP GET/POST──▶ Go http.ListenAndServe(127.0.0.1:PORT)
Electron main ◀──SSE stream───── Go SSE endpoint for operation progress
```

### For

- **Best debuggability.** curl, Postman, browser devtools all work against
  `http://127.0.0.1:PORT`. Fastest way to debug Go responses without Electron.
- **Familiar HTTP semantics.** Every Go and TypeScript developer knows REST.
  No JSON-RPC library needed. Standard `net/http` in Go.
- **Easy mocking.** In test, start a mock Go HTTP server. Electron client code
  is identical to production.
- **OpenAPI.** If you want codegen (Decision 4 from brainstorm), OpenAPI tooling
  is mature for HTTP. Generate Go server stubs and TypeScript client.

### Against

- **Port discovery is awkward.** Go picks a random port to avoid conflicts.
  How does Electron know it? Options:
  - Go writes port as first stdout line, then Electron parses it before
    setting up the rest. This works but mixes protocol concerns with startup.
  - Go writes to a temp file, Electron reads it. Race condition on startup.
  - Fixed port (e.g., 37291): simpler but risks collision with other apps.
  None of these are elegant. All add startup complexity.
- **Security exposure.** `127.0.0.1:PORT` is reachable by **any process on
  the machine**, not just Electron. A malicious local app could call the Go
  HTTP server directly (SSRF from browser extensions, other apps). For a skill
  management tool with filesystem write access, this is a real surface.
  Mitigation: token-based auth on every request (Go generates a one-time token,
  passes it to Electron at startup). Adds complexity.
- **SSE for progress.** Server-Sent Events work but need a persistent connection
  per operation. Multiple simultaneous operations need multiple SSE streams or
  a multiplexed event endpoint. More moving parts.
- **Process lifecycle.** If Electron crashes without closing the HTTP server,
  Go keeps listening. Next app launch may find the port occupied (if using
  fixed port) or produce a zombie Go process (if using random port and no
  cleanup). Need explicit Go shutdown signal or port file cleanup.
- **Renderer security risk.** If renderer ever gets the port (accidental leak
  via contextBridge), it can call Go directly, bypassing the Electron main
  security boundary. Must ensure port never reaches renderer.

### Failure modes

| Failure | Consequence | Mitigation |
|---|---|---|
| Port collision (fixed port) | Go fails to bind, app won't start | Use random port with discovery |
| Port discovery race (temp file) | Electron reads before Go writes | Retry loop with timeout |
| Go crashes, port not released | Next launch fails to bind | Random port avoids this; fixed port needs cleanup |
| Another process calls Go HTTP | Unauthorized operation or data access | Auth token per session |
| SSE connection drops mid-operation | Progress lost | Client reconnects; Go re-sends current operation state |

### Dev ergonomics

```bash
# Debug without Electron — excellent
curl http://127.0.0.1:37291/api/getDashboard | jq .
curl -N http://127.0.0.1:37291/api/operation/abc123/progress  # SSE stream
```

Best option for Go solo development. Developer doesn't need Electron running
to explore API behavior.

### Packaging/security

- Firewall: macOS will prompt user on first launch ("Do you want app to accept
  incoming network connections?") if Go opens a server socket. This prompt
  appears even for localhost. Bad first-run UX. Workaround: skip firewall
  registration for loopback, but this requires macOS entitlement config.
- Auth token: must be generated per-session, passed from Go to Electron at
  startup, included in every Electron→Go request, never exposed to renderer.
  Adds implementation surface.
- Signing: same as stdio. Binary must be signed. No additional entitlements
  beyond what Hardened Runtime normally requires.

---

## Option C: gRPC

### How it works

Go runs a gRPC server (usually over TCP localhost or unix socket). TypeScript
client uses `@grpc/grpc-js` in Electron main process. Proto files define the
service contract.

### For

- **Strongest typing.** Protobuf IDL is the contract. Generate Go server stubs
  and TypeScript client from the same `.proto` files. No drift possible.
- **Bidirectional streaming.** gRPC streaming RPCs handle operation progress
  natively without SSE/WebSocket/notifications.
- **Well-documented.** gRPC is industry standard. Plenty of tooling and docs.

### Against

- **Heavy toolchain.** `protoc` compiler, `protoc-gen-go`, `protoc-gen-go-grpc`,
  `@grpc/grpc-js`. Every developer needs the protobuf toolchain. CI needs it.
  Wrong protoc version produces subtly different code.
- **TypeScript in Electron renderer cannot use gRPC directly.** `@grpc/grpc-js`
  works in Node (Electron main), but not in the renderer (browser context).
  So you still need an IPC bridge from renderer to Electron main, and Electron
  main calls gRPC to Go. This is the same shape as stdio with extra steps.
- **Significant setup cost for ~20 methods.** The productivity gain from
  protobuf codegen pays off at scale. For Phase 1 with ~20 commands/queries,
  the setup cost outweighs the benefit.
- **Same port/security issues as HTTP.** gRPC over TCP has the same
  localhost exposure as HTTP. gRPC over unix socket avoids this but adds
  cross-platform complexity (Option D territory).
- **Proto evolution.** Adding a field to a proto message is backward-compatible,
  but changing types or removing fields breaks clients. For an internal IPC
  this is less of an issue but requires migration discipline.

### Dev ergonomics

```bash
# Requires grpcurl
grpcurl -plaintext -d '{}' 127.0.0.1:PORT skillbox.v1.DashboardService/GetDashboard
```

Worse than HTTP (needs specialized tool), better than stdio (no piping).

### Packaging/security

Same as HTTP for TCP. Same firewall prompt issue on macOS.
If gRPC over unix socket: same as Option D.

---

## Option D: Unix Socket / Named Pipe + JSON-RPC

### How it works

Go creates a unix socket at a temp path (e.g., `/tmp/skillbox-{pid}.sock`).
Electron connects via `net.Socket`. JSON-RPC 2.0 protocol over the socket.
On Windows: named pipe (`\\.\pipe\skillbox-{pid}`).

```
Electron main ──net.Socket──▶ Go net.Listen("unix", "/tmp/skillbox-123.sock")
Electron main ◀─────────────── Go writes JSON-RPC responses and notifications
```

### For

- **No network exposure.** Unix sockets are filesystem-level. Only processes
  with filesystem access to the socket path can connect. No firewall prompt.
  No network listener.
- **Better than stdio for multiple consumers.** Multiple connections can open
  the same socket. Future CLI tool can connect to a running Go server.
- **Same JSON-RPC protocol as Option A.** The protocol layer is identical.
  Only the transport differs. Migrating from stdio to unix socket later is
  a small change in Go and Electron main.
- **No stdout reservation.** Go can log to stdout normally; logs don't conflict
  with the protocol channel.
- **High performance.** Unix sockets are the fastest IPC on macOS/Linux.

### Against

- **Socket file lifecycle.** Go must:
  - Remove stale socket file from previous crash before binding.
  - Remove socket file on clean shutdown.
  - Handle the case where another process took the path.
  Electron must retry connection if Go hasn't created the socket yet
  (startup race).
- **Address discovery.** Electron needs to know the socket path. Options:
  - Go writes path to stdout (first line). Same as HTTP port discovery.
  - Fixed path: `/tmp/skillbox.sock`. Simpler but one socket for all instances
    (problematic if user runs two app instances, unlikely but possible).
  - `{pid}` in socket path: unique per session, Electron reads from stdout.
- **Cross-platform complexity.** macOS/Linux: `net.Listen("unix", path)`.
  Windows: named pipes have different API (`\\.\pipe\name`). Go's `net` package
  supports named pipes on Windows, but Electron's `net.Socket` needs `path`
  set to the pipe name. Different code paths or a cross-platform abstraction.
- **Cleanup on crash.** If Go crashes, socket file remains. Next launch must
  detect and remove it. Small but real robustness requirement.

### Failure modes

| Failure | Consequence | Mitigation |
|---|---|---|
| Stale socket file | Go fails to bind on next launch | Check + remove stale file at startup |
| Electron connects before socket ready | `ENOENT` on connect | Retry with backoff (max 5s) |
| Go crashes mid-write | Partial message, socket EOF | Electron detects EOF, shows error |
| Two app instances | Both try to use same socket (if fixed path) | Use pid in socket path |
| Windows named pipe cross-session | Session isolation issues | Use per-user pipe names |

### Dev ergonomics

```bash
# Test Go server standalone
socat - UNIX-CONNECT:/tmp/skillbox.sock <<< \
  '{"jsonrpc":"2.0","id":1,"method":"getDashboard","params":{}}'

# Or use nc (netcat) on some systems
```

Better than stdio for interactive debugging (connect from terminal without
piping), slightly worse than HTTP (no curl/Postman, need socat or custom tool).

### Packaging/security

- No firewall prompt. Unix socket is not a network interface.
- Socket path in temp directory: no special permissions needed.
- macOS: socket created in `$TMPDIR` (per-user temp). Correct.
- Windows: named pipe in user namespace. No admin required.
- Signing: same as all other options. No additional entitlements.

---

## Head-to-Head Comparison

| Criterion | stdio JSON-RPC | Local HTTP | gRPC | Unix socket |
|---|---|---|---|---|
| Setup complexity | Low | Medium | High | Medium |
| Port/address discovery | None needed | Required | Required | Required (socket path) |
| Server push (progress) | Native (notifications) | SSE or WebSocket | Native (streaming) | Native (notifications) |
| Debug without Electron | Pipe JSON to stdin | curl / Postman | grpcurl | socat |
| Security (localhost exposure) | None | Moderate risk | Moderate risk | None |
| macOS firewall prompt | No | Yes | Yes (TCP) / No (unix) | No |
| Multiple consumers | No | Yes | Yes | Yes |
| Cross-platform complexity | Low | Low | Medium | Medium (Windows pipes) |
| stdout discipline required | Yes | No | No | No |
| Startup race condition | None | Port not ready | Port not ready | Socket not ready |
| Codegen / contract | JSON Schema (manual) | OpenAPI possible | Protobuf native | JSON Schema (manual) |
| Phase 2 CLI reuse | Awkward | Easy | Easy | Easy |
| Zombie process risk | Low (pipe EOF) | Medium (needs explicit shutdown) | Medium | Medium (socket cleanup) |

---

## Recommendation

**Phase 1: stdio JSON-RPC 2.0. Migrate to unix socket in Phase 2 if CLI reuse or multi-window become real requirements.**

Reasoning:

1. **Security wins without configuration.** No network listener, no firewall
   prompt, no localhost exposure. For a desktop app that writes to the user's
   filesystem, reducing attack surface is worth the stdout discipline cost.

2. **Simplest startup.** No port discovery, no socket path negotiation, no
   connection retry loop. Electron spawns Go, streams are ready immediately.

3. **The LSP precedent is definitive.** VS Code runs gopls, tsserver, pyright
   over stdio JSON-RPC and ships to millions of users. The failure modes are
   known and the workarounds are documented.

4. **Stdout discipline is manageable.** The rule is simple: Go writes JSON-RPC
   to stdout, logs to stderr. Enforced by team convention and a CI check that
   scans for `fmt.Print*` calls in protocol-layer code. Libraries that write
   to stdout (rare) are caught at integration test time.

5. **Protocol is transport-agnostic.** JSON-RPC 2.0 works identically over
   stdio or unix socket. Migrating transport in Phase 2 means changing
   `os.Stdin`/`os.Stdout` to `net.Conn` in Go and `spawn().stdout` to
   `net.Socket` in Electron main. The JSON-RPC handler code doesn't change.

6. **HTTP is the wrong shape.** HTTP request/response semantics fit queries well
   but are awkward for progress streams (need SSE or WebSocket, which are
   separate connections). The macOS firewall prompt alone is a UX blocker.

7. **gRPC is out of scope.** The setup cost (protobuf toolchain, codegen CI)
   is not justified for ~20 internal API methods between two local processes.

---

## If We're Wrong

If stdio proves limiting in Phase 2, the migration path is:

```
Go side:    Replace os.Stdin/os.Stdout reader/writer with net.UnixConn
Electron:   Replace child.stdout stream with net.Socket connection
Protocol:   Unchanged (JSON-RPC 2.0 stays)
```

The JSON-RPC handler logic in Go (`handleRequest`, `handleNotification`)
does not need to change. The only change is the I/O source/sink.

Risk of being wrong: Low. Phase 1 has one window, user-triggered operations,
no background daemon, no CLI. All the stdio limitations are Phase 2+ concerns.

---

## Open Questions for Codex

1. **JSON-RPC library choice in Go:**
   - `sourcegraph/jsonrpc2`: mature, used in gopls. Last commit activity?
   - `creachadair/jrpc2`: actively maintained, clean API, supports bidirectional.
   - Custom minimal handler (~150 lines): no dependency, full control.
   Which does Codex prefer? Tradeoff: dependency vs maintenance burden.

2. **NDJSON vs Content-Length framing:**
   - NDJSON (one JSON per line): simpler, requires Go to never write bare
     newlines in string values. Go's json encoder handles this correctly.
   - LSP-style (`Content-Length: N\r\n\r\n{...}`): more robust, slightly more
     parsing code. What language servers use.
   Does Codex want the LSP framing for rigor, or NDJSON for simplicity?

3. **Concurrent request handling in Go:**
   - JSON-RPC 2.0 allows multiple in-flight requests. Go server can handle each
     in a goroutine. Responses may arrive out of order (matched by id).
   - Or: serialize all requests (one at a time). Simpler but slower if a query
     blocks behind a slow command.
   Recommend: concurrent goroutines per request, with operation locking at
   the service layer (not transport layer). Agree?

4. **Dev debug server:**
   - In dev mode only: Go could start a secondary HTTP debug server on a fixed
     port (e.g., `SKILLBOX_DEBUG_PORT=37291`) that mirrors the JSON-RPC API
     over HTTP. This gives curl/Postman access without changing production path.
   - Or: just accept that debug = pipe JSON to stdin.
   Does Codex want the debug HTTP server?

5. **Startup handshake:**
   - Should Go send a `ready` notification as the first message on stdout?
     `{"jsonrpc":"2.0","method":"server.ready","params":{"version":"0.1.0"}}`
   - Electron main waits for this before forwarding renderer requests.
   - Prevents race condition where Electron sends a request before Go's
     handlers are registered.
   Recommend: yes, implement a `server.ready` notification. Simple and safe.
