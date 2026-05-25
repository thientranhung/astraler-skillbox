# UI Wireframes

Tài liệu này phác thảo UI ở mức text wireframe. Mục tiêu là kiểm tra thông tin,
action, trạng thái empty/loading/warning/error trước khi đi vào technical
architecture hoặc visual design.

## Design Principles

- GUI là interface chính.
- UI ưu tiên quản trị rõ ràng hơn trang trí.
- Mọi màn hình quan trọng phải có action kế tiếp rõ.
- Warning phải đi kèm action cụ thể nếu có thể.
- Project và provider scope phải hiển thị rõ để tránh nhầm skill trùng tên.
- Symlink, rsync/copy, direct là khái niệm kỹ thuật được hiển thị trực tiếp.

## Navigation Shell

Layout đề xuất:

```text
┌─────────────────────────────────────────────────────────────┐
│ Skillbox                                      Global Actions │
├───────────────┬─────────────────────────────────────────────┤
│ Dashboard     │                                             │
│ Skills        │ Main content                                │
│ Global Skills │                                             │
│ Projects      │                                             │
│ Updates       │                                             │
│ Settings      │                                             │
└───────────────┴─────────────────────────────────────────────┘
```

Sidebar items:

- Dashboard
- Skills Library
- Global Skills
- Projects
- Updates
- Settings

Global actions:

- Scan
- Fetch
- Open Skill Host Folder

## Dashboard

Purpose: hiển thị trạng thái tổng quan khi mở app.

Wireframe:

```text
Dashboard

Skill Host Folder
  Path: /absolute/path/to/host
  Status: active / missing / unreadable / empty
  [Open Folder] [Change Folder] [Scan]

Summary
  Skills: 42
  Global skills: 6
  Projects: 12
  Updates available: 3
  Warnings: 2

Warnings
  [warning] Project missing: /path/to/project        [Update Path] [Remove]
  [warning] Broken symlink: project-a / skill-x      [Relink] [Remove]

Recent Operations
  Fetch all           success    2 minutes ago
  Scan project-a      success    5 minutes ago
```

Primary states:

- No Skill Host Folder configured.
- Skill Host Folder missing.
- Skill Host Folder empty.
- Normal state with summary counts.
- Warning state with scoped actions.

Primary actions:

- Choose Skill Host Folder.
- Scan Skill Host Folder.
- Add Project.
- Fetch All.
- Scan Global.

## Skills Library

Purpose: quản lý skill trong Skill Host Folder.

Wireframe:

```text
Skills Library

[Add / Import Skill] [Fetch All] [Open Skill Host Folder]

Filters
  Source: all / GitHub / Vercel / Local / Manual
  Status: all / update available / local modified / unreadable
  Search: __________________

Table
  Name                    Source    Version      Fetch Status      Projects
  documentation-writer    GitHub    a1b2c3       up to date        4
  browser-automation      Local     -            not fetchable     2
  adr-helper              GitHub    d4e5f6       update available  7
```

Row actions:

- View detail.
- Fetch.
- Update host copy.
- Open folder.
- Show projects using this skill.

Empty state:

```text
No skills in Skill Host Folder.
[Add / Import Skill]
```

## Skill Detail

Purpose: xem metadata và impact của một skill.

Wireframe:

```text
Skill Detail: documentation-writer

Metadata
  Host path: /host/.agents/skills/documentation-writer
  Source: GitHub
  Repo: owner/repo
  Path: skills/documentation-writer
  Current commit: a1b2c3
  Checksum: sha256:...
  Last fetched: 2026-05-22 10:31

Actions
  [Fetch] [Update Host Copy] [Open Folder]

Projects Using This Skill
  Project              Provider          Mode        Status
  project-a            Shared Agent Skills (.agents)    symlink     current
  project-b            Claude            rsync/copy  needs sync
  project-c            Shared Agent Skills (.agents)    direct      current

Global Usage
  Provider          Location               Mode        Status
  Shared Agent Skills (.agents)    User Global            direct      current
  Claude            Claude Global          symlink     external symlink
```

Warnings:

- Source not fetchable.
- Local modifications need review.
- Host path missing/unreadable.

## Global Skills

Purpose: xem skill/config ở provider global level trên máy.

Wireframe:

```text
Global Skills

[Scan Global] [Open Selected Folder]

Global Locations
  Provider          Path                         Status          Entries
  Shared Agent Skills (.agents)    ~/.agents/skills             active          4
  Claude            ~/.claude/...                not configured  0

Global Entries
  Provider          Skill/Entry             Mode        Status             Actions
  Shared Agent Skills (.agents)    research-writer         direct      current            [Open]
  Shared Agent Skills (.agents)    adr-helper              symlink     current            [Relink] [Remove]
  Claude            old-command             symlink     broken symlink     [Relink] [Remove]
```

Warnings:

```text
[info] Global skill also exists in 3 projects.
[warning] Broken global symlink. [Relink] [Remove]
[warning] Global provider location missing. [Update Path] [Disable]
```

Rules:

- Global entries are never merged with project installs.
- Global direct entries are shown as unmanaged/direct.
- Removing a global entry never removes the Skill Host Folder source.
- Global/project overlap is informational unless provider behavior makes it
  blocking later.
- Phase 1 does not include an Add Skill to Global Location flow. Global Skills
  focuses on scan, visibility, and remediation actions.

## Projects

Purpose: danh sách project được add vào Skillbox.

Wireframe:

```text
Projects

[Add Project] [Scan All]

Filters
  Provider: all / Shared Agent Skills (.agents) / Claude / unsupported
  Status: all / active / missing / warnings
  Search: __________________

Table
  Project        Path                    Providers                 Skills  Warnings
  skillbox       /repo/skillbox          Shared Agent Skills (.agents)            3       0
  content-lab    /repo/content-lab       Claude, Shared Agent Skills (.agents)    8       1
  old-project    /repo/old-project       -                         0       missing
```

Row actions:

- Open Project Detail.
- Scan.
- Open folder.
- Remove from Skillbox database.

## Project Detail

Purpose: điều phối skill trong một project.

Wireframe:

```text
Project Detail: content-lab

Path: /repo/content-lab
Status: active

Providers
  [Shared Agent Skills (.agents)] supported    .agents/skills     5 skills
  [Claude]         experimental .claude/...        3 skills

Actions
  [Add Skill] [Scan Project] [Open Folder]

Installed Skills
  Provider          Skill                 Mode        Status            Actions
  Shared Agent Skills (.agents)    documentation-writer  symlink     current           [Switch] [Remove]
  Shared Agent Skills (.agents)    adr-helper            rsync/copy  needs sync        [Sync] [Switch] [Remove]
  Claude            old-skill             symlink     broken symlink    [Relink] [Remove]
  Claude            manual-note           direct      current           [Open]
```

Grouping:

- Group by provider by default.
- Allow filter by provider.
- Do not merge same skill name across providers.

Provider warning examples:

```text
[warning] Claude provider is experimental.
[warning] Provider path missing. [Rescan] [Update Path]
[warning] Unsupported provider detected. Install disabled.
```

## Add Skill Flow

Purpose: cài skill từ Skill Host Folder vào project.

Flow screens:

```text
Add Skill

Step 1: Select Skills
  [ ] documentation-writer
  [ ] adr-helper
  [ ] browser-automation

Step 2: Select Provider
  ( ) Shared Agent Skills (.agents)   supported      .agents/skills
  ( ) Claude           experimental   .claude/...

Step 3: Select Mode
  ( ) symlink
  ( ) rsync/copy

Step 4: Confirm
  Install 2 skills into content-lab / Shared Agent Skills (.agents) using symlink.
  Existing conflicts: none
  Affected paths:
    /repo/content-lab/.agents/skills/documentation-writer
    /repo/content-lab/.agents/skills/adr-helper

  [Install] [Cancel]
```

Rules:

- If project has one valid provider, provider step can be skipped.
- Unsupported/disabled providers cannot be selected.
- If target path exists, show replace/skip/cancel decision.
- Validate install target stays inside project root.

## Updates

Purpose: kiểm tra và xử lý upstream updates.

Wireframe:

```text
Updates

[Fetch All]

Available Updates
  Skill                 Current      Latest       Affected Projects
  adr-helper            a1b2c3       d4e5f6       7
  research-writer       v1.2         v1.3         2

Affected Projects: adr-helper
  Project       Provider          Mode        Result after host update
  project-a     Shared Agent Skills (.agents)    symlink     updates immediately
  project-b     Shared Agent Skills (.agents)    rsync/copy  needs sync
  project-c     Claude            direct      unaffected

Affected Global Installs
  Location              Provider          Mode        Result after host update
  User Global           Shared Agent Skills (.agents)    symlink     updates immediately
  Claude Global         Claude            rsync/copy  needs sync

Actions
  [Update Host Copy] [Sync rsync/copy Projects]
```

