# Product Brief: Astraler Skillbox

## Problem

Agent skills are becoming an important part of workflows with AI agents. Users
are increasingly experimenting with many skills, many projects, and many agent
providers such as Claude, Codex, opencode, Antigravity CLI, etc.

Agent coding providers are also converging around project-local skill folders.
Many providers can load skills from project folders such as `.agents/skills`,
while skill installers increasingly ask whether a skill should be installed
globally or into the current project.

Managing skills is currently fragmented:

- Skills live in many different places.
- Global skills and project-level skills are easily confused.
- Each project needs its own set of skills.
- Global installs can pollute projects that do not need a skill.
- Project-by-project installs can create copied skills that drift apart.
- Each provider has its own conventions for folder, path, naming, or format.
- Not only developers use skills, so CLI-only is not enough.
- Updating skills is inconvenient when multiple projects share the same skill.

Astraler Skillbox solves this problem with a GUI-first app for managing skills
locally across multiple projects and providers.

## Product Positioning

```text
Skillbox is a local-first control center for agent skills.
```

Skillbox manages:

- Skill Host Folder: a folder the user selects in the GUI as the source of truth
  for skills on the machine.
- Skills: skills present in the host.
- Sources: GitHub, Vercel skills, local/manual.
- Projects: projects added to the app.
- Global Skills: skills/config at the provider global level on the machine.
- Providers: Claude, Codex, opencode, Antigravity CLI, and other providers.
- Installs: which skills are installed into which project/provider, and by what
  mode.
- Updates: fetch upstream to find out which skills have new versions.

## Target Users

Users are not only developers.

User groups may include:

- Developer
- Content creator
- Researcher
- Marketer
- Operator
- PM
- Founder
- Analyst

Their common ground is that they use many AI agent workflows and need controlled
skill management.

## Confirmed Pain Points

- Users experiment with many skills; over time they lose track of where each
  skill is.
- Each project needs its own set of skills.
- Global skills and project-level skills are easily confused, causing context
  noise and overlapping behavior.
- A globally installed skill may affect projects that do not need it.
- Installing the same skill separately into many projects creates duplicate
  copies and update drift.
- Users have difficulty seeing which skills/config exist at the provider global
  level.
- Many agent providers have different conventions for folder, path, naming.
- Not only developers use skills, so CLI-only is not sufficient.
- Updating skills is inconvenient when multiple projects share the same skill.
- Users have difficulty knowing which project is using which skill.
- Skill discovery and skill management are currently fragmented.

## Confirmed Design Decisions

- Skillbox is GUI-first.
- Skill Host Folder is a folder the user selects and configures in the GUI.
- Skill content source of truth lives in the Skill Host Folder.
- Global Skills is a separate area for observing the provider global level, not
  mixed with the Skill Host Folder or project-level installs.
- The app uses SQLite from the start to store management metadata.
- Skill sources prioritize GitHub and Vercel skills.
- There is a Fetch button to check for upstream updates.
- Detailed health checks are not the current focus.
- Users need to understand technical concepts such as symlink, provider, and
  Skill Host Folder.

## Skill Host Folder

The Skill Host Folder is a folder the user selects in the GUI as the source of
truth for skills on the machine.

```text
<skill-host-folder>/
  .agents/
    skills/
      documentation-and-adrs/
      documentation-writer/
      browser-automation/
```

Skillbox reads the skill list from this folder and installs skills to other
projects via symlink.

## Project Install

A project install is a skill from the Skill Host Folder being installed into a
specific project/provider.

Main flow:

```text
<skill-host-folder>/.agents/skills/<skill>
        |
        | symlink
        v
target-project/.agents/skills/<skill>
```

Install mode:

- `symlink`: the project points directly to the skill in the Skill Host Folder.
  This is the currently supported mode.
- `direct`: the skill already exists in the project but is not managed by
  Skillbox.

## Provider Model

Skillbox needs provider adapters to understand which folder/path/convention each
provider uses.

Current assumptions:

- Claude has its own world.
- Many other providers may share a common convention like `.agents`.
- Nevertheless, the adapter layer must exist from the start to avoid being locked
  into a single convention.

## Updates

Skillbox has a Fetch button to check for upstream updates.

Priority skill sources:

- GitHub repo directly.
- GitHub repo + subfolder.
- GitHub repo + branch/tag/commit.
- Vercel skills ecosystem.
- Local/manual skill.

With symlink installs, updating the Skill Host Folder immediately affects the
project.
