# Data Model

Tài liệu này phác thảo data model cấp cao cho SQLite. Mục tiêu là đủ chặt để
support UI, user flows, edge cases, fetch/update/sync, và provider adapters,
nhưng chưa khóa chi tiết implementation như migration syntax hay ORM.

Filesystem vẫn là source of truth cho skill content. SQLite là source of truth
cho metadata quản trị.

## Design Principles

- Skill content nằm trong Skill Host Folder, không nằm trong database.
- Database lưu metadata để UI biết skill, source, global provider location,
  project, provider, global install, project install, scan, fetch, update, sync,
  và warning state.
- Filesystem là trạng thái thật khi scan project hoặc Skill Host Folder.
- Scan có quyền reconcile database với filesystem.
- Mọi path lưu trong database nên là absolute path để UI/scan ổn định.
- Các bảng nên có `created_at` và `updated_at`.
- Các enum nên được lưu dạng text để dễ debug.

## Core Entities

```text
app_settings
api_credentials
skill_host_folders
skills
skill_sources
projects
provider_definitions
provider_path_candidates
project_providers
global_provider_locations
installs
global_installs
fetch_results
scan_results
warnings
operations
```

## 1. app_settings

Lưu cấu hình app cấp global.

Fields đề xuất:

```text
id
active_skill_host_folder_id
default_install_mode
database_version
created_at
updated_at
```

Notes:

- `active_skill_host_folder_id` trỏ tới Skill Host Folder hiện tại.
- `active_skill_host_folder_id` nullable để support first-time setup trước khi
  user chọn Skill Host Folder.
- Sản phẩm hiện tại chỉ hỗ trợ một active host.
- `default_install_mode` có thể là `symlink` hoặc `rsync_copy`.

## 2. api_credentials

Lưu metadata về credentials dùng cho GitHub/Vercel fetch. Giá trị secret thực tế
nên ưu tiên nằm trong OS keychain. Nếu implementation chọn lưu trong SQLite thì
phải lưu dạng encrypted value.

Fields đề xuất:

```text
id
provider_key
credential_type
storage_type
credential_ref
value_encrypted
status
last_validated_at
created_at
updated_at
```

Provider key:

```text
github
vercel
```

Credential type:

```text
token
oauth
ssh_key
```

Storage type:

```text
os_keychain
encrypted_sqlite
environment
```

Status:

```text
active
missing
invalid
expired
```

Notes:

- `credential_ref` trỏ tới keychain item hoặc environment variable name.
- `value_encrypted` chỉ dùng nếu `storage_type = encrypted_sqlite`.
- Không lưu plaintext token trong SQLite.

## 3. skill_host_folders

Lưu các folder từng được user chọn làm Skill Host Folder.

Fields đề xuất:

```text
id
name
path
skills_path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
missing
unreadable
unwritable
invalid_structure
empty
inactive
```

Notes:

- `path` là folder user chọn.
- `skills_path` thường là `<skill-host-folder>/.agents/skills`.
- `status` giúp Dashboard và Settings hiển thị warning nhanh.
- Khi đổi Skill Host Folder, host cũ không nhất thiết bị xóa khỏi database.

## 4. skills

Đại diện cho một skill trong Skill Host Folder.

Fields đề xuất:

```text
id
skill_host_folder_id
name
display_name
relative_path
absolute_path
status
source_id
current_version
current_commit
current_checksum
last_scanned_at
created_at
updated_at
```

Status:

```text
available
missing
unreadable
local_modified
unknown
```

Notes:

- `name` là folder name hoặc canonical skill id.
- `relative_path` thường là `.agents/skills/<skill-name>`.
- `absolute_path` là path thật trong Skill Host Folder.
- `source_id` nullable để support local/manual skill.
- `current_version` hoặc `current_commit` dùng cho Fetch/Update nếu có.
- `current_checksum` dùng để phát hiện local modification với các source không
  có git commit rõ ràng.

## 5. skill_sources

Lưu upstream/source metadata cho skill.

Fields đề xuất:

