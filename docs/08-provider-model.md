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
- Detect provider global location nếu provider có global level.
- Resolve provider paths từ project root.
- Resolve provider global paths từ user/machine conventions nếu có.
- Resolve skill install path.
- Scan installed skills trong provider scope.
- Classify install state.
- Báo provider folder structure có thể tạo được không.
- Báo unsupported/invalid/missing state.
- Cung cấp metadata cho UI như display name, icon, support status.
- Detect skill format trong provider scope khi Phase 2 conversion bắt đầu.

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
has_global_level
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

`can_create_structure` cho biết provider có thể được core Skillbox logic
scaffold folder/path cần thiết hay chỉ được dùng khi structure đã tồn tại.

`has_global_level` cho biết provider có global/user-level location mà Skillbox
có thể scan hoặc cấu hình. Global scan chỉ load provider có
`has_global_level = 1` hoặc đã có configured global location.

`key` là stable identifier để lưu config, seed data, và external references.
`provider_type` là enum/category để app dispatch adapter implementation. Hai giá
trị này có thể giống nhau ở provider built-in, nhưng không bắt buộc giống nhau
với custom provider sau này.

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
- Priority thấp hơn thắng. Adapter kiểm tra `priority = 1` trước `priority = 10`.
- `project_providers.detected_path` nên lấy từ candidate `purpose = detect` phù
  hợp nhất và tồn tại trên disk.
- `project_providers.skills_path` nên lấy từ candidate `purpose = skills` đã
  resolve cho provider đó.
- Nếu nhiều candidate cùng purpose cùng tồn tại, adapter chọn candidate có
  priority thấp nhất.
- Nếu nhiều candidate cùng purpose có cùng priority, adapter chọn theo thứ tự
  path alphabet để Phase 1 không cần thêm UI chọn path.
- `commands` và `config` được giữ để chuẩn bị cho future phases. Phase 1 adapter
  chỉ bắt buộc cần `detect` và `skills`.

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

`configured` là future/manual setup state. Phase 1 chưa cần flow riêng để user
manually configure provider target; nếu chưa làm UI này thì adapter không nên tự
set `configured` tùy tiện.

## Global Provider Location

Global provider location là provider scope ở cấp user/máy, không thuộc một
project cụ thể.

Provider adapter có thể hỗ trợ global scan nếu provider có global convention.
Kết quả scan được ghi vào `global_provider_locations` và `global_installs`.

Global detection không được gộp với project detection:

- Project provider scope ghi vào `project_providers`.
- Global provider scope ghi vào `global_provider_locations`.

UI phải hiển thị global entries riêng trong Global Skills để user biết global
skill nào có thể ảnh hưởng nhiều project.

## Detection Flow

Flow:

```text
Project scan bắt đầu
  -> Load provider_definitions có status khác disabled
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

Detection được phép nhận diện `unsupported` providers để UI báo rõ cho user.
Install target resolution mới là nơi chặn write vào provider chưa support.

Khi rescan thấy provider path cũ đã missing, `project_providers.detection_status`
nên chuyển thành `missing`, và các installs thuộc provider đó nên được đánh dấu
`install_status = missing` cho tới khi user relink/rescan được path mới.

## Global Detection Flow

Flow:

```text
Global scan bắt đầu
  -> Load provider_definitions có has_global_level = 1 hoặc configured global paths
  -> Resolve global provider locations
  -> Scan global skills_path nếu có
  -> Tạo/cập nhật global_provider_locations
  -> Tạo/cập nhật global_installs
  -> Ghi warnings nếu missing/unreadable/unmanaged/overlap
```

Global scan phải giữ scope riêng với project scan. Một global entry không được
tự động coi là project install.

Global provider paths không dùng `provider_path_candidates.relative_path` vì
field đó là project-root relative path. Global paths được resolve bởi adapter từ
user/machine conventions hoặc từ `global_provider_locations.path` đã được user
cấu hình trong Settings.

Global scan dùng cùng rule với project rsync/copy detection: nếu entry là folder
thường và có `global_installs` DB record cho path đó với
`install_mode = rsync_copy`, mode là `rsync_copy`; nếu không có record thì mode
là `direct`.

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
- `skills_path` nằm trong project root sau khi canonicalize/normalize path.
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
- Nếu entry là folder thường và có `installs` DB record cho path đó với
  `install_mode = rsync_copy`, mode là `rsync_copy`.
- Nếu entry là folder thường không có Skillbox metadata, mode là `direct`.
- Nếu entry không phân loại an toàn được, status là `error`.

Phase 1 chọn DB record làm Skillbox metadata cho rsync/copy detection, không
ghi marker file vào project folder. Nếu database bị mất và app scan lại từ đầu,
các rsync/copy installs cũ có thể bị phân loại thành `direct`; user cần sync lại
bằng Skillbox nếu muốn đưa chúng về managed state.

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
has_global_level = true
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
has_global_level = true
```

Path candidates should be finalized after provider convention research.
Không implement Claude scan/install cho tới khi convention này được xác minh từ
documentation hoặc local provider behavior.

