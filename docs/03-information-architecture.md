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
Global Plugins
Projects
Project Detail
Skill Detail
Updates
Settings
```

Sidebar navigation thứ tự: Dashboard → Host Skills → Global Skills → Global Plugins → Projects → Settings → About.

## Dashboard

Dashboard hiển thị tổng quan:

- Tổng số skill trong Skill Host Folder.
- Tổng số global skills được phát hiện.
- Tổng số project đã add.
- Lối tắt đến Global Plugins (navigation row, mirror Global Skills pattern).
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

## Global Plugins

Global Plugins là nơi xem và quản lý plugin ở global (user) layer cho các provider hỗ
trợ plugin convention (Claude, Codex, Antigravity CLI).

File: `apps/desktop/renderer/src/screens/plugins-screen.tsx`.

Hiển thị (group theo provider):

- Settings file path đang được Skillbox scan (ví dụ `~/.claude/settings.json`).
- Layer scan status: ok, not configured, unreadable, malformed, too large,
  symlink, path escape.
- Danh sách plugin với name, marketplace name, status enabled/disabled.
- Danh sách marketplace với name, source type, source summary.

Action:

- Rescan user-layer settings file của một provider.
- Toggle enable/disable globally cho một plugin (chỉ provider có write
  support: Claude, Codex, Antigravity CLI).

Phase 1 scope:

- Chỉ global (user) layer được hiển thị ở Global Plugins. Project layer và effective
  state per project nằm trong Project Detail.
- Local layer (`settings.local.json`) là read-only.
- Managed settings (enterprise config) là out-of-scope.

> **Naming note:** UI hiển thị label `Global` cho layer mà code/contract dùng
> identifier `user` (`layer: "user"`, `PluginLayerUser`, SQL `settings_layer =
> 'user'`). End-user terminology favors `Global`; code/data terminology giữ
> `user` để không phá contract và DB.

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
- Plugin tab: plugin version column — Claude lấy từ `installed_plugins.json`
  (user + project scope); Codex lấy từ cache dir `~/.codex/plugins/cache/`
  (cache là global, apply cho cả user layer và project layer); Antigravity CLI
  không có nguồn version → hiển thị `—`.

Action:

- Add skill.
- Remove skill.
- Switch mode giữa symlink và rsync/copy.
- Rescan.
- Open project folder.

## Add Skill Flow

Flow mở Add Skill Wizard từ Project Detail.

```text
Project Detail
  -> Add Skill
  -> Wizard mở tab strip, mỗi tab là một installable provider
     (tab header: ProviderIcon + display name + skills path badge + "experimental" badge nếu có)
  -> User chọn tab provider muốn install vào
  -> User tick skill trong danh sách của tab đó
     (skill đã installed ở provider đó bị disable + "Installed" badge)
  -> Footer hiển thị path hint của tab đang active
  -> User nhấn Install
  -> Skill được install vào provider của tab đang active
```

Nếu không có installable provider nào (0 provider hợp lệ), wizard hiển thị empty
state "No provider is ready for install." kèm CTA "Scan project".

Selection reset khi user chuyển tab.

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

## About

About screen hiển thị thông tin về ứng dụng và tác giả.

File: `apps/desktop/renderer/src/screens/about-screen.tsx`.

Hiển thị:

- Tên ứng dụng và version (từ `VITE_APP_VERSION`).
- Author links: Email, GitHub, Blog — click mở browser.
- Update check: nút "Check for Updates" gọi `app.checkUpdate` RPC.
  - Trạng thái: idle / checking / up-to-date / available / disabled / error.
  - Khi có bản mới: hiển thị `latestVersion` và link "View release" đến GitHub Releases.
  - Khi network bị tắt (Settings → Network): hiển thị thông báo hướng user bật lên.

`app.checkUpdate` gọi GitHub Releases API, chỉ hoạt động khi `network_settings.update_check_enabled = true`.

<!-- DOC-VERIFIED: about-screen, use-check-app-update, method-allowlist app.checkUpdate -->
