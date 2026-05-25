# Slice 2A: Read-Only Projects And Generic Agents Scan — Design

Date: 2026-05-25
Status: draft, pending user approval
Scope: read-only project foundation — Projects nav/list/detail, Add Project, read-only project scan, `generic_agents` detection only. KHÔNG có bất kỳ write-path nào vào project filesystem.

## 1. Purpose And User Value

Sau Slice 1, user có một Skill Host Folder làm source of truth và xem được Skills Library. Nhưng app chưa biết gì về **project** — nơi skill thực sự được dùng. Slice 2A đưa khái niệm Project vào app ở mức an toàn nhất:

- User add một project folder vào Skillbox và app nhớ nó trong DB.
- User scan project để **thấy** project đang có provider nào (`generic_agents`) và đang chứa skill entries nào trong `.agents/skills`, kèm trạng thái thật từ filesystem (symlink/direct, broken/external/old-host).
- User thấy warnings rõ ràng (no provider, project missing, broken symlink) mà không bị app tự ý sửa gì.

Giá trị: **quan sát và chẩn đoán** trạng thái skill của nhiều project trước khi app được phép thao tác. Đây cũng là bước dựng provider abstraction và projects/installs model làm nền cho Slice 2B (symlink install) mà không gánh rủi ro ghi file.

Non-goal cốt lõi: Slice 2A **không cài, không gỡ, không sửa, không tạo** bất cứ thứ gì trong project folder. Mọi thao tác filesystem với project là read-only.

## 2. Current Slice 1 Baseline

Đã có và sẽ tái dùng:

- Walking skeleton: Electron + React + Go sidecar, JSON-RPC stdio NDJSON, handshake `server.ready`, process lifecycle.
- DB (migration `000001_init`): `app_settings`, `skill_host_folders`, `skill_sources`, `skills`, `operations`, `warnings`.
- Patterns đã chạy: filesystem gateway (read/scan/validate/init), repository layer, operation runner + per-target lock + `operation.progress`/`operation.cancel`, CQRS, error taxonomy (`validation_error=1001` … `unknown_error=1099`), contract-first API với `shared/api-contracts` + generated TS.
- RPC: `ping`, `host.choose`, `host.scan`, `skill.list`, `operation.cancel`, `settings.get`.
- React: routes `/`, `/setup`, `/skills`, `/settings`; sidebar = Skills + Settings; TanStack Router (memory) + Query; `lib/core-client`.

Slice 2A mở rộng các layer này, **không refactor** chúng. Mọi thứ thêm vào phải tương thích pattern Slice 1.

## 3. Scope: In / Out

### In

- Sidebar thêm mục `Projects`. Route `/projects` (list) và `/projects/$projectId` (detail).
- **Add Project**: native folder picker (Electron-handled dialog) → persist `projects` row. Idempotent theo `path`.
- **Project list**: hiển thị projects với provider badges, skill/entry count, warning count, status.
- **Project Detail**: project path/status, providers detected, entries đã phân loại group-by-provider, warnings.
- **Read-only project scan**: detect provider, đọc `.agents/skills` top-level, classify entries, reconcile DB. Không ghi vào project FS.
- Provider detection **chỉ `generic_agents`**.
- Classification entries: `direct` folder, `symlink` (current / old_host / external_symlink / broken_symlink), `missing`, `error`.
- `projects.status = missing` + warning khi project path biến mất; bỏ qua provider scan trong trường hợp này.
- Empty `.agents/skills`: provider `detected` với zero entries, không phải error.
- Reuse operation lock per-project, `operation.progress`, `operation.cancel`.
- Seed `provider_definitions` + `provider_path_candidates` cho `generic_agents`; provider khác chỉ được seed nếu Settings metadata hiện có cần, không có adapter và không ảnh hưởng Projects.

### Out (tường minh, defer 2B+)

- `install.create` / symlink creation / bất kỳ FS write nào vào project.
- Remove/unlink install, relink, switch install mode.
- "Set Up Provider" (scaffold `.agents/skills` — cần `can_create_structure` write).
- `project.remove` (xóa project khỏi DB) — **deferred**.
- "Update Path" cho project missing — **deferred**.
- rsync/copy detection-as-managed, sync, checksum/drift.
- Global Skills scan, `global_provider_locations`, `global_installs`.
- Fetch/Updates/sources/credentials/keychain.
- Add/Import Skill to host.
- Adapter cho provider thật khác `generic_agents` (Claude/Codex/opencode/Antigravity). Marker của chúng (vd `.claude`) bị **ignore** trong 2A.
- Dashboard aggregate đầy đủ. Tối đa: nav count nhẹ cho Projects (không bắt buộc; xem §4).
- Packaging/signing, Windows, Skill Detail full.

