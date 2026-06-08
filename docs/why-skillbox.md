# Why Skillbox

Agent skills are becoming part of the development workspace.

That sounds simple until you use more than one project, more than one provider,
and more than one evolving skill.

## The Problem

Agent coding providers are moving toward project-local skill folders. Many tools
can load skills from a folder inside the project, commonly under conventions
such as `.agents/skills`.

At the same time, skill installers and registries are starting to ask a familiar
question:

```text
Install this skill globally or into this project?
```

Both answers are useful, and both can become messy.

Global installs can affect projects that do not need the skill. Project installs
can create many copies of the same skill. When that skill changes, every copied
project becomes a place where drift can happen.

After a few weeks of agentic coding, the real questions become harder than the
installation command:

- Which project is using which skill?
- Which provider folders exist in this repository?
- Is this skill global, project-local, or both?
- Where is the version I should edit?
- Did I update every project that depends on it?
- Which global skills and plugins are affecting this workspace?
- Which skills are experiments, and which are shared working tools?

## The Skillbox Answer

Astraler Skillbox starts from a different center of gravity:

```text
One local Skill Host Folder.
Many projects linked from it.
```

The Skill Host Folder is where you keep, study, install, and develop skills.
Each project receives only the skills it needs. Skillbox distributes those skills
through symlink, so the project sees the skill while the source remains in one
place.

```text
Skill Host Folder
  .agents/skills/code-review
        |
        | symlink
        v
Project A
  .agents/skills/code-review

Project B
  .agents/skills/code-review
```

Update the skill once in the host folder, and every linked project receives the
update immediately.

## Why Not Only Global Skills?

Global skills are convenient, but not every project should inherit every skill.

One project may need a browser automation skill. Another may need a release QA
skill. A third may need a local research workflow. Installing everything
globally can pollute provider context and make project behavior harder to reason
about.

Skillbox keeps global skills visible, but it does not make global state the
center of the system. The center is the host folder plus explicit project
distribution.

## Why Not Copy Skills Into Every Project?

Copying feels safe at first because each project owns its files.

The cost appears later:

- bug fixes must be copied everywhere
- experiments fork silently
- old versions stay hidden in old projects
- it becomes unclear which copy is the real one

Symlink keeps the project-local provider convention while avoiding scattered
manual copies.

## What Skillbox Shows

Skillbox is not only an installer. It is a visibility tool.

It scans the host folder, project folders, global provider folders, and plugin
settings so you can see the actual skill picture across your machine:

- host skills available for distribution
- project skills installed through symlink
- provider folders detected in a project
- global skills and global plugin configuration
- project-level provider configuration
- missing or unsupported provider states

The goal is simple: make skill state visible before it becomes chaos.
