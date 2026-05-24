# Skillbox Scaffold + Slice 1 — Design

Date: 2026-05-25
Status: draft, pending user approval
Scope: scaffold toàn bộ monorepo + 1 vertical slice end-to-end (Skill Host Folder setup + Skills Library list)

## Goal

Đưa Astraler Skillbox từ docs-only sang một codebase chạy được:

- Monorepo skeleton (Electron + React + Go sidecar) compile và launch được.
- JSON-RPC handshake giữa Electron main và Go core hoạt động ổn định.
- Một use case hoàn chỉnh end-to-end: user chọn Skill Host Folder qua native dialog → app init `.agents/skills` → scan → hiển thị Skills Library với status, warnings, last-scanned-at.
- TDD strict cho Go core (services, repositories, filesystem gateway, operation runner). Test-after cho React UI, test-first cho `lib/core-client/`.
- Đặt nền tảng patterns (filesystem gateway, repository, operation runner, CQRS, error taxonomy, contract-first API) để slice 2+ chỉ cần extend, không cần refactor.

Non-goals trong slice này: packaging/signing, provider adapters, projects/installs/global skills, fetch/updates, mock-core fixtures đầy đủ, Playwright e2e, Windows support.

## Constraints

- Tech stack chốt theo `docs/11-tech-stack-and-scaffold-decisions.md`: Electron + electron-vite + electron-builder (defer), React, TanStack Router/Query, shadcn/ui + Radix + Tailwind, react-hook-form + Zod, Golang + modernc.org/sqlite + golang-migrate + zalando/go-keyring (defer) + creachadair/jrpc2.
- Architecture boundaries chốt theo `docs/10-technical-architecture.md` và 16 patterns trong `docs/12-implementation-patterns.md`.
- Schema SQLite phải khớp 100% với `docs/06-data-model.md` và `docs/07-schema-dictionary.md` (ID = integer auto-increment, enums theo docs).
- Electron security defaults bake từ M1: contextIsolation=true, nodeIntegration=false, sandbox=true (fallback false nếu incompatible), narrow preload bridge, CSP.
- Stdout của Go chỉ chứa JSON-RPC protocol bytes; logs đi stderr/file.
- Native dialog: Electron mở dialog trước, pass absolute path vào `host.choose`. Không reverse RPC từ Go sang Electron trong slice 1.

## Approach

Outside-in vertical slice: build mỏng end-to-end trước để de-risk integration, sau đó deepen từng layer TDD. 5 milestones sequential, mỗi milestone là 1 PR/branch, merge không squash để giữ history per layer.

## Milestones

### M1 — Walking Skeleton

Bootstrap monorepo (`apps/desktop/` + `core-go/`), wire JSON-RPC handshake hoạt động, 1 ping/pong method round-trip từ React → preload → Electron main → Go → trở lại.

**Repo layout tạo trong M1:**

```text
astraler-skillbox/
  pnpm-lock.yaml
  apps/desktop/
    package.json                    deps: electron, electron-vite, react, react-dom, vite
    tsconfig.json + tsconfig.node.json + tsconfig.web.json
    electron.vite.config.ts         3 targets: main, preload, renderer
    electron/main/
      index.ts                      app lifecycle, BrowserWindow, security defaults, CSP
      core-process/
        manager.ts                  spawn Go, NDJSON parse, server.ready handshake (10s timeout),
                                    SIGTERM→3s→SIGKILL on quit, restart up to 3 times mid-session
        method-allowlist.ts         {"ping"} cho M1; mở rộng ở M3
        ipc-bridge.ts               route invoke/onEvent giữa preload và Go (hoặc dialog handlers)
    electron/preload/
      index.ts                      contextBridge.exposeInMainWorld("core", {invoke, onEvent})
    renderer/
      index.html
      src/main.tsx                  React mount
      src/App.tsx                   1 button "Ping Go" + display response (M1 only)
      src/lib/core-client/
        client.ts                   typed wrapper trên window.core.invoke
        types.ts                    PingRequest/PingResponse (placeholder, M2 sẽ generate)
  core-go/
    go.mod                          module github.com/astraler/skillbox/core-go
    cmd/skillbox-core/main.go       entry: build app, register handlers, send server.ready,
                                    serve trên stdin/stdout
    internal/app/wire.go            composition root
    internal/rpc/server.go          creachadair/jrpc2 wiring với NDJSON channel
    internal/rpc/handlers/ping.go   trả {"pong": true, "ts": <iso8601>}
  scripts/
    build-go.sh                     go build -o dist/skillbox-core ./cmd/skillbox-core
  shared/                           folder placeholder (dùng từ M2)
  fixtures/                         folder placeholder (dùng từ M3)
```

**Handshake protocol:**

```text
1. Electron main spawn() Go bằng `go run ./cmd/skillbox-core` (dev mode);
   prod path defer sau M5.
2. Electron main start 10s timer chờ server.ready.
3. Go core: setup logger (stderr), khởi tạo jrpc2 với NDJSON channel,
   register "ping", gửi server.ready notification, Serve() blocking.
4. Electron main parse stdout từng line (NDJSON):
   - JSON-RPC valid → forward vào jrpc2 client
   - server.ready notification → clear timer, mở BrowserWindow
   - Non-parseable → log warning, skip
5. Timeout/exit trước server.ready → kill child, hiển thị blocking error window
   với stderr tail + log path + nút Quit. KHÔNG silent retry ở startup.
```