```text
id
source_type
url
github_owner
github_repo
github_path
github_ref
vercel_skill_id
local_source_path
resolved_version
resolved_commit
last_fetched_at
last_successful_fetch_at
last_fetch_status
last_fetch_error
created_at
updated_at
```

Source type:

```text
github
vercel_skills
local
manual
```

Fetch status:

```text
never_fetched
up_to_date
update_available
failed
auth_required
not_found
network_error
needs_review
not_fetchable
```

Notes:

- GitHub source có thể là repo root hoặc subfolder.
- `github_ref` có thể là branch, tag, hoặc commit.
- Vercel skills dùng `vercel_skill_id` hoặc metadata tương đương.
- `last_fetched_at` là lần fetch attempt gần nhất, kể cả failed attempt.
- `last_successful_fetch_at` là lần fetch thành công gần nhất.
- Local/manual source có thể dùng `not_fetchable`.

## 6. projects

Lưu các project được user add vào Skillbox.

Fields đề xuất:

```text
id
name
path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
missing
unreadable
removed
```

Notes:

- `path` là project root absolute path.
- Warning presence và `no_provider_detected` là derived state từ bảng
  `warnings`, không nằm trong `projects.status`.
- Project bị remove khỏi database nên có thể hard delete hoặc soft delete bằng
  `removed`, tùy implementation.

## 7. provider_definitions

Lưu danh sách provider/convention mà Skillbox biết.

Fields đề xuất:

```text
id
key
display_name
provider_type
icon_key
status
can_create_structure
has_global_level
created_at
updated_at
```

Provider type:

```text
claude
codex
opencode
antigravity_cli
generic_agents
custom
unsupported
```

Status:

```text
supported
experimental
unsupported
disabled
```

Notes:

- Provider adapter implementation sẽ dùng bảng này như metadata UI/config.
- `can_create_structure` cho biết core Skillbox logic có thể scaffold provider
  folder cho provider này hay chỉ được scan/install vào structure đã tồn tại.
- `has_global_level` cho biết provider có global/user-level location mà
  Skillbox có thể scan hoặc cấu hình.

## 8. provider_path_candidates

Lưu các path candidate mà một provider adapter dùng để detect hoặc install skill.
Một provider có thể có nhiều candidate path.

Fields đề xuất:

```text
id
provider_definition_id
relative_path
purpose
priority
description
created_at
updated_at
```

Purpose:

```text
detect
skills
commands
config
```

Notes:

- `relative_path` là path tương đối từ project root.
- `priority` giúp adapter chọn candidate chính khi có nhiều path hợp lệ.
- Bảng này tránh khóa provider vào một `default_relative_skills_path` duy nhất.
- Với provider đơn giản, chỉ cần một row `purpose = skills`.

## 9. project_providers

Lưu provider được phát hiện hoặc cấu hình trong từng project.

Fields đề xuất:

```text
id
project_id
provider_definition_id
detected_path
skills_path
detection_status
last_scanned_at
created_at
updated_at
```

Detection status:

```text
detected
configured
missing
unsupported
invalid_structure
format_unknown
```

Notes:

- Một project có thể có nhiều provider.
- Add Skill flow dùng bảng này để chọn provider target.
- `skills_path` là nơi install skill vào provider đó.
- Khi scan provider, `detected_path` nên lấy từ candidate `purpose = detect`
  có priority thấp nhất và tồn tại trên disk.
- `skills_path` nên lấy từ candidate `purpose = skills` đã resolve cho provider
  đó.

## 10. global_provider_locations

Lưu provider global locations ở cấp user/máy.

Fields đề xuất:

```text
id
provider_definition_id
name
path
skills_path
status
last_scanned_at
created_at
updated_at
```

Status:

```text
active
not_configured
missing
unreadable
invalid_structure
empty
disabled
```

Notes:

- `path` là absolute path tới global provider root hoặc global convention path.
  Nullable khi `status = not_configured`.
- `skills_path` là absolute path nơi provider global level nhận skill/global
  entries nếu có.