## 4. UX Design

Theo `docs/09-ui-wireframes.md`, cắt bỏ mọi action ghi-FS.

### Navigation Shell

Sidebar (thứ tự): `Skills`, `Projects`, `Settings`. (Dashboard/Global/Updates vẫn defer.) Mục Projects có thể kèm count nhẹ (số projects) — optional, không phải Dashboard.

### Projects List (`/projects`)

```text
Projects

[Add Project] [Scan All]

Filters
  Provider: all / Generic Agents / none
  Status: all / active / missing / warnings
  Search: __________________

Table
  Project        Path                  Providers          Skills   Warnings
  content-lab    /repo/content-lab     Generic Agents     8        1
  old-project    /repo/old-project     -                  0        missing
```

- Row actions trong 2A: **Open Detail**, **Scan**, **Open Folder**. KHÔNG có Remove (deferred).
- `Scan All`: scan tuần tự từng project (mỗi project một operation, tôn trọng lock).
- `Skills` count = số observed entries của project.
- `Warnings` count = số active warnings scope project + project_provider + install.

Empty state:

```text
No projects added yet.
[Add Project]
```

### Add Project

1. Click `Add Project` → Electron mở native directory dialog (`dialog.openProjectFolder`).
2. User cancel → no-op.
3. Có path → gọi `project.add { path }`.
4. App persist `projects` row (idempotent theo path), navigate `/projects/$projectId`.
5. **Không** auto-scan ngay (để scan là hành động tường minh, progress quan sát được). UI gợi ý "Scan to detect providers and skills." (Quyết định auto-scan-on-add nằm ở §14.)

Validation: path phải absolute, tồn tại, là directory. Sai → `validation_error` toast.

### Project Detail (`/projects/$projectId`)

```text
Project Detail: content-lab

Path: /repo/content-lab
Status: active                 last scanned 2026-05-25 10:31

Providers
  [Generic Agents] supported   .agents/skills    5 entries

Actions
  [Scan Project] [Open Folder]

Entries (Generic Agents)
  Skill                  Mode        Status
  documentation-writer   symlink     current
  adr-helper             symlink     old host
  legacy-note            direct      current
  ghost                  symlink     broken symlink
  out-there             symlink     external symlink
```

- Group-by-provider. Trong 2A chỉ có nhiều nhất một provider (`generic_agents`).
- **Cột Actions ẩn/empty** — không có Remove/Relink/Switch (toàn bộ là write, defer 2B).
- Warnings hiển thị dạng banner phía trên entries, mỗi warning chỉ kèm read-only action (`Rescan`, `Open Folder`).

Provider/empty edge states:

```text
No provider detected in this project.
[Rescan]
```

(KHÔNG hiện `[Set Up Provider]` vì cần write.)

```text
[warning] Project path missing: /repo/old-project   [Rescan]
```

(KHÔNG hiện `[Update Path]` / `[Remove]` — deferred.)

### Loading / Error / Warning States

- Loading: skeleton rows cho list và detail; scan chạy → `OperationProgressToast` (reuse Slice 1) hiển thị phases (`reading_project`, `detecting_providers`, `classifying_entries`, `done`).
- Query error (core unavailable / DB): `ErrorDisplay` với `userMessage` + Retry.
- Warning vs blocking: warning (broken/external/old_host/no_provider/project_missing/project_unreadable) hiển thị non-blocking; DB/core error dừng action liên quan, không mutate state cũ.

## 5. Domain / Data Model

Migration mới (đề xuất `000002_projects.up.sql`). Tất cả bảng và field bám **đúng** `docs/06-data-model.md` + `docs/07-schema-dictionary.md`. Không phát minh field ngoài docs.

### Bảng activate trong 2A