**Process lifecycle:**

```text
Mid-session    Go exit unexpected → restart count++, respawn nếu count ≤ 3.
               count == 4 → blocking error.
Quit           before-quit handler: SIGTERM → wait 3s → SIGKILL nếu còn alive.
Go SIGTERM     M1: ngay exit 0. M3+: mark running ops failed, close SQLite, exit 0.
```

**Method allowlist & IPC bridge:**

```text
electron/main/core-process/method-allowlist.ts:
  export const ALLOWLIST = new Set(["ping"]);    // M1

electron/main/core-process/ipc-bridge.ts:
  ipcMain.handle("core:invoke", async (_, method, params) => {
    if (!ALLOWLIST.has(method)) throw new Error(`method_not_allowed: ${method}`);
    return await goClient.call(method, params);
  });

  goClient.on("notification", (method, params) => {
    if (method.startsWith("operation.")) {
      mainWindow.webContents.send("core:event", method, params);
    }
  });

electron/preload/index.ts:
  contextBridge.exposeInMainWorld("core", {
    invoke: (method, params) => ipcRenderer.invoke("core:invoke", method, params),
    onEvent: (event, cb) => { /* register/unregister handler */ },
  });
```

**Electron security defaults (bake từ M1):**

```text
BrowserWindow.webPreferences:
  contextIsolation: true
  nodeIntegration: false
  sandbox: true                     fallback false nếu incompatible
  preload: <path>
CSP injected vào HTML head:
  default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
session.defaultSession.setPermissionRequestHandler: deny all
```

**Logging:**

```text
M1:
  Go stderr → Electron main capture → console.log + file write
  Path: app.getPath('logs') + '/core.log'
  Format: [skillbox-core][timestamp][level] message
Electron main logs vào cùng folder: main.log
```

**Tests trong M1:**

```text
Go (TDD):
  internal/rpc/handlers/ping_test.go        ping trả pong:true với timestamp hợp lệ
  internal/app/wire_test.go                 wire trả server có "ping" registered

Electron (test-after, wire 1 test sớm để chứng minh harness):
  electron/main/core-process/manager.test.ts  mock spawn, verify timeout + SIGTERM logic
  renderer/src/lib/core-client/client.test.ts  mock window.core, verify invoke wrapper
```

**Acceptance:**

```text
[ ] pnpm dev mở Electron window, Go sidecar được spawn, server.ready ≤ 10s
[ ] Click "Ping" trong renderer → hiển thị {pong:true, ts:"..."} ≤ 500ms
[ ] Cmd+Q → process Go không leak
[ ] Stdout Go chỉ chứa JSON-RPC bytes (verify qua log dump)
[ ] Sửa Go panic startup → blocking error window xuất hiện với stderr tail
[ ] go test ./... xanh; pnpm test xanh
```

### M2 — API Contracts Cho Slice 1

JSON Schema cho 3 RPC methods + 2 notifications + shared types + 1 Electron-handled method. Generate TypeScript types vào `shared/generated/`. CI script check drift.

**File layout:**

```text
shared/api-contracts/
  README.md                       naming conventions, versioning, contributing
  index.json                      manifest list tất cả schemas
  package.json                    scripts: generate, validate-drift
  methods/
    host.choose.json
    host.scan.json
    skill.list.json
    operation.cancel.json
  notifications/
    server.ready.json
    operation.progress.json
  shared/
    operation.json                OperationId, OperationStatus enum
    error.json                    Error code taxonomy
    skill.json                    Skill entity shape cho view model
    warning.json                  Warning shape cho view model
  electron/
    dialog.openHostFolder.json    Namespace riêng, đánh dấu "Electron-handled,
                                  không forward sang Go"

shared/generated/
  index.ts
  methods/{host-choose, host-scan, skill-list, operation-cancel}.ts
  notifications/{server-ready, operation-progress}.ts
  shared/{operation, error, skill, warning}.ts
  electron/{dialog-open-host-folder}.ts
```

**Method shapes:**

```text
host.choose
  Request: { path: string }       // absolute path từ Electron native dialog
  Response: {
    hostId: number,
    path: string,
    skillsPath: string,           // <path>/.agents/skills
    initialized: boolean,         // true nếu .agents/skills vừa tạo
    status: "active" | "missing" | "unreadable" | "unwritable"
          | "invalid_structure" | "empty" | "inactive"
  }
  Errors: validation_error (path không absolute / không tồn tại / không phải dir / không writable)
          filesystem_error (không tạo được .agents/skills)

host.scan
  Request: { hostId: number }
  Response: { operationId: number }   // long-running, progress qua notification
  Errors: validation_error (hostId không tồn tại)
          conflict_error (host này đã có scan operation active)

operation.cancel
  Request: { operationId: number }
  Response: { acknowledged: boolean }   // true nếu operation tồn tại và cancel signal được gửi;
                                        // false nếu operation đã kết thúc (success/failed/cancelled)
  Errors: validation_error (operationId không tồn tại)

skill.list
  Request: { hostId: number }
  Response: {
    hostPath: string,
    skills: Array<{
      id: number, name: string, relativePath: string,
      status: "available" | "missing" | "unreadable" | "local_modified" | "unknown",
      sourceLabel: string | null, lastScannedAt: string | null
    }>,
    totals: { available: number, missing: number, unreadable: number, local_modified: number, unknown: number },
    lastScanAt: string | null,
    warnings: Array<{ code: string, message: string, scopeRef: string | null }>
  }
  Errors: validation_error (hostId không tồn tại)
```