Fetch states:

- Fetch running.
- Up to date.
- Update available.
- Auth required.
- Network error.
- Not fetchable.
- Needs review because local modified.

## Settings

Purpose: cấu hình app-level settings.

Wireframe:

```text
Settings

Skill Host Folder
  Current: /absolute/path/to/host
  Status: active
  [Change Folder] [Open Folder] [Scan]

Default Install Mode
  ( ) symlink
  ( ) rsync/copy

Providers
  Provider          Status          Create Structure   Icon
  Shared Agent Skills (.agents)    supported       yes                agents
  Claude            experimental    no                 claude
  Codex             experimental    yes                codex
  opencode          experimental    yes                opencode
  Antigravity CLI   experimental    yes                antigravity

Global Provider Locations
  Provider          Path                         Status          Actions
  Shared Agent Skills (.agents)    ~/.agents/skills             active          [Change] [Scan]
  Claude            -                            not configured  [Configure]

Credentials
  GitHub            active          [Validate] [Change]
  Vercel            missing         [Configure]

Database
  Location: /path/to/skillbox.sqlite
  Version: 1
  [Open Folder] [Export Diagnostics]
```

Settings warnings:

- Skill Host Folder missing.
- Credential invalid/expired.
- Database migration failed.

## Empty States

### No Skill Host Folder

```text
Skillbox needs a Skill Host Folder.
[Choose Skill Host Folder]
```

### Empty Skill Host Folder

```text
No skills found in this Skill Host Folder.
[Add / Import Skill]
```

### No Projects

```text
No projects added yet.
[Add Project]
```

### No Global Skills

```text
No global skills found.
[Scan Global] [Configure Global Location]
```

### No Provider Detected

```text
No provider detected in this project.
[Set Up Provider] [Rescan]
```

`Set Up Provider` only appears if at least one provider has
`can_create_structure = true`.

## Loading States

Loading states:

- Scanning Skill Host Folder.
- Scanning global locations.
- Scanning project.
- Fetching updates.
- Updating host skill.
- Syncing rsync/copy install.
- Installing skill.

UI rules:

- Show operation target.
- Disable duplicate dangerous action on the same target.
- Keep navigation usable unless the operation is blocking.

## Warning And Error States

Recoverable warning examples:

```text
Broken symlink       [Relink] [Remove]
Project missing      [Update Path] [Remove]
Fetch failed         [Retry] [Configure Source]
Unsupported provider [Open Path]
Needs sync           [Sync]
```

Blocking error examples:

```text
Database corrupt
Skill Host Folder unreadable
Install target outside project root
Permission denied while writing
```

Blocking errors should stop the related action and keep existing metadata
unchanged.

## Confirmations

Confirm before:

- Remove skill from project.
- Replace existing target folder.
- Change Skill Host Folder when existing symlinks point to old host.
- Update host skill with many symlinked projects affected.
- Delete/remove project from Skillbox database.

Confirmation should show:

- Object being changed.
- Filesystem path.
- Affected projects/providers.
- Whether action changes Skill Host Folder or only project install.

## Impact Preview

Impact preview is required for:

- Update Host Copy.
- Change Skill Host Folder.
- Switch Install Mode.
- Remove Skill.

Example:

```text
Update adr-helper

Symlink projects updated immediately:
  project-a
  project-b

Rsync/copy projects needing sync after update:
  project-c
  project-d

Direct installs unaffected:
  project-e

Global symlink installs updated immediately:
  User Global (Shared Agent Skills (.agents))

Global rsync/copy installs needing sync:
  Claude Global

[Update Host Copy] [Cancel]
```

## Implementation Notes

- Dashboard should be driven by aggregate queries over `skills`, `projects`,
  `installs`, `warnings`, and `fetch_results`.
- Project Detail should be scoped by `project_providers`.
- Global Skills should be scoped by `global_provider_locations`.
- Updates view should use latest `fetch_results` per `source_id`.
- Warnings should use `action_key` for quick actions.
- Operations should drive loading/progress state.
