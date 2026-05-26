# Agent Orchestration Playbook

This playbook hardens the multi-agent workflow used for Astraler Skillbox. It exists to prevent repeated tmux/TUI failures, stale prompts, and uncontrolled phase changes.

## Roles

- `agent-tech-skillbox`: senior developer and implementor. Use for scoped drafting, specs, plans, and implementation.
- `agent-lead-skillbox`: reviewer, QA, and tester. Use for code review, spec review, risk review, and smoke-test validation.
- Orchestrator: coordinates work, owns scope control, decides when to advance phases, and prevents agents from editing the same file at the same time.

## Phase Gates

Use this sequence for substantial work:

1. Brainstorm and scope the next slice.
2. Produce or revise the design spec.
3. Ask lead to review the spec.
4. Ask user to approve the spec.
5. Write the implementation plan.
6. Use `/goal` only for a planned milestone or work package with clear acceptance criteria.
7. Implement, review, test, and smoke-test.

Do not use `/goal` during brainstorming, sketching, quick fixes, or small review follow-ups. Do not move to the next slice until the current slice is reviewed and explicitly closed.

## tmux Hygiene Checklist

Before sending any task:

```sh
tmux capture-pane -t agent-tech-skillbox -p | tail -80
tmux capture-pane -t agent-lead-skillbox -p | tail -80
git status --short
```

Confirm the target pane is in the expected app, not a shell. Confirm there is no stale prompt in the input area. If the agent is in a shell, start the TUI first and verify it loaded before sending work.

For `agent-lead-skillbox`, `cmd=codex-aarch64-a` and the correct `cwd` are necessary but not sufficient. Codex can be running while its input area still contains a suggestion or stale text such as `Run /review on my current changes` or `Summarize recent commits`. Treat that state as not ready.

Before every lead handoff:

1. Capture the pane and inspect the bottom input area.
2. Send `C-u` to clear any visible input.
3. Send the prompt from a short text file or short literal string.
4. Press Enter once. If the text remains in the input area and the TUI is waiting, press Enter one more time.
5. Confirm the prompt moved into the transcript or the pane shows active work.

Use this audit command when the session state is unclear:

```sh
tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} cmd=#{pane_current_command} active=#{pane_active} title=#{pane_title} cwd=#{pane_current_path}' | rg 'agent-lead-skillbox|agent-tech-skillbox'
```

## Context And Model Switching

Before handing off a new task, decide whether the agent should run `/clear`. Use `/clear` when switching phases, changing from brainstorm to implementation, after a long or noisy thread, or after a failed/stale TUI interaction. Do not clear context in the middle of an active goal unless the current work is explicitly stopped or superseded.

Use `opus` for `agent-tech-skillbox` when brainstorming, scoping, or writing plans. Switch `agent-tech-skillbox` back to `sonnet` for implementing an approved plan, running focused fixes, and test/build loops. Keep model changes explicit in the orchestration notes or prompt so the handoff state is clear.

## Prompt Delivery Rules

Do not paste long prompts directly into TUI input. For long instructions, write a task brief file:

```sh
/tmp/skillbox-agent-task.md
```

Then send only a short TUI prompt:

```text
Read /tmp/skillbox-agent-task.md and follow it exactly. Stop after the requested checkpoint.
```

If the TUI shows pasted text and does not run, send Enter once more only after confirming it is waiting for submission. Do not spam Enter.

Do not run heredoc commands or non-interactive review commands inside an agent pane, for example `codex review --commit ... <<EOF`. Those commands can drop the agent out of the TUI workflow and pollute the pane with shell state. If a non-interactive command is needed, run it from the orchestrator shell instead, not from `agent-tech-skillbox` or `agent-lead-skillbox`.

## When To Use `/goal`

Use `/goal` as a durable execution contract, not as the default way to ask an agent to do work. It is appropriate when the agent should keep working across multiple steps until a defined outcome is reached.

Use `/goal` for:

- a slice, milestone, or phase from an approved implementation plan,
- a bounded work package that touches multiple files or layers,
- a smoke-test-and-fix loop with a clear stop condition,
- work that has explicit deliverables, verification commands, and a report/commit checkpoint.

Do not use `/goal` for:

- brainstorm/proposal work,
- asking for status or clarification,
- one-file or two-file quick edits,
- narrow review findings,
- reviewer prompts,
- small follow-up commits after lead review.

For narrow review findings, send a normal prompt to `agent-tech-skillbox`:

```text
Fix lead finding in commit abc123. Scope: file.go only. Run go test ./...
Commit a small fix and report SHA. Do not touch CLAUDE.md.
```

For lead review, always use a normal prompt:

```text
Review commit abc123 only. Do not edit files. Findings first. Verify: [commands].
```

If a small finding expands into a larger design or cross-layer change, stop and create a new plan or planned milestone before using `/goal`.

## Orchestrator Implementation Boundary