**Notifications:**

```text
server.ready
  { version: string, pid: number, capabilities: string[] }

operation.progress
  {
    operationId: number,
    status: "queued" | "running" | "success" | "failed" | "cancelled" | "partial",
    phase: string,                vd "reading_host_folder", "classifying_entries", "done"
    processed: number | null,
    total: number | null,
    message: string | null
  }
```

**Electron-handled method:**

```text
dialog.openHostFolder
  Request: {}
  Response: { path: string | null }       null nếu user cancel
  Handled bởi: Electron main (KHÔNG forward sang Go)
```

**Error taxonomy + JSON-RPC code mapping (tránh reserved -32768..-32000):**

```text
"validation_error" | "filesystem_error" | "provider_error" | "database_error"
| "auth_error" | "network_error" | "conflict_error" | "operation_cancelled"
| "user_cancelled" | "unknown_error"

validation_error    = 1001
filesystem_error    = 1002
provider_error      = 1003          (chưa dùng, định nghĩa sẵn)
database_error      = 1004
conflict_error      = 1005
user_cancelled      = 1006
operation_cancelled = 1007
unknown_error       = 1099

error.data: {
  code: <error_code_string>,
  userMessage: string,        UI hiển thị
  technicalMessage: string,   log
  operationId?: number,
  entityRef?: string
}
```

**Method routing trong Electron main:**

```text
DIALOG_METHODS = new Set(["dialog.openHostFolder"])
GO_FORWARDED   = new Set(["ping", "host.choose", "host.scan", "skill.list", "operation.cancel"])

ipcMain.handle("core:invoke", (_, method, params) => {
  if (DIALOG_METHODS.has(method)) return handleDialog(method, params);
  if (GO_FORWARDED.has(method))   return goClient.call(method, params);
  throw new Error(`method_not_allowed: ${method}`);
});
```

**Generation pipeline:**

```text
Tool: json-schema-to-typescript (npm)

apps/desktop/package.json scripts:
  "generate:contracts":      "node scripts/generate-contracts.mjs"
  "check:contracts-drift":   "node scripts/generate-contracts.mjs --check"

scripts/generate-contracts.mjs:
  - đọc shared/api-contracts/index.json
  - cho mỗi schema, compile bằng json-schema-to-typescript
  - write vào shared/generated/<corresponding>.ts
  - --check mode: generate vào temp, diff với committed, exit 1 nếu khác
```

**Contract tests (Go side, chạy ở M3):**

```text
core-go/internal/rpc/handlers/contract_test.go
  - Load tất cả schemas từ shared/api-contracts qua go:embed
  - Cho mỗi method: build sample valid request, gọi handler,
    validate response payload qua jsonschema/v5
```

**Decisions M2:**

- Go structs hand-written Phase 1, defer codegen sang Phase 2 nếu drift đau.
- Không add `$id` versioning. Khi nào break, tăng version trong file name (`host.choose.v2.json`).
- `additionalProperties: false` ở mọi response (catch typo). Request mở (forward compat).

**Acceptance:**

```text
[ ] 3 method schemas + 2 notification schemas + 4 shared types + 1 electron schema committed
[ ] pnpm generate:contracts chạy được, regenerate giống file đã commit
[ ] Drift check fail nếu sửa schema mà chưa regen
```

### M3 — Go Core Slice 1 (TDD Layer-By-Layer)

Build order: domain → migrations + skeleton → filesystem gateway → repositories → operation runner → application services → JSON-RPC handlers → wire trong main.

#### Step 3.1 — Domain layer

```text
core-go/internal/domain/
  errors.go         AppError struct với taxonomy khớp M2
                    factory: NewValidationError, NewFilesystemError, ...
  errors_test.go    verify Error() format, JSON marshal khớp shared/error.json
  app_settings.go   AppSettings { ActiveSkillHostFolderId *int64,
                                  DefaultInstallMode, DatabaseVersion,
                                  CreatedAt, UpdatedAt }
  skill_host.go     SkillHostFolder + SkillHostStatus enum:
                    active|missing|unreadable|unwritable|invalid_structure|empty|inactive
  skill.go          Skill + SkillStatus enum:
                    available|missing|unreadable|local_modified|unknown
  operation.go      Operation + OperationStatus (queued|running|success|failed|cancelled|partial)
                              + OperationType (slice 1: scan, change_skill_host_folder)
  warning.go        Warning + WarningScopeType (slice 1: app, skill_host_folder, skill)
                            + WarningSeverity (info|warning|error|blocking)

KHÔNG dùng UUID. ID = int64. Domain pure Go, no external deps.
```

#### Step 3.2 — SQLite migrations + skeleton

