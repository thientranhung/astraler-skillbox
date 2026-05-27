# Current Handoff

Updated: 2026-05-27

## Repository

Path: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox`

Current branch state at handoff: `main...origin/main [ahead 263]`

Recent commits:

- `f059c16 Connect plugin scans to settings registry for dynamic path resolution`
- `c0b3d20 Update current handoff after adding plugin toggle smoke docs`
- `65ed5df Add provider plugin toggle smoke test checklist`
- `f14e23e Add TOML plugin write support for Codex`
- `d6a02a6 Update current handoff after project-level toggles`
- `32a561f Implement project-level plugin toggles for JSON providers`

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

- Connected global and project plugin scans to the database settings registry (commit `f059c16`). Config files paths are now resolved dynamically from `provider_path_candidates` (by matching scope, purpose: `"config"`, and sorting by priority) rather than using hardcoded maps.
- Hardened security in `SetPluginEnabled` project-level writes by verifying path confinement against the project root container (`project.Path`) directly.
- Created manual UI smoke test checklist section in `SMOKE.md` (commit `65ed5df`).
- Implemented write support for Codex config files (TOML format) under `providerPlugin.setEnabled` in commit `f14e23e`.
- Created a comment-preserving TOML editor in the Go backend (`toml_plugin_writer.go`) using line-based regex replacements and double validation passes via `toml.Unmarshal`.

## Next Work

| Priority | Task | Notes |
| --- | --- | --- |
| P0 | Check repo and tmux health | Ensure clean state and both agents are usable |
| P1 | Apple Developer ID and notarization | Accepted release/distribution tech debt |

## Suggested Tech Prompt

Use a task file if the prompt becomes long.

```text
Inspect release scripts and codesigning configurations. Standardize environment checking and prepare Apple Developer ID notarization steps.
```

## Suggested Lead Prompt

Use after the tech agent reports a commit.

```text
Review the latest tech commit only. Do not edit files. Findings first. Check correctness, tests, regressions. Approve or block.
```
