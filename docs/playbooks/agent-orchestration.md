# Agent Orchestration Playbook

> If stuck, read the TL;DR. When you need detail, jump by heading.

## TL;DR

- **Tom** codes, **Larry** reviews, **Orchestrator** coordinates only (does not make technical decisions).
- Long / multi-step / cross-layer plans or cross-phase handoffs -> use **`/goal`** to lock context. Short, clear work -> a normal message is enough.
- Standard phase flow: Brainstorm -> Branch? -> Spec -> Spec review -> User approval -> Plan -> Implement -> Review -> Smoke -> PR -> Docs.
- Branch + PR is **required** for schema migrations or breaking changes. Use the table for everything else.
- Each orchestration failure adds one rule to this playbook. Do not repeat a failure twice.

## Roles

| Name | tmux pane | Role |
|---|---|---|
| **Tom** | `agent-tech-skillbox` | Senior dev. Brainstorm, spec, plan, implement. |
| **Larry** | `agent-lead-skillbox` | Reviewer & QA. Code/spec/security review, smoke. **Does not edit files.** |
| **Orchestrator** | (this session) | PM & coordinator. Decides **who** and **when**, not **what** or **how**. |

Orchestrator may edit directly only: this playbook, process docs, tiny doc fixes, or anything the user explicitly allows.

## `/goal` for Long Work

`/goal` is how Orchestrator packages a large task for Tom/Larry when context can drift. **It is not a raw task dump; it locks role + phase + expected output.** A weak goal makes the agent guess, causes context drift, and forces redo. For long plans, spending 2 minutes writing a good goal can save 30 minutes of cleanup. This is a tool to use at the right time, not a mandatory gate for everything.

### When To Use `/goal`

- Long / multi-step plan, cross-layer slice, or >1 commit -> should use it (context can drift).
- Cross-phase handoff (Spec -> Plan -> Implement), where the previous phase output becomes the next input -> should use it.
- Single edit, small follow-up, very clear scope -> normal message is enough; **no `/goal` needed**.

### Anatomy: 5 Blocks (for file-backed goals)

A full file-backed `/goal` contains the 5 blocks below. Inline goals use the shorter version (see "Two flavors") and do not need every block. This structure inherits the user's original `/goal` template and adapts it for this repo's phase-gated workflow.

1. **CONTEXT**: project, slice, relevant stack, current state, working dir/branch, inputs (path to spec/plan/brainstorm), constraints (MUST + MUST NOT), audience. Lock role + phase at the top (`You are Tom. Phase = Spec.`).
2. **SUCCESS CRITERIA**: numbered list, each line measurable. Must include one line for "build/test pass without errors" and one line for "evidence shown" (diff, log, screenshot, smoke result). If a concept changes, add one docs line per `documentation.md`.
3. **OPERATING RULES**: 10 non-negotiable rules (Plan first, Respect phase gate, Self-verify, Debug yourself, Use every tool, No placeholders, Progress log, Stay on scope, If blocked, Check success before stopping). Adapted from the original template: rule #2 replaces "WORK AUTONOMOUSLY" with **RESPECT PHASE GATE** so the agent stops at the correct phase and does not collapse spec -> code.
4. **QUALITY BAR**: code conventions, architecture hard rules (`docs/10`), Larry-review-passable output, docs updated, `DOC-VERIFIED` commit trailer.
5. **FINAL DELIVERABLE**: `[OK]` per criterion, `[FILE]` paths, `[RUN]` verification command, `[PROOF]` evidence, `[LOG]` decisions, `[WARN]` limitations, `[STOP]` next phase + assignee.

### Two Flavors

**Inline**: < 500 characters, sent directly through tmux. Use for small same-phase follow-ups, single edits, very clear scope. Keep only the essentials (outcome + context + success + stop); it does not need all 5 blocks.

**File-backed**: write `.scratch/goal-<slice>-<phase>.md`, then send a one-line `/goal` pointing to the file. Use when context is long, there are many paths/snippets, or the handoff crosses phases. Naming: `goal-<slice>-<phase>.md` (for example `goal-slice-b.md`, `goal-naming-ui-fixes.md`). See existing `.scratch/goal-*.md` files for real examples.

### Templates

Copy from `docs/playbooks/templates/`:

- [`goal-inline.md`](templates/goal-inline.md): short template.
- [`goal-file.md`](templates/goal-file.md): full template.

### Anti-Patterns

- Sending large work raw, such as "Tom, implement skill provider tabs": missing CONTEXT/SUCCESS/RULES lets the agent merge brainstorm + spec + implementation and lose phase control. (Small clear work can still be sent raw; this is a problem for long plans.)
- Pasting 50 lines of context into tmux instead of linking a file: easy to corrupt the input area and hard to rerun.
- Goal has no **MUST NOT** block in CONTEXT.constraints: agent may expand into unrelated refactors.
- Dropping **RESPECT PHASE GATE** when copying the original template: agent may autonomously run spec -> code -> PR and skip Larry review/user approval.
- SUCCESS CRITERIA says "implement feature X" instead of measurable outcomes: no way to verify done.