```text
core-go/migrations/0001_init.sql
  Bảng cho slice 1 (khớp 100% docs/07):
    app_settings              singleton row (id=1, default_install_mode='symlink',
                                              database_version=1)
    skill_host_folders        full schema theo docs/07
    skills                    full schema theo docs/07 (source_id nullable, slice 1 không seed)
    operations                full schema theo docs/07 (polymorphic target)
    warnings                  full schema theo docs/07 (polymorphic scope)

  DEFER cho slice 2+:
    api_credentials, projects, provider_definitions, provider_path_candidates,
    project_providers, global_provider_locations, installs, global_installs,
    fetch_results, skill_sources, scan_results

  Note: scan_results defer. Slice 1 lưu scan summary trong operations.metadata_json.

  Indexes:
    CREATE INDEX idx_skills_host ON skills(skill_host_folder_id);
    CREATE UNIQUE INDEX uq_skills_host_relpath ON skills(skill_host_folder_id, relative_path);
    CREATE INDEX idx_operations_target ON operations(target_type, target_id, status);
    CREATE INDEX idx_warnings_scope ON warnings(scope_type, scope_id, is_resolved);
    CREATE UNIQUE INDEX uq_skill_host_path ON skill_host_folders(path);

core-go/internal/repositories/db.go
  Open() function:
    - sql.Open("sqlite", path) qua modernc.org/sqlite
    - Apply PRAGMAs: journal_mode=WAL, foreign_keys=ON,
                     busy_timeout=5000, synchronous=NORMAL
    - Run migrations qua golang-migrate
    - Return *sql.DB

core-go/internal/repositories/db_test.go
  - Open() với temp DB path
  - Migrations chạy idempotent
  - PRAGMA verify sau Open: journal_mode='wal', foreign_keys=1
```

#### Step 3.3 — Filesystem gateway

```text
core-go/internal/filesystem/
  gateway.go        Gateway struct (inject allowed roots policy)
  paths.go          NormalizeAbs(path), Realpath(path)
  validate.go       ValidateHostPath(path):
                      absolute? exists? is dir? writable?
                      → FilesystemError với sub-code nếu fail
  scan.go           ScanHostFolder(absPath) → []HostEntry
                    HostEntry { Name, RelativePath, IsDir, IsSymlink,
                                SymlinkTarget, Broken, External }
  init_host.go      EnsureAgentsSkills(hostPath) → (created bool, err)
  errors.go         FilesystemError + sub-codes:
                    PathNotFound, PermissionDenied, NotADirectory,
                    OutsideAllowedRoot, NotWritable

  gateway_test.go         constructor, allowed roots validation
  validate_test.go        từng failure mode dùng t.TempDir() fixtures
  scan_test.go            t.TempDir() + dirs/files/symlinks (broken/external/valid),
                          verify output
  init_host_test.go       fresh folder → create; lần 2 → no-op
  paths_test.go           Normalize: trailing slash, ../, symlink edge cases

Slice 1 chỉ cần: scan, init, normalize, realpath, validate.
Hoãn: copy folder, symlink create/remove, remove managed install — slice 2.
```

#### Step 3.4 — Repositories

```text
core-go/internal/repositories/
  app_settings_repo.go       Get() *AppSettings    (singleton id=1)
                             UpdateActiveHost(hostId *int64)
                             UpdateDefaultInstallMode(mode string)
  skill_host_folder_repo.go  Insert(SkillHostFolder) → int64
                             GetByID(id) → *SkillHostFolder
                             GetByPath(path) → *SkillHostFolder | nil
                             GetActive() → *SkillHostFolder | nil  (join app_settings)
                             SetActive(hostId)            // TRANSACTION:
                                                          // clear host cũ → inactive,
                                                          // set host mới → active,
                                                          // update app_settings
                             UpdateStatus(id, status)
                             UpdateLastScannedAt(id, t)
  skill_repo.go              UpsertMany(hostId, []Skill) trong transaction
                             ListByHost(hostId) → []Skill
                             MarkMissing(hostId, presentIds) trong transaction
                             ListIDsByHost(hostId) → []int64
  operation_repo.go          Insert(targetType, targetId, opType) → int64
                             UpdateStatus(id, status, errMsg, metadataJSON, finishedAt)
                             GetByID(id) → *Operation
                             ListActiveByTarget(targetType, targetId) → []Operation
  warning_repo.go            Insert(Warning) → int64
                             ListByScope(scopeType, scopeId, includeResolved bool)
                             ClearByScope(scopeType, scopeId)   // mark all active resolved

  *_test.go                  helper NewTestDB() trả temp *sql.DB đã migrate,
                             t.Cleanup() đóng DB; mỗi test DB riêng
```

#### Step 3.5 — Operation runner