The orchestrator is PM and coordinator first. "Do not use `/goal`" means use a normal prompt to the appropriate agent, not that the orchestrator should automatically implement the work inline.

Default ownership:

- `agent-tech-skillbox` implements approved work, including small fixes, using normal prompts or `/goal` depending on scope.
- `agent-lead-skillbox` reviews, tests, and QA-checks using normal prompts.
- Orchestrator writes plans/specs, manages scope, verifies independently, handles tmux hygiene, and only edits code directly for emergency unblock, tiny documentation/playbook changes, or user-approved inline fixes.

If the orchestrator has already created partial local changes before handing off, state that clearly in the tech prompt and ask tech to continue from the current worktree state.

## `/goal` Prompt Shape

Treat `/goal` prompts as scoped execution contracts, not full brainstorm documents. Use the Obsidian template as the canonical checklist, then compress it before sending to the agent:

```text
/Users/tranthien/Library/Mobile Documents/iCloud~md~obsidian/Documents/Obsidian/40-Collection/References/Slash goal prompt template.md
```

Do not rely on memory when writing an important goal. Re-read the template first, especially before large slice work, migrations, smoke-test-and-fix loops, or long autonomous runs.

Use the full template structure when creating a task brief file for a large run:

- context,
- success criteria,
- operating rules,
- quality bar,
- final deliverable.

Use a compressed version when sending directly to tmux. Direct TUI goal prompts must stay short enough to avoid input/paste failures.

A good `/goal` includes:

1. Final outcome in one sentence.
2. Exact scope: files, layer, commit range, or slice.
3. Non-goals and off-limits files, especially `CLAUDE.md`.
4. Success criteria that can be verified.
5. Required commands or smoke steps.
6. Commit message or report-only instruction.
7. Stop condition: report hash, tests, findings, or blocker.

Prefer this compact shape:

```text
/goal [Outcome]. Scope: [files/layer]. Constraints: [non-goals/off-limits].
Success: [observable criteria]. Verify: [commands/smoke]. Commit/report: [message or no-edit review].
Stop after [checkpoint] and report [hash/tests/findings/blockers].
```

Do not overload `/goal` with broad product context if the agent can read the relevant spec or plan file. Reference the file instead. For large goals, create `/tmp/skillbox-agent-task.md` and send a short `/goal` that tells the agent to read it.

## Editing Ownership

Only one actor may edit a file at a time. If `agent-tech-skillbox` is editing a file, the orchestrator must not edit that same file unless the tech task has been interrupted and the pane is idle.

For reviewer work, `agent-lead-skillbox` must not edit files unless explicitly asked. Review prompts should include `Do not edit files`.

## Recovery Procedures

If a TUI prompt is stale or wrong:

1. Capture the pane.
2. Send `C-c` once.
3. Capture again.
4. If still unsafe, send `C-c C-c` to exit the TUI.
5. Restart the TUI cleanly and verify an empty prompt before continuing.

If a prompt is accidentally sent to the shell, stop immediately. Do not try to continue from mixed shell/TUI state; restart the agent TUI.

If the pane shows a suggestion or placeholder such as `Summarize recent commits`, treat it as unsafe until verified. Clear or exit before sending a real task.

If `C-c` exits Codex to a shell, restart with `codex --yolo`, then capture the pane again before sending work. Do not trust the process name alone; inspect the visible input area.

## Review Loop

Lead reviews must be scoped to a commit or file and must start with findings. Example:

```text
Review commit abc123 only. Do not edit files. Findings first. Approve or block.
```

When the lead finds an issue, send it to tech as a small scoped task. After the fix commit, ask lead to review only the follow-up commit.

Do not ask for a final verdict after interrupting, clearing, or restarting the lead unless the lead has already shown evidence that it inspected the diff and considered verification. If the lead reports `No verdict` because it did not inspect the change, treat that as a correct guardrail and rerun a scoped review from scratch.

Slow review is acceptable when the pane shows active reading, searches, or test reasoning. Wait for the actual review instead of forcing a quick approval.

When lead findings identify docs/source-of-truth drift, fix the documentation before closing the task. Rerun targeted searches after the fix, for example:

```sh
rg "old provider label|stale UI text" docs apps core-go
```

## Migration And Bulk Edit Safety

When adding a later migration that changes data created by an older migration, keep the older migration test faithful to the old SQL. Add a new test for the new migration and update only latest-state expectations where appropriate.

After broad search-and-replace work, inspect nearby tests and numeric assertions before committing. Run targeted searches for accidental replacements and verify migration test fixtures still match their migration files.

## Hardening Notes

Every orchestration failure should become a rule here. Typical hardening updates include:

- stale prompt prevention,
- long-prompt delivery via file,
- phase-gate enforcement,
- review-only guardrails,
- recovery steps for shell/TUI drift,
- ownership rules for shared files.