- Global locations không thuộc project nào.
- Provider adapter chịu trách nhiệm resolve/cấu hình candidate global paths,
  core Skillbox logic chịu trách nhiệm scan/write nếu được phép.

## 11. installs

Lưu việc một skill được cài vào một project/provider.

Fields đề xuất:

```text
id
project_provider_id
skill_id
skill_name
install_mode
install_status
project_skill_path
source_skill_path
symlink_target_path
installed_from_host_folder_id
installed_version
installed_commit
installed_checksum
last_synced_at
last_scanned_at
created_at
updated_at
```

Install mode:

```text
symlink
rsync_copy
direct
```

Install status:

```text
current
outdated
missing
broken_symlink
old_host
external_symlink
conflict
needs_sync
error
```

Notes:

- `project_id` không được lưu trực tiếp vì `project_provider_id` đã suy ra
  project qua `project_providers.project_id`.
- `skill_id` nullable cho `direct` hoặc unknown skill.
- `skill_name` vẫn cần lưu để hiển thị khi không map được `skill_id`.
- `skill_name` được ghi tại thời điểm scan/install và không tự động sync ngược
  từ `skills.name`.
- `project_skill_path` là entry trong provider folder.
- `source_skill_path` là path trong Skill Host Folder nếu managed.
- `install_mode` chỉ lưu cơ chế quản lý/install intent, không lưu detected
  filesystem anomaly.
- Khi scan thấy một symlink trên disk, `install_mode = symlink` bất kể symlink
  đó do Skillbox tạo hay do user tạo thủ công. `install_status` phân biệt trạng
  thái managed/current, old host, broken, hoặc external symlink.
- `symlink_target_path` giúp phân biệt valid symlink, old host,
  external_symlink, và broken_symlink trong `install_status`.
- `installed_checksum` hữu ích cho rsync/copy outdated detection.
- Phase 1 dùng hard delete cho install khi user remove skill bằng Skillbox.
- `missing` đại diện cho install record còn trong database nhưng filesystem đã
  bị sửa/xóa ngoài app.
- `error` là catch-all cho filesystem entry không thể phân loại an toàn trong
  quá trình scan.

## 12. global_installs

Lưu skill/global entry tồn tại trong provider global location.

Fields đề xuất:

```text
id
global_provider_location_id
skill_id
skill_name
install_mode
install_status
global_skill_path
source_skill_path
symlink_target_path
installed_from_host_folder_id
installed_version
installed_commit
installed_checksum
last_synced_at
last_scanned_at
created_at
updated_at
```

Install mode:

```text
symlink
rsync_copy
direct
```

Install status:

```text
current
outdated
missing
broken_symlink
old_host
external_symlink
conflict
needs_sync
error
```

Notes:

- Global installs dùng cùng semantics với project installs, nhưng scope là
  provider global level.
- `skill_id` nullable cho direct/unmanaged global entries.
- Global install cần UI phân biệt rõ với project-level install để tránh nhầm
  lẫn global contamination với project-specific behavior.

## 13. fetch_results

Lưu kết quả fetch upstream cho skill/source.

Fields đề xuất:

```text
id
source_id
status
host_version_at_fetch
upstream_version_at_fetch
host_commit_at_fetch
upstream_commit_at_fetch
fetched_at
error_message
raw_metadata_json
created_at
```

Status:

```text
up_to_date
update_available
failed
auth_required
not_found
network_error
needs_review
not_fetchable
```

Notes:

- Bảng này cho phép Updates view hiển thị lịch sử fetch gần nhất.
- `source_id` là FK chính. Skill context được suy ra qua `skills.source_id`.
- Nếu cần query nhanh theo skill trong implementation, có thể thêm helper
  denormalized `skill_id`, nhưng không nên coi nó là FK độc lập.
- `raw_metadata_json` giúp debug mà không cần schema hóa mọi field provider
  ngay từ đầu.
- Phase 1 nên giới hạn retention, ví dụ chỉ giữ N fetch results gần nhất theo
  `source_id`, để tránh bảng này tăng không giới hạn.