### Codex

```text
key = codex
display_name = Codex
provider_type = codex
icon_key = codex
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until Codex
requires a distinct adapter.
Phase 1 không nên seed `.agents` path candidates riêng cho Codex nếu
`generic_agents` đã cover cùng convention, để tránh một `.agents` folder tạo
nhiều provider detections trùng nhau.

### opencode

```text
key = opencode
display_name = opencode
provider_type = opencode
icon_key = opencode
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until opencode
requires a distinct adapter.
Phase 1 không nên seed `.agents` path candidates riêng cho opencode nếu
`generic_agents` đã cover cùng convention, để tránh duplicate provider badges.

### Antigravity CLI

```text
key = antigravity_cli
display_name = Antigravity CLI
provider_type = antigravity_cli
icon_key = antigravity
status = experimental
can_create_structure = true
has_global_level = false
```

Initial path candidates may use the generic `.agents` convention until
Antigravity CLI requires a distinct adapter.
Phase 1 không nên seed `.agents` path candidates riêng cho Antigravity CLI nếu
`generic_agents` đã cover cùng convention, để tránh duplicate provider badges.

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

Provider `experimental` nên hiển thị badge/tooltip nhẹ để user biết adapter có
thể thay đổi. Provider `disabled` nên ẩn khỏi install target list, nhưng có thể
hiện trong Settings để user bật lại nếu app support provider toggles.

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

Adapter cũng không tự thực hiện filesystem writes. Các thao tác như `mkdir`,
symlink creation, rsync/copy, delete, relink đều do core Skillbox logic thực
hiện sau khi adapter trả về path và capability metadata. Điều này giúp adapter
dễ test và giảm rủi ro ghi nhầm vào project.

Ví dụ adapter output:

```text
provider_key
detected_path
skills_path
detection_status
warnings
entries
```

Minimum output contract:

```text
provider_key: text
detected_path: absolute path | null
skills_path: absolute path | null
detection_status: detected | configured | missing | unsupported | invalid_structure | format_unknown
warnings: list of {
  code: text
  severity: info | warning | error | blocking
  message: text
  action_key: text | null
}
entries: list of {
  name: text
  path: absolute path to the skill entry within the provider skills_path
  entry_type: symlink | directory | unknown
  symlink_target: path | null
}
```

Global adapter output contract:

```text
provider_key: text
global_path: absolute path | null
global_skills_path: absolute path | null
global_status: active | not_configured | missing | unreadable | invalid_structure | empty | disabled
warnings: list of {
  code: text
  severity: info | warning | error | blocking
  message: text
  action_key: text | null
}
entries: list of {
  name: text
  path: absolute path to the global skill entry within global_skills_path
  entry_type: symlink | directory | unknown
  symlink_target: path | null
}
```

Core Skillbox logic chịu trách nhiệm:

- Ghi `project_providers`.
- Ghi `installs`.
- Ghi `warnings`.
- Chạy install/sync/remove.
- Thực hiện filesystem writes sau khi validate output của adapter.

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

## Provider Plugin Layer Model

Một số provider (ban đầu là Claude, Codex, Antigravity CLI) hỗ trợ khái niệm
**plugin** thông qua settings file riêng (`~/.claude/settings.json`,
`~/.codex/config.toml`, `~/.gemini/antigravity-cli/settings.json`, ...). Plugin
khác với skill: plugin là một extension được provider khai báo trong settings
file, có thể đến từ marketplace bên ngoài, và có thể được enable/disable mà
không cần xóa khỏi disk.

Skillbox Phase 1 chỉ đọc/ghi settings file để hiển thị trạng thái và cho phép
toggle enable/disable. Skillbox không quản lý download/install marketplace nội
dung; provider tự xử lý.

### Layer Precedence

Plugin state được khai báo trên ba layer có precedence rõ ràng:

```text
local   (project-scoped, máy này, không commit)
project (project-scoped, commit chung)
user    (global ở cấp user/máy)
```

Effective rule: `local > project > user`. Layer có precedence cao hơn override
khai báo của layer thấp hơn. Vắng mặt khai báo ở một layer = `absent` ở layer
đó (rơi xuống layer kế tiếp).

Effective status sau merge:

```text
enabled
disabled
absent   (không khai báo ở bất kỳ layer nào)
unknown  (có khai báo nhưng layer chứa khai báo có scan_status != ok)
```

### Toggle Semantics

UI cho phép user thao tác plugin state ở hai scope:

- **User layer (Global Plugins screen)**: toggle 2-state. Enable / Disable
  globally. Ghi vào `~/.claude/settings.json` (hoặc tương đương) ở user scope.
- **Project layer (Project Detail screen)**: cycle 3-state. Inherit
  (không khai báo ở project layer, fall through xuống user) → Enable (force
  enable ở project) → Disable (force disable ở project) → Inherit. "Inherit"
  được thực thi bằng cách xóa entry khỏi `.claude/settings.json` của project.

