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
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## provider_path_candidates

Purpose: lưu các candidate path mà provider adapter dùng để detect/config/install.

| Field | Type | Nullable | Description |
|---|---|---:|---|
| `id` | integer | no | Primary key. |
| `provider_definition_id` | integer | no | FK tới `provider_definitions.id`. |
| `relative_path` | text | no | Path tương đối từ project root. |
| `purpose` | text | no | Candidate purpose. Allowed: `detect`, `skills`, `commands`, `config`. |
| `priority` | integer | no | Priority thấp/cao theo convention implementation chọn. Dùng để resolve candidate chính khi nhiều path tồn tại. |
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
| `target_type` | text | no | Scan target. Allowed: `skill_host_folder`, `project`, `project_provider`. |
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
| `operation_type` | text | no | Operation kind. Allowed: `scan`, `fetch`, `update_host_skill`, `sync_install`, `install_skill`, `remove_install`, `switch_install_mode`, `change_skill_host_folder`. |
| `target_type` | text | no | Target object type của operation, ví dụ `project`, `skill`, `install`, `skill_host_folder`. |
| `target_id` | integer | yes | ID của target. Polymorphic FK, validate ở app layer. |
| `status` | text | no | Operation status. Allowed: `queued`, `running`, `success`, `failed`, `cancelled`, `partial`. |
| `started_at` | datetime | yes | Thời điểm operation bắt đầu. |
| `finished_at` | datetime | yes | Thời điểm operation kết thúc. |
| `error_message` | text | yes | Error message nếu failed hoặc partial. |
| `metadata_json` | json | yes | Operation-specific metadata như counts, affected projects, changed paths. |
| `created_at` | datetime | no | Thời điểm tạo row. |
| `updated_at` | datetime | no | Thời điểm cập nhật row gần nhất. |

## Polymorphic References

SQLite không enforce được các polymorphic references như:

- `warnings.scope_type` + `warnings.scope_id`
- `operations.target_type` + `operations.target_id`
- `scan_results.target_type` + `scan_results.target_id`

App layer phải validate các reference này.