## 14. scan_results

Lưu kết quả scan gần nhất cho Skill Host Folder hoặc project.

Fields đề xuất:

```text
id
target_type
target_id
status
started_at
finished_at
summary_json
error_message
created_at
```

Target type:

```text
skill_host_folder
project
project_provider
global_provider_location
```

Status:

```text
success
partial
failed
cancelled
```

Notes:

- UI không cần lưu mọi scan detail trong bảng này nếu detail đã reconcile vào
  `skills`, `project_providers`, và `installs`.
- `summary_json` có thể lưu counts như skills found, providers found, warnings.
- Nếu `operations` đã đủ cho audit trail, implementation có thể gộp scan result
  vào `operations.metadata_json`. Tài liệu giữ entity này để làm rõ dữ liệu scan
  cần có.

## 15. warnings

Lưu warning/recoverable error để Dashboard, Projects, và Project Detail hiển thị
nhất quán.

Fields đề xuất:

```text
id
scope_type
scope_id
severity
code
message
action_key
source_operation_id
is_resolved
created_at
updated_at
resolved_at
```

Scope type:

```text
app
skill_host_folder
skill
project
project_provider
install
global_provider_location
global_install
source
database
```

Severity:

```text
info
warning
error
blocking
```

Code examples:

```text
skill_host_missing
skill_host_unwritable
project_missing
no_provider_detected
unsupported_provider
broken_symlink
old_host_symlink
external_symlink
rsync_outdated
fetch_failed
database_corrupt
```

Action key examples:

```text
choose_folder
rescan
retry
relink
sync
remove
configure_source
open_folder
```

Notes:

- Warnings có thể được regenerate sau scan.
- `source_operation_id` nullable, trỏ tới operation/scan tạo ra warning nếu có.
- `is_resolved` giúp UI ẩn warning cũ mà vẫn giữ lịch sử nếu cần.
- Phase 1 nên ưu tiên regenerate active warnings sau scan thay vì giữ warning
  history dài hạn.

## 16. operations

Lưu các operation dài hoặc quan trọng như scan, fetch, update, install, remove.

Fields đề xuất:

```text
id
operation_type
target_type
target_id
status
started_at
finished_at
error_message
metadata_json
created_at
updated_at
```

Operation type:

```text
scan
fetch
update_host_skill
sync_install
install_skill
remove_install
switch_install_mode
change_skill_host_folder
scan_global_skills
```

Status:

```text
queued
running
success
failed
cancelled
partial
```

Notes:

- Dùng cho loading state, progress, audit trail nhẹ, và debug.
- Không nhất thiết phải build job system ngay; bảng này vẫn hữu ích cho UI.

## 17. provider_user_settings

Lưu user-level preference cho từng provider, ví dụ provider có được enable hay không.

Fields đề xuất:

```text
id
provider_definition_id
enabled
created_at
updated_at
```

Notes:

- Một row duy nhất cho mỗi `provider_definition_id` (UNIQUE).
- `enabled` là boolean `0/1`. Phase 1 chỉ dùng để user bật/tắt provider khỏi global
  scan và install target list.
- Khác với `provider_definitions.status`: `status` là support state do app quyết
  định, còn `enabled` ở đây là preference do user quyết định.
- Khi provider definition bị xóa, row tương ứng cascade delete.

## 18. provider_path_overrides

Lưu override của user cho danh sách path candidate built-in của provider. Một row
cho mỗi tổ hợp `(provider_definition_id, scope, purpose)`. `paths_json` là JSON
array path strings thay thế (không merge) candidate built-in.

Fields đề xuất:

```text
id
provider_definition_id
scope
purpose
paths_json
created_at
updated_at
```

Scope:

```text
project
global
```

Purpose:

```text
detect
skills
config
commands
```

Notes:

- Khi có override, adapter dùng `paths_json` thay vì `provider_path_candidates`
  cho `(scope, purpose)` đó. Built-in candidate không được merge ngầm.
