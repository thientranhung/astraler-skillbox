# Technical Architecture

Tài liệu này phác thảo kiến trúc kỹ thuật của Astraler Skillbox ở mức module và
boundary. Mục tiêu là giúp team build app mà không trộn lẫn UI, database,
filesystem writes, provider conventions, và operation/audit logic.

Đây chưa phải implementation spec cho framework cụ thể. Nếu sau này chọn
Electron, Tauri, hoặc desktop web shell khác, các boundary bên dưới vẫn nên giữ.

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
Desktop App
  -> UI Layer
  -> Application Services
  -> Domain Services
  -> Data Access Layer
  -> SQLite
  -> Filesystem Gateway
  -> Provider Adapters
  -> External Sources
```

Ý nghĩa:

- UI Layer chỉ render state, nhận input, gọi commands/queries.
- Application Services là entry point cho từng use case.
- Domain Services chứa logic nghiệp vụ dùng chung.
- Data Access Layer đọc/ghi SQLite.
- Filesystem Gateway gom mọi thao tác đọc/ghi file/folder/symlink/copy.
- Provider Adapters hiểu convention của Claude, Generic Agents, Codex, v.v.
- External Sources xử lý GitHub, Vercel skills, local import.

## Runtime Processes

Skillbox nên tách tư duy thành hai phía:

```text
Renderer/UI process
  -> hiển thị màn hình
  -> gửi command/query
  -> nhận progress/result/warning

Core process
  -> SQLite
  -> filesystem access
  -> provider adapters
  -> fetch/update/sync jobs
  -> operation audit
```

Nếu framework cho phép chạy một process duy nhất ở Phase 1, vẫn nên giữ boundary
code như có hai phía để sau này không phải refactor lớn.

UI không nên import trực tiếp database client, filesystem APIs, hoặc provider
adapter implementation.

## Module Boundaries

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

Boundary đề xuất:

- `ui`: screens, components, view models, client API.
- `core/services`: use case orchestration.
- `core/domain`: pure-ish business rules, enums, validation.
- `core/repositories`: SQLite queries and transactions.
- `core/providers`: provider definitions, adapters, detection contracts.
- `core/filesystem`: safe path, symlink, copy, remove, scan helpers.
- `core/sources`: GitHub/Vercel/local/manual source integrations.
- `core/operations`: job runner, progress, cancellation, audit.
- `core/migrations`: SQLite schema migrations and seed data.

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
- `OperationService`: start/read/cancel operation records.
- `WarningService`: list/resolve/dismiss warning state nếu cần.

## Command And Query Pattern

UI nên gọi core qua hai loại API:

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
```

Query không nên tạo side effect. Command có thể tạo `operations` record, ghi DB,
và thao tác filesystem.

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

## Open Architecture Questions

- Desktop framework: Electron, Tauri, or another shell.
- Language/runtime for core: TypeScript, Rust, or hybrid.
- SQLite library and migration tool.
- OS keychain library.
- Whether operation runner needs a worker process from day one.
- Whether provider definitions are seeded from code, bundled JSON, or migrations.