- **`projects`** — `id, name, path (UNIQUE), status (active|missing|unreadable|removed), last_scanned_at, created_at, updated_at`. 2A dùng `active`, `missing`, `unreadable`. `removed` không dùng (project.remove deferred).
- **`provider_definitions`** — seed data. 2A bắt buộc seed `generic_agents` và chỉ `generic_agents` có adapter/được detect trong Projects. Seed provider khác chỉ được phép nếu đã cần cho metadata Settings hiện có; dữ liệu đó **không** được ảnh hưởng Projects scan/UI.
- **`provider_path_candidates`** — seed cho `generic_agents`: `(purpose=detect, relative_path=.agents, priority=10)`, `(purpose=skills, relative_path=.agents/skills, priority=10)`.
- **`project_providers`** — `project_id, provider_definition_id, detected_path, skills_path, detection_status (detected|missing|invalid_structure), last_scanned_at, …`. 2A dùng `detected`, `missing`, `invalid_structure`. KHÔNG dùng `configured` (cần manual config UI, defer), `unsupported`/`format_unknown` (không phát sinh với generic_agents-only).
- **`installs` — dùng như "observed entries", KHÔNG phải Skillbox-managed installs.**
- **`warnings`** — reuse, scope mở rộng sang `project`, `project_provider`, `install`.
- **`operations`** — reuse, `operation_type=scan`, `target_type=project`.

### `installs` as observed entries (quan trọng)

Trong 2A, mỗi row `installs` đại diện cho **một entry quan sát được trên filesystem** trong `project_providers.skills_path`, KHÔNG phải bằng chứng rằng Skillbox đã cài/quản lý nó.

- `install_mode` lưu **cơ chế quan sát**: `symlink` (entry là symlink) hoặc `direct` (folder thường). 2A **không bao giờ** ghi `rsync_copy` vì không có managed-copy record nào do app tạo.
- `install_status` lưu **trạng thái phát hiện**: `current | old_host | external_symlink | broken_symlink | missing | error`. (`outdated`, `needs_sync`, `conflict` không phát sinh trong read-only 2A.)
- `installed_from_host_folder_id` chỉ được set khi symlink resolve vào một known host (xem §7). Với `direct`/`external` để `null`.
- `source_skill_path` set khi target nằm trong bất kỳ known host `skills_path` (`current` hoặc `old_host`). `install_status` mới thể hiện active-vs-old distinction. `symlink_target_path` set raw cho mọi symlink (kể cả broken/external) để chẩn đoán.
- `skill_id` chỉ set theo rule chặt ở §7 (exact relative path match trong known host); ngược lại `null` và giữ `skill_name`.

Hệ quả thiết kế cần truyền đạt cho user: vì không có managed metadata, **mọi folder thường = `direct`**, và một symlink do user tạo tay trỏ vào active host vẫn hiện `current` (app không phân biệt "ai tạo"). Điều này nhất quán với `docs/08-provider-model.md`.

### `scan_results` — deferred tactical shortcut

`scan_results` (docs/06 §14, docs/07) **không** được tạo trong 2A.

- Trạng thái reconciled hiện tại sống trong core tables (`projects`, `project_providers`, `installs`, `warnings`).
- Summary của một lần scan (counts: providers found, entries classified, warnings created) lưu trong `operations.metadata_json` — nhất quán với quyết định Slice 1.
- **UI/API trong 2A KHÔNG được phụ thuộc `scan_results`.** Nếu Updates/history sau này cần, sẽ thêm bảng ở slice riêng.

### Relationships (2A subset)

```text
projects.id            -> project_providers.project_id
provider_definitions.id-> project_providers.provider_definition_id
provider_definitions.id-> provider_path_candidates.provider_definition_id
project_providers.id   -> installs.project_provider_id
skills.id              -> installs.skill_id           (nullable, chặt theo §7)
skill_host_folders.id  -> installs.installed_from_host_folder_id  (nullable)
operations.id          -> warnings.source_operation_id
```

Polymorphic refs (`warnings.scope_type/scope_id`, `operations.target_type/target_id`) validate ở app layer (theo docs/07 §Polymorphic References).

## 6. Provider Detection: generic_agents Rules

Theo `docs/08-provider-model.md`. 2A chỉ load `generic_agents` (status `supported`).

Adapter `GenericAgentsAdapter` resolve candidates từ project root:

- `detect`: `<root>/.agents` (priority 10).
- `skills`: `<root>/.agents/skills` (priority 10).

Rules:

1. `<root>/.agents` không tồn tại → **không** tạo `project_providers` row → warning `no_provider_detected` (scope `project`, severity `warning`, action_key `rescan`). Không blocking; `projects.status` vẫn `active`.
2. `<root>/.agents` tồn tại & là directory → tạo/update `project_providers`: `detection_status=detected`, `detected_path=<root>/.agents`, `skills_path=<root>/.agents/skills`.
3. `<root>/.agents` tồn tại nhưng `.agents/skills` chưa có → provider `detected`, `skills_path` vẫn trỏ `<root>/.agents/skills`, entries = 0. KHÔNG scaffold (write). Không error.
4. `<root>/.agents` tồn tại nhưng không đọc được (permission) → `detection_status=invalid_structure` + warning (severity `warning`, action_key `rescan`).
5. `<root>/.agents` tồn tại nhưng là **file** (không phải dir) → `detection_status=invalid_structure` + warning.
6. Marker provider khác (`.claude`, …) → **ignore** hoàn toàn trong 2A (không detect, không warning).
7. Rescan thấy `.agents` từng có nay mất → `project_providers.detection_status=missing`, các `installs` thuộc provider đó → `install_status=missing` (reconcile, không xóa row trừ khi entry biến mất nhưng provider còn — xem §7).

Adapter chỉ trả **structured facts** (detected_path, skills_path, detection_status, entries, warnings) theo minimum output contract ở `docs/08`. Adapter KHÔNG ghi DB, KHÔNG ghi FS. Core service ghi DB.

## 7. Read-Only Scan Classification Semantics

Scan đọc **top-level entries** trong `skills_path` (skill = folder một cấp; KHÔNG đệ quy vào trong skill). Với mỗi entry, gateway trả raw facts (is dir / is symlink / symlink target raw / resolved realpath / broken?), service phân loại:

### Symlink entries → `install_mode = symlink`

Resolve target qua canonicalize/realpath, rồi so với tập **known hosts** = tất cả rows `skill_host_folders` (cả `active` và `inactive`):

- Target không resolve được (đích không tồn tại) → `broken_symlink`.
- Target nằm trong **active** host `skills_path` → `current`.
- Target nằm trong một **inactive/known** host `skills_path` (không phải active) → `old_host`.
- Target tồn tại nhưng **ngoài mọi known host** → `external_symlink`.
- Không phân loại an toàn được (symlink loop, lỗi IO khi resolve) → `error`.

`symlink_target_path` luôn lưu raw target. `installed_from_host_folder_id` set tới host khớp khi `current`/`old_host`.

### skill_id matching (chặt, không fuzzy)

Chỉ set `installs.skill_id` khi **đồng thời**:

1. Symlink resolve vào trong một known host (`current` hoặc `old_host`), và
2. Target đã canonicalize khớp một skill đã scan trong host đó bằng cùng một dạng path canonical: hoặc `canonical(resolved_target) == canonical(skills.absolute_path)`, hoặc `rel(canonical(resolved_target), canonical(matched_host.path)) == skills.relative_path`. KHÔNG so `rel(resolved_target, host.skills_path)` với `skills.relative_path`, vì `skills.relative_path` được lưu relative từ Skill Host Folder.

Ngược lại `skill_id = null`, giữ `skill_name` (= tên entry trên disk). KHÔNG match theo tên-gần-đúng. Không có "management semantics" suy ra từ tên.

### Folder thường (non-symlink directory) → `install_mode = direct`

`install_status = current`, `skill_id = null`, `installed_from_host_folder_id = null`. (Theo docs/08: thiếu managed metadata thì folder thường luôn `direct`.)

### Entry khác (file thường, socket, …)

Không phân loại an toàn được → `install_status = error` + warning (info/warning, action_key `open_folder`).

### Reconcile

- Entry còn trên disk → upsert row tương ứng.
- Row `installs` cũ trong DB mà entry đã biến mất trên disk → `install_status = missing` (không hard delete — hard delete chỉ dành cho user-initiated remove ở 2B).
- Warnings regenerate theo scope sau mỗi scan (clear active rồi insert lại) — nhất quán Slice 1.
- `project_providers.last_scanned_at`, `projects.last_scanned_at` cập nhật.
- Clear/insert warnings và upsert/reconcile rows chỉ xảy ra trong **final DB transaction** sau khi classification hoàn tất, để cancel/failure không làm UI mất trạng thái cũ giữa chừng. Ngoại lệ duy nhất là terminal state có chủ ý: project root `missing` hoặc `unreadable`, khi service cập nhật `projects.status` + warning tương ứng và bỏ qua provider/entry mutation.

### Project path missing

Nếu project root không tồn tại lúc scan → `projects.status=missing` + warning `project_missing` (scope `project`, action_key `rescan`). **Bỏ qua** provider detection/entry classification. Không mutate provider/install rows cũ ngoài việc giữ nguyên (chúng phản ánh lần scan trước).

