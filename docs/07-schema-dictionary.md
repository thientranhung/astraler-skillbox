# Schema Dictionary

Tài liệu này mô tả chi tiết các table và field dự kiến cho SQLite. Đây là tài
liệu field-level để AI, reviewer, và developer hiểu đúng ý nghĩa nghiệp vụ của
từng cột.

`06-data-model.md` là conceptual model. File này là schema reference.

## Conventions

- `integer` dùng cho primary key, foreign key, boolean dạng `0/1`, hoặc priority.
- `text` dùng cho enum, path, id ngoài hệ thống, message, version, commit, hash.
- `datetime` lưu ISO-8601 string hoặc SQLite-compatible timestamp.
- `json` là text chứa JSON hợp lệ.
- Path trong database nên là absolute path trừ các field có tên `relative_path`.
- Enum được lưu dạng text để dễ debug và thân thiện với AI.

## app_settings

Purpose: lưu cấu hình global của app.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. Thường chỉ có một row active cho app settings. |
| `active_skill_host_folder_id` | integer | yes | FK tới `skill_host_folders.id`. Nullable trong first-time setup trước khi user chọn Skill Host Folder. |
| `default_install_mode` | text | no | Install mode mặc định khi user cài skill. Allowed: `symlink`, `rsync_copy`. |
| `database_version` | integer | no | Version schema hiện tại, dùng cho migration. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## api_credentials

