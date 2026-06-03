# Product Notes

## Product Thesis

Skillbox is a GUI-first local skill manager for the era of multiple AI agent
providers.

It helps users manage, install, update, check, and observe agent skills across
multiple projects, multiple providers, and multiple formats.

## Current Scope

Skillbox is not a small utility. It is a large product with GUI as the primary
experience.

## Core Product Pieces

```text
Skill Host Folder
  Folder the user selects in the GUI as the source of truth for skills on the machine.

Skillbox GUI
  Primary interface for managing skills, global skills, projects, providers, installs,
  and updates.

Provider Adapters
  Mapping providers to their corresponding folder/path/convention.

SQLite Database
  Stores management metadata such as projects, skills, global installs, project installs,
  sources, and fetch results.
```

## Decisions

- Use SQLite from the start.
- Skill sources prioritize GitHub and Vercel skills.
- Use Fetch to check which skills have new versions.
- Symlink is the currently supported install mode.
- If the Skill Host Folder is moved/deleted, the app will warn when opening or
  scanning.
- Detailed health checks are deferred; not the current product focus.
- Non-developer users still need to understand technical concepts.

## Install Model

Symlink:

- The project points directly to the Skill Host Folder.
- Editing one place in the Skill Host Folder means multiple projects receive the
  change immediately.
- This is the currently supported install mode.

## Update Model

Fetch only checks whether upstream has changed.

Update brings the change from upstream into the Skill Host Folder.

With projects installed via symlink, updating the Skill Host Folder means those
projects receive the change immediately.

## Plugins and Marketplaces

Some providers (Claude, Codex, Antigravity CLI) have their own **plugin** concept,
distinct from skills. A plugin is a provider extension declared in the provider's
own settings file (e.g. `~/.claude/settings.json`), typically fetched from a
**marketplace** (a named source such as a GitHub repo, git URL, or local
directory).

Skillbox reads the provider's settings file to display which plugins are
enabled/disabled at which layer (user/project/local) and allows the user to
toggle quickly without manually opening the settings file. Skillbox does not
download marketplace content itself; the provider handles that.

Phase 1 scope:

- Scan settings file at user + project layer. Local layer is scan-only, no write.
- Toggle enable/disable at the global (user) layer or per-project (project layer,
  3-state cycle: inherit → enabled → disabled).
- Managed settings (provider-managed enterprise config) are out of scope.

`provider_plugin` is the domain object representing a plugin in code. Data model
details are in [`06-data-model.md`](06-data-model.md) § Provider Plugin Layer
System; flow details are in [`08-provider-model.md`](08-provider-model.md) §
Provider Plugin Layer Model.

## Remaining Tradeoffs

### Provider Convention Drift

Providers may change their folder/path/convention. Skillbox needs an adapter
layer to isolate this.

### Source Metadata

For Fetch to work well, each skill should have source metadata:

- GitHub repo
- Subfolder if applicable
- Branch/tag/commit
- Vercel skills identifier if applicable
- Local/manual if no clear upstream

### Visibility on Update

Symlink is the primary design, but the UI should still display which projects
will be affected when the Skill Host Folder is updated.
