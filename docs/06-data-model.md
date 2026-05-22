# Data Model

Tài liệu này phác thảo data model cấp cao cho SQLite. Mục tiêu là đủ chặt để
support UI, user flows, edge cases, fetch/update/sync, và provider adapters,
nhưng chưa khóa chi tiết implementation như migration syntax hay ORM.

Filesystem vẫn là source of truth cho skill content. SQLite là source of truth
cho metadata quản trị.

## Design Principles

- Skill content nằm trong Skill Host Folder, không nằm trong database.
- Database lưu metadata để UI biết skill, source, project, provider, install,
  scan, fetch, update, sync, và warning state.
- Filesystem là trạng thái thật khi scan project hoặc Skill Host Folder.
- Scan có quyền reconcile database với filesystem.
- Mọi path lưu trong database nên là absolute path để UI/scan ổn định.
- Các bảng nên có `created_at` và `updated_at`.
- Các enum nên được lưu dạng text để dễ debug.

## Core Entities

```text
app_settings
skill_host_folders
skills
skill_sources
projects
provider_definitions
project_providers
installs
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
- Phase đầu chỉ cần một active host, nhưng model không chặn multi-host sau này.
- `default_install_mode` có thể là `symlink` hoặc `rsync_copy`.

## 2. skill_host_folders

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

## 3. skills

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

## 4. skill_sources

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
```

Notes:

- GitHub source có thể là repo root hoặc subfolder.
- `github_ref` có thể là branch, tag, hoặc commit.
- Vercel skills dùng `vercel_skill_id` hoặc metadata tương đương.
- Local/manual source có thể không fetch được.

## 5. projects

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
no_provider_detected
has_warnings
removed
```

Notes:

- `path` là project root absolute path.
- Project bị remove khỏi database nên có thể hard delete hoặc soft delete bằng
  `removed`, tùy implementation.

## 6. provider_definitions

Lưu danh sách provider/convention mà Skillbox biết.

Fields đề xuất:

```text
id
key
display_name
provider_type
default_relative_skills_path
icon_key
status
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
- Path convention cụ thể có thể cần table phụ sau này nếu provider có nhiều
  candidate paths.

## 7. project_providers

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

## 8. installs

Lưu việc một skill được cài vào một project/provider.

Fields đề xuất:

```text
id
project_id
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
external_symlink
unknown
```

Install status:

```text
current
outdated
missing
broken_symlink
old_host
external
conflict
needs_sync
unmanaged
error
```

Notes:

- `skill_id` nullable cho `direct` hoặc unknown skill.
- `skill_name` vẫn cần lưu để hiển thị khi không map được `skill_id`.
- `project_skill_path` là entry trong provider folder.
- `source_skill_path` là path trong Skill Host Folder nếu managed.
- `symlink_target_path` giúp phân biệt valid symlink, old host, external
  symlink, và broken symlink.
- `installed_checksum` hữu ích cho rsync/copy outdated detection.

## 9. fetch_results

Lưu kết quả fetch upstream cho skill/source.

Fields đề xuất:

```text
id
skill_id
source_id
status
current_version
latest_version
current_commit
latest_commit
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
- `raw_metadata_json` giúp debug mà không cần schema hóa mọi field provider
  ngay từ đầu.

## 10. scan_results

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

## 11. warnings

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
- `is_resolved` giúp UI ẩn warning cũ mà vẫn giữ lịch sử nếu cần.

## 12. operations

Lưu các operation dài hoặc quan trọng như scan, fetch, update, sync, install,
remove, switch mode.

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

## Relationship Overview

```text
app_settings.active_skill_host_folder_id
  -> skill_host_folders.id

skill_host_folders.id
  -> skills.skill_host_folder_id

skill_sources.id
  -> skills.source_id

projects.id
  -> project_providers.project_id

provider_definitions.id
  -> project_providers.provider_definition_id

projects.id
  -> installs.project_id

project_providers.id
  -> installs.project_provider_id

skills.id
  -> installs.skill_id

skills.id / skill_sources.id
  -> fetch_results.skill_id / fetch_results.source_id
```

## Data Needed By Main Views

### Dashboard

Needs:

- Active Skill Host Folder status.
- Count skills.
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
- Latest/current version or commit.
- Affected projects and install modes.
- Rsync/copy installs needing sync.

Tables:

```text
skills
skill_sources
fetch_results
installs
projects
project_providers
```

### Settings

Needs:

- Active Skill Host Folder.
- Database version/location.
- Default install mode.
- Provider definitions/config.

Tables:

```text
app_settings
skill_host_folders
provider_definitions
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
- `skill_sources.resolved_version/resolved_commit`
- `operations`
- `installs.install_status = needs_sync` for affected rsync/copy installs

### Sync Rsync / Copy Project

Writes:

- `installs.installed_version/installed_commit/installed_checksum`
- `installs.last_synced_at`
- `installs.install_status`
- `operations`

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

### No Provider Detected

Represented by:

```text
projects.status = no_provider_detected
warnings.code = no_provider_detected
```

### Broken Symlink

Represented by:

```text
installs.install_mode = symlink
installs.install_status = broken_symlink
warnings.code = broken_symlink
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
installs.install_mode = external_symlink
installs.install_status = external
warnings.code = external_symlink
```

### Direct Install

Represented by:

```text
installs.install_mode = direct
installs.install_status = unmanaged
installs.skill_id = null
```

### Rsync / Copy Outdated

Represented by:

```text
installs.install_mode = rsync_copy
installs.install_status = outdated
```

### Fetch Failure

Represented by:

```text
fetch_results.status = failed | auth_required | not_found | network_error
warnings.code = fetch_failed
```

### Unsupported Provider

Represented by:

```text
provider_definitions.status = unsupported
project_providers.detection_status = unsupported
warnings.code = unsupported_provider
```

## Open Questions

- Có cần soft delete cho projects/skills/installs không, hay scan sẽ hard delete
  records không còn tồn tại?
- Checksum cho rsync/copy nên tính toàn folder hay dựa vào manifest/snapshot
  metadata?
- Provider path conventions nên nằm trong `provider_definitions` hay tách bảng
  riêng nếu một provider có nhiều candidate paths?
- `warnings` nên lưu lịch sử dài hạn hay regenerate theo scan và chỉ giữ trạng
  thái hiện tại?
- Phase 2 convert skill format có cần bảng `skill_variants` hoặc
  `provider_skill_formats` không?
