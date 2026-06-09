<!--
Template: file-backed /goal for Astraler Skillbox.
Save as `.scratch/goal-<slice>-<phase>.md`, then send one tmux line:
  /goal read `.scratch/goal-<slice>-<phase>.md` and proceed.

Origin: the user's original `/goal` template (Obsidian References).
Adapted for this repo's Tom/Larry phase-gated workflow.

Read anatomy + when/anti-patterns in docs/playbooks/agent-orchestration.md, section `/goal`.
Do not use /goal for tiny fixes, exploratory reading, review comments, or tasks without clear success criteria.
-->

/goal [THE FINAL OUTCOME: what "done" looks like in one line]

You are **Tom** (or **Larry**). Phase = <Brainstorm | Spec | Plan | Implement | Review | Fix>.

-- CONTEXT --

- Project: Astraler Skillbox
- Slice / Feature: <slice name>
- Stack: <only relevant parts: Electron + React renderer, Go core, SQLite, etc.>
- Current state: <where the slice stands: nothing yet / brainstorm exists / spec exists>
- Working dir: <branch + relevant subpath, for example `feat/dashboard-plugins-metric` / `apps/desktop/renderer/src/screens`>
- Inputs:
  - Spec: <path if any>
  - Plan: <path if any>
  - Brainstorm note: <path if any>
  - Related doc: <docs/...>
- Constraints:
  - MUST: <required work>
  - MUST NOT: <negative scope: do not touch module X, do not refactor Y, do not change contract Z>
- Audience: <product user / team / Larry review / orchestrator handoff>

-- SUCCESS CRITERIA (ALL MUST BE TRUE) --

1. <Specific measurable outcome>
2. <Specific measurable outcome>
3. <Specific measurable outcome>
4. Final deliverable runs/builds without errors (tests pass if in scope)
5. Evidence is shown clearly (diff, log, screenshot, scenario pass/fail)
6. Docs are updated per `docs/playbooks/documentation.md` if a concept changed

-- OPERATING RULES: NON-NEGOTIABLE --

1. PLAN FIRST. Output a task list / outline before writing code or the spec body.
2. RESPECT PHASE GATE. If phase = Spec, stop after the spec and do not implement. If phase = Plan, stop after the plan and do not code. If phase = Implement, stop before PR/merge and wait for approval.
3. SELF-VERIFY. After each step, run relevant test/typecheck/lint, capture output, and confirm pass.
4. DEBUG YOURSELF. If something fails, diagnose and fix within the same phase. Do not hand back to Orchestrator unless truly blocked.
5. USE EVERY TOOL. Read code, grep, run commands, use MCP, use fixtures. Do not guess.
6. NO PLACEHOLDERS. No TODOs in commits, no stubs, no "// implement later".
7. PROGRESS LOG. Track completed / in-flight / decisions / blockers in the phase final response.
8. STAY ON SCOPE. If you find off-spec work, note it and do not expand scope. Tell Orchestrator if critical.
9. IF BLOCKED. Log the specific blocker, continue all parallelizable branches, and do not idle.
10. CHECK SUCCESS BEFORE STOPPING. Re-read SUCCESS CRITERIA and confirm every line is met; if not, fix before reporting done.
11. CAPTURE HARNESS LESSONS. If an operational failure recurs, note a candidate lesson for `.scratch/YYYY-MM-DD-harness-retro.md`; do not rewrite governance directly.

-- QUALITY BAR --

- Code: clean, typed, follows conventions (`AGENTS.md`, Conventions section).
- Architecture: respects hard rules in `docs/10-technical-architecture.md` (renderer / main / Go boundaries).
- Output: can pass Larry code review + smoke tests.
- Docs: new concept / new RPC / new migration -> update canonical docs according to the `documentation.md` map.
- Commit: short imperative message, with `DOC-VERIFIED: <reason>` trailer if the push range touches `main`.

-- FINAL DELIVERABLE --

- [OK] Confirm every Success Criterion is met.
- [FILE] List every created/modified file (relative path).
- [RUN] Exact verification commands (test command, smoke scenario, screenshot path).
- [PROOF] Diff / log output / screenshot / scenario result.
- [LOG] Decisions made, trade-offs chosen, important context.
- [WARN] Known limitations, follow-up tasks, deferred scope.
- [LEARN] Candidate harness lesson, or `none`.
- [STOP] Next phase + assignee (for example "Larry reviews spec next").
