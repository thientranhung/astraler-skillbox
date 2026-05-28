# Agent Orchestration Playbook

## Roles

| Name | tmux pane | Role |
|------|-----------|------|
| **Tom** | `agent-tech-skillbox` | Senior developer. Brainstorm, specs, plans, implementation. |
| **Larry** | `agent-lead-skillbox` | Reviewer & QA. Code review, spec review, smoke tests. Does NOT edit files. |
| **Orchestrator** | (this session) | PM & coordinator. Decides **who** and **when**, never **what** or **how**. No technical opinions — route all analysis/design to Tom or Larry. |

Orchestrator may directly edit: this playbook, process docs, tiny doc fixes, or user-approved exceptions only.

## `/goal` Template

Before using `/goal`, read and follow this template:

```
/Users/tranthien/Library/Mobile Documents/iCloud~md~obsidian/Documents/Obsidian/40-Collection/References/Slash goal prompt template.md
```

For large goals, write a descriptive file in `.scratch/` and send a short `/goal` referencing it.

## Phase Gates

Recommended sequence for substantial work. Compress for small slices, but never skip user approval on specs.

> A **slice** is a thin end-to-end piece of work (UI → service → data, or any cross-layer cut) — bigger than a single edit, smaller than a feature.

1. Brainstorm & scope → Tom
2. Design spec → Tom
3. Spec review → Larry
4. User approval
5. Implementation plan → Tom
6. Implement → Tom, review → Larry, test & smoke-test
7. Update docs / source-of-truth (see [Docs & Source of Truth](#docs--source-of-truth))

## tmux Rules

### Before Every Handoff

```sh
tmux capture-pane -t <pane> -p | tail -80
git status --short
```

Confirm: TUI is running (not shell), input area is empty, no stale text.

### Sending a Prompt

Clear stale input, send the prompt, send Enter separately to submit, then capture the pane to verify it moved into the transcript. The TUI's first Enter only confirms multi-line input — submission needs a second Enter call.

**Short vs file delivery:** Send prompts inline by default. Only write to `.scratch/` when the message is too long for tmux input (~500+ chars). Name files descriptively, e.g. `.scratch/fix-useeffect-regression.md`, `.scratch/slice-3k-impl-plan.md`.

**Never send keys into an interactive selection prompt blindly.** If a capture shows the agent is at a `Enter to select · ↑/↓ to navigate · Esc to cancel` prompt, do NOT send arbitrary text or Enter — it will register as a selection (often the default/Recommended option) and silently corrupt the agent's answer record. Instead: `Escape` first to dismiss the prompt, verify input area is clean, then send a text message. To relay a user's choice into a selection prompt, send the exact navigation keys (`Down`/`Up` × N then `Enter`) — and capture immediately after to verify the recorded answer matches what the user picked.

### Context & Model Switching

- `/clear` before unrelated tasks or phase switches. Never clear mid-goal.
- Match model strength to task type: stronger / deep-thinking model for brainstorm/scope/plan, faster / cheaper model for implementation, fixes, and test loops. For Tom (Claude Code): **opus** ↔ **sonnet**. For other runtimes, map equivalently.

### Waiting for an Agent to Finish

User should not have to manually poll agent status. Orchestrator auto-polls via background Bash + tmux capture:

```sh
# Idle detection: no spinner pattern (Cogitating/Fermenting/… (Xs · tokens)) for 30s
stable=0; iters=0; max=180
while [ $stable -lt 6 ] && [ $iters -lt $max ]; do
  out=$(tmux capture-pane -t <pane> -p 2>/dev/null)
  if echo "$out" | grep -qE '… \([0-9]+[smh]'; then stable=0; else stable=$((stable+1)); fi
  iters=$((iters+1)); sleep 5
done
echo "agent idle"
```

Run with `run_in_background: true` — single notification when script exits. Max iters caps runaway (180×5s = 15min). If agent is still working after timeout, re-poll.

Pattern `… \([0-9]+[smh]` matches Claude Code's active spinner line (`✶ Cogitating… (11s · ↑ 1.3k tokens)`). Adapt for other runtimes if their spinner differs. Status footer (timer, %, model) updates every second but never matches the spinner regex, so footer churn doesn't reset the stable counter.

### Audit Command

```sh
tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} cmd=#{pane_current_command} cwd=#{pane_current_path}' | rg 'agent-tech|agent-lead'
```

## Review

Larry handles reviews. There are several review types — pick the right one and trust Larry's skills/tools to execute it:

- **Code review** — diff/commit-level correctness, style, regressions.
- **PR review** — full PR scope, cross-commit consistency, merge readiness.
- **Spec/design review** — architecture, risks, missing cases before implementation.
- **Security review** — auth, data exposure, injection surfaces.

Leverage whatever review tooling Larry's runtime provides — built-in slash commands, skills, MCP review servers, or provider-native review flows. Larry picks the best tool for the review type and the codebase; the orchestrator only states the target and intent.

Principles (apply to any review type):
- Scope to a specific target (commit, PR, file, spec).
- Findings first; Larry decides approve / block / needs-discussion.
- Larry does NOT edit files unless explicitly asked.
- Lead finding → Tom fixes in a scoped commit → Larry re-reviews only the follow-up.
- If Larry reports "No verdict" (didn't inspect), rerun from scratch.
- Docs/source-of-truth drift found → fix before closing.

## Smoke Tests

Smoke tests verify end-to-end behavior — UI, CLI, API, data flow, IPC, whatever the slice touches. Not UI-only.

Test scenarios are designed **before implementation**, during Tom's spec/plan phase:

- Tom brainstorms smoke scenarios as part of the spec.
- Scenarios are stored with the spec (so user can review and approve them upfront).
- During implementation phase, Larry (or user) just executes the approved scenarios.

Principles:
- Scenarios cover the slice's external surface, not internal units (that's what unit tests are for).
- Larry executes and reports pass/fail with evidence; Larry does not invent scenarios on the fly.
- Failure → Tom fixes. Larry never edits.
- If a scenario gap is found during execution, log it back to the spec for next iteration.

## Ownership

- One actor per file at a time.
- Larry does NOT edit files unless explicitly asked.
- Orchestrator does NOT implement product code. Agent failure → restore agent first (clear, restart, split task, switch model), then ask user if still stuck.

## Recovery

**Stale/wrong prompt:** `C-c` → capture → if still broken, `C-c C-c` to exit TUI → restart TUI → verify empty input.

**Tom degraded (stale behavior, wrong scope, corrupted context):**
1. `C-c`, capture pane
2. `/clear` or restart TUI
3. Re-send smaller task with explicit stop condition
4. If repeats, ask user — never self-implement

**Shell leak:** If agent drops to shell, restart its TUI with the runtime's standard "uninterrupted" launch flags so permission prompts don't stall work. Verify the input area is clean before sending work. Common launches:

- Claude Code: `claude --dangerously-skip-permissions`
- Codex: `codex --yolo`
- OpenCode / agy: `agy --dangerously-skip-permissions`

Do not trust the process name alone — inspect the visible input area.

## Docs & Source of Truth

Every slice must end with documentation and source-of-truth aligned to the implementation. Treat this as part of "done", not a follow-up.

- Tom updates the relevant docs as part of the implementation commit (or a paired commit in the same slice): architecture docs, `CLAUDE.md`/`AGENTS.md`, schema dictionary, contracts/types, README, changelogs.
- Larry checks for docs drift during review and blocks if implementation diverges from spec or if spec/source-of-truth wasn't updated.
- Source-of-truth wins: if code and docs disagree, fix the side that's wrong — don't silently let drift accumulate.
- When a slice changes a public contract, schema, or convention, update both the canonical file and any examples/quickstart that reference it.
- Re-run targeted searches after doc updates to catch stale labels/paths, e.g. `rg "old term|old path" docs apps core-go`.

## Maintenance

Every orchestration failure → add a rule here. Keep this playbook lean: principles over recipes, references over duplication.
