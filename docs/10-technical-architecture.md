# Technical Architecture

Tài liệu này phác thảo kiến trúc kỹ thuật của Astraler Skillbox ở mức module và
boundary. Mục tiêu là giúp team build app mà không trộn lẫn UI, database,
filesystem writes, provider conventions, và operation/audit logic.

Stack đã chốt:

- Desktop framework: Electron.
- UI framework: React.
- Core runtime language: Golang.

Các boundary bên dưới vẫn quan trọng vì Electron/React không nên trực tiếp sở
hữu database, filesystem writes, provider adapters, hoặc long-running jobs.

Implementation decisions bên dưới có hai loại:

- Chốt: stack nền tảng và responsibility boundary.
- Recommended defaults: đề xuất kỹ thuật cần được brainstorm/chốt trước khi
  scaffold code thật.

Brainstorm kỹ thuật hiện tại được ghi ở:

- `docs/archive/review-results/technical-architecture-brainstorm.md`
- `docs/archive/review-results/transport-decision-brainstorm.md`

## Architecture Goals

- GUI là trải nghiệm chính.
- Skill Host Folder là source of truth cho skill content.
- SQLite là source of truth cho metadata quản trị.
- Filesystem scan luôn có quyền reconcile database với trạng thái thật.
- Provider convention nằm trong adapter, không hardcode rải rác trong UI.
- Filesystem writes phải đi qua service có validation và audit.
- Long-running work như scan, fetch, update, sync phải tạo operation record.
- UI không tự thao tác filesystem trực tiếp.

## High-Level Shape

```text
Electron Desktop App
  -> React UI Layer
  -> Electron Bridge / IPC Client
  -> Golang Core Runtime
     -> Application Services
     -> Domain Services
     -> Data Access Layer
     -> SQLite
     -> Filesystem Gateway
     -> Provider Adapters
     -> External Sources
```

Ý nghĩa:

- React UI Layer chỉ render state, nhận input, gọi commands/queries.
- Electron Bridge / IPC Client là boundary giữa UI và Golang core.
- Golang Core Runtime giữ application services và mọi thao tác có side effect.
- Application Services là entry point cho từng use case.
- Domain Services chứa logic nghiệp vụ dùng chung.
- Data Access Layer đọc/ghi SQLite.
- Filesystem Gateway gom mọi thao tác đọc/ghi file/folder/symlink/copy.
- Provider Adapters hiểu convention của Claude, Generic Agents, Codex, v.v.
- External Sources xử lý GitHub, Vercel skills, local import.

## Runtime Processes

Skillbox nên tách tư duy thành ba phần runtime:

```text
Electron main process
  -> app window lifecycle
  -> launch/manage Golang core runtime
  -> expose narrow IPC bridge to renderer

Electron renderer process / React UI
  -> hiển thị màn hình
  -> gửi command/query qua bridge
  -> nhận progress/result/warning

Golang core runtime
  -> SQLite
  -> filesystem access
  -> provider adapters
  -> fetch/update/sync jobs
  -> operation audit
```

Decision: Golang core chạy như sidecar process do Electron main process quản lý
trong Phase 1.

Decision: transport giữa Electron main process và Golang core là stdio
JSON-RPC 2.0 trong Phase 1. Lý do chính: không cần mở local port, không có port
conflict, không có macOS firewall prompt, dễ đóng gói desktop app, và hỗ trợ
request/response lẫn server-push notifications.

Decision: JSON-RPC Phase 1 dùng NDJSON framing và `creachadair/jrpc2`, trừ khi
spike implementation phát hiện blocker cụ thể.

Migration path: nếu Phase 2 cần CLI reuse, multi-window, hoặc nhiều consumer
cùng kết nối tới một Go core đang chạy, giữ JSON-RPC protocol và đổi transport
sang unix socket/named pipe.

Dù transport là stdio, UI contract vẫn nên là command/query API thay vì gọi
trực tiếp implementation detail.

UI không nên import trực tiếp database client, filesystem APIs, hoặc provider
adapter implementation.

## Transport Decision

Phase 1 dùng stdio JSON-RPC 2.0:

```text
Electron main process
  -> spawn Go core binary
  -> write JSON-RPC requests to child stdin
  -> read JSON-RPC responses/notifications from child stdout
  -> forward safe events to React renderer through preload bridge

Go core runtime
  -> read JSON-RPC requests from stdin
  -> write only JSON-RPC protocol messages to stdout
  -> write logs/debug output to stderr or log file
```