### Project path unreadable

`project.add` chỉ yêu cầu path tồn tại và là directory; không yêu cầu đọc sâu toàn bộ cây. Nếu `project.scan` thấy root tồn tại nhưng không đọc được, set `projects.status=unreadable` + warning `project_unreadable` (scope `project`, action_key `open_folder` hoặc `rescan`) và bỏ qua provider detection/entry classification. Khi scan sau đọc được root, status quay lại `active` trong final transaction.

## 8. API / RPC Contract Sketch (conceptual)

Đủ cụ thể để chốt JSON Schema ở milestone contracts sau, nhưng chưa khóa shape. Naming dot-namespaced, nhất quán Slice 1. Error codes reuse taxonomy hiện có; lần đầu dùng `provider_error=1003`.

### Electron-handled

```text
dialog.openProjectFolder
  Request:  {}
  Response: { path: string | null }     // null nếu user cancel; Electron-handled, không forward Go
```

### Queries (no side effect)

```text
project.list
  Request:  {}
  Response: {
    projects: Array<{
      id, name, path,
      status: "active" | "missing" | "unreadable",
      providers: Array<{ key, displayName, providerStatus, detectionStatus }>,
      skillCount: number,         // observed entries
      warningCount: number,
      lastScannedAt: string | null
    }>
  }

project.get
  Request:  { projectId: number }
  Response: {
    project: {
      id, name, path,
      status: "active" | "missing" | "unreadable",  // same enum as project.list
      lastScannedAt
    },
    providers: Array<{
      projectProviderId, providerKey, displayName,
      providerStatus: "supported" | "experimental" | "unsupported" | "disabled",
      detectionStatus: "detected" | "missing" | "invalid_structure",
      detectedPath: string | null, skillsPath: string | null,
      entryCount: number
    }>,
    entries: Array<{                 // installs-as-observed-entries
      id, projectProviderId, providerKey,
      name: string,                  // skill_name on disk
      mode: "symlink" | "direct",
      status: "current" | "old_host" | "external_symlink" | "broken_symlink" | "missing" | "error",
      projectSkillPath: string,
      symlinkTargetPath: string | null,
      skillId: number | null
    }>,
    warnings: Array<{ code, severity, message, scopeType, scopeRef: string | null, actionKey: string | null }>
  }
  Errors: validation_error (projectId không tồn tại)
```

### Commands

```text
project.add
  Request:  { path: string }            // absolute, từ dialog.openProjectFolder
  Response: { projectId, name, path, status }
  Behavior: normalize to an absolute clean path before storage; idempotent by that
            normalized path (UNIQUE). Do not use realpath for uniqueness in 2A,
            so symlinked project roots are preserved as user-selected paths.
  Errors:   validation_error (không absolute / không tồn tại / không phải dir),
            database_error

project.scan
  Request:  { projectId: number }
  Response: { operationId: number }     // long-running; progress qua operation.progress
  Errors:   validation_error (projectId không tồn tại),
            conflict_error (project này đang có scan active)
```

### Reused (Slice 1)

```text
operation.cancel { operationId } -> { acknowledged }
operation.progress (notification)
  { operationId, status, phase, processed|null, total|null, message|null }
  phases 2A: "reading_project" | "detecting_providers" | "classifying_entries" | "done"
```

`additionalProperties: false` ở response (catch typo), request mở (forward-compat) — nhất quán quyết định Slice 1.

## 9. Go Architecture Shape

Bám CLAUDE.md boundaries và 16 patterns. Mọi FS access qua gateway; SQL chỉ ở repositories; adapter trả facts.

### Provider adapter

```text
internal/providers/
  adapter.go            interface ProviderAdapter:
                          Key() string
                          Detect(projectRoot string, fs FsReader) (DetectResult, error)
                          // DetectResult: detectedPath, skillsPath, detectionStatus, entries[], warnings[]
                          // entries = raw facts (name, path, entryType, symlinkTargetRaw, resolvedTarget, broken)
  registry.go           Registry: map provider key -> adapter (2A: chỉ generic_agents)
  generic_agents.go     GenericAgentsAdapter — resolve .agents / .agents/skills,
                        list top-level entries, KHÔNG ghi DB/FS, KHÔNG phân loại managed-state
```

Adapter trả raw facts + provider-level detection_status. Việc so target với known hosts và quyết định `current/old_host/external` thuộc **core service** (vì cần truy vấn DB hosts), không thuộc adapter (giữ adapter pure & testable).