Purpose: lưu metadata về credentials cho GitHub/Vercel fetch. Không lưu plaintext
token trong SQLite.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_key` | text | no | Provider dùng credential. Allowed: `github`, `vercel`. |
| `credential_type` | text | no | Loại credential. Allowed: `token`, `oauth`, `ssh_key`. |
| `storage_type` | text | no | Nơi lưu secret thật. Allowed: `os_keychain`, `encrypted_sqlite`, `environment`. |
| `credential_ref` | text | yes | Reference tới keychain item hoặc environment variable name. |
| `value_encrypted` | text | yes | Secret encrypted nếu `storage_type = encrypted_sqlite`. Không lưu plaintext. |
| `status` | text | no | Trạng thái credential. Allowed: `active`, `missing`, `invalid`, `expired`. |
| `last_validated_at` | datetime | yes | Lần gần nhất app kiểm tra credential còn hợp lệ. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## skill_host_folders

Purpose: lưu các folder từng được user chọn làm Skill Host Folder.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `name` | text | yes | Tên hiển thị do user đặt hoặc app suy ra từ folder name. |
| `path` | text | no | Absolute path tới folder user chọn làm Skill Host Folder. |
| `skills_path` | text | no | Absolute path tới nơi chứa skill, thường là `<skill-host-folder>/.agents/skills`. |
| `status` | text | no | Host state. Allowed: `active`, `missing`, `unreadable`, `unwritable`, `invalid_structure`, `empty`, `inactive`. |
| `last_scanned_at` | datetime | yes | Lần gần nhất app scan Skill Host Folder. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## skills

Purpose: đại diện cho một skill trong Skill Host Folder. Skill content nằm trên
filesystem, database chỉ lưu metadata.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `skill_host_folder_id` | integer | no | FK tới `skill_host_folders.id`. |
| `name` | text | no | Canonical skill name, thường là folder name trong `.agents/skills`. |
| `display_name` | text | yes | Tên hiển thị thân thiện hơn nếu app đọc được metadata. |
| `relative_path` | text | no | Relative path từ Skill Host Folder, thường là `.agents/skills/<skill-name>`. |
| `absolute_path` | text | no | Absolute path tới folder skill trong Skill Host Folder. |
| `status` | text | no | Skill state. Allowed: `available`, `missing`, `unreadable`, `local_modified`, `unknown`. |
| `source_id` | integer | yes | FK tới `skill_sources.id`. Nullable cho local/manual skill chưa có source metadata. |
| `current_version` | text | yes | Version hiện tại của skill trong Skill Host Folder nếu source có version. |
| `current_commit` | text | yes | Commit hiện tại của skill trong Skill Host Folder nếu source là git/GitHub. |
| `current_checksum` | text | yes | Hash/checksum nội dung hiện tại, dùng cho local modification và rsync/copy drift detection. |
| `last_scanned_at` | datetime | yes | Lần gần nhất app scan skill này trong Skill Host Folder. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## skill_sources

Purpose: lưu source/upstream metadata cho skill.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `source_type` | text | no | Loại source. Allowed: `github`, `vercel_skills`, `local`, `manual`. |
| `url` | text | yes | URL source gốc nếu có. |
| `github_owner` | text | yes | GitHub owner/org nếu `source_type = github`. |
| `github_repo` | text | yes | GitHub repo nếu `source_type = github`. |
| `github_path` | text | yes | Subfolder trong repo nếu skill không nằm ở repo root. |
| `github_ref` | text | yes | Branch, tag, hoặc commit ref đang theo dõi. |
| `vercel_skill_id` | text | yes | Identifier trong Vercel skills ecosystem nếu có. |
| `local_source_path` | text | yes | Absolute path tới local source nếu `source_type = local`. |
| `resolved_version` | text | yes | Version hiện tại đã resolve từ source. |
| `resolved_commit` | text | yes | Commit hiện tại đã resolve từ source. |
| `last_fetched_at` | datetime | yes | Lần fetch attempt gần nhất, kể cả failed attempt. |
| `last_successful_fetch_at` | datetime | yes | Lần fetch thành công gần nhất. |
| `last_fetch_status` | text | no | Latest fetch summary. Allowed: `never_fetched`, `up_to_date`, `update_available`, `failed`, `auth_required`, `not_found`, `network_error`, `needs_review`, `not_fetchable`. |
| `last_fetch_error` | text | yes | Error message của fetch attempt gần nhất nếu failed. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## projects

Purpose: lưu các project được user add vào Skillbox.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `name` | text | no | Tên project hiển thị trong UI, thường suy ra từ folder name. |
| `path` | text | no | Absolute path tới project root. |
| `status` | text | no | Project lifecycle/filesystem state. Allowed: `active`, `missing`, `unreadable`, `removed`. |
| `last_scanned_at` | datetime | yes | Lần gần nhất app scan project. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

Notes:

- `has_warnings` và `no_provider_detected` không nằm trong `projects.status`.
  Chúng là derived state từ bảng `warnings`.

## provider_definitions

Purpose: lưu danh sách provider/convention mà Skillbox biết.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `key` | text | no | Stable provider key dùng trong code/config, ví dụ `claude`, `generic_agents`. |
| `display_name` | text | no | Tên hiển thị trong UI. |
| `provider_type` | text | no | Provider category. Allowed: `claude`, `codex`, `opencode`, `antigravity_cli`, `generic_agents`, `custom`, `unsupported`. |
| `icon_key` | text | yes | Key để UI chọn icon phù hợp. |
| `status` | text | no | Adapter support state. Allowed: `supported`, `experimental`, `unsupported`, `disabled`. |
| `can_create_structure` | integer | no | Boolean `0/1`. Cho biết core Skillbox logic có thể scaffold provider folder structure cho provider này hay không. |
| `has_global_level` | integer | no | Boolean `0/1`. Cho biết provider có global/user-level location mà Skillbox có thể scan hoặc cấu hình. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

Notes:

- `generic_agents` uses `can_create_structure = 1` so project installs may
  create the selected project's `.agents/skills` folder. Provider-specific
  definitions such as `claude` must remain `0` until their conventions are
  verified and documented.

## provider_path_candidates

Purpose: lưu các candidate path mà provider adapter dùng để detect/config/install.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. |
| `relative_path` | text | no | Path tương đối từ project root. |
| `purpose` | text | no | Candidate purpose. Allowed: `detect`, `skills`, `commands`, `config`. |
| `priority` | integer | no | Lower value wins. Priority `1` is checked before priority `10`. Dùng để resolve candidate chính khi nhiều path tồn tại. |
| `description` | text | yes | Mô tả vì sao path này tồn tại hoặc provider dùng nó cho việc gì. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## project_providers

Purpose: lưu provider được phát hiện hoặc cấu hình trong từng project.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `project_id` | integer | no | FK tới `projects.id`. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. |
| `detected_path` | text | yes | Absolute path được resolve từ candidate `purpose = detect`. |
| `skills_path` | text | yes | Absolute path nơi provider này nhận skill installs. Resolve từ candidate `purpose = skills`. |
| `detection_status` | text | no | Detection state trong project. Allowed: `detected`, `configured`, `missing`, `unsupported`, `invalid_structure`, `format_unknown`. |
| `last_scanned_at` | datetime | yes | Lần gần nhất app scan provider scope này. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## installs

Purpose: lưu việc một skill được cài vào một project/provider.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `project_provider_id` | integer | no | FK tới `project_providers.id`. Project được suy ra qua provider scope này. |
| `skill_id` | integer | yes | FK tới `skills.id`. Nullable cho direct/manual/unknown installs không map được skill trong host. |
| `skill_name` | text | no | Skill name ghi tại thời điểm scan/install. Không tự động sync ngược từ `skills.name`. |
| `install_mode` | text | no | Install mechanism/intent only. Allowed: `symlink`, `rsync_copy`, `direct`. Không lưu filesystem anomaly ở đây. |
| `install_status` | text | no | Detected current state. Allowed: `current`, `outdated`, `missing`, `broken_symlink`, `old_host`, `external_symlink`, `conflict`, `needs_sync`, `error`. |
| `project_skill_path` | text | no | Absolute path tới skill entry trong provider folder của project. |
| `source_skill_path` | text | yes | Absolute path tới skill trong Skill Host Folder nếu managed. |
| `symlink_target_path` | text | yes | Symlink target nếu `project_skill_path` là symlink. Dùng để detect broken/old/external symlink. |
| `installed_from_host_folder_id` | integer | yes | FK tới `skill_host_folders.id` tại thời điểm install. Dùng cho old host detection. |
| `installed_version` | text | yes | Version đã install/sync vào project nếu biết. |
| `installed_commit` | text | yes | Commit đã install/sync vào project nếu biết. |
| `installed_checksum` | text | yes | Hash/checksum snapshot trong project, dùng cho rsync/copy drift detection. |
| `last_synced_at` | datetime | yes | Lần gần nhất rsync/copy install được sync từ Skill Host Folder. |
| `last_scanned_at` | datetime | yes | Lần gần nhất install này được scan từ filesystem. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

Notes:

- Scan thấy symlink trên disk thì `install_mode = symlink`, bất kể symlink do
  Skillbox hay user tạo. `install_status` phân biệt state thật.
- Phase 1 dùng hard delete khi user remove install bằng Skillbox.

## global_provider_locations

Purpose: lưu provider global locations ở cấp user/máy.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. |
| `name` | text | yes | Tên hiển thị cho global location, ví dụ Claude Global hoặc Generic Agents Global. |
| `path` | text | yes | Absolute path tới provider global root/location. Nullable khi global location chưa được cấu hình. |
| `skills_path` | text | yes | Absolute path nơi provider global level nhận skill/global entries nếu có. |
| `status` | text | no | Global location state. Allowed: `active`, `not_configured`, `missing`, `unreadable`, `invalid_structure`, `empty`, `disabled`. |
| `last_scanned_at` | datetime | yes | Lần gần nhất app scan global location này. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## global_installs

Purpose: lưu skill/global entry tồn tại ở provider global level.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `global_provider_location_id` | integer | no | FK tới `global_provider_locations.id`. |
| `skill_id` | integer | yes | FK tới `skills.id`. Nullable cho direct/manual global entries không map được skill trong host. |
| `skill_name` | text | no | Skill/global entry name ghi tại thời điểm scan/install. |
| `install_mode` | text | no | Install mechanism/intent only. Allowed: `symlink`, `rsync_copy`, `direct`. |
| `install_status` | text | no | Detected current state. Allowed: `current`, `outdated`, `missing`, `broken_symlink`, `old_host`, `external_symlink`, `conflict`, `needs_sync`, `error`. |
| `global_skill_path` | text | no | Absolute path tới global skill/entry trong provider global location. |
| `source_skill_path` | text | yes | Absolute path tới skill trong Skill Host Folder nếu managed. |
| `symlink_target_path` | text | yes | Symlink target nếu `global_skill_path` là symlink. |
| `installed_from_host_folder_id` | integer | yes | FK tới `skill_host_folders.id` tại thời điểm install. |
| `installed_version` | text | yes | Version đã install/sync vào global location nếu biết. |
| `installed_commit` | text | yes | Commit đã install/sync vào global location nếu biết. |
| `installed_checksum` | text | yes | Hash/checksum snapshot trong global location. |
| `last_synced_at` | datetime | yes | Lần gần nhất rsync/copy global install được sync từ Skill Host Folder. |
| `last_scanned_at` | datetime | yes | Lần gần nhất global install này được scan từ filesystem. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

Notes:

- Global installs phải được UI phân biệt rõ với project-level installs.

## fetch_results

Purpose: lưu kết quả fetch upstream cho một source.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `source_id` | integer | no | FK chính tới `skill_sources.id`. Skill context suy ra qua `skills.source_id`. |
| `status` | text | no | Fetch result. Allowed: `up_to_date`, `update_available`, `failed`, `auth_required`, `not_found`, `network_error`, `needs_review`, `not_fetchable`. |
| `host_version_at_fetch` | text | yes | Version trong Skill Host Folder tại thời điểm fetch. |
| `upstream_version_at_fetch` | text | yes | Version upstream được phát hiện tại thời điểm fetch. |
| `host_commit_at_fetch` | text | yes | Commit trong Skill Host Folder tại thời điểm fetch. |
| `upstream_commit_at_fetch` | text | yes | Commit upstream được phát hiện tại thời điểm fetch. |
| `fetched_at` | datetime | no | Thời điểm fetch attempt. |
| `error_message` | text | yes | Error message nếu fetch failed. |
| `raw_metadata_json` | json | yes | Metadata thô từ GitHub/Vercel/source adapter để debug. |
| `created_at` | datetime | no | Thời điểm tạo row. |

Notes:

- Phase 1 nên giới hạn retention theo `source_id`, ví dụ giữ N row gần nhất.

## scan_results

Purpose: lưu kết quả scan gần nhất hoặc lịch sử scan nhẹ cho host/project/provider.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `target_type` | text | no | Scan target. Allowed: `skill_host_folder`, `project`, `project_provider`, `global_provider_location`. |
| `target_id` | integer | no | ID của target tương ứng. Polymorphic FK, validate ở app layer. |
| `status` | text | no | Scan result. Allowed: `success`, `partial`, `failed`, `cancelled`. |
| `started_at` | datetime | no | Thời điểm scan bắt đầu. |
| `finished_at` | datetime | yes | Thời điểm scan kết thúc. Nullable khi đang chạy. |
| `summary_json` | json | yes | Counts và summary như skills found, providers found, warnings. |
| `error_message` | text | yes | Error message nếu scan failed hoặc partial. |
| `created_at` | datetime | no | Thời điểm tạo row. |

## warnings

Purpose: lưu warning/recoverable/blocking states để UI hiển thị nhất quán.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `scope_type` | text | no | Scope của warning. Allowed: `app`, `skill_host_folder`, `skill`, `project`, `project_provider`, `install`, `global_provider_location`, `global_install`, `source`, `database`. |
| `scope_id` | integer | yes | ID của scoped object. Nullable cho app-level hoặc database-level warning. Polymorphic FK, validate ở app layer. |
| `severity` | text | no | Severity. Allowed: `info`, `warning`, `error`, `blocking`. |
| `code` | text | no | Stable warning code, ví dụ `broken_symlink`, `fetch_failed`, `project_missing`. |
| `message` | text | no | Message hiển thị hoặc debug-friendly text. |
| `action_key` | text | yes | Suggested action key cho UI, ví dụ `rescan`, `retry`, `relink`, `sync`, `choose_folder`. |
| `source_operation_id` | integer | yes | FK tới `operations.id` nếu warning được tạo bởi một operation/scan. |
| `is_resolved` | integer | no | Boolean `0/1`. Cho biết warning đã được resolve hoặc superseded. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |
| `resolved_at` | datetime | yes | Thời điểm warning được resolve nếu có. |

## operations

Purpose: lưu operation dài hoặc quan trọng để UI có loading/progress/debug state.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `operation_type` | text | no | Operation kind. Allowed: `scan`, `fetch`, `update_host_skill`, `sync_install`, `install_skill`, `remove_install`, `switch_install_mode`, `change_skill_host_folder`, `scan_global_skills`. |
| `target_type` | text | no | Target object type của operation, ví dụ `project`, `skill`, `install`, `skill_host_folder`. |
| `target_id` | integer | yes | ID của target. Polymorphic FK, validate ở app layer. |
| `status` | text | no | Operation status. Allowed: `queued`, `running`, `success`, `failed`, `cancelled`, `partial`. |
| `started_at` | datetime | yes | Thời điểm operation bắt đầu. |
| `finished_at` | datetime | yes | Thời điểm operation kết thúc. |
| `error_message` | text | yes | Error message nếu failed hoặc partial. |
| `metadata_json` | json | yes | Operation-specific metadata như counts, affected projects, changed paths. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## provider_user_settings

Purpose: lưu user-level preference cho từng provider (Phase 1 chỉ enable/disable).

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. UNIQUE — một row duy nhất per provider. ON DELETE CASCADE. |
| `enabled` | integer | no | Boolean `0/1`. User preference cho provider. CHECK `enabled IN (0, 1)`. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## provider_path_overrides

Purpose: lưu override của user cho path candidate của provider. Một row cho mỗi
`(provider_definition_id, scope, purpose)`. Khi có override, adapter dùng
`paths_json` thay vì `provider_path_candidates` cho slot đó.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. |
| `scope` | text | no | Scope của override. Allowed: `project`, `global`. |
| `purpose` | text | no | Slot được override. Allowed: `detect`, `skills`, `config`, `commands`. |
| `paths_json` | json | no | JSON array path strings thay thế built-in candidates. Mặc định `'[]'`. CHECK `json_valid` AND `json_type = 'array'`. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

Notes:

- UNIQUE `(provider_definition_id, scope, purpose)` đảm bảo mỗi slot chỉ có
  một override active.
- Path trong `paths_json` có thể là absolute hoặc bắt đầu bằng `~` (user home);
  adapter resolve trước khi dùng.

## provider_plugin_layer_scans

Purpose: lưu kết quả scan một settings file ở một layer của provider plugin
system. Một row mỗi `(provider_definition_id, project_id, settings_layer)`.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. ON DELETE CASCADE. |
| `project_id` | integer | yes | FK tới `projects.id`. NULL khi `settings_layer = user`. Non-null bắt buộc cho `project`/`local`. ON DELETE CASCADE. |
| `settings_layer` | text | no | Layer precedence. Allowed: `user`, `project`, `local`. |
| `scan_status` | text | no | Kết quả đọc settings file. Allowed: `ok`, `missing`, `unreadable`, `malformed`, `too_large`, `symlink`, `path_escape`. |
| `settings_file_path` | text | no | Absolute path tới settings file scanner đã thử đọc. |
| `last_scanned_at` | datetime | no | Thời điểm scan gần nhất. Default `now()`. |
| `source_operation_id` | integer | yes | FK tới `operations.id` của lần scan tạo row này. ON DELETE SET NULL. |
| `scan_warnings` | json | no | JSON array string các parse-time warnings. Mặc định `'[]'`. Không lưu raw file content. |

Notes:

- Partial unique indexes:
  - `(provider_definition_id, settings_layer)` WHERE `project_id IS NULL` (user
    layer).
  - `(provider_definition_id, project_id, settings_layer)` WHERE `project_id IS
    NOT NULL` (project/local layer).
- Table CHECK ràng buộc: user layer phải null `project_id`; project/local layer
  phải non-null `project_id`.
- `scan_status = ok` là điều kiện duy nhất để các entries/marketplaces phát
  sinh từ scan này có hiệu lực.

## provider_plugin_entries

Purpose: lưu khai báo plugin (enabled/disabled) trong một settings file scan.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `layer_scan_id` | integer | no | FK tới `provider_plugin_layer_scans.id`. ON DELETE CASCADE. |
| `plugin_name` | text | no | Tên plugin do settings file khai báo. |
| `marketplace_name` | text | no | Tên marketplace mà plugin được resolve từ đó. |
| `declaration` | text | no | Khai báo trong file. Allowed: `enabled`, `disabled`. |
| `version` | text | yes | *(migration 000021)* Installed version từ `installed_plugins.json`. `NULL` khi không có record (non-Claude providers, plugin chưa cài). `"unknown"` là literal hợp lệ khi Claude không xác định được version. |

Notes:

- UNIQUE `(layer_scan_id, plugin_name, marketplace_name)`.
- Effective status (`enabled`/`disabled`/`absent`/`unknown`) được resolve ở
  application layer bằng cách merge entries theo precedence `local > project >
  user`; không lưu trực tiếp trong bảng.
- Vắng mặt entry trong một layer scan = `absent` ở layer đó.
- `version` chỉ được populate cho Claude provider (user layer): đọc từ
  `~/.claude/plugins/installed_plugins.json` tại thời điểm scan.

## provider_plugin_marketplaces

Purpose: lưu marketplace declaration trong một settings file scan.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `layer_scan_id` | integer | no | FK tới `provider_plugin_layer_scans.id`. ON DELETE CASCADE. |
| `marketplace_name` | text | no | Tên marketplace (named source) trong settings file. |
| `source_type` | text | no | Loại nguồn. Validate ở application layer. Common values: `github`, `git`, `directory`, `url`, `settings`, `hostPattern`. |
| `source_summary` | text | no | Mô tả nguồn (owner/repo, URL, path). Không lưu raw credentials. |

Notes:

- `source_type` không có CHECK constraint trong migration; enum value được
  validate ở application layer theo format settings file của từng provider.
- Một marketplace có thể xuất hiện ở nhiều layer scans (user/project/local);
  effective marketplace list resolve ở application layer.

## plugin_update_check_cache

*(migration 000022)* Purpose: cache kết quả `git ls-remote` cho từng plugin đã cài, TTL mặc định 6 giờ. Upsert theo UNIQUE key mỗi lần `updateCheck.run` chạy thành công hoặc thất bại.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_key` | text | no | Provider của plugin. Phase 1: luôn `"claude"`. |
| `plugin_name` | text | no | Tên plugin (phần trước `@` trong plugin key). |
| `marketplace_name` | text | no | Tên marketplace (phần sau `@` trong plugin key). |
| `source_url` | text | no | HTTPS URL từ `marketplace.json`. Luôn HTTPS (non-HTTPS bị reject trước khi subprocess). |
| `source_ref` | text | yes | Tag hoặc branch (`"v1.5.5"`, `"main"`). `NULL` khi source không khai báo ref. |
| `installed_sha` | text | yes | `gitCommitSha` từ `installed_plugins.json`. `NULL` khi không có. |
| `installed_version` | text | yes | Version string từ `installed_plugins.json`. Reserved — Phase 1 không dùng để so sánh. |
| `remote_sha` | text | yes | SHA trả về bởi `git ls-remote`. `NULL` khi check thất bại hoặc ref không tìm thấy. |
| `remote_latest_tag` | text | yes | Reserved Phase 2 (semver tag scan). Luôn `NULL` trong Phase 1. |
| `update_available` | integer | yes | `0`=up-to-date, `1`=update có sẵn, `NULL`=unknown (thiếu SHA hoặc check lỗi). |
| `checked_at` | text | no | ISO-8601 UTC timestamp lần check gần nhất. |
| `error` | text | yes | Error code nếu check thất bại. Ví dụ: `non_https_scheme_rejected`, `timeout`, `git_not_found`, `ref_not_found`, `host_backoff`. `NULL` khi thành công. |

