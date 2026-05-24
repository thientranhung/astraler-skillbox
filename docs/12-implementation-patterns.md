# Implementation Patterns

Tài liệu này chốt các pattern sẽ dùng khi implement Astraler Skillbox. Mục tiêu
là biến architecture trong `10-technical-architecture.md` và tech stack trong
`11-tech-stack-and-scaffold-decisions.md` thành quy tắc code cụ thể.

## Pattern Principles

- Pattern phục vụ boundary, không phải để làm code phức tạp hơn.
- UI không sở hữu business rules, filesystem writes, SQLite, provider logic.
- Electron main không chứa nghiệp vụ; nó giữ lifecycle, preload bridge, native
  dialogs, và allowlist.
- Go core là nơi giữ command/query handlers, services, repositories, provider
  adapters, filesystem gateway, operation runner.
- Mọi operation có side effect phải có validation, audit/log, và error path rõ.

## 1. Process Coordinator

Nơi áp dụng:

```text
apps/desktop/electron/main/
apps/desktop/electron/core-process/
```

Responsibility:

- Spawn Go sidecar bằng `spawn()`, không dùng `exec()`.
- Parse stdout như JSON-RPC NDJSON protocol stream.
- Forward responses/notifications an toàn tới renderer qua preload bridge.
- Read stderr như log stream.
- Chờ `server.ready` tối đa 10 giây.
- Nếu Go exit trước `server.ready` hoặc timeout, show blocking startup error.
- Khi app quit, gửi SIGTERM, chờ 3 giây, rồi SIGKILL nếu cần.
- Mid-session crash có thể restart tối đa 3 lần, sau đó show blocking error.

Không làm:

- Không chứa business logic.
- Không đọc/ghi SQLite.
- Không tự thao tác Skill Host Folder hoặc project files.
- Không expose raw Go transport details cho renderer.

## 2. Narrow Preload Bridge

Nơi áp dụng:

```text
apps/desktop/electron/preload/
apps/desktop/renderer/src/lib/core-client/
```

Responsibility:

- Expose API hẹp kiểu `invoke(method, params)` và `onEvent(event, callback)`.
- Renderer không import `ipcRenderer` trực tiếp.
- Renderer không biết Go binary path, stdin/stdout, hoặc process lifecycle.
- Electron main validate method allowlist trước khi forward sang Go.

Security defaults:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

## 3. JSON-RPC Command/Query Boundary

Nơi áp dụng:

```text
core-go/internal/rpc/
shared/api-contracts/
shared/generated/
```

Decision:

```text
transport = stdio
protocol = JSON-RPC 2.0
framing = NDJSON
go_library = creachadair/jrpc2
ready_notification = server.ready
progress_notification = operation.progress
```

Rules:

- Stdout chỉ dành cho JSON-RPC protocol messages.
- Logs đi vào stderr hoặc log file.
- Requests có `id`; notifications không có `id`.
- App error codes không dùng JSON-RPC reserved range `-32768` đến `-32000`.
- Long-running commands trả `operation_id`.
- Progress đi bằng server-push notifications, không dùng polling làm primary
  model.
- Contract schemas nằm trong `shared/api-contracts`.
- Generated TypeScript types được commit.
- Go structs viết tay trong Phase 1, có contract tests validate JSON Schema.

## 4. CQRS For UI-Facing API

Nơi áp dụng:

```text
core-go/internal/services/
core-go/internal/rpc/
apps/desktop/renderer/src/
```

Queries:

- Không có side effect.
- Trả view model đã chuẩn bị sẵn cho UI.
- Có thể join nhiều bảng thông qua repository layer.
- React render dữ liệu, không tự join hoặc suy luận nghiệp vụ phức tạp.

Examples:

```text
getDashboard()
listSkills()
getProjectDetail(projectId)
getGlobalSkills()
getUpdateOverview()
```

Commands:

- Validate input.
- Có thể ghi SQLite hoặc filesystem.
- Với tác vụ dài, tạo operation và trả `operation_id`.
- Không chạy write operation trực tiếp trong renderer.

Examples:

```text
chooseSkillHostFolder(path)
scanSkillHostFolder(hostId)
addProject(path)
scanProject(projectId)
scanGlobalSkills()
installSkillToProject(input)
syncInstall(installId)
updateSkill(skillId)
operation.cancel(operationId)
```

## 5. Application Service Layer

Nơi áp dụng:

```text
core-go/internal/services/
```

Responsibility:

- Orchestrate use cases.
- Gọi repositories, filesystem gateway, provider adapters, operation runner.
- Quyết định product policy.
- Không viết SQL trực tiếp.
- Không bypass filesystem gateway.

Examples:

```text
SkillHostService
SkillLibraryService
ProjectService
GlobalSkillsService
InstallService
UpdateService
OperationService
SettingsService
```

Rule:

- Provider adapter trả facts/capabilities.
- Service quyết định action nào được phép.

## 6. Repository Pattern

Nơi áp dụng:

```text
core-go/internal/repositories/
```

Responsibility:

- Là nơi duy nhất viết SQL trực tiếp.
- Quản lý transactions.
- Cung cấp query methods cho services.
- Giữ SQLite details ra khỏi service layer.

Rules:

- Multi-table writes dùng transaction.
- Scan/reconcile update entity status và stale rows trong transaction.
- Hot queries cần index và được kiểm tra bằng `EXPLAIN QUERY PLAN` khi cần.
- Repository tests dùng temp SQLite database.

SQLite startup:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

## 7. Filesystem Gateway

Nơi áp dụng:

```text
core-go/internal/filesystem/
```

Responsibility:

- Normalize absolute paths.
- Resolve realpath khi cần.
- Validate allowed roots trước mọi write.
- Detect symlink, broken symlink, external symlink.
- Create/remove symlink.
- Copy folder cho rsync/copy.
- Remove managed install entries.
- Read directory entries cho scan.

Hard rules:

- Không service nào được gọi trực tiếp `os.WriteFile`, `os.Remove`,
  `os.Rename`, hoặc copy/symlink helpers ngoài gateway khi thao tác lên skill,
  project, provider, hoặc host folder.
- Invalid write outside allowed root bị block cứng, không có "continue anyway".
- Direct/unmanaged entries cần confirmation policy ở service/UI trước khi
  gateway được gọi.
- Khi remove symlink, không follow target rồi xóa target; chỉ remove link.
- Path từ renderer luôn là untrusted input.

## 8. Provider Adapter Pattern

Nơi áp dụng:

```text
core-go/internal/providers/
```

Responsibility:

- Detect provider trong project.
- Resolve provider project paths.
- Resolve provider global locations nếu provider có global level.
- Scan entries trong provider scope.
- Classify facts: detected paths, entries, capabilities, warnings.

Không làm:

- Không ghi DB.
- Không viết filesystem.
- Không quyết định install/update policy.
- Không render UI state.

Adapter output nên là facts:

```text
provider key
detected path
skills path
entries
warnings
capabilities
global locations if applicable
```

## 9. Source Adapter Pattern

Nơi áp dụng:

```text
core-go/internal/sources/
```

Responsibility:

- GitHub/Vercel/local/manual source metadata.
- Fetch latest version metadata.
- Download/update host copy khi user xác nhận.
- Map auth/network/not-fetchable errors thành taxonomy chung.

Không làm:

- Không biết project providers.
- Không quyết định affected projects.
- Không sync rsync/copy installs.

UpdateService sẽ dùng DB để tính affected project installs và global installs.

## 10. Operation Runner And State Machine

Nơi áp dụng:

```text
core-go/internal/operations/
```

State:

```text
queued
running
succeeded
failed
cancelled
```

Responsibility:

- Tạo operation record.
- Chạy long-running tasks trong goroutine có context.
- Emit `operation.progress` notifications.
- Hỗ trợ `operation.cancel`.
- Lock theo target để tránh operation xung đột.
- Mark running operations failed khi sidecar shutdown/crash path cần cleanup.

Phase 1 locking:

- Single active operation per target.
- Nếu target đang bận, trả `conflict_error` fail-fast.
- Không queue tự động trong Phase 1.

Examples:

```text
target = skill_host_folder:{id}
target = project:{id}
target = global_provider_location:{id}
target = install:{id}
```

Progress rules:

- Không flood IPC.
- Progress nên theo phase/entry, không cần percent giả nếu không đo được.
- Khi operation xong, UI re-fetch view model.

## 11. Manual Constructor DI

Nơi áp dụng:

```text
core-go/cmd/skillbox-core/main.go
core-go/internal/app/
```

Decision:

- Phase 1 dùng manual constructor dependency injection.
- Không dùng `google/wire`, `uber-go/dig`, hoặc DI container.

