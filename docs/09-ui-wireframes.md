# UI Wireframes

This document outlines the UI at the text wireframe level. The goal is to verify
information, actions, and empty/loading/warning/error states before moving to
technical architecture or visual design.

## Design Principles

- GUI is the primary interface.
- UI prioritizes clear management over decoration.
- Every key screen must have a clear next action.
- Warnings must include a specific action where possible.
- Project and provider scope must be displayed clearly to avoid confusing skills
  with the same name.
- Symlink and direct are technical concepts displayed directly. (rsync/copy is
  deferred — not in the current release)

## Table Notation

Use **Shared Agents** as the user-facing label for the shared `.agents`
provider. Keep `generic_agents` as the internal stable provider key in code,
contracts, database rows, and verifier output.

## Navigation Shell

Proposed layout:

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

Sidebar items (actual nav order):

- Dashboard
- Host Skills
- Global Skills
- Global Plugins
- Projects
- Settings

Global actions:

- Scan
- Fetch
- Open Skill Host Folder

## Dashboard

Purpose: display a summary overview when the app opens.

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

Purpose: manage skills in the Skill Host Folder.

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

Purpose: view metadata and impact for a skill.

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
  Project              Provider        Mode        Status
  project-a            Shared Agents   symlink     current
  project-c            Shared Agents   direct      current

Global Usage
  Provider        Location               Mode        Status
  Shared Agents   User Global            direct      current
  Claude          Claude Global          symlink     external symlink
```

Warnings:

- Source not fetchable.
- Local modifications need review.
- Host path missing/unreadable.

## Global Skills

Purpose: view skills/config at the provider global level on the machine.

Wireframe:

```text
Global Skills

[Scan Global] [Open Selected Folder]

[Shared Agents] [Claude]

Global Locations
  Provider        Path                   Status          Entries
  Shared Agents   ~/.agents/skills       active          4

Global Entries
  Provider        Skill/Entry             Mode        Status             Actions
  Shared Agents   research-writer         direct      current            [Open]
  Shared Agents   adr-helper              symlink     current            [Relink] [Remove]
```

Warnings:

```text
[info] Global skill also exists in 3 projects.
[warning] Broken global symlink. [Relink] [Remove]
[warning] Global provider location missing. [Update Path] [Disable]
```

Rules:

- Global entries are never merged with project installs.
- Provider tabs are provider-specific only; do not show an aggregate "All" tab.
- The Shared Agents provider is selected first when available.
- Global direct entries are shown as unmanaged/direct.
- Removing a global entry never removes the Skill Host Folder source.
- Global/project overlap is informational unless provider behavior makes it
  blocking later.
- Phase 1 does not include an Add Skill to Global Location flow. Global Skills
  focuses on scan, visibility, and remediation actions.

## Projects

Purpose: list of projects added to Skillbox.

Wireframe:

```text
Projects

[Add Project] [Scan All]

Filters
  Provider: Shared Agents / Claude / unsupported
  Status: all / active / missing / warnings
  Search: __________________

Table
  Project        Path                    Providers             Skills                    Plugins
  skillbox       /repo/skillbox          [icon] Shared Agents  [icon] 3 Shared Agents    —
  content-lab    /repo/content-lab       [icon] Claude, …      [icon] 5 Claude, 3 Agents [icon] 2 Claude
  old-project    /repo/old-project       —                     —                         —

Skills and Plugins columns show provider icon + count + truncated displayName per provider.
```

Row actions:

- Open Project Detail.
- Scan.
- Open folder.
- Remove from Skillbox database.

## Project Detail

Purpose: coordinate skills within a project.

Wireframe:

```text
Project Detail: content-lab

Path: /repo/content-lab
Status: active

Providers
  [Shared Agents] supported    .agents/skills     5 skills
  [Claude]         experimental .claude/...        3 skills

Actions
  [Add Skill] [Scan Project] [Open Folder]

Installed Skills
  Provider        Skill                 Mode        Status            Actions
  Shared Agents   documentation-writer  symlink     current           [Remove]
  Shared Agents   adr-helper            symlink     current           [Remove]
  Claude          old-skill             symlink     broken symlink    [Relink] [Remove]
  Claude          manual-note           direct      current           [Open]