### Services (use-case orchestration)

```text
internal/services/
  project_service.go
    AddProject(ctx, path) -> ProjectResult          // validate path, upsert idempotent
    ScanProject(ctx, projectId) -> operationId       // runner.Start, lock target=project
    scanProjectInternal(ctx, progress) -> (summary, err)
        - load project; nếu path missing -> status=missing + warning, return
        - adapter.Detect -> facts
        - classify entries: resolve symlink targets vs known hosts (hostRepo), skill_id match (skillRepo)
        - transaction: upsert project_providers, upsert/reconcile installs,
          clear+insert warnings, update last_scanned_at
        - summary -> operations.metadata_json
  project_detail_service.go (hoặc query trong project_service)
    ListProjects(ctx) -> []ProjectListItem
    GetProject(ctx, projectId) -> ProjectDetailView
```

### Repositories (chỉ nơi viết SQL)

```text
internal/repositories/
  project_repo.go            Upsert idempotent theo path, GetByID, List, UpdateStatus, UpdateLastScannedAt
  provider_definition_repo.go  GetByKey, ListActiveAdapters (seed-aware)
  provider_path_candidate_repo.go  ListByProvider
  project_provider_repo.go   UpsertByProjectAndProvider, ListByProject, MarkMissing
  install_repo.go            UpsertObservedMany (transaction), ListByProjectProvider, MarkMissingAbsent
  (reuse) warning_repo, operation_repo, skill_repo (ListByHost for skill_id match), skill_host_folder_repo (known hosts)
```

Seed `provider_definitions` + `provider_path_candidates`: idempotent seed chạy trong migration hoặc app bootstrap (quyết định ở §14), bằng cách upsert theo `key`.

### Filesystem gateway — chỉ thêm READ methods

```text
internal/filesystem/
  scan_project.go    ScanProjectSkills(skillsPath) -> []Entry
                     Entry { Name, Path, IsDir, IsSymlink, SymlinkTargetRaw, ResolvedTarget, Broken, Kind }
  (reuse) paths.go NormalizeAbs/Realpath, validate.go cho project root readability
```

**KHÔNG** thêm method ghi (create symlink, copy, remove) trong 2A. Đây là ranh giới read-only được enforce ở mức gateway API surface (xem §12).

### Operation runner

Reuse. `Target{Type:"project", ID:projectId}`, `OperationType=scan`. Per-target lock → scan trùng project trả `conflict_error`. `Scan All` lặp tuần tự, mỗi project một operation.

## 10. React Architecture Shape

Bám pattern Slice 1 (`features/`, `screens/`, `lib/core-client`).

### Routes

```text
/projects               Projects list
/projects/$projectId     Project detail
```

Sidebar thêm `Projects` (icon lucide, vd `FolderGit2`).

### core-client

```text
lib/core-client/methods.ts (thêm)
  openProjectFolder: () => invoke("dialog.openProjectFolder", {})
  addProject:        (req) => invoke("project.add", req)
  listProjects:      () => invoke("project.list", {})
  getProject:        (req) => invoke("project.get", req)
  scanProject:       (req) => invoke("project.scan", req)
  // cancelOperation, subscribeOperationProgress reuse
```

### Hooks / features

```text
features/projects/
  use-projects-list.ts     useQuery(queryKey: projects.list)
  use-project-detail.ts    useQuery(queryKey: projects.detail(projectId))
  use-add-project.ts       useMutation; onSuccess invalidate projects.list; navigate detail
  use-scan-project.ts      useMutation -> operationId; subscribe progress;
                           on terminal status invalidate projects.detail + projects.list
  entry-status-badge.tsx   Badge variants theo install_status enum
  provider-badge.tsx       Badge theo provider status + detection status
```

### Query keys & invalidation

```text
query-keys.ts (thêm)
  projects.list
  projects.detail(projectId)

Invalidation:
  addProject success      -> invalidate projects.list
  scanProject terminal    -> invalidate projects.detail(projectId) + projects.list
```

### Screens / components

```text
screens/projects-screen.tsx          list + filters + Add/Scan All
screens/project-detail-screen.tsx     providers + entries + warnings, read-only actions only
components/ (reuse) operation-progress-toast, error-display, empty-state, warning-banner
```

Không có form ghi (Add Skill wizard) — defer 2B.

## 11. Error Handling And Warnings

### Errors (typed, taxonomy Slice 1)