## Phase Gates

1. Brainstorm & scope -> Tom. Output includes **Risk Classification** (see table below).
2. Branch decision -> Orchestrator applies rule, creates branch if needed, **before** Spec.
3. Spec -> Tom.
4. Spec review -> Larry.
5. User approval.
6. Implementation plan -> Tom.
7. Implement -> Tom; **Tom runs `gh pr create`** if on a branch.
8. **Larry runs `gh pr review` on the PR**, not just a tmux report. If there are issues -> Larry comments `--request-changes` + `file:line` -> Tom fixes on branch + pushes -> Larry re-reviews. Repeat until clean -> Larry `--approve`.
9. **Tom runs `gh pr merge`** after the PR has Larry approval.
10. Docs / source-of-truth update -> Tom updates canonical docs **in the same slice** (do not defer); see `documentation.md` for which docs.

> Slice = thin cross-layer cut (UI -> service -> data). Compress phases for small slices, but **do not skip user approval at Spec**.

## Branch & PR Workflow

### Risk Classification (Tom closes at the end of brainstorm)

| Field | Value |
|---|---|
| Layers | UI / contract / Go / SQL / docs |
| Breaking change | yes / no |
| Schema/migration | yes / no |
| Est. LOC | <50 / 50-300 / >300 |
| Workflow | direct-to-main / branch + PR |

### Decision Rule (Orchestrator applies before Spec)

| Condition | Workflow |
|---|---|
| Schema/migration: yes **OR** Breaking change: yes | **MUST** branch + PR |
| Layers >= 3 **OR** Est. LOC > 300 | **SHOULD** branch + PR |
| Independent multi-slice parallel work | worktree per slice |
| UI-only / docs-only, < 50 LOC | OK direct-to-main |

Branch naming: `<type>/<kebab-slug>` (for example `feat/dashboard-plugins-metric`). PR target is always `main`. Tom creates the PR immediately after Larry approves the final commit.

**Standard PR flow (do not combine create + merge):**

Tom pushes branch -> `gh pr create` -> Larry runs `gh pr review` on the PR (issues: `--request-changes` + `file:line` -> Tom fixes + pushes -> Larry re-reviews; clean: `--approve`) -> Tom runs `gh pr merge`.

Review must happen **on the PR** (`gh pr review`), not only in tmux, so the PR has a real trace. `reviews: []` after merge means the gate was skipped.

> **Same-owner gotcha (will recur):** GitHub blocks `gh pr review --approve` / `--request-changes` on PRs from the *same account* ("Can not approve/request changes on your own pull request"). Reviewer switches to `gh pr comment` and writes a clear **APPROVE / BLOCK + file:line** verdict; Orchestrator reads that comment as the verdict. Note that `reviewDecision` will be **empty** even after review, so do not treat that as "not reviewed". Still use CI checks (`gh pr checks`) + `mergeStateStatus=CLEAN` as merge conditions.

## tmux Handoff Contract

### Before Every Handoff

```sh
tmux capture-pane -t <pane> -p | tail -80
git status --short
```

Verify: TUI is running (not a shell), input area is empty, and no stale text remains.

### Sending A Prompt

Clear stale input -> send prompt -> send Enter **separately** to submit -> capture again to confirm it entered the transcript and the receiver started processing. The first Enter may only confirm multi-line input; a second Enter is often required. If the prompt remains in the input area, send Enter again and recapture before walking away.

### Selection Prompt = Dangerous

If capture shows the agent at `Enter to select · ↑/↓ to navigate · Esc to cancel`:

- **Do not** send text directly; it may register as a selection (often the default/Recommended option) and corrupt the answer.
- Correct path: send `Escape` -> verify clean input -> send text.
- To relay a user choice into a selection prompt: send exact `Down`/`Up` x N, then `Enter`, then capture immediately to verify the right choice was recorded.

### Context & Model Switching

- `/clear` before unrelated tasks or when changing phase. **Do not clear in the middle of a goal.**
- Map model to task: opus/strong for brainstorm/scope/plan, sonnet/fast for implement and fix loop.

### Waiting For An Agent

Auto-poll with background bash; do not make the user poll manually. **But polling must have checkpoints and must not wait blindly.**

**Principle:** split polling into short windows (~3 minutes). After each window, return to Orchestrator to inspect instead of blocking for 20-30 minutes. The loop exits on **three conditions**: (a) agent idle, (b) detected **stuck-state** requiring intervention, or (c) window elapsed (return, inspect, then decide whether to poll again).