Rules:

- Stdout là protocol boundary. Không dùng `fmt.Print*` hoặc log output thường
  vào stdout trong Go core.
- Go core phải gửi `server.ready` notification trước khi Electron main forward
  renderer requests.
- Electron main chờ `server.ready` tối đa 10 giây sau khi spawn Go core.
- Nếu timeout hoặc Go exit trước `server.ready`, Electron main kill child nếu
  còn sống, hiển thị blocking error window, và surface stderr/log path.
- Operation progress dùng JSON-RPC notifications như `operation.progress`.
- Request/response dùng `id` của JSON-RPC để support multiple in-flight
  requests.
- Operation locking nằm ở service layer, không nằm ở transport layer.
- Production không mở local HTTP server.
- App error codes không dùng JSON-RPC reserved range `-32768` đến `-32000`.

Implementation details:

- JSON-RPC Go library: `creachadair/jrpc2`.
- Framing: NDJSON, one JSON object per line.
- Dev-only debug server có thể thêm sau qua `SKILLBOX_DEBUG_PORT`, nhưng không
  thuộc production path.

## Module Boundaries

```text
app/
  ui/
  electron/
  core-go/
  shared/
```

Candidate module shape trong Golang core:

```text
core-go/
  services/
  domain/
  repositories/
  providers/
  filesystem/
  sources/
  operations/
  migrations/
```

Candidate module shape trong React/Electron side:

```text
ui/
  screens/
  components/
  view-models/
  client/

electron/
  main/
  preload/
  core-process/

shared/
  api-contracts/
```

Boundary đề xuất:

- `ui`: React screens, components, view models, client API.
- `electron/main`: window lifecycle, app menu, native dialogs, core process
  lifecycle.
- `electron/preload`: narrow bridge exposed to renderer.
- `electron/core-process`: start/stop/monitor Golang core runtime. Folder này có
  thể đổi tên hoặc tách nhỏ sau khi transport được chốt.
- `shared/api-contracts`: command/query request and response shapes.
- `core-go/services`: use case orchestration.
- `core-go/domain`: business rules, enums, validation.
- `core-go/repositories`: SQLite queries and transactions.
- `core-go/providers`: provider definitions, adapters, detection contracts.
- `core-go/filesystem`: safe path, symlink, copy, remove, scan helpers.
- `core-go/sources`: GitHub/Vercel/local/manual source integrations.
- `core-go/operations`: job runner, progress, cancellation, audit.
- `core-go/migrations`: SQLite schema migrations and seed data.

Previous conceptual grouping:

```text
app/
  ui/
  core/
    services/
    domain/
    repositories/
    providers/
    filesystem/
    sources/
    operations/
    migrations/
```

Conceptual boundary này vẫn đúng, nhưng implementation folder vẫn là candidate
shape cho tới khi scaffold code thật.

## Application Services

Application Services là API mà UI gọi. Mỗi service nên expose command/query rõ
ràng, không leak SQL hoặc raw filesystem detail lên UI.

Services chính:

```text
SettingsService
SkillHostService
SkillLibraryService
ProjectService
ProviderService
GlobalSkillsService
InstallService
UpdateService
ProviderPluginService
OperationService
WarningService
```

Mapping:

- `SettingsService`: app settings, active Skill Host Folder, default install
  mode, global provider location settings.
- `SkillHostService`: chọn host folder, init `.agents/skills`, scan host.
- `SkillLibraryService`: list/import/fetch/update skills.
- `ProjectService`: add project, scan project, project detail queries.
- `ProviderService`: provider detection, provider definitions, icons/status.
- `GlobalSkillsService`: scan global locations, list global entries, remediation.
- `InstallService`: install, sync, switch mode, remove project install.
- `UpdateService`: fetch all, update host copy, impact preview.
- `ProviderPluginService`: scan, toggle, and remove plugin overrides across
  layers (user/project/local). Owns `pluginWriterFn` and `pluginRemoverFn`
  abstractions for JSON and TOML config files.
- `OperationService`: start/read/cancel operation records.
- `WarningService`: list/resolve/dismiss warning state nếu cần.

## Command And Query Pattern

React UI nên gọi Golang core qua hai loại API:

```text
Query:
  getDashboard()
  listSkills()
  getProjectDetail(projectId)
  getGlobalSkills()
  getUpdateOverview()

Command:
  chooseSkillHostFolder(path)
  scanSkillHostFolder(hostId)
  addProject(path)
  scanProject(projectId)
  scanGlobalSkills()
  installSkillToProject(input)
  syncInstall(installId)
  updateSkill(skillId)
  providerPlugin.setEnabled(input)
  providerPlugin.removeOverride(input)
```

