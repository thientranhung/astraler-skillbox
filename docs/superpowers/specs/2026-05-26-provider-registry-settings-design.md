# Provider Registry Settings Design

## Goal

Skillbox must show how each LLM or agent provider organizes skills at both global level and project level. Settings becomes the provider registry: the place where Skillbox exposes built-in provider defaults, user overrides, icons, enablement, and the path candidates used by project/global scans.

## Product Model

Skillbox manages three related concepts:

- Skill Host: the user's source library, for example `<global-documents>/my-agent-skills`.
- Provider global scope: where a provider loads user-level skills, for example `~/.claude/skills`.
- Provider project scope: where a provider loads project-level skills, for example `<project>/.claude/skills`.

Provider identity is the main axis. UI must not merge skills by name across providers or scopes.

## Provider Registry

Built-in providers are owned by Skillbox and ship with defaults: Claude, Codex, Gemini, Antigravity CLI, OpenCode, and Generic Agents. Built-ins expose a stable key, display name, icon key, support status, capability flags, and path candidates. Users may override paths and reset to defaults, but built-in provider keys are not editable.

Custom providers are deferred. The registry shape must still allow future user-defined providers with custom key, name, icon, and path candidates.

Provider `status` and user `enabled` are separate. `status` describes Skillbox's confidence/capability (`supported`, `experimental`, `unsupported`). `enabled` is a user preference and must not overwrite provider capability truth.

## Path Model

The registry must support arrays of path candidates rather than one path per provider. Each candidate has:

```text
scope: project | global
purpose: detect | skills | config | commands
path
priority
source: builtin | override | custom
verification_status: verified | assumed | experimental
```

Project scan resolves project candidates against the project root. Global scan resolves global candidates against the user home or absolute paths. Effective paths are `override ?? builtin default`.

## Settings UI

Add a Providers section in Settings. It should show a dense table with icon, display name, key, status, enabled toggle, project detect candidates, project skills candidates, global skills candidates, and actions. Built-in rows show reset controls when overridden. Experimental or assumed conventions must be visually distinct without noisy warning text.

## Screen Integration

Global Skills groups entries by provider and displays the effective global skills path candidates. Project List and Project Detail use provider registry metadata for icons and labels. Add Skill target selection uses the provider registry rather than scattered hard-coded maps.

## Slice Plan

PR-1 is read-only registry hardening: add a `provider.list` contract/API, seed icon keys and OpenCode if verified, render the Settings provider table, and update provider icons to use registry metadata. It must not change scan/install behavior.

PR-2 adds overrides and reset storage. PR-3 separates enabled state and connects global/project scans to the registry. PR-4 adds custom providers.

## Verification Notes

PR-1 must include contract drift checks, Go repository/service/handler tests, renderer Settings tests, and a registry-vs-seed drift test. Before hard-coding provider defaults, verify provider conventions from official documentation when available. Antigravity CLI remains experimental until its project and global skill paths are confirmed.
