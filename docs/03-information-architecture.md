# Information Architecture

## Core Concepts

### Skill Host Folder

A folder the user selects and configures in the GUI to store the source of truth
for skills on the machine.

```text
<skill-host-folder>/
  .agents/
    skills/
      skill-a/
      skill-b/
```

### Skill

A specific skill residing in the Skill Host Folder.

A skill may have a source from GitHub, Vercel skills, or local/manual.

### Source

The origin of a skill.

Initial source types:

- `github`
- `vercel_skills`
- `local`
- `manual`

### Project

A project the user adds to Skillbox.

Skillbox scans a project to find out which providers are present and which skills
are installed.

### Global Provider Location

A provider-level/global location is where a provider stores skills, commands, or
config at the user/machine level, not belonging to any specific project.

Skillbox scans global locations so the user can see which global skills exist and
may affect multiple projects.

### Provider

An agent provider or convention that a project is using.

Examples:

- Claude
- Codex
- opencode
- Antigravity CLI
- Generic `.agents`

### Install

The act of a skill being installed from the Skill Host Folder into a
project/provider.

Install mode:

- `symlink` — current stable path.
- `direct` — unmanaged skill already in project.
- `rsync/copy` — deferred; not current UI or RPC support.

### Global Install

A skill/config that exists in a global provider location.

A global install may be managed or direct, similar to a project install, but
scoped to the provider global level rather than a project/provider.

### Fetch

Check upstream to find out if a skill has a new version.

### Update

Bring the change from upstream into the Skill Host Folder.

### Sync

> **Deferred.** Rsync/copy sync is a future flow; not implemented in the current
> release.

## Main App Areas

```text
Dashboard
Host Skills
Global Skills
Global Plugins
Projects
Project Detail
Skill Detail
Settings
About
Setup / Startup Error
```

Sidebar navigation order: Dashboard → Host Skills → Global Skills → Global
Plugins → Projects → Settings → About.

## Dashboard

Dashboard displays an overview:

- Total number of skills in the Skill Host Folder.
- Total number of global skills discovered.
- Total number of projects added.
- Shortcut to Global Plugins (navigation row, mirrors Global Skills pattern).
- Skills with updates since the last Fetch.
- Projects using symlink.
- Basic warnings such as host missing, broken path, provider path missing.

## Host Skills

Host Skills is where skills in the Skill Host Folder are viewed and managed.
The route is `/skills` and the renderer component is still named
`SkillsLibraryScreen`, but the user-facing navigation label is **Host Skills**.

Displays:

- Skill name.
- Source: GitHub, Vercel skills, local, manual.
- Provider compatibility if known.
- Last fetched.
- Whether an update is available.
- Number of projects using the skill.

Actions:

- Open skill folder.
- View skill detail.
- Scan the active Skill Host Folder.

Scope:

- The screen focuses the Shared Agents host (`.agents/skills`) and does not show
  an aggregate "All skills" provider tab.
- Add/import and upstream skill fetch/update workflows are deferred from the
  current shipped UI.

## Global Skills

Global Skills is where skills/config at the provider global level on the machine
are viewed.

Displays:

- Provider-specific tabs with provider icons/counts. Shared Agents is selected
  first when available; there is no aggregate "All" tab.
- Provider.
- Global location path.
- Skill/global entry name.
- Mode: symlink, direct. (rsync/copy: reserved, not current UI support)
- Status: current, missing, external symlink, broken symlink, unmanaged.
- Skill Host Folder source if mappable.
- Warning if a global skill may interfere with project-level behavior.

Actions:

- Scan global locations.
- Open global provider folder.

Phase 1 scope:

- Global Skills is a scan, visibility, and remediation surface.
- No Install Skill To Global Location flow yet.
- Global entry remove/relink/adopt actions are deferred from the current shipped
  UI.
- Add Skill flow only targets project providers.

## Global Plugins

Global Plugins is where plugins at the global (user) layer are viewed and managed
for providers that support the plugin convention (Claude, Codex, Antigravity CLI).

File: `apps/desktop/renderer/src/screens/plugins-screen.tsx`.

Displays (grouped by provider):

- Provider-specific tabs with provider icons/counts. Shared Agents is selected
  first when available; there is no aggregate "All" tab.