Query không nên tạo side effect. Command có thể tạo `operations` record, ghi DB,
và thao tác filesystem.

IPC/transport rules:

- Renderer chỉ gọi API đã expose qua Electron preload bridge.
- Renderer không được gọi Node filesystem API trực tiếp.
- Electron main không nên chứa business logic, chỉ làm lifecycle/bridge/native
  integration.
- Golang core trả typed response cho mọi command/query.
- Long-running command phải trả `operation_id`.
- Nếu dùng stdio JSON-RPC, progress nên đi qua JSON-RPC server-push
  notifications như `operation.progress`, không dùng polling làm primary model.

## Data Access Layer

Repository layer là nơi duy nhất viết SQL trực tiếp.

Repository groups:

```text
AppSettingsRepository
SkillHostRepository
SkillRepository
SkillSourceRepository
ProjectRepository
ProviderRepository
ProjectProviderRepository
GlobalProviderLocationRepository
InstallRepository
GlobalInstallRepository
FetchResultRepository
ScanResultRepository
WarningRepository
OperationRepository
```

Rules:

- Mỗi command lớn nên dùng transaction khi update nhiều bảng.
- Scan commands nên ghi `scan_results`, update entity status, và reconcile stale
  rows trong cùng một transaction sau khi filesystem read hoàn tất.
- Filesystem write nên được validate trước, thực hiện write, rồi update DB trong
  transaction ngay sau đó.
- Không lưu plaintext secrets trong SQLite.
- Migrations phải chạy trước khi UI vào app chính.

SQLite startup sequence:

```text
Open SQLite connection
  -> Apply connection PRAGMAs
  -> Run migrations
  -> Seed provider definitions through migration
  -> Open app main window only after success
```

Required PRAGMAs for every connection, including tests:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

SQLite file path:

```text
macOS:   ~/Library/Application Support/Astraler Skillbox/skillbox.db
Windows: %APPDATA%\Astraler Skillbox\skillbox.db
Linux:   ~/.config/astraler-skillbox/skillbox.db
Tests:   SKILLBOX_DB_PATH override to temp database path
```

## Filesystem Gateway

Filesystem Gateway là boundary bắt buộc cho mọi thao tác path.

Responsibilities:

- Normalize absolute paths.
- Resolve realpath khi cần.
- Validate path nằm trong allowed root trước khi write.
- Detect symlink, broken symlink, external symlink.
- Copy folder cho rsync/copy mode.
- Create/remove symlink.
- Remove managed install entry.
- Read directory entries cho scan.
- Open folder bằng OS shell nếu UI yêu cầu.

Write safety rules:

- Không write vào project/provider path nếu provider adapter chưa resolve target.
- Không remove folder/file nếu entry không được nhận diện là managed install,
  trừ khi user xác nhận rõ đó là direct/unmanaged entry.
- Không follow symlink khi remove symlink; chỉ remove link.
- Không overwrite direct install nếu chưa có confirmation và impact preview.
- Không tạo path ngoài project root cho project install.
- Không tạo path ngoài configured global provider location cho global
  remediation.

## Provider Adapter Boundary

Provider adapters không truy cập UI và không ghi database trực tiếp.

Adapter input:

```text
project_root
provider_definition
path_candidates
configured_paths
skill_host_folder
```

Adapter output:

```text
detected project providers
resolved skills paths
installed entries
global provider locations
global entries
warnings
capabilities
```

Core Skillbox logic nhận output này rồi quyết định:

- Ghi bảng nào.
- Warning nào được tạo.
- Install target nào hợp lệ.
- Action nào được enable/disable trong UI.

Provider adapters chỉ trả về facts và capabilities. Product policy nằm ở core
services.

## Operation Model

Các thao tác sau nên chạy qua Operation runner:

- Scan Skill Host Folder.
- Scan project.
- Scan global skills.
- Fetch updates.
- Update Skill Host Folder copy.
- Sync rsync/copy install.
- Switch install mode.
- Remove managed install.
- Change Skill Host Folder.

Operation lifecycle:

```text
queued
running
succeeded
failed
cancelled
```

Operation runner nên:

- Ghi `operations` trước khi chạy.
- Emit progress cho UI.
- Ghi result/error summary.
- Không để hai operation xung đột chạy cùng lúc trên cùng target.
- Cho phép retry nếu lỗi không phải validation error.

