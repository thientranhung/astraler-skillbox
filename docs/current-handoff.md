# Current Handoff

Updated: 2026-05-27

## Repository

Path: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox`

Current branch state at handoff: `main...origin/main [ahead 258]`

Recent commits:

- `32a561f Implement project-level plugin toggles for JSON providers`
- `d137801 Update current handoff document`
- `1e163ea Update playbook to support agy reviewer process`
- `e350b3a Implement global plugin enable/disable for JSON-format providers`
- `5cb2bc6 Keep Antigravity lead note out of playbook`

## Operating Model

The orchestrator is PM/coordinator, not the default implementor.

| Role | Session | Command | Responsibility |
| --- | --- | --- | --- |
| Tech | `agent-tech-skillbox` | `claude --dangerously-skip-permissions` | Implementation |
| Lead | `agent-lead-skillbox` | `agy --dangerously-skip-permissions` | Review, QA, testing |
| Orchestrator | current session | shell/tmux | Scope, handoff, verification, hardening |

If the tech agent is degraded or context-poisoned, restore the agent workflow first: interrupt, `/goal clear`, `/clear`, restart Claude, split the task smaller, or ask the user before any orchestrator implementation exception.

Read `docs/agent-orchestration-playbook.md` before continuing work.

## Environment Checks

Run before the next handoff:

```sh
cd /Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox
git status --short --branch
tmux capture-pane -t agent-tech-skillbox -p | tail -80
tmux capture-pane -t agent-lead-skillbox -p | tail -80
```

## Completed Recently

- Implemented project-level plugin enable/disable write actions for JSON-format providers (`claude` and `antigravity_cli`) in commit `32a561f`.
- Added Enable/Disable action toggles on the Project Detail screen, with automatic local override warnings and button disabling.
- Updated `useSetProviderPluginEnabled` hook to invalidate project detail queries on mutation success.
- Implemented global (user-layer) plugin enable/disable write actions for JSON-format providers in commit `e350b3a`.
- Updated the agent orchestration playbook to support both `agy` and `codex` reviewer processes in commit `1e163ea`.

## Next Work

| Priority | Task | Notes |
| --- | --- | --- |
| P0 | Check repo and tmux health | Ensure clean state and both agents are usable |
| P1 | Codex/TOML write support | Support TOML comment-preserving writes for Codex config |
| P1 | Connect Global/Project Scans to the Registry | Ensure scans read paths from settings registry instead of hardcoded paths |
| P2 | Apple Developer ID and notarization | Accepted release/distribution tech debt |

## Suggested Tech Prompt

Use a task file if the prompt becomes long.

```text
Implement write support for Codex config files (TOML format) under providerPlugin.setEnabled. The implementation should preserve comments, formatting, and unrelated keys in the TOML file. Add boundary validation, error handling, and unit/Vitest tests. Commit one focused implementation commit and report SHA.
```

## Suggested Lead Prompt

Use after the tech agent reports a commit.

```text
Review the latest tech commit only. Do not edit files. Findings first. Check correctness, tests, regressions, and whether Codex/TOML comment-preserving config writes are safe and correctly scoped. Approve or block.
```