- `validation_error` — projectId/path không hợp lệ.
- `filesystem_error` — lỗi IO bất thường khi scan; root unreadable dự kiến được hạ thành `projects.status=unreadable` + warning.
- `provider_error` — adapter resolve thất bại bất thường (lần đầu dùng code 1003).
- `conflict_error` — scan trùng target.
- `database_error` — DB unavailable.
- `operation_cancelled` — user cancel scan.

Blocking errors dừng action, **không** mutate state cũ (giữ metadata lần scan trước).

### Warnings (regenerate sau scan)

| code | scope_type | severity | action_key (2A) |
|---|---|---|---|
| `no_provider_detected` | project | warning | rescan |
| `project_missing` | project | warning | rescan |
| `project_unreadable` | project | warning | open_folder/rescan |
| `broken_symlink` | install | warning | rescan |
| `external_symlink` | install | info/warning | open_folder |
| `old_host_symlink` | install | info/warning | rescan |
| `invalid_structure` (provider) | project_provider | warning | rescan |
| entry `error` (unclassifiable) | install | info | open_folder |

2A chỉ sinh `action_key` thuộc nhóm **read-only** (`rescan`, `open_folder`). Write action_keys (`relink`, `remove`, `sync`) **không** được sinh ở 2A. UI render warning chỉ với read-only actions; nếu một warning row có action_key ngoài nhóm này thì coi như bug.

## 12. Security / Path-Safety / Read-Only Guarantees

- **Read-only guarantee enforced ở gateway surface**: trong 2A, filesystem gateway **không expose** bất kỳ method ghi nào liên quan project (no create symlink, no copy, no remove). Không có code path nào trong services/adapters có thể ghi vào project folder. Đây là invariant chính của slice.
- **Canonicalize trước khi so sánh**: mọi so sánh "target nằm trong host?" dùng realpath/canonicalize cả target lẫn host skills_path để tránh nhầm do symlink trung gian / `..` / trailing slash.
- **Project path identity**: `project.add` lưu normalized absolute clean path và dùng chính path đó để idempotency/UNIQUE. Không realpath project root trong 2A, để app không đổi path user chọn và không merge hai path symlink khác nhau ngoài ý muốn. Realpath chỉ dùng cho symlink target/host comparison.
- **Symlink resolution an toàn**: guard symlink loop và độ sâu resolve; lỗi → `error` status, không panic.
- **Absolute paths trong DB** (nguyên tắc docs/06). `relative_path`-named fields là ngoại lệ.
- **Electron security defaults** giữ nguyên Slice 1 (contextIsolation, sandbox, CSP, method allowlist). Method mới (`project.*`, `dialog.openProjectFolder`) phải thêm vào allowlist; `dialog.openProjectFolder` Electron-handled, không forward Go.
- **Stdout Go** chỉ JSON-RPC; logs đi stderr/file (giữ Slice 1).
- Không đọc nội dung bên trong skill (chỉ top-level entry metadata) → giảm bề mặt rủi ro và chi phí scan.

## 13. Testing Strategy And Smoke Checklist

### Discipline (theo Slice 1)

- Go: TDD strict cho domain/services/repositories/filesystem/operations; `-race` cho operations.
- React: test-after cho features/screens; test-first cho `lib/core-client`.
- Contract tests: response validate qua JSON Schema (khi contracts được chốt ở milestone sau).

### Go test focus

- `GenericAgentsAdapter`: `.agents` missing / present / `.agents` is file / `.agents/skills` empty / permission denied — dùng `t.TempDir()` fixtures + symlinks.
- Classification: symlink → active host (`current`), → inactive host (`old_host`), → outside (`external_symlink`), broken (`broken_symlink`), folder thường (`direct`), file thường (`error`), symlink loop (`error`).
- skill_id match: exact relative path match → set; tên trùng nhưng path khác → null; target ngoài host → null.
- Reconcile: entry biến mất → `missing` (không hard delete); rescan idempotent.
- Project missing path → status `missing` + warning, bỏ qua provider scan, không crash.
- Lock: 2 `project.scan` cùng project → `conflict_error`.

### Smoke checklist (manual, read-only)

