# Information Architecture

## Core Concepts

### Skill Host Folder

Folder do user chọn và cấu hình trong GUI để lưu source of truth cho skill trên
máy.

```text
<skill-host-folder>/
  .agents/
    skills/
      skill-a/
      skill-b/
```

### Skill

Một skill cụ thể nằm trong Skill Host Folder.

Skill có thể có source từ GitHub, Vercel skills, hoặc local/manual.

### Source

Nguồn gốc của skill.

Các source type ban đầu:

- `github`
- `vercel_skills`
- `local`
- `manual`

### Project

Một project được user add vào Skillbox.

Skillbox scan project để biết provider nào có trong project và skill nào đang
được cài.

### Global Provider Location

Provider-level/global location là nơi một provider lưu skill, command, hoặc
config ở cấp user/máy, không thuộc riêng project nào.

Skillbox scan global locations để user biết global skill nào đang tồn tại và có
thể ảnh hưởng nhiều project.

### Provider

Agent provider hoặc convention mà project đang dùng.

Ví dụ:

- Claude
- Codex
- opencode
- Antigravity CLI
- Generic `.agents`

### Install

Việc một skill được cài từ Skill Host Folder vào một project/provider.

Install mode:

- `symlink`
- `rsync/copy`
- `direct`

### Global Install

Skill/config tồn tại ở global provider location.

Global install có thể là managed hoặc direct, tương tự project install, nhưng
scope là provider global level thay vì project/provider.

### Fetch

Kiểm tra upstream để biết skill có bản mới không.

### Update

Đưa thay đổi từ upstream về Skill Host Folder.

### Sync

Đưa thay đổi từ Skill Host Folder sang project cài bằng rsync/copy.

## Main App Areas

```text
Dashboard
Skills Library
Global Skills
Projects
Project Detail
Skill Detail
Updates
Settings
```

## Dashboard

Dashboard hiển thị tổng quan:

- Tổng số skill trong Skill Host Folder.
- Tổng số global skills được phát hiện.
- Tổng số project đã add.
- Skill có update sau lần Fetch gần nhất.
- Project đang dùng symlink.
- Project đang dùng rsync/copy.
- Warning cơ bản như host missing, broken path, provider path missing.

## Skills Library

Skills Library là nơi quản lý skill trong Skill Host Folder.

Hiển thị:

- Tên skill.
- Source: GitHub, Vercel skills, local, manual.
- Provider compatibility nếu biết.
- Last fetched.
- Update available hay không.
- Số project đang dùng skill.

Action:

- Add/import skill vào host.
- Fetch update.
- Open skill folder.
- View skill detail.

## Global Skills

Global Skills là nơi xem skill/config ở provider global level trên máy.

Hiển thị:

- Provider.
- Global location path.
- Skill/global entry name.
- Mode: symlink, rsync/copy, direct.
- Status: current, missing, external symlink, broken symlink, unmanaged.
- Skill Host Folder source nếu map được.
- Warning nếu global skill có thể gây nhiễu project-level behavior.

Action:

- Scan global locations.
- Open global provider folder.
- Remove global entry nếu user xác nhận.
- Relink hoặc sync nếu entry được Skillbox quản lý.
- Adopt/import sau này nếu feature này được support.

Phase 1 scope:

- Global Skills là scan, visibility, và remediation surface.
- Chưa có flow Install Skill To Global Location.
- Add Skill flow chỉ target project providers.

## Projects

Projects là danh sách project được add vào Skillbox.

Hiển thị:

- Project name.
- Project path.
- Providers detected.
- Số skill đang cài.
- Warning status nếu có.

Action:

- Add project.
- Scan project.
- Open project detail.
- Remove project khỏi Skillbox database.

## Project Detail

Project Detail là màn hình chính để điều phối skill trong một project.

Hiển thị:

- Project path.
- Provider detected.
- Skills installed.
- Mode: symlink, rsync/copy, direct.
- Source skill trong host nếu map được.
- Update/sync status nếu là rsync/copy.

Action:

- Add skill.
- Remove skill.
- Switch mode giữa symlink và rsync/copy.
- Rescan.
- Open project folder.

## Add Skill Flow

Flow không cần wizard nặng.

```text
Project Detail
  -> Add Skill
  -> Chọn skill từ Skills Library
  -> Chọn provider target nếu project có nhiều provider
  -> Chọn mode symlink hoặc rsync/copy
  -> Install
```

Nếu project chỉ có một provider thì bỏ qua bước chọn provider.

## Skill Detail

Hiển thị:

- Tên skill.
- Host path.
- Source type.
- Source URL hoặc Vercel source id.
- Current version/commit nếu có.
- Last fetched.
- Projects using this skill.

Action:

- Fetch.
- Update host copy.
- Open folder.
- Show affected projects.

## Updates

Updates là màn hình tập trung cho việc kiểm tra và xử lý update.

Hiển thị:

- Nút Fetch All.
- Danh sách skill có bản mới.
- Current version.
- Latest version.
- Affected projects.
- Project install modes.

Action:

- Update skill in host.
- Sync rsync/copy projects.
- View affected projects.

## Settings

Settings quản lý:

- Skill Host Folder path.
- Default install mode.
- Provider configs.
- Database location.
- GitHub/Vercel settings nếu cần.