- Settings file path being scanned by Skillbox (e.g. `~/.claude/settings.json`).
- Layer scan status: ok, not configured, unreadable, malformed, too large,
  symlink, path escape.
- Plugin list with name, marketplace name, status enabled/disabled.
- Marketplace list with name, source type, source summary.

Actions:

- Rescan user-layer settings file for a provider.
- Check installed plugin sources for updates (manual trigger only).
- Toggle enable/disable at the global/user layer for a plugin (only for
  providers with write support: Claude, Codex, Antigravity CLI).

Phase 1 scope:

- Only the global (user) layer is shown in Global Plugins. The project layer and
  effective state per project are in Project Detail.
- Local layer (`settings.local.json`) is read-only.
- Managed settings (enterprise config) are out of scope.

> **Naming note:** The UI displays the label `Global` for the layer that
> code/contracts use the identifier `user` (`layer: "user"`, `PluginLayerUser`,
> SQL `settings_layer = 'user'`). End-user terminology favors `Global`;
> code/data terminology keeps `user` to avoid breaking contracts and the DB.

## Projects

Projects is the list of projects added to Skillbox.

Displays:

- Project name.
- Project path.
- Providers detected.
- Number of skills installed.
- Warning status if any.

Actions:

- Add project.
- Scan project.
- Open project detail.
- Remove project from the Skillbox database.

## Project Detail

Project Detail is the primary screen for coordinating skills within a project.

Displays:

- Project path.
- Provider detected.
- Skills installed.
- Mode: symlink, direct. (rsync/copy: reserved, not current UI support)
- Source skill in host if mappable.
- Plugin tab: plugin version column — Claude reads from `installed_plugins.json`
  (user + project scope); Codex reads from cache dir
  `~/.codex/plugins/cache/` (cache is global, applies to both user layer and
  project layer); Antigravity CLI has no version source → displays `—`.
- Provider Plugins section uses provider-specific tabs with icons/counts. Shared
  Agents is selected first when available; there is no aggregate "All" tab.
- In the Provider Plugins table, Project cells without a project-layer plugin
  override render as an explicit `No override` control, not a bare dash.

Actions:

- Add skill.
- Remove skill.
- Rescan.
- Open project folder.

## Add Skill Flow

Flow to open the Add Skill Wizard from Project Detail.

```text
Project Detail
  -> Add Skill
  -> Wizard opens a tab strip, each tab is an installable provider
     (tab header: ProviderIcon + display name + skills path badge + "experimental" badge if applicable)
  -> User selects the tab of the provider they want to install into
  -> User checks skills in the list for that tab
     (already installed skills at that provider are disabled + "Installed" badge)
  -> Footer shows the path hint for the active tab
  -> User clicks Install
  -> Skill is installed into the provider of the active tab
```

If there are no installable providers (0 valid providers), the wizard shows the
empty state "No provider is ready for install." with CTA "Scan project".

Selection is reset when the user switches tabs.

## Skill Detail

Displays:

- Skill name.
- Host path.
- Source type.
- Source URL or Vercel source id.
- Current version/commit if available.
- Last fetched.
- Projects using this skill.

Actions:

- Fetch.
- Update host copy.
- Open folder.
- Show affected projects.

## Deferred: Skill Source Updates

There is no standalone Updates route in the current app. Upstream skill
fetch/update workflows remain a product concept but are not part of the shipped
screen set.

## Settings

Settings manages:

- Skill Host Folder path.
- Default install mode.
- Provider configs.
- Database location.
- GitHub/Vercel settings if needed.

## About

About screen displays information about the app and its author.

File: `apps/desktop/renderer/src/screens/about-screen.tsx`.

Displays:

- App name and version (from `VITE_APP_VERSION`).
- Author links: Email, GitHub, Blog — click to open in browser.
- Update check: "Check for Updates" button calls `app.checkUpdate` RPC.
  - States: idle / checking / up-to-date / available / error.
  - When a new version is available: displays `latestVersion` and a "View
    release" link to GitHub Releases.

`app.checkUpdate` calls the GitHub Releases API only when the user clicks "Check
for Updates". There is no opt-in gate, but there is also no automatic check when
the About screen opens.

<!-- DOC-VERIFIED: about-screen, use-check-app-update, method-allowlist app.checkUpdate -->