```text
core-go/internal/operations/
  runner.go         Runner struct
                    Start(ctx, target Target, opType OperationType,
                          fn func(ctx, progressFn) (metadata any, err error)) → operationId
                    Cancel(operationId)
                    GetStatus(operationId)
  target.go         Target { Type string, ID int64 }
                    vd {Type:"skill_host_folder", ID:42}
  locks.go          per-target mutex map; Lock fail-fast → conflict_error
  progress.go       ProgressFn type: func(phase string, processed, total int, msg string)
                    Runner inject channel → Step 3.7 wire vào JSON-RPC notification

  runner_test.go    happy path: Start → fn chạy → status=success, metadata persisted
                    fail path:  fn return error → status=failed, error mapped
                    cancel:     cancel khi đang chạy → status=cancelled, fn nhận ctx.Done()
                    lock:       2 Start cùng target → cái thứ 2 conflict_error
                    panic:      fn panic → status=failed, không crash runner

  -race flag bật cho package này.

Status mapping: success/failed/cancelled (KHÔNG "succeeded").
```

#### Step 3.6 — Application services

```text
SkillHostService (constructor inject: hostRepo, appSettings, fs, runner)
  ChooseHost(ctx, path string) → ChooseHostResult
    1. fs.ValidateHostPath(path) → validation_error nếu sai
    2. initialized := fs.EnsureAgentsSkills(path) → filesystem_error nếu fail
    3. Transaction:
       existing := repo.GetByPath(path)
       if existing == nil:
         hostId := repo.Insert(SkillHostFolder{
           Name: filepath.Base(path), Path: path,
           SkillsPath: filepath.Join(path, ".agents/skills"),
           Status: "active"
         })
       else:
         hostId := existing.ID
         repo.UpdateStatus(hostId, "active")
       currentActive := appSettings.Get().ActiveSkillHostFolderId
       if currentActive != nil && *currentActive != hostId:
         repo.UpdateStatus(*currentActive, "inactive")
       appSettings.UpdateActiveHost(&hostId)
    4. Return { HostId, Path, SkillsPath, Initialized, Status }

  Idempotent theo path. Switch host inline. KHÔNG conflict_error cho host.choose.

  ScanHost(ctx, hostId) → operationId
    Validate hostId tồn tại.
    Runner.Start(target={skill_host_folder, hostId}, opType=scan, fn=scanHostInternal)

  scanHostInternal(ctx, progress) → (summary, err)
    host := repo.GetByID(hostId)
    progress("reading_host_folder", 0, 0, "")
    entries := fs.ScanHostFolder(host.SkillsPath)
    progress("classifying_entries", len(entries), len(entries), "")
    skills := convertEntriesToSkills(entries, host)
    warnings := generateWarnings(entries, host)
    Transaction:
      skillRepo.UpsertMany(hostId, skills)
      skillRepo.MarkMissing(hostId, presentIds)
      hostRepo.UpdateLastScannedAt(hostId, now)
      warningRepo.ClearByScope("skill_host_folder", hostId)
      for w := range warnings: warningRepo.Insert(w)
    progress("done", len(skills), len(skills), "")
    summary := { skillsFound, warningsCreated, ... }
    return summary, nil   // Runner persist metadata vào operations.metadata_json

SkillLibraryService (constructor inject: skillRepo, hostRepo, warningRepo)
  List(ctx, hostId) → SkillsLibraryView
    Validate hostId.
    skills := skillRepo.ListByHost(hostId)
    warnings := warningRepo.ListByScope("skill_host_folder", hostId, false)
    totals := count theo status
    Return view model khớp skill.list response schema.

*_test.go
  - Mock fs, repos qua interface
  - ChooseHost: happy, validation_error, filesystem_error, switch-host transaction
  - ScanHost: operation queued, fn gọi đúng order, progress emitted, metadata persisted
  - List: empty host, host với skills, host với warnings
```

#### Step 3.7 — JSON-RPC handlers + contract tests

```text
core-go/internal/rpc/handlers/
  host_choose.go        wrap SkillHostService.ChooseHost
  host_scan.go          wrap SkillHostService.ScanHost
  skill_list.go         wrap SkillLibraryService.List
  operation_cancel.go   wrap Runner.Cancel
  ping.go               (M1, giữ cho health check)
  contract_test.go      go:embed shared/api-contracts/,
                        cho mỗi handler validate response qua JSON Schema

core-go/internal/rpc/notifications/
  progress_dispatcher.go
    Subscribe Runner progress channel → forward operation.progress qua jrpc2.Notify
    Validate notification payload qua schema (bật khi test)

Method allowlist update trong Electron main:
  ["ping", "host.choose", "host.scan", "skill.list", "operation.cancel"]
```

#### Step 3.8 — Wire trong cmd/skillbox-core

```text
core-go/cmd/skillbox-core/main.go
  1. Setup logger (stderr, slog)
  2. Resolve DB path (env SKILLBOX_DB_PATH hoặc OS app data dir)
  3. db := repositories.OpenDatabase(dbPath)
  4. fs := filesystem.NewGateway()
  5. operationRepo, hostRepo, skillRepo, warningRepo, appSettingsRepo
  6. runner := operations.NewRunner(operationRepo)
  7. hostService := services.NewSkillHostService(hostRepo, appSettingsRepo, fs, runner)
  8. libraryService := services.NewSkillLibraryService(skillRepo, hostRepo, warningRepo)
  9. rpcServer := rpc.NewServer()
 10. rpcServer.Register("ping", ping.Handler)
 11. rpcServer.Register("host.choose", host_choose.NewHandler(hostService))
 12. rpcServer.Register("host.scan", host_scan.NewHandler(hostService))
 13. rpcServer.Register("skill.list", skill_list.NewHandler(libraryService))
 14. rpcServer.Register("operation.cancel", operation_cancel.NewHandler(runner))
 15. progressDispatcher.Subscribe(runner, rpcServer)
 16. rpcServer.Notify("server.ready", {version, pid, capabilities})
 17. rpcServer.Serve(stdin, stdout)

Defer:
  db.Close()
  SIGTERM handler: runner.MarkAllRunningAsFailed("shutdown"), db.Close, exit 0
  panic recovery ở top level → log stack trace stderr → exit 1
```