```text
Pre: app từ Slice 1 chạy được, có active host với vài skills.

Add Project:
  [ ] Tạo /tmp/proj-a với .agents/skills chứa:
        - symlink documentation-writer -> <active-host>/.agents/skills/documentation-writer
        - folder thường legacy-note
        - symlink ghost -> /nonexistent (broken)
        - symlink out-there -> /tmp/outside-x (ngoài host)
  [ ] Add Project -> dialog -> chọn /tmp/proj-a -> navigate detail
  [ ] sqlite3: projects 1 row status=active

Scan (read-only):
  [ ] Ghi lại inode/mtime của /tmp/proj-a/.agents/skills trước scan
  [ ] Scan Project -> toast phases reading_project..classifying_entries..done
  [ ] Detail: provider Generic Agents detected, skills_path đúng
  [ ] entries: documentation-writer symlink/current (skill_id set),
              legacy-note direct/current,
              ghost symlink/broken_symlink,
              out-there symlink/external_symlink
  [ ] So sánh inode/mtime sau scan -> KHÔNG đổi (chứng minh read-only)
  [ ] sqlite3: installs rows khớp; operations có scan/success + metadata_json counts

Old host:
  [ ] Đổi active host sang host khác (Slice 1), rescan proj-a
  [ ] documentation-writer -> status old_host

No provider:
  [ ] Tạo /tmp/proj-b (không có .agents) -> Add + Scan
  [ ] warning no_provider_detected, status active, không crash

Project missing:
  [ ] rm -rf /tmp/proj-a -> Rescan -> status missing + warning project_missing

Concurrency:
  [ ] Trigger scan 2 lần liên tiếp cùng project -> lần 2 conflict_error

Cancel:
  [ ] Start scan rồi cancel qua operation.cancel
  [ ] operation cancelled; previous visible project/provider/entry state vẫn nhất quán,
      không bị clear warnings/entries giữa chừng

Regression:
  [ ] go test -race ./... xanh; pnpm test xanh
  [ ] Skills Library (Slice 1) vẫn hoạt động
```

## 14. Open Decisions Deferred To Slice 2B+

Quyết định product/UX cần chốt trước hoặc trong slice sau, **không** chặn 2A design:

1. `project.remove` (DB-only) — deferred (đã chốt OUT 2A).
2. `Update Path` cho project missing — deferred.
3. **Auto-scan-on-add**: 2A mặc định KHÔNG auto-scan (scan tường minh). Nếu PM muốn auto-scan ngay sau add, quyết định ở 2B.
4. **Seed location**: seed `provider_definitions`/`provider_path_candidates` trong migration SQL hay app bootstrap upsert — quyết định kỹ thuật ở milestone implement.
5. **Nav count badge** cho Projects — optional, chưa chốt.
6. Provider thật khác generic_agents (Claude convention) + `unsupported_provider` surfacing — defer cho slice provider.
7. rsync/copy managed detection, `installed_checksum` — defer (cần marker/DB record do app tạo ở 2B+).
8. Symlink target form khi **tạo** (absolute vs relative) — không áp dụng 2A (không tạo symlink); quyết định ở 2B.
9. Conflict policy khi target tồn tại lúc install — 2B.

## 15. Acceptance Criteria

```text
[ ] Migration 000002 tạo projects, provider_definitions, provider_path_candidates,
    project_providers, installs; seed generic_agents definition + 2 path candidates.
    KHÔNG tạo scan_results.
[ ] Sidebar có Projects; /projects và /projects/$projectId render.
[ ] Add Project (idempotent theo path) persist projects row; re-add cùng path không tạo trùng.
[ ] project.scan là read-only: entries của project KHÔNG bị app sửa (verify inode/mtime).
[ ] generic_agents detection đúng cho: present / missing / .agents-is-file / empty-skills.
[ ] Classification đúng: current / old_host / external_symlink / broken_symlink / direct / error.
[ ] skill_id chỉ set khi exact relative-path match trong known host; ngược lại null.
[ ] old_host vs external dựa trên active + inactive known skill_host_folders.
[ ] Empty .agents/skills -> provider detected, zero entries, không error.
[ ] Marker provider khác (.claude) bị ignore, không sinh warning/row.
[ ] Project path missing -> status missing + warning project_missing, bỏ qua provider scan.
[ ] Reconcile: entry biến mất -> install_status=missing (không hard delete).
[ ] Scan summary trong operations.metadata_json; UI/API không phụ thuộc scan_results.
[ ] Warnings chỉ kèm read-only action_key (rescan/open_folder).
[ ] Per-project scan lock -> conflict_error khi trùng.
[ ] operation.cancel cho project.scan không làm mất previous visible state giữa chừng.
[ ] go test -race ./... xanh; pnpm test xanh; smoke checklist pass trên macOS.
[ ] KHÔNG có filesystem write method nào cho project trong gateway (read-only invariant).
```