- `paths_json` phải parse được thành JSON array string (CHECK constraint).
- Dùng cho user cần trỏ tới layout phi tiêu chuẩn (ví dụ Claude settings ở
  custom path) mà không cần Skillbox release version mới.
- UNIQUE `(provider_definition_id, scope, purpose)` đảm bảo mỗi slot chỉ có một
  override active.

## 19. Provider Plugin Layer System

Một số provider (ban đầu là Claude, Codex, Antigravity CLI) hỗ trợ khái niệm
**plugin** thông qua settings file riêng (`~/.claude/settings.json`,
`~/.codex/config.toml`, ...). Một plugin có thể được khai báo ở nhiều layer
khác nhau (user/project/local) và effective state được merge theo precedence
`local > project > user`.

Skillbox scan các settings file đó để hiển thị view enabled/disabled per
project và per provider. Ba bảng phối hợp:

```text
provider_plugin_layer_scans
provider_plugin_entries
provider_plugin_marketplaces
```

Một `provider_plugin_layer_scan` row đại diện cho **một lần đọc** một settings
file ở một layer cụ thể (`user`, `project`, hoặc `local`) cho một provider.
Mỗi scan sinh ra (nếu file đọc được) các `provider_plugin_entries` (khai báo
enabled/disabled cho từng plugin) và `provider_plugin_marketplaces` (các
marketplace bổ sung nơi plugin được tải về).

Layer rules:

- `settings_layer = 'user'` → `project_id IS NULL`. Đây là layer global ở cấp
  user/máy.
- `settings_layer IN ('project', 'local')` → `project_id IS NOT NULL`.
- Unique theo `(provider_definition_id, settings_layer)` cho user layer (partial
  index where `project_id IS NULL`), và `(provider_definition_id, project_id,
  settings_layer)` cho project/local layer.

### 19.1 provider_plugin_layer_scans

Fields đề xuất:

```text
id
provider_definition_id
project_id
settings_layer
scan_status
settings_file_path
last_scanned_at
source_operation_id
scan_warnings
```

Settings layer:

```text
user
project
local
```

Scan status:

```text
ok
missing
unreadable
malformed
too_large
symlink
path_escape
```

Notes:

- `scan_status = ok` là điều kiện duy nhất để các entries/marketplaces phát
  sinh từ scan này được coi là có hiệu lực.
- `missing` là trạng thái hợp lệ, nghĩa là settings file chưa tồn tại ở layer
  đó (không phải lỗi).
- `symlink` và `path_escape` là defensive: scanner từ chối đọc symlink và path
  thoát khỏi user/project root để tránh leak.
- `too_large` chặn file lớn bất thường để giới hạn parse cost.
- `scan_warnings` là JSON array string chứa parse-time warnings (không chứa
  raw file content), giới hạn kích thước.
- `source_operation_id` nullable, FK tới `operations.id` đại diện cho lần scan
  đã tạo row này.

### 19.2 provider_plugin_entries

Fields đề xuất:

```text
id
layer_scan_id
plugin_name
marketplace_name
declaration
version          -- TEXT nullable (migration 000021); NULL = not available
```

Declaration:

```text
enabled
disabled
```

Notes:

- Một entry là một dòng khai báo plugin trong settings file của layer tương
  ứng. Vắng mặt khai báo = `absent` (không có entry row).
- UNIQUE `(layer_scan_id, plugin_name, marketplace_name)` đảm bảo settings
  file không thể khai báo trùng cùng plugin trong cùng marketplace.
- Effective status (`enabled` / `disabled` / `absent` / `unknown`) được resolve
  ở application layer bằng cách merge các entry rows theo precedence layer; không
  lưu trực tiếp trong bảng này.
- `version`: được đọc từ `~/.claude/plugins/installed_plugins.json` khi scan
  user layer của Claude provider. NULL cho Codex/Antigravity (không có file
  tương đương) và khi plugin không có bản ghi install. Giá trị `"unknown"` là
  literal hợp lệ (Claude báo cáo khi không xác định được version).