**Manual smoke test M3 (standalone, không qua Electron):**

```text
1. SKILLBOX_DB_PATH=/tmp/skillbox-test.db go run ./cmd/skillbox-core
2. Send qua stdin (NDJSON):
   {"jsonrpc":"2.0","id":1,"method":"ping","params":{}}
   {"jsonrpc":"2.0","id":2,"method":"host.choose","params":{"path":"/tmp/test-host"}}
   {"jsonrpc":"2.0","id":3,"method":"host.scan","params":{"hostId":1}}
   wait operation.progress notifications
   {"jsonrpc":"2.0","id":4,"method":"skill.list","params":{"hostId":1}}
3. Verify response shape match schema
4. sqlite3 /tmp/skillbox-test.db ".schema" — verify migrations
5. sqlite3 ... "SELECT * FROM skills" — verify rows
```

**Acceptance M3:**

```text
[ ] Mỗi layer có test trước implementation
[ ] go test -race ./... xanh
[ ] Contract tests pass (response validate qua JSON Schema)
[ ] Manual smoke standalone pass đầy đủ
```

### M4 — React Slice 1

Skill Host setup screen + Skills Library screen + Settings (minimal) + app shell. TanStack Router (memory) + TanStack Query. Test-after cho UI, test-first cho `lib/core-client/`.

**Deps mới (apps/desktop/package.json):**

```text
@tanstack/react-router @tanstack/router-devtools
@tanstack/react-query @tanstack/react-query-devtools
react-hook-form @hookform/resolvers zod
tailwindcss @tailwindcss/vite
lucide-react
class-variance-authority clsx tailwind-merge

shadcn/ui init (style="default", baseColor="zinc", cssVariables=true)
components đầu: Button, Card, Dialog, AlertDialog, Badge, Skeleton,
                Tooltip, ScrollArea, Separator, Sonner (toast)
```

**Routes (createMemoryRouter):**

```text
/                redirect → /skills nếu có active host, → /setup nếu chưa
/setup           First-time setup wizard
/skills          Skills Library
/settings        Active host + Change Host Folder
```

**Folder layout sau M4:**

```text
apps/desktop/renderer/src/
  main.tsx                            mount providers
  app/
    router.tsx                        route tree
    query-client.ts                   QueryClient config
    providers.tsx                     QueryClientProvider, RouterProvider, Toaster
  lib/
    core-client/
      client.ts                       typed invoke wrapper
      methods.ts                      per-method typed wrappers
      progress.ts                     subscribeOperationProgress(id, cb)
    query-keys.ts                     skills.list(hostId), settings.app
  features/
    skill-host/
      use-choose-host.ts              useMutation
      use-active-host.ts              useQuery for settings + active host
      use-scan-host.ts                useMutation + operationId tracking
    skills-library/
      use-skills-list.ts              useQuery
      skill-status-badge.tsx          Badge variants per status enum
      skill-row.tsx
  screens/
    setup-screen.tsx
    skills-library-screen.tsx
    settings-screen.tsx
  components/
    app-shell.tsx                     Sidebar + content
    sidebar.tsx                       Navigation: Skills, Settings
    operation-progress-toast.tsx
    error-display.tsx                 Render AppError (userMessage + retry)
    empty-state.tsx
  styles/globals.css
```

**Screen specs:**

```text
/setup
  Render khi appSettings.activeSkillHostFolderId == null
  Centered card, không sidebar
  - Heading "Choose your Skill Host Folder"
  - Button "Choose Folder":
    1. await methods.openHostFolder()    (Electron dialog)
    2. nếu cancel → no-op
    3. await methods.chooseHost({path})
    4. invalidate queries: app.settings, skills.list
    5. router.navigate('/skills')
  - Error display nếu fail

/skills (AppShell)
  Header: "Skills" + scan status pill + button "Rescan"
  Body:
    Loading: Skeleton rows
    Empty: EmptyState với hint
    List: ScrollArea + SkillRow[] (name, relativePath, StatusBadge, lastScannedAt)
    WarningBanner ở trên list nếu warnings.length > 0
  Rescan action:
    methods.scanHost({hostId}) → operationId
    subscribe progress → OperationProgressToast
    khi nhận success/failed/cancelled → invalidate skills.list → close toast

/settings (minimal slice 1)
  Section "Skill Host":
    Path (monospace), Status badge, Last scanned at
    Button "Change Host Folder" → reuse setup flow
```

**core-client wrapper (test-first):**

