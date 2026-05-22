# Provider Model

Provider Model mô tả cách Skillbox hiểu và thao tác với các agent provider khác
nhau. Mục tiêu là không hardcode path/convention rải rác trong app, mà gom logic
provider vào adapter rõ ràng.

## Provider Là Gì

Provider là một agent hoặc convention mà project dùng để chứa skill, command,
config, hoặc workflow files.

Ví dụ:

- Claude
- Codex
- opencode
- Antigravity CLI
- Generic `.agents`
- Custom/unsupported provider

Một project có thể có nhiều provider cùng lúc. Ví dụ project có cả Claude
convention và shared `.agents` convention.

## Provider Adapter

Provider adapter là lớp biết cách làm việc với một provider cụ thể.

Responsibilities:

- Detect provider trong project.
- Resolve provider paths từ project root.
- Resolve skill install path.
- Scan installed skills trong provider scope.
- Classify install state.
- Tạo provider folder structure nếu adapter được phép.
- Báo unsupported/invalid/missing state.
- Cung cấp metadata cho UI như display name, icon, support status.

Adapter không nên tự quyết định product policy như update/sync strategy. Những
policy đó thuộc core Skillbox logic.

## Provider Definitions

`provider_definitions` là lookup table cho provider mà Skillbox biết.

Các field quan trọng:

```text
key
display_name
provider_type
icon_key
status
can_create_structure
```

Status:

```text
supported
experimental
unsupported
disabled
```

Ý nghĩa:

- `supported`: adapter đủ ổn định để scan/install.
- `experimental`: adapter dùng được nhưng convention có thể còn thay đổi.
- `unsupported`: Skillbox nhận diện được provider nhưng chưa biết cách thao tác
  an toàn.
- `disabled`: provider bị tắt trong config hoặc chưa bật cho user.

`can_create_structure` cho biết adapter có thể scaffold folder/path cần thiết
hay chỉ được dùng khi structure đã tồn tại.

## Provider Path Candidates

`provider_path_candidates` lưu các path mà adapter dùng để detect hoặc thao tác.

Fields:

```text
provider_definition_id
relative_path
purpose
priority
description
```

Purpose:

```text
detect
skills
commands
config
```

Ý nghĩa:

- `detect`: path dùng để phát hiện provider có tồn tại trong project không.
- `skills`: path nơi provider nhận skill installs.
- `commands`: path dành cho command-style files nếu provider có convention này.
- `config`: path tới config file/folder của provider.

Resolution rules:

- Adapter resolve candidate path từ project root.
- Candidate có priority cao hơn được xét trước.
- `project_providers.detected_path` nên lấy từ candidate `purpose = detect` phù
  hợp nhất và tồn tại trên disk.
- `project_providers.skills_path` nên lấy từ candidate `purpose = skills` đã
  resolve cho provider đó.
- Nếu provider có nhiều skills path hợp lệ, adapter phải chọn một path chính
  hoặc báo state cần user chọn, tùy product decision sau này.

## Project Provider

`project_providers` là provider đã được detect hoặc cấu hình trong một project.

Một project có thể có nhiều row `project_providers`.

Fields chính:

```text
project_id
provider_definition_id
detected_path
skills_path
detection_status
last_scanned_at
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

Ý nghĩa:

- `detected`: provider convention tồn tại và adapter hiểu được.
- `configured`: user hoặc app đã cấu hình provider target, kể cả khi path chưa
  tự detect được.
- `missing`: provider từng tồn tại/cấu hình nhưng path hiện không còn.
- `unsupported`: project có dấu hiệu provider, nhưng Skillbox chưa có adapter
  thao tác an toàn.
- `invalid_structure`: path tồn tại nhưng cấu trúc không đúng expectation.
- `format_unknown`: structure tồn tại nhưng format bên trong chưa đọc được.

## Detection Flow

Flow:

```text
Project scan bắt đầu
  -> Load provider_definitions đang enabled/supported/experimental
  -> Với mỗi provider, load provider_path_candidates
  -> Resolve candidate paths từ project root
  -> Kiểm tra candidate detect paths
  -> Nếu match, tạo/cập nhật project_providers
  -> Resolve skills_path
  -> Scan installed skills trong skills_path nếu có
  -> Ghi warnings nếu missing/unsupported/invalid
```

Nếu không detect được provider nào:

```text
projects.status vẫn là active
warnings.scope_type = project
warnings.code = no_provider_detected
```

Provider absence không phải lỗi blocking. User có thể chọn setup provider nếu
adapter hỗ trợ `can_create_structure`.

## Install Target Resolution

Khi user cài skill vào project:

```text
User mở Project Detail
  -> Chọn Add Skill
  -> Skillbox lấy danh sách project_providers
  -> Nếu chỉ có một provider target hợp lệ, có thể auto-select
  -> Nếu có nhiều provider, user phải chọn provider target
  -> Adapter resolve skills_path
  -> Core install logic tạo symlink hoặc rsync/copy