Notes:

- UNIQUE `(provider_key, plugin_name, marketplace_name)` — mỗi plugin chỉ có 1 row; upsert ghi đè kết quả cũ.
- Không có FK tới bảng plugin: cache là snapshot độc lập; xóa plugin không cascade-delete cache.
- `update_available` được tính bằng `installed_sha != remote_sha` khi cả hai non-NULL; otherwise `NULL`.
- `source_url` được derive từ `marketplace.json` trên disk mỗi lần check — không cache URL lâu dài (ADR-0001 allowlist-from-disk requirement).

## network_settings

*(migration 000022; cột `update_check_enabled` bị drop ở 000023)* Purpose: bảng singleton lưu `cache_ttl_hours` cho update-check. Luôn có đúng 1 row (`id = 1`) được insert bởi migration.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. CHECK `(id = 1)` — đảm bảo singleton. |
| `cache_ttl_hours` | integer | no | TTL cache update-check tính bằng giờ. Default `6`. |
| `created_at` | text | no | ISO-8601 UTC; set bởi migration. |
| `updated_at` | text | no | ISO-8601 UTC; cập nhật khi `SetCacheTTLHours` được gọi. |

Notes:

- Cột `update_check_enabled` đã bị drop ở migration 000023 (ADR-0002): update-check là always-on, không còn opt-in gate. `UpdateCheckService.RunUpdateCheck` không đọc bảng này nữa.
- Không bao giờ delete row này; chỉ UPDATE.

## Polymorphic References

SQLite không enforce được các polymorphic references như:

- `warnings.scope_type` + `warnings.scope_id`
- `operations.target_type` + `operations.target_id`
- `scan_results.target_type` + `scan_results.target_id`

App layer phải validate các reference này.