```text
lib/core-client/client.ts
  invoke<TReq, TRes>(method, params): Promise<TRes>
  - validate window.core exists
  - error response → throw AppClientError extends Error
                     với code, userMessage, technicalMessage

lib/core-client/methods.ts
  export const methods = {
    openHostFolder:  () => invoke<{}, {path: string|null}>("dialog.openHostFolder", {}),
    chooseHost:      (req) => invoke<ChooseHostRequest, ChooseHostResponse>("host.choose", req),
    scanHost:        (req) => invoke<ScanHostRequest, ScanHostResponse>("host.scan", req),
    listSkills:      (req) => invoke<SkillListRequest, SkillListResponse>("skill.list", req),
    cancelOperation: (req) => invoke<{operationId: number}, void>("operation.cancel", req),
  };
  TS types import từ shared/generated/.

lib/core-client/progress.ts
  subscribeOperationProgress(operationId, onProgress):
    return window.core.onEvent("operation.progress", (params) => {
      if (params.operationId === operationId) onProgress(params);
    });

Tests:
  client.test.ts     invoke args, error mapping, missing window.core
  methods.test.ts    mock invoke, verify params shape khớp schema
  progress.test.ts   mock onEvent, verify filter theo operationId
```

**QueryClient config:**

```text
new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      gcTime: 5 * 60 * 1000,
      retry: (failureCount, error) => {
        if (error.code === 'validation_error') return false;
        if (error.code === 'conflict_error')   return false;
        return failureCount < 1;
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      onError: (error) => toast.error(error.userMessage),
    },
  },
})
```

**Test scope M4:**

```text
Test-first (TDD): lib/core-client/*
Test-after: features/*/use-*.test.tsx (mocked methods, verify invalidate + error)
Screen tests defer trừ khi cần verify acceptance.
Playwright defer.
```

**Acceptance M4:**

```text
[ ] Fresh install → /setup screen
[ ] Choose Folder dialog → chọn /tmp/test-host → navigate /skills
[ ] /tmp/test-host/.agents/skills/ tạo được sau chọn
[ ] Tạo foo/bar trong .agents/skills → Rescan → /skills hiển thị foo/bar/Available
[ ] rm foo → Rescan → foo status=Missing
[ ] ln -s /nonexistent broken → Rescan → WarningBanner hiển thị broken symlink
[ ] Cmd+Q → reopen → /skills với active host nhớ từ DB
[ ] /settings → Change Host Folder → chọn host khác → /skills refresh, host cũ inactive
```

### M5 — End-To-End Smoke + Scaffold Docs

Viết SMOKE.md (checklist manual đầy đủ) và SCAFFOLD.md (setup, 3 dev modes, troubleshooting). Tag commit `slice-1-skills-library`.

**SMOKE.md** — checklist manual chi tiết, viết để chạy được mà không cần đọc spec này:

```text
Pre-conditions:
  - Fresh clone
  - rm -rf ~/Library/Application Support/Astraler\ Skillbox/   (clean DB)
  - macOS hoặc Linux test env

Setup smoke:
  [ ] pnpm install hoàn tất < 2 phút
  [ ] cd core-go && go mod download hoàn tất < 1 phút
  [ ] pnpm dev mở app window trong < 10s
  [ ] Console không có lỗi đỏ
  [ ] core.log có line server.ready received

Handshake smoke:
  [ ] App render /setup screen (chưa có active host)
  [ ] core.log có line server.ready với version + pid
  [ ] Stdout dump Go không chứa non-JSON-RPC bytes

Choose host smoke:
  [ ] Tạo /tmp/skillbox-test-host/ (folder rỗng)
  [ ] Click "Choose Folder" → dialog mở
  [ ] Chọn /tmp/skillbox-test-host → app navigate /skills
  [ ] /tmp/skillbox-test-host/.agents/skills/ đã được tạo
  [ ] sqlite3 ...skillbox.db "SELECT * FROM skill_host_folders" → 1 row, status='active'
  [ ] sqlite3 ... "SELECT active_skill_host_folder_id FROM app_settings" → not null

Scan smoke:
  [ ] mkdir /tmp/skillbox-test-host/.agents/skills/{foo,bar,baz}
  [ ] Click Rescan
  [ ] Toast hiển thị phases (reading_host_folder → classifying_entries → done)
  [ ] /skills hiển thị 3 rows foo/bar/baz status=Available
  [ ] sqlite3 ... "SELECT name,status FROM skills" → 3 rows available
  [ ] sqlite3 ... "SELECT operation_type,status,metadata_json FROM operations
                   ORDER BY id DESC LIMIT 1" → scan/success/{skillsFound:3,...}

Reconcile smoke:
  [ ] rm -rf /tmp/skillbox-test-host/.agents/skills/foo
  [ ] Click Rescan → foo hiển thị status=Missing
  [ ] sqlite3 ... skills WHERE name='foo' → status='missing'

Warning smoke:
  [ ] ln -s /nonexistent /tmp/skillbox-test-host/.agents/skills/broken
  [ ] Click Rescan → WarningBanner hiển thị broken symlink
  [ ] sqlite3 ... warnings → 1 row scope_type=skill_host_folder, code='broken_symlink'

Switch host smoke:
  [ ] mkdir /tmp/skillbox-test-host-2/.agents/skills/qux
  [ ] /settings → Change Host Folder → chọn /tmp/skillbox-test-host-2
  [ ] /skills hiển thị qux only
  [ ] sqlite3 ... skill_host_folders: host cũ inactive, host mới active,
                   app_settings.active_skill_host_folder_id trỏ host mới

Lifecycle smoke:
  [ ] Cmd+Q → ps aux | grep skillbox → không còn process Go
  [ ] Reopen → /skills với active host nhớ từ DB
  [ ] kill -9 <go_pid> khi app đang chạy → Electron restart Go, server.ready lại
  [ ] Force-kill 4 lần → blocking startup error window

Validation smoke:
  [ ] Setup screen, dialog trả path file (không phải dir) →
      Error toast "Path must be a directory" (validation_error)
  [ ] Choose folder không có write permission →
      Error toast "Cannot create .agents/skills" (filesystem_error)
```