```sh
# One poll window ~= 3 minutes. exit_reason tells Orchestrator what to do.
stable=0; iters=0; max=36   # 36*5s = 3 minutes
exit_reason=window_elapsed
while [ $iters -lt $max ]; do
  out=$(tmux capture-pane -t <pane> -p 2>/dev/null)
  # stuck-state: selection/permission prompt, shell leak, obvious error -> stop immediately
  if echo "$out" | grep -qE 'Enter to select|Do you want to proceed|❯ 1\.|Yes, |No, go back|\$ $|command not found|Error:'; then
    exit_reason=needs_attention; break
  fi
  # busy while spinner remains; idle after spinner absent >=40s (8 loops)
  # Pattern MUST catch elapsed format "Verb... (3m 0s · ↓ 1.2k tokens)" using "... *\(" + token counters.
  # Do NOT use ✻/✶ as busy signal: they persist on completed summary lines ("✻ Churned for 4m") and cause false-busy loops.
  if echo "$out" | grep -qE '… *\([0-9]+[smh]|esc to interrupt|◎ /goal active|↓ [0-9]|↑ [0-9]|· [0-9.]+k? tokens'; then stable=0; else stable=$((stable+1)); fi
  if [ $stable -ge 8 ]; then exit_reason=idle; break; fi
  iters=$((iters+1)); sleep 5
done
echo "EXIT=$exit_reason after ~$((iters*5))s"; tmux capture-pane -t <pane> -p | grep -n '[^[:space:]]' | tail -40
```

- `run_in_background: true`. Each window is ~3 minutes; when notified, read `EXIT`:
  - `idle` -> agent done, read report.
  - `needs_attention` -> selection/permission/shell-leak/error -> inspect + recover (see Recovery). Do not keep polling blindly.
  - `window_elapsed` -> still busy -> note progress and launch a new window (repeat up to N times).
- **Hard wall-clock budget:** set a total cap (for example 6 windows ~= 18 minutes for a large task). If still busy at budget, capture pane, decide whether real progress is happening (tokens/spinner changed), and if it appears stuck -> `C-c` + inspect. Do not leave it running forever.
- Pattern `… \([0-9]+[smh]` matches Claude Code spinner; other runtimes may need adaptation. The stuck-state grep is also TUI-specific.

### Audit

```sh
tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} cmd=#{pane_current_command} cwd=#{pane_current_path}' | rg 'agent-tech|agent-lead'
```

## Review & Smoke

Larry chooses review type by target. Orchestrator states target + intent only and does not prescribe the tool.

| Type | Target |
|---|---|
| Code review | diff/commit |
| PR review | full PR scope |
| Spec/design review | architecture, risk, missing case |
| Security review | auth, data exposure, injection |

**Smoke scenarios are designed in the Spec phase**, not during execution. Tom proposes scenarios with the spec; user/Larry approves them; Larry executes and reports pass/fail with evidence. Scenario gaps found during execution are logged back into the spec for next time. Smoke verifies end-to-end behavior (UI, CLI, API, IPC, data flow), not just UI, and not unit behavior.

**Driving the Electron app:** drive the running `pnpm dev` instance through CDP. Read [`agent-browser-smoke.md`](agent-browser-smoke.md) first (connect port 49222, **do not** launch a second instance).

**General rules for every review/smoke:**

- Scope is specific (commit, PR, file, spec).
- Larry verdict: approve / block / needs-discussion. **Does not edit files.**
- Lead finding -> Tom fixes with scoped commit -> Larry only re-reviews that follow-up.
- "No verdict" (Larry did not inspect) -> rerun from the beginning.
- Docs drift found during review -> fix before close.
- **Reviewer does not self-poll for fixes and does not self-drive the loop.** When review finds an issue, report verdict (`BLOCK` + `file:line`) and **stop**. Orchestrator controls handoff: receive finding -> dispatch Tom fix -> verify fix -> re-engage reviewer to re-smoke. (Lesson G3c: Larry opened a `while` loop waiting for Tom commit -> context ran out + overstepped into **editing production files**. It was reverted and the branch survived, but the role boundary was violated. Reviewer self-drive risks role drift + context burn. One review = one verdict, one pass.)

## Ownership

- One actor per file at a time.
- Larry does not edit files unless the user explicitly asks.
- Orchestrator does not implement product code. If an agent fails, restore the agent first (clear, restart, split task, switch model). If still stuck, ask the user.

## Recovery

**Stale/wrong prompt:** `C-c` -> capture -> if still broken, `C-c C-c` to exit TUI -> restart -> verify empty input.

**Tom degraded** (stale behavior, wrong scope, corrupt context):

1. `C-c`, capture pane.
2. `/clear` or restart TUI.
3. Re-send a smaller task with clear stop condition.
4. If it repeats, ask the user. **Do not self-implement.**

**Shell leak:** agent dropped to shell -> restart TUI with uninterrupted flag so permission prompts do not stall:

- Claude Code: `claude --dangerously-skip-permissions`
- Codex: `codex --yolo`
- OpenCode / agy: `agy --dangerously-skip-permissions`

Do not trust process name; inspect the input area.

## Maintenance

Each orchestration failure should add one rule to this playbook. Playbook principle: **principles over recipes, references over duplication**. If a rule applies only once, do not add it.
