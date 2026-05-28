# Product Notes

## Product Thesis

Skillbox là một GUI-first local skill manager cho thời đại nhiều AI agent
provider.

Nó giúp người dùng quản lý, cài đặt, cập nhật, kiểm tra và quan sát agent skills
trên nhiều project, nhiều provider, nhiều format khác nhau.

## Scope Hiện Tại

Skillbox không phải một utility nhỏ. Đây là một product lớn với GUI là trải
nghiệm chính.

CLI có thể được phát triển sau như một interface phụ cho automation và power
users.

## Core Product Pieces

```text
Skill Host Folder
  Folder do user chọn trong GUI để làm source of truth chứa skill trên máy.

Skillbox GUI
  Giao diện chính để quản trị skill, global skills, project, provider, install,
  update.

Provider Adapters
  Mapping provider sang folder/path/convention tương ứng.

SQLite Database
  Lưu metadata quản trị như projects, skills, global installs, project installs,
  sources, fetch results.

CLI
  Interface phụ, phát triển sau.
```

## Decisions

- Dùng SQLite ngay từ đầu.
- Source skill ưu tiên GitHub và Vercel skills.
- Dùng Fetch để kiểm tra skill nào có bản mới.
- Symlink và rsync/copy là core design, không phải workaround.
- Skill Host Folder bị move/delete thì app sẽ warning khi mở hoặc scan.
- Convert format giữa provider để Phase 2.
- Health check chi tiết để sau, không phải trọng tâm product hiện tại.
- Non-developer users vẫn cần làm quen với khái niệm kỹ thuật.

## Symlink vs Rsync / Copy

Symlink:

- Project trỏ trực tiếp về Skill Host Folder.
- Sửa một chỗ trong Skill Host Folder thì nhiều project nhận thay đổi ngay.
- Đây là cơ chế chính để dùng chung source of truth.

Rsync/copy:

- Project nhận snapshot riêng.
- Update Skill Host Folder không tự động đổi project.
- Dùng khi project cần ổn định hoặc muốn kiểm soát thời điểm sync.

## Update Model

Fetch chỉ kiểm tra upstream có thay đổi không.

Update là đưa thay đổi từ upstream về Skill Host Folder.

Sync là đưa thay đổi từ Skill Host Folder sang project đã cài bằng rsync/copy.

Với project cài bằng symlink, update Skill Host Folder đồng nghĩa project nhận
thay đổi ngay.

## Plugins và Marketplaces

Một số provider (Claude, Codex, Antigravity CLI) có khái niệm **plugin** riêng,
khác với skill. Plugin là extension provider khai báo trong settings file của
mình (ví dụ `~/.claude/settings.json`), thường được tải về từ một
**marketplace** (named source như GitHub repo, git URL, hoặc local directory).

Skillbox đọc settings file của provider để hiển thị plugin nào đang
enabled/disabled ở layer nào (user/project/local) và cho phép user toggle
nhanh mà không cần mở settings file thủ công. Skillbox không tự download
marketplace content; phần đó provider tự xử lý.

Phase 1 scope:

- Scan settings file user + project layer. Local layer chỉ scan, không write.
- Toggle enable/disable ở global (user) layer hoặc per-project (project
  layer, 3-state cycle: inherit → enabled → disabled).
- Managed settings (provider-managed enterprise config) là out-of-scope.

`provider_plugin` là domain object đại diện cho plugin trong code. Chi tiết
data model nằm ở [`06-data-model.md`](06-data-model.md) § Provider Plugin
Layer System; chi tiết flow ở [`08-provider-model.md`](08-provider-model.md) §
Provider Plugin Layer Model.

## Remaining Tradeoffs

### Provider Convention Drift

Provider có thể đổi folder/path/convention. Skillbox cần adapter layer để cô lập
phần này.

### Skill Format Diversity

Skill format giữa provider có thể khác nhau. Phase 1 chỉ quản lý/cài đặt theo
provider adapter. Phase 2 mới convert format.

### Source Metadata

Để Fetch hoạt động tốt, mỗi skill nên có metadata về source:

- GitHub repo
- Subfolder nếu có
- Branch/tag/commit
- Vercel skills identifier nếu có
- Local/manual nếu không có upstream rõ ràng

### Visibility Khi Update

Symlink là thiết kế chính, nhưng UI vẫn nên hiển thị project nào sẽ bị ảnh hưởng
khi update Skill Host Folder.