**SCAFFOLD.md** — coverage:

```text
Prerequisites: Node 20+, pnpm 9+, Go 1.22+, macOS/Linux (Windows defer)
Install: pnpm install, cd core-go && go mod download

Three dev modes:
  Full-stack (default):     pnpm dev
  Go-only (TDD):            cd core-go && go test -race ./...
                            SKILLBOX_DB_PATH=/tmp/dev.db go run ./cmd/skillbox-core
  UI-only (mock core):      SKILLBOX_USE_MOCK_CORE=1 pnpm dev
                            (flag wired, fixtures minimal trong slice 1)

Database:
  Default: ~/Library/Application Support/Astraler Skillbox/skillbox.db
  Override: SKILLBOX_DB_PATH=/tmp/test.db pnpm dev
  Inspect: sqlite3 <path>
  Reset: rm -rf folder

Logs:
  ~/Library/Logs/Astraler Skillbox/main.log     Electron main
  ~/Library/Logs/Astraler Skillbox/core.log     Go core stderr

Contracts:
  pnpm generate:contracts        regenerate TS từ JSON Schema
  pnpm check:contracts-drift     CI check

Tests:
  pnpm test                      Vitest
  cd core-go && go test ./...    Go
  cd core-go && go test -race ./internal/operations/...

Troubleshooting:
  "server.ready timeout"         check core.log; thường do go binary build fail
  "method_not_allowed"           method thiếu trong ALLOWLIST
  SQLite "database is locked"    check WAL pragma applied
```

**Branch strategy gợi ý:**

```text
main = stable scaffold
Mỗi milestone là 1 PR/branch:
  feat/m1-walking-skeleton
  feat/m2-contracts
  feat/m3-go-slice-1
  feat/m4-react-slice-1
  feat/m5-smoke-and-docs
Merge sequential, không squash để giữ history per layer.
```

**Acceptance M5:**

```text
[ ] SMOKE.md checklist pass 100% trên macOS
[ ] Fresh clone + pnpm install + pnpm dev → app chạy được trong ≤ 5 phút
[ ] SCAFFOLD.md cover đủ 3 dev modes
[ ] Tag slice-1-skills-library push lên remote
```

## What's Deferred Sau Slice 1

```text
electron-builder packaging + signing/notarization
Hot reload cho Go (air)
Mock-core fixtures đầy đủ
Windows support
Playwright e2e
Provider adapters (Claude, Codex, ...)
projects table + add project flow
installs (symlink/rsync-copy) — slice 2
global_skills — slice 3
fetch/updates — slice 4
api_credentials + keychain integration
Multi-host UI
CLI layer
Skill format conversion
```

## Open Risks

```text
R1  jrpc2 NDJSON framing
    Cần verify creachadair/jrpc2 hỗ trợ NDJSON channel hoặc cần custom channel.
    Spike 1-2 giờ đầu M1; nếu không khả thi → fallback custom channel hoặc
    library khác (sourcegraph/jsonrpc2).

R2  SQLite WAL trong sandbox
    modernc.org/sqlite + WAL trong macOS sandbox cần verify journal file (.wal)
    ghi được trong sandbox dir. Test sớm trong Step 3.2.

R3  Electron sandbox=true compatibility
    Một số preload patterns có thể incompatible. Fallback sandbox=false nếu cần,
    log decision rõ ràng.

R4  Native dialog absolute path consistency
    Trên Linux có thể không trả absolute path consistent. fs.ValidateHostPath
    phải handle.
```

## Decisions Log

```text
2026-05-25  Approach C (outside-in vertical slice) chốt
2026-05-25  M0+M1 gộp thành "walking skeleton" milestone
2026-05-25  Test discipline: TDD strict Go, test-after UI, test-first lib/core-client
2026-05-25  Solo sequential execution; mỗi milestone là 1 PR
2026-05-25  Native dialog: Electron mở trước, pass path vào host.choose
            (KHÔNG reverse RPC trong slice 1)
2026-05-25  ID = integer auto-increment (theo docs/06-07), không UUID
2026-05-25  scan_results table defer; summary trong operations.metadata_json
2026-05-25  host.choose idempotent theo path, switch host inline,
            KHÔNG conflict_error (chỉ validation_error / filesystem_error)
2026-05-25  dialog.openHostFolder contract ở shared/api-contracts/electron/
            namespace riêng, không trộn vào methods/ Go RPC
```
