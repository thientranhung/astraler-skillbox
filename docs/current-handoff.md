# Current Handoff

Updated: 2026-05-27

## Repository

Path: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox`

Current branch state at handoff: `main...origin/main [ahead 261]`

Recent commits:

- `65ed5df Add provider plugin toggle smoke test checklist`
- `f14e23e Add TOML plugin write support for Codex`
- `d6a02a6 Update current handoff after project-level toggles`
- `32a561f Implement project-level plugin toggles for JSON providers`
- `d137801 Update current handoff document`
- `1e163ea Update playbook to support agy reviewer process`

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

- Created manual UI smoke test checklist section in `SMOKE.md` (commit `65ed5df`).
- Implemented write support for Codex config files (TOML format) under `providerPlugin.setEnabled` in commit `f14e23e`.
- Created a comment-preserving TOML editor in the Go backend (`toml_plugin_writer.go`) using line-based regex replacements and double validation passes via `toml.Unmarshal`.
- Enabled Codex plugin Enable/Disable toggling in the React frontend UI by whitelisting `"codex"` in both `plugins-screen.tsx` and `project-detail-screen.tsx`.
- Audited the implementation commit via `agent-lead-skillbox` (`agy` reviewer) and received full approval.
- Implemented project-level plugin enable/disable write actions for JSON-format providers (`claude` and `antigravity_cli`) in commit `32a561f`.

## Next Work

| Priority | Task | Notes |
| --- | --- | --- |
| P0 | Check repo and tmux health | Ensure clean state and both agents are usable |
| P1 | Connect Global/Project Scans to the Registry | Ensure scans read paths from settings registry instead of hardcoded paths |
| P2 | Apple Developer ID and notarization | Accepted release/distribution tech debt |

## Suggested Tech Prompt

Use a task file if the prompt becomes long.

```text
Implement connecting global and project scans to the settings registry. The scanner must resolve paths using definitions from the settings registry database table instead of hardcoded paths. Add validation and unit/Vitest tests. Commit one focused implementation commit and report SHA.
```

## Suggested Lead Prompt

Use after the tech agent reports a commit.

```text
Review the latest tech commit only. Do not edit files. Findings first. Check correctness, tests, regressions, and whether paths resolution via registry is correct and safe. Approve or block.
```