```

Provider Plugins (per provider):

```text
Provider Plugins — Claude

  Layers
    Layer    Status          File                                    Last Scanned
    local    not configured  .claude/settings.local.json             5/28/2026
    project  ok              .claude/settings.json                   5/28/2026
    user     ok              ~/.claude/settings.json                 5/27/2026

  Plugin              Marketplace              Version   Project          User         Effective
  superpowers          claude-plugins-official   1.0.0     —  (not set)    [enabled]    enabled
  my-plugin            custom                   —         [disabled]       [enabled]    disabled
  other-plugin         official                 unknown   [enabled]        [disabled]   enabled
```

Plugin column rules:

- Version column: read-only, shows installed version from `installed_plugins.json` for the
  winning provenance layer. Claude only (Phase 1). Column hidden when all rows have null version.
  `—` rendered when version is null or unavailable. `"unknown"` displayed as-is (valid literal).
- Project column: 3-state cycle toggle (not set → enabled → disabled → not set).
  "not set" means no override at project layer — user layer value takes effect.
- User column: 2-state toggle (enabled/disabled). Visually dimmed when project
  layer has a value (project overrides user).
- Effective column: read-only, shows resolved status after layer precedence.
- When provenance is local, both Project and User columns show "overridden" (no toggle).

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

## Global Plugins

Purpose: view and toggle plugin enable/disable at the user (global) layer.

Wireframe:

```text
Provider Plugins                                    [Scan Global]

[Shared Agents] [Claude] [Codex]

── Shared Agents ───────────────────────────────────
  Path: ~/.agents/plugins.json   Status: ok   Last scan: 2026-05-29 15:02

  Plugin              Marketplace              Version    Status    Action
  shared-tools        local                    1.0.0      enabled   [Disable]
```

Version column rules:

- Provider tabs are provider-specific only; do not show an aggregate "All" tab.
- The Shared Agents provider is selected first when available.
- Version column appears only when at least one plugin in that provider section
  has a non-null version. Column hidden for providers where version is
  unavailable (Codex, Antigravity CLI in Phase 1).
- `"unknown"` is a valid literal shown as-is; NULL or undefined → `—`.
- Version is read from `~/.claude/plugins/installed_plugins.json` at scan time.
  Persisted in `provider_plugin_entries.version`.

## Add Skill Flow

Purpose: install a skill from the Skill Host Folder into a project.

Dialog: fixed-position overlay, max-h 90vh, flex-col layout — does not overflow
the viewport.

Flow screens:

```text
Add Skills                                                   [X]

Tab strip (logo + displayName, hover shows full path):
  [claude-icon] Claude | [agents-icon] Shared Agents

Skill list (scrollable, max-h 48):
  [ ] documentation-writer            docs/documentation-writer
  [ ] adr-helper                      docs/adr-helper
  [x] browser-automation (Installed)

Footer:
  Will install to: /repo/content-lab/.claude/

  [Cancel] [Install]
```

Rules:

- Tab strip: icon + displayName only; path in title tooltip and footer hint.
- If project has no valid provider, show empty state + Scan CTA.
- Unsupported/disabled providers do not appear as tabs.
- Dialog covers full viewport (fixed overlay, z-50).

## Updates

Purpose: check and handle upstream updates.

Wireframe:

```text
Updates

[Fetch All]

Available Updates
  Skill                 Current      Latest       Affected Projects
  adr-helper            a1b2c3       d4e5f6       7
  research-writer       v1.2         v1.3         2

Affected Projects: adr-helper
  Project       Provider        Mode        Result after host update
  project-a     Shared Agents   symlink     updates immediately
  project-c     Claude          direct      unaffected

Affected Global Installs
  Location              Provider        Mode        Result after host update
  User Global           Shared Agents   symlink     updates immediately

Actions
  [Update Host Copy]
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

Purpose: configure app-level settings.

Wireframe:

```text
Settings

Skill Host Folder
  Current: /absolute/path/to/host
  Status: active
  [Change Folder] [Open Folder] [Scan]

Default Install Mode
  (•) symlink

Providers
  Provider        Status          Create Structure   Icon
  Shared Agents   supported       yes                agents
  Claude          experimental    no                 claude
  Codex           experimental    yes                codex
  opencode        experimental    yes                opencode
  Antigravity CLI experimental    yes                antigravity

Global Provider Locations
  Provider        Path                   Status          Actions
  Shared Agents   ~/.agents/skills       active          [Change] [Scan]
  Claude          -                      not configured  [Configure]

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
- Remove Skill.

Example:

```text
Update adr-helper

Symlink projects updated immediately:
  project-a
  project-b

Direct installs unaffected:
  project-e

Global symlink installs updated immediately:
  User Global (Shared Agents)

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