Why:

- Dễ đọc.
- Dễ review bởi AI/người.
- Ít magic.
- Phù hợp khi số lượng services còn kiểm soát được.

Recommended shape:

```go
db := repositories.OpenDatabase(dbPath)
fs := filesystem.NewGateway()
providers := providers.NewRegistry(...)
ops := operations.NewRunner(...)

projectService := services.NewProjectService(db.ProjectRepo, providers, fs, ops)
installService := services.NewInstallService(db.InstallRepo, providers, fs, ops)

rpcServer.Register("project.scan", projectService.ScanProject)
rpcServer.Register("install.skill", installService.InstallSkill)
```

Nếu composition root phình quá lớn, tạo `internal/app` để gom wiring logic,
không đưa DI framework vào sớm.

## 12. View Model Composition

Nơi áp dụng:

```text
core-go/internal/services/
apps/desktop/renderer/src/screens/
```

Rules:

- Go query handlers trả view model phù hợp màn hình.
- React không tự join `skills`, `installs`, `projects`, `warnings`.
- View model gồm action availability, warnings, empty-state reason, loading
  state nếu cần.
- TanStack Query cache view models và invalidate sau command/operation.

Examples:

```text
DashboardView
SkillsLibraryView
SkillDetailView
ProjectsView
ProjectDetailView
GlobalSkillsView
UpdatesView
SettingsView
```

## 13. UI Component Composition

Nơi áp dụng:

```text
apps/desktop/renderer/src/components/
apps/desktop/renderer/src/screens/
```

Stack:

```text
shadcn/ui
Radix UI
Tailwind CSS
lucide-react
```

Patterns:

- App shell with sidebar navigation.
- Screen-level layout components.
- Detail panes for selected entities.
- Status badges.
- Warning banners with actions.
- Dialogs/AlertDialog for destructive actions.
- Popovers/DropdownMenu for scoped actions.
- Tooltip for icon-only buttons.

Avoid:

- Generic SaaS dashboard template assumptions.
- Hero/marketing layout.
- Cards inside cards.
- Renderer-only business rules.

## 14. Form Validation Pattern

Nơi áp dụng:

```text
apps/desktop/renderer/src/
core-go/internal/services/
shared/api-contracts/
```

Decision:

- React Hook Form + Zod cho UI/form validation.
- JSON Schema cho wire/API contract.
- Go validates params again in command/query handlers.

Duplication is intentional:

- UI validation tối ưu user experience.
- JSON Schema tối ưu contract.
- Go validation bảo vệ core khỏi untrusted renderer input.

## 15. Error Taxonomy Pattern

Nơi áp dụng:

```text
core-go/internal/domain/errors.go
core-go/internal/rpc/
apps/desktop/renderer/src/
```

Error categories:

```text
validation_error
filesystem_error
provider_error
database_error
auth_error
network_error
conflict_error
operation_cancelled
unknown_error
```

Rules:

- JSON-RPC error response map từ domain error taxonomy.
- `conflict_error` dùng khi operation target đang bận.
- UI hiển thị user message; logs giữ technical message.
- Không log secrets hoặc full payload chứa sensitive data.

## 16. Testing Pattern

Nơi áp dụng:

```text
core-go/
apps/desktop/
fixtures/
shared/api-contracts/
```

Required:

- Go unit tests for pure domain logic.
- Repository tests with temp SQLite.
- Filesystem gateway tests in temp directories.
- Provider adapter tests with fixture folders.
- JSON-RPC contract tests against JSON Schema.
- `go test -race` for operation runner/provider scan/filesystem gateway code.
- React component tests for complex UI states.

Deferred:

- Playwright until first UI shell exists.

## What We Keep From The External Pattern Report

Useful ideas:

- Process Coordinator for Go sidecar lifecycle.
- JSON-RPC bidirectional notifications.
- CQRS distinction between queries and commands.
- Adapter pattern for providers/sources.
- Filesystem Gateway as security boundary.
- Operation Runner as state machine.
- Repository pattern for SQLite.
- Manual constructor DI in Go.

Corrections applied:

- Use `server.ready`, not `system.ready`.
- Invalid filesystem writes are blocked, not confirmed-through.
- Phase 1 operation locking is fail-fast per target, not an open queueing
  decision.
- Avoid `os.Exit(0)` in normal shutdown prose; prefer normal return after
  cleanup.