Local layer (`.claude/settings.local.json`) chỉ được scan, không được Skillbox
write ở Phase 1. User vẫn có thể chỉnh tay file local để override tạm thời.

### Scan Flow

Một plugin scan operation:

```text
Trigger (manual hoặc auto sau khi mở project / Global Plugins screen)
  -> Với mỗi provider có plugin support:
       -> Resolve settings file path cho layer được scan
            (user: từ ~/.<provider>/settings.json,
             project: từ <project>/.<provider>/settings.json,
             local: từ <project>/.<provider>/settings.local.json)
       -> Defensive checks:
            * file phải nằm trong user home / project root (path_escape)
            * không follow symlink (symlink)
            * size phải dưới ngưỡng (too_large)
       -> Đọc + parse file (JSON/TOML tuỳ provider)
       -> Tạo/cập nhật provider_plugin_layer_scans với scan_status phù hợp
       -> DELETE toàn bộ provider_plugin_entries cho layer_scan_id này
       -> DELETE toàn bộ provider_plugin_marketplaces cho layer_scan_id này
       -> Nếu scan_status = ok:
            -> Reinsert provider_plugin_entries từ parsed content
            -> Reinsert provider_plugin_marketplaces từ parsed content
       -> Ghi parse-time warnings vào scan_warnings (JSON array; bounded)
```

Replace-by-scan strategy: DELETE xảy ra **unconditionally** mỗi lần scan,
bất kể scan_status. Reinsert chỉ xảy ra khi `scan_status = ok`. Kết quả:
nếu file trở thành `missing`, `malformed`, v.v., entries + marketplaces cũ
của layer đó bị xóa sạch thay vì được giữ nguyên. Không cần diff/migrate
khai báo trong code.

### Settings File Paths

Provider settings file paths được seed trong `provider_path_candidates` với
`purpose = config`. Hai layer:

- `scope = global`: user-level settings (ví dụ `~/.claude/settings.json`).
- `scope = project`: project-level settings. Hai path candidate trong scope này
  không cạnh tranh nhau — chúng fill **hai layer slot riêng biệt** qua sort
  `ORDER BY priority DESC`: `.claude/settings.json` (priority = 10) → index 0
  → `project` layer slot; `.claude/settings.local.json` (priority = 9) →
  index 1 → `local` layer slot. Priority column cao hơn được xử lý trước
  (DESC sort), xác định slot nào là project layer và slot nào là local layer
  — không liên quan tới layer merge precedence. Khi merge effective state,
  `local` vẫn có **precedence cao hơn** `project` (rule `local > project >
  user` không đổi).

User có thể override các path này qua `provider_path_overrides` với cùng
`(scope, purpose = config)`.

### Marketplace Concept

Marketplace là **nguồn được đặt tên** mà plugin được resolve từ đó. Khái niệm
do provider định nghĩa, Skillbox chỉ record metadata. Source types thường gặp:

```text
github      (owner/repo)
git         (git URL)
directory   (local path)
url         (HTTP URL)
settings    (marketplace metadata định nghĩa trong settings tree)
hostPattern (provider-specific routing)
```

`source_type` không enforce CHECK trong migration; mỗi provider có thể có
source type riêng. Marketplace metadata không chứa credentials.

### Provider Plugin Service Boundary

Provider plugin scanner/service responsibilities:

- Đọc settings file theo layer.
- Validate defensive rules trước khi parse.
- Persist scan result vào 3 bảng (`provider_plugin_layer_scans`,
  `provider_plugin_entries`, `provider_plugin_marketplaces`).
- Resolve effective state per project / global view.
- Write enable/disable thay đổi vào settings file của layer phù hợp khi user
  toggle (chỉ user/project layer).

Provider plugin service KHÔNG:

- Tải/install marketplace content (provider tự xử lý).
- Edit local layer (`settings.local.json`).
- Modify managed settings (out of scope cho Phase 1; `ManagedOutOfScope =
  true` luôn trả về để UI hiển thị).

### Domain Object: provider_plugin

Domain layer expose các struct chính (xem `core-go/internal/domain/provider_plugin.go`):

- `PluginLayerScan` — kết quả scan một layer.
- `PluginEntry` — một khai báo plugin trong một scan.
- `PluginMarketplace` — một marketplace declaration.
- `PluginEffectiveEntry` — plugin sau khi resolve effective status, kèm
  per-layer provenance.
- `GlobalPluginView` — view cho Global Plugins screen (user layer per
  provider).
- `ProjectPluginView` — view cho project (merge local + project + user).
- `PluginCount` / `PluginProviderCount` — aggregate cho Dashboard / Projects.

## Open Questions

- Claude convention chính xác nên là gì và path nào nên được adapter support?
- Codex/opencode/Antigravity CLI có cần adapter riêng ngay, hay dùng
  `generic_agents` trước?
- Có nên cho user tạo custom provider trong UI ở Phase 1 không?
- Provider icon nên dùng bundled asset, icon key, hay package icon set?