Recommended progress model nếu dùng stdio JSON-RPC:

- Go core gửi `operation.progress` notifications qua stdout.
- Electron main parse notification và forward qua preload bridge.
- React UI subscribe theo `operation_id`.
- Khi operation kết thúc, UI re-fetch view model liên quan để lấy state đã
  reconcile từ SQLite.
- Cancel dùng command `operation.cancel` với `operation_id`; Go dùng
  `context.WithCancel` và check cancel ở natural checkpoints.
- Retry là command mới từ UI, không auto-retry âm thầm trong Go.

Startup and shutdown lifecycle:

- Electron main spawns Go with `spawn()`, not `exec()`.
- Electron main waits up to 10 seconds for `server.ready`.
- If Go exits or times out before `server.ready`, show blocking startup error;
  do not silently retry.
- During an app session, if Go exits unexpectedly, Electron main may restart up
  to 3 times before showing blocking error.
- On `before-quit`, Electron main sends SIGTERM, waits 3 seconds, then SIGKILL.
- Go handles SIGTERM and stdin EOF by marking running operations as failed,
  closing SQLite, and exiting.

Locking gợi ý:

- Một Skill Host Folder chỉ nên có một scan/update operation active.
- Một project chỉ nên có một scan/install/sync/remove operation active.
- Một global provider location chỉ nên có một scan/remediation operation active.

## Scan And Reconcile

Scan là cơ chế đưa database về gần trạng thái thật của filesystem.

Project scan:

```text
read project path
detect providers
scan provider skills paths
classify entries
compare with installs table
mark missing/stale records
upsert project_providers and installs
write warnings
write scan_results
```

Global scan:

```text
load providers with has_global_level or configured global paths
resolve global locations
scan global skills paths
classify entries
compare with global_installs table
upsert global_provider_locations and global_installs
write warnings
write scan_results
```

Skill Host scan:

```text
read active Skill Host Folder
ensure or validate .agents/skills
scan skill folders
read source metadata when available
upsert skills and skill_sources
mark missing/unreadable/local_modified
write warnings
write scan_results
```

Reconcile rule:

- Filesystem state wins for existence/status.
- SQLite metadata wins for management intent, source mapping, and operation
  history.
- If filesystem and database disagree, UI shows explicit status instead of
  silently hiding the mismatch.

## Install, Sync, Remove

Install to project:

```text
validate skill exists in active host
validate project exists and provider target is supported
resolve target path via provider adapter
show impact preview if target exists
write symlink or rsync/copy through filesystem gateway
upsert installs
write operation result
refresh project detail
```

Sync rsync/copy:

```text
validate install is managed rsync_copy
validate source skill exists
validate target path belongs to provider scope
copy host skill to target
update checksum/version metadata
write operation result
```

Remove project install:

```text
validate target install
if managed symlink/copy, remove target entry
if direct/unmanaged, require stronger confirmation
mark or delete install metadata based on product policy
write operation result
```

Phase 1 does not include Install Skill To Global Location. Global remediation
can support safe actions such as open folder, update configured path, relink
managed broken symlink, sync managed global rsync/copy entry if that entry was
previously created/adopted by Skillbox.

## Fetch And Update Sources

Source integrations should be separate from provider adapters.

```text
GitHubSourceAdapter
VercelSkillSourceAdapter
LocalSourceAdapter
ManualSourceAdapter
```

Responsibilities:

- Fetch latest version metadata.
- Compare current version/commit/checksum.
- Report auth/network/not-fetchable states.
- Download/update Skill Host Folder copy when user confirms.

GitHub/Vercel source logic should not know project providers. After host update,
UpdateService computes affected project installs and global installs from DB.

## UI State Composition

Screens should be backed by view models assembled from queries, not by UI joining
raw tables manually.

View models:

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

Each view model should include:

- Primary entities.
- Counts.
- Action availability.
- Warning summaries.
- Loading/operation state.
- Empty state reason.
- Next recommended action.

Action availability should come from core rules, not UI-only checks.

## Error Handling

Use typed application errors:

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

Every command result should return:

```text
status
operation_id
changed_entities
warnings_created
user_message
technical_message
```

UI shows `user_message`. Logs/debug tools can show `technical_message`.

## Security And Privacy

- Do not store plaintext tokens in SQLite.
- Prefer OS keychain for GitHub/Vercel credentials.
- Treat project paths and skill content as local private data.
- Do not send local file content to external service unless user explicitly
  triggers a source/fetch/conversion feature that requires it.