```

Provider target hợp lệ khi:

- `project_providers.detection_status` là `detected` hoặc `configured`.
- `provider_definitions.status` là `supported` hoặc `experimental`.
- `skills_path` resolve được.
- Nếu skills path chưa tồn tại, adapter phải có `can_create_structure = 1` mới
  được scaffold.

Nếu provider là `unsupported`, Skillbox không được tự ghi file vào provider path.

## Scan Installed Skills

Adapter cung cấp provider scope. Core scan logic phân loại install state dựa vào
filesystem và Skillbox metadata.

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

Rule:

- Nếu entry là symlink, `install_mode = symlink`.
- Nếu symlink trỏ vào active Skill Host Folder, status có thể là `current`.
- Nếu symlink trỏ vào Skill Host Folder cũ, status là `old_host`.
- Nếu symlink trỏ ngoài Skill Host Folder, status là `external_symlink`.
- Nếu symlink target không tồn tại, status là `broken_symlink`.
- Nếu entry là folder thường có Skillbox metadata, mode là `rsync_copy`.
- Nếu entry là folder thường không có Skillbox metadata, mode là `direct`.
- Nếu entry không phân loại an toàn được, status là `error`.

## Initial Provider Assumptions

Các giả định hiện tại:

- Claude có convention riêng và cần adapter riêng.
- Generic `.agents` là shared convention cho nhiều provider.
- Codex, opencode, Antigravity CLI có thể bắt đầu bằng generic `.agents` nếu
  chưa cần adapter riêng.
- Khi provider convention thay đổi, adapter layer là nơi cập nhật, không sửa
  rải rác trong UI/core logic.

## Suggested Initial Provider Definitions

### Generic Agents

```text
key = generic_agents
display_name = Generic Agents
provider_type = generic_agents
icon_key = agents
status = supported
can_create_structure = true
```

Path candidates:

```text
purpose = detect, relative_path = .agents, priority = 10
purpose = skills, relative_path = .agents/skills, priority = 10
```

### Claude

```text
key = claude
display_name = Claude
provider_type = claude
icon_key = claude
status = experimental
can_create_structure = false
```

Path candidates should be finalized after provider convention research.

### Codex

```text
key = codex
display_name = Codex
provider_type = codex
icon_key = codex
status = experimental
can_create_structure = true
```

Initial path candidates may use the generic `.agents` convention until Codex
requires a distinct adapter.

### opencode

```text
key = opencode
display_name = opencode
provider_type = opencode
icon_key = opencode
status = experimental
can_create_structure = true
```

Initial path candidates may use the generic `.agents` convention until opencode
requires a distinct adapter.

### Antigravity CLI

```text
key = antigravity_cli
display_name = Antigravity CLI
provider_type = antigravity_cli
icon_key = antigravity
status = experimental
can_create_structure = true
```

Initial path candidates may use the generic `.agents` convention until
Antigravity CLI requires a distinct adapter.

## UI Representation

Provider UI nên hiển thị:

- Provider badge/icon từ `icon_key`.
- Provider display name.
- Support state: supported, experimental, unsupported, disabled.
- Detection status trong project.
- Skill count theo provider.
- Warning nếu provider missing/unsupported/invalid.

Trong Project Detail, installed skills nên được group hoặc filter theo provider.
Không nên gộp các skill trùng tên ở nhiều provider thành một row mơ hồ.

## Unsupported Provider Policy

Nếu scan phát hiện dấu hiệu provider chưa support:

```text
project_providers.detection_status = unsupported
provider_definitions.status = unsupported
warnings.code = unsupported_provider
```

UI nên:

- Hiển thị provider là unsupported.
- Không cho install skill vào provider đó.
- Cho user xem path liên quan.
- Có thể cho user gửi/report provider convention sau này.

## Provider Adapter Boundary

Provider adapter nên trả về structured result, không tự mutate database trực
tiếp.

Ví dụ adapter output:

```text
provider_key
detected_path
skills_path
detection_status
warnings
entries
```

Core Skillbox logic chịu trách nhiệm:

- Ghi `project_providers`.
- Ghi `installs`.
- Ghi `warnings`.
- Chạy install/sync/remove.

Boundary này giúp adapter testable và tránh database logic bị phân tán.

## Phase 2 Conversion

Phase 2 có thể thêm skill format conversion giữa provider.

Các khái niệm có thể cần:

```text
skills.detected_format
skill_variants
provider_skill_formats
convert_skill operation
```

Provider Model hiện tại không block Phase 2 vì:

- Provider đã là entity riêng.
- Install đã scoped theo project provider.
- Path candidates đã tách khỏi provider definition.
- Operations có thể thêm `convert_skill`.

Phase 1 chưa cần lưu converted variants.

## Open Questions

- Claude convention chính xác nên là gì và path nào nên được adapter support?
- Codex/opencode/Antigravity CLI có cần adapter riêng ngay, hay dùng
  `generic_agents` trước?
- Khi một provider có nhiều skills path hợp lệ, app nên auto chọn theo priority
  hay yêu cầu user chọn?
- Có nên cho user tạo custom provider trong UI ở Phase 1 không?
- Provider icon nên dùng bundled asset, icon key, hay package icon set?