### 19.3 provider_plugin_marketplaces

Fields đề xuất:

```text
id
layer_scan_id
marketplace_name
source_type
source_summary
```

Source type:

```text
github
git
directory
url
settings
hostPattern
```

Notes:

- Mỗi row là một marketplace được khai báo trong cùng settings file scan đại
  diện bởi `layer_scan_id`. Marketplace là nguồn (named source) mà plugin được
  resolve từ đó.
- `source_summary` là string mô tả nguồn (ví dụ `owner/repo`, git URL, path).
  Không lưu raw credentials.
- `source_type` không có CHECK constraint trong migration: enum value được
  validate ở application layer dựa trên format của settings file provider.

## 20. Plugin Update-Check Cache & Network Settings

*(migration 000022 — 2026-05-29)*

Hai bảng hỗ trợ tính năng **G3c plugin update-check**. Tính năng này là
**always-on, manual-trigger-only** (ADR-0002, supersedes ADR-0001): không còn
opt-in gate; network chỉ được gọi khi user bấm "Check Updates".

> **Migration 000023 (2026-05-31):** cột `network_settings.update_check_enabled`
> đã bị **drop** (gate cũ vô dụng). Bảng `network_settings` được giữ cho
> `cache_ttl_hours`.

```text
plugin_update_check_cache
network_settings
```

### 20.1 plugin_update_check_cache

Lưu kết quả mỗi lần `updateCheck.run` kiểm tra upstream SHA cho một plugin cụ
thể. Cache có TTL 6 giờ (configurable qua `network_settings.cache_ttl_hours`);
mỗi click "Check Updates" upsert lại row theo UNIQUE key.

Fields:

```text
id                 -- INTEGER PRIMARY KEY
provider_key       -- TEXT NOT NULL; "claude" (Phase 1 chỉ hỗ trợ Claude)
plugin_name        -- TEXT NOT NULL
marketplace_name   -- TEXT NOT NULL
source_url         -- TEXT NOT NULL; HTTPS URL từ marketplace.json (allowlist từ disk)
source_ref         -- TEXT nullable; tag/branch (ví dụ "v1.5.5", "main")
installed_sha      -- TEXT nullable; gitCommitSha từ installed_plugins.json
installed_version  -- TEXT nullable; version string từ installed_plugins.json
remote_sha         -- TEXT nullable; SHA trả về bởi git ls-remote
remote_latest_tag  -- TEXT nullable; reserved Phase 2 (semver tag scan)
update_available   -- INTEGER nullable; 0=false / 1=true / NULL=unknown
checked_at         -- TEXT NOT NULL; ISO-8601 UTC timestamp của lần check
error              -- TEXT nullable; error code nếu check thất bại
UNIQUE(provider_key, plugin_name, marketplace_name)
```

Update-available logic:

```text
installed_sha IS NOT NULL AND remote_sha IS NOT NULL
  → update_available = (installed_sha != remote_sha)
otherwise
  → update_available = NULL (unknown)
```

Notes:

- `source_url` phải là HTTPS — `GitLsRemoteClient` từ chối non-HTTPS trước khi
  spawn subprocess.
- `error` chứa error code string, ví dụ: `non_https_scheme_rejected`, `timeout`,
  `git_not_found`, `ref_not_found`, `host_backoff`.
- Row không có FK tới bảng plugin nào: cache là snapshot độc lập — plugin bị
  xóa không cascade-delete cache.

### 20.2 network_settings

Bảng singleton (luôn có đúng 1 row với `id = 1`, được insert bởi migration
000022). Giữ cài đặt cache cho update-check.

Fields (sau migration 000023):

```text
id                    -- INTEGER PRIMARY KEY CHECK (id = 1); luôn = 1
cache_ttl_hours       -- INTEGER NOT NULL DEFAULT 6; TTL cache update-check (giờ)
created_at            -- TEXT NOT NULL; ISO-8601 UTC
updated_at            -- TEXT NOT NULL; ISO-8601 UTC; cập nhật khi set_ttl
```