- Log paths and operation metadata, but avoid logging secret values.
- Any future telemetry must be opt-in.

Electron security decisions:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true if compatible
preload exposes narrow typed bridge only
renderer never receives Go process path or transport details
Electron main validates JSON-RPC method allowlist before forwarding to Go
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
dev only: remote-debugging-port on 127.0.0.1 (default 49222, gated on ELECTRON_RENDERER_URL); packaged builds never open a debugging port
```

Dev exposes a Chrome DevTools Protocol port (default `49222`, override `SKILLBOX_CDP_PORT`) so browser-automation agents can `connect` to the running `pnpm dev` instance instead of launching a second app. It is gated on `ELECTRON_RENDERER_URL` (set by electron-vite only in dev) and binds to loopback only — packaged builds never open it. See `AGENTS.md` → "Agent Browser".

## Testing Strategy

Core test layers:

- Domain unit tests for install mode classification and impact preview.
- Provider adapter tests using fixture folders.
- Filesystem gateway tests in temp directories.
- Repository tests against temporary SQLite database.
- Service tests for scan/install/sync/update flows.
- UI tests for view states and disabled/enabled actions.

Critical fixtures:

- Empty Skill Host Folder.
- Missing Skill Host Folder.
- Project with `.agents/skills`.
- Project with multiple providers.
- Managed symlink install.
- Broken symlink install.
- External symlink install.
- Managed rsync/copy install.
- Direct/unmanaged install.
- Global provider location missing.
- Global/project overlap.

## Phase Boundaries

Phase 1:

- GUI-first app.
- One active Skill Host Folder.
- SQLite metadata.
- Scan Skill Host Folder.
- Add project and scan project providers.
- Project install via symlink or rsync/copy.
- Global Skills scan/visibility/remediation surface.
- Fetch/update source metadata for GitHub/Vercel/local where supported.
- Updates impact preview.

Phase 2:

- Skill format conversion.
- Install Skill To Global Location if product decides to support it.
- Custom provider UI.
- More source registries.
- CLI layer over the same Application Services.
- Multi-host support if needed.

## Architecture Decisions To Confirm

Các decision dưới đây là các điểm còn cần chốt trước khi scaffold code thật.
Transport Phase 1 đã chốt là stdio JSON-RPC 2.0; các chi tiết framing/library
vẫn còn mở.

```text
IPC transport:
  phase_1_decision = stdio JSON-RPC 2.0
  phase_2_migration_path = unix socket/named pipe if CLI or multi-window needs it
  library = creachadair/jrpc2
  framing = NDJSON
  open = dev debug server

Go core lifecycle:
  phase_1_decision = sidecar process managed by Electron main
  alternative = persistent daemon if background work becomes product requirement

Operation progress:
  phase_1_decision = JSON-RPC server-push notifications
  avoid = polling as primary progress model

API contract:
  recommended = JSON Schema in shared/api-contracts, generate TypeScript types
  open = whether Go structs are generated or hand-matched

SQLite:
  recommended = modernc.org/sqlite for no-CGO Phase 1 builds
  migrations = embedded SQL migrations
  pragmas = WAL, foreign_keys=ON, busy_timeout=5000, synchronous=NORMAL
  path = OS app data directory, SKILLBOX_DB_PATH override for dev/test

Keychain:
  recommended = Go core owns credentials via zalando/go-keyring
  fallback = SKILLBOX_GITHUB_TOKEN, SKILLBOX_VERCEL_TOKEN for dev/CI

Packaging:
  recommended = electron-builder with bundled Go binary
  high-risk = macOS code signing and notarization for both app and Go binary

Provider seed data:
  recommended = seed via migration
  alternatives = bundled JSON or code seed

Outbound Network:
  default = OFF (local-first invariant, see ADR-0001)
  opt_in = network.update_check.enabled via network_settings table (migration 000022)
  mechanism = git ls-remote via system git (no new SDK); HTTPS URLs only
  security = HTTPS-only validation before subprocess; env-stripped (PATH + GIT_TERMINAL_PROMPT=0 only)
  timeout = 8s per-request, 60s batch deadline; max 4 concurrent subprocesses
  cache = plugin_update_check_cache table, 6h TTL default
  renderer_boundary = renderer never calls network; all outbound via Go core (UpdateCheckService)
  see = docs/decisions/0001-outbound-network-update-check.md
```