Notes:

- Cột `update_check_enabled` đã bị drop ở migration 000023 (ADR-0002): update-check
  là always-on, không còn gate. `UpdateCheckService.RunUpdateCheck` không đọc
  setting nào nữa — chạy mỗi khi user trigger.
- `cache_ttl_hours` hiện read-only từ UI (Phase 1); Phase 2 có thể expose slider.

## Provider Plugin Relationships

```text
provider_definitions.id
  -> provider_user_settings.provider_definition_id (UNIQUE)

provider_definitions.id
  -> provider_path_overrides.provider_definition_id

projects.id
  -> provider_plugin_layer_scans.project_id (nullable; user layer = null)

provider_definitions.id
  -> provider_plugin_layer_scans.provider_definition_id

provider_plugin_layer_scans.id
  -> provider_plugin_entries.layer_scan_id (CASCADE DELETE)

provider_plugin_layer_scans.id
  -> provider_plugin_marketplaces.layer_scan_id (CASCADE DELETE)

operations.id
  -> provider_plugin_layer_scans.source_operation_id
```

## Relationship Overview

```text
app_settings.active_skill_host_folder_id
  -> skill_host_folders.id

skill_host_folders.id
  -> skills.skill_host_folder_id

skill_host_folders.id
  -> installs.installed_from_host_folder_id

skill_sources.id
  -> skills.source_id

projects.id
  -> project_providers.project_id

provider_definitions.id
  -> project_providers.provider_definition_id

provider_definitions.id
  -> provider_path_candidates.provider_definition_id

provider_definitions.id
  -> global_provider_locations.provider_definition_id

project_providers.id
  -> installs.project_provider_id

skills.id
  -> installs.skill_id

global_provider_locations.id
  -> global_installs.global_provider_location_id

skills.id
  -> global_installs.skill_id

skill_host_folders.id
  -> global_installs.installed_from_host_folder_id

skill_sources.id
  -> fetch_results.source_id

operations.id
  -> warnings.source_operation_id
```

## Data Needed By Main Views

### Dashboard

Needs:

- Active Skill Host Folder status.
- Count skills.
- Count global installs.
- Count projects.
- Count installs by mode.
- Count warnings by severity.
- Count update_available fetch results.

Tables:

```text
app_settings
skill_host_folders
skills
projects
installs
global_provider_locations
global_installs
fetch_results
warnings
```

### Skills Library

Needs:

- Skill list from active Skill Host Folder.
- Source type and fetch status.
- Project count per skill.
- Last fetched/update status.

Tables:

```text
skills
skill_sources
fetch_results
installs
```

### Projects

Needs:

- Project list.
- Provider badges.
- Skill/install counts.
- Warning status.

Tables:

```text
projects
project_providers
provider_definitions
installs
warnings
```

### Global Skills

Needs:

- Global provider locations.
- Global entries grouped by provider.
- Mode/status/source path per global install.
- Warning status.

Tables:

```text
global_provider_locations
global_installs
provider_definitions
skills
warnings
```

### Project Detail

Needs:

- Project path/status.
- Providers detected.
- Installed skills grouped by provider.
- Mode/status/source path per install.
- Warnings and available actions.

Tables:

```text
projects
project_providers
provider_definitions
installs
skills
warnings
```

### Updates

Needs:

- Skills with update available.
- Host/upstream version or commit from latest fetch result.
- Affected projects and install modes.
- Affected global installs and install modes.
- Rsync/copy installs needing sync.

Tables:

```text
skills
skill_sources
fetch_results
installs
projects
project_providers
global_installs
global_provider_locations
```

### Settings

Needs:

- Active Skill Host Folder.
- Database version/location.
- Default install mode.
- Provider definitions/config.
- GitHub/Vercel credential metadata if configured.

Tables:

```text
app_settings
api_credentials
skill_host_folders
provider_definitions
provider_path_candidates
global_provider_locations
```

## Mapping From User Flows

### First-Time Setup

Writes:

- `skill_host_folders`
- `app_settings.active_skill_host_folder_id`
- `skills` after initial scan
- `scan_results`

### Add Project

Writes:

- `projects`
- `project_providers`
- `installs` discovered during scan
- `warnings` if provider/path issues exist

### Scan Global Skills

Writes:

- `global_provider_locations`
- `global_installs` discovered during scan
- `warnings` if global path/provider issues exist
- `scan_results`

### Install Skill To Project

Writes:

- `installs`
- `operations`
- `warnings` if conflict or filesystem error occurs

### Fetch Skill Updates

Writes:

- `fetch_results`
- `skill_sources.last_fetched_at`
- `skill_sources.last_fetch_status`
- `warnings` for fetch failures

### Update Skill Host Folder

Writes:

- `skills.current_version/current_commit`
- `skills.current_checksum`
- `skill_sources.resolved_version/resolved_commit`
- `operations`
- Sản phẩm hiện tại không ghi sync rows. Project dùng symlink nhận update host
  thông qua filesystem link.

### Change Skill Host Folder

Writes:

- `skill_host_folders`
- `app_settings.active_skill_host_folder_id`
- `skills` after host scan
- `warnings` for old host symlinks

## Mapping From Edge Cases

### Missing Skill Host Folder

Represented by:

```text
skill_host_folders.status = missing
warnings.code = skill_host_missing
```

### Missing Project

Represented by:

```text
projects.status = missing
warnings.code = project_missing
```

### Missing Global Provider Location

Represented by:

```text
global_provider_locations.status = missing
warnings.code = global_provider_location_missing
warnings.scope_type = global_provider_location
```

### No Provider Detected

Represented by:

```text
warnings.code = no_provider_detected
warnings.scope_type = project
```

### Broken Symlink

Represented by:

```text
installs.install_mode = symlink
installs.install_status = broken_symlink
warnings.code = broken_symlink
```

### Global Direct Install

Represented by:

```text
global_installs.install_mode = direct
global_installs.install_status = current
global_installs.skill_id = null
```

### Global Skill Overlap

Represented by:

```text
global_installs.skill_name
installs.skill_name
warnings.code = global_project_skill_overlap
warnings.scope_type = global_install | project | install
```

### Old Host Symlink

Represented by:

```text
installs.install_status = old_host
warnings.code = old_host_symlink
```

### External Symlink

Represented by:

```text
installs.install_mode = symlink
installs.install_status = external_symlink
warnings.code = external_symlink
```

### Direct Install

Represented by:

```text
installs.install_mode = direct
installs.install_status = current
installs.skill_id = null
```

### Fetch Failure

Represented by:

```text
fetch_results.status = failed | auth_required | not_found | network_error
warnings.code = fetch_failed
warnings.scope_type = source
```

### Unsupported Provider

Represented by:

```text
provider_definitions.status = unsupported
project_providers.detection_status = unsupported
warnings.code = unsupported_provider
```

### Multi-Path Provider Detection

Represented by:

```text
provider_path_candidates.provider_definition_id
provider_path_candidates.relative_path
provider_path_candidates.priority
project_providers.detected_path
project_providers.skills_path
```

### Fetch Attempt Failed But Previous Fetch Was Successful

Represented by:

```text
skill_sources.last_fetched_at
skill_sources.last_successful_fetch_at
skill_sources.last_fetch_status = failed | network_error | auth_required
```

## Open Questions

- Projects/skills có cần soft delete dài hạn không? Phase 1 đã chọn hard delete
  cho user-initiated install removal.
- Checksum cho rsync/copy nên tính toàn folder hay dựa vào manifest/snapshot
  metadata?
- GitHub/Vercel auth credentials nên lưu trong OS keychain, SQLite encrypted
  table, hay environment?
- Phase 2 convert skill format có cần bảng `skill_variants` hoặc
  `provider_skill_formats` không?
- Có nên thêm `skills.detected_format` ngay từ Phase 1 để chuẩn bị cho convert
  Phase 2 không?
