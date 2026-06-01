# Project Governance

> Operating rules for every agent working in this repository. Read this before coding, reviewing, or opening a PR. This document defines the rules of play: phases, gates, ownership, and quality expectations.

## TL;DR

- Standard phase flow: **Brainstorm -> Branch? -> Spec -> Spec review -> User approval -> Plan -> Implement + Docs -> PR -> Review + Smoke/QA -> Merge.**
- **Do not skip user approval at Spec** for features, cross-layer work, behavior changes, schema, RPC, providers, filesystem, security, or process changes. Tiny low-risk work may compress phases under the rule below.
- Branch + PR is **required** for schema/migration or breaking changes. Use the Decision Rule table for everything else.
- **One actor per file.** Implementers code; Reviewers review only and do not edit files.
- Canonical docs are updated **in the same slice, before review**. Do not defer them.

## Roles

| Role | Responsibility |
|---|---|
| **Implementer** | Brainstorm, spec, plan, implement, open PRs, merge after approval, update docs. |
| **Reviewer** | Code/spec/security review and smoke testing. Verdict: approve / block / needs-discussion. **Does not edit production files.** |

> Slice = a thin cross-layer cut (UI -> service -> data). A slice runs all the way from Spec to Docs/QA evidence.

## `/goal` for Long Handoffs

`/goal` is a handoff capsule for long or stateful work, not a generic task command. Use it only when the outcome, scope, negative scope, success criteria, stop condition, and phase boundary are clear.

Do not use `/goal` for subagents, background threads, same-session worker prompts, initial brainstorming, exploratory reading, review comments, tiny fixes, or tasks without success criteria.

When you need to write a long handoff prompt, use the templates:

- [`templates/goal-file.md`](templates/goal-file.md): long handoff, rich context, many paths, or cross-phase work.
- [`templates/goal-inline.md`](templates/goal-inline.md): small same-phase follow-up with very clear scope.

File-backed `/goal` prompts must be written under `.scratch/` and follow the convention below.

## `.scratch/` Workspace

`.scratch/` is a temporary, gitignored workspace for:

- long prompts / file-backed `/goal` before sending them to a tmux agent;
- brainstorm notes, draft specs/plans, or AI sketches between user and agent;
- temporary run notes, context packs, checklists, or handoff capsules that are not source-of-truth yet.

Do not use `.scratch/` as canonical docs, approved specs, ADRs, official QA reports, or final decision storage. Once `.scratch/` content is approved, digest/sync it into the appropriate canonical document under `docs/`.

**Required naming:** every `.scratch/` file must use a date prefix so files sort well and remain traceable:

```text
YYYY-MM-DD-<topic>.md
YYYY-MM-DD-<topic>-<phase>.md
YYYY-MM-DD-goal-<slice>-<phase>.md
```

Examples:

```text
.scratch/2026-06-01-governance-project-review.md
.scratch/2026-06-01-goal-plugin-settings-spec.md
.scratch/2026-06-01-dashboard-slice-plan.md
```

Use lowercase kebab-case filenames. If the file is for handoff, its content should state phase, owner, input paths, constraints, success criteria, and stop condition.

## Workflow Skills / Superpowers

Governance does not depend on a specific workflow engine, but agents **must use available workflow skills** when the task matches a phase. If Superpowers is available, use it as the default workflow engine. If it is not available, the agent must reproduce the same outputs and gates with a normal prompt or the relevant playbook.

Minimum mapping:

| Need | Preferred workflow skill |
|---|---|
| Clarify intent, explore approaches, write spec/design | `brainstorming` |
| Convert an approved spec into an implementation plan | `writing-plans` |
| Execute a plan with independent tasks and clear ownership | `subagent-driven-development` |
| Execute a tightly coupled plan in one context | `executing-plans` |
| Request/reconcile review loop | `requesting-code-review`, `receiving-code-review` |
| Verify before reporting done | `verification-before-completion` |

Workflow skills may not bypass governance: user approval gates, ownership, docs/ADR, review, PR, QA, and `.scratch/` conventions still apply.

When delegating to a subagent/worker, the prompt must be bounded and include at least:

```text
Objective:
Owned files/modules:
Context:
Constraints:
Verification:
Expected final report:
```

Subagents/workers do not expand scope, change file ownership, or merge phases on their own. Final status should use one of:

- `DONE`
- `DONE_WITH_CONCERNS`
- `NEEDS_CONTEXT`
- `BLOCKED`

## Phase Gates

1. **Brainstorm & scope**: output includes **Risk Classification** (table below).
2. **Branch decision**: apply the Decision Rule and create a branch if needed, **before** Spec.
3. **Spec**: design includes smoke scenarios.
4. **Spec review**: Reviewer.
5. **User approval**: hard gate for all work outside the tiny low-risk exception.
6. **Implementation plan**.
7. **Implement + docs**: code/tests/docs in the same slice; self-verify before opening a PR.
8. **PR create**: if on a branch, push and create a PR. Do not combine create + merge.
9. **Review + smoke/QA**: Reviewer reviews on the PR when a PR exists. Findings -> `BLOCK`/request changes + `file:line`; Implementer fixes and pushes; Reviewer re-reviews. Repeat until clean -> approve.
10. **Merge**: Implementer merges after review/QA gates pass.

Tiny low-risk work may compress phases: docs-only/test-only/small UI polish, no behavior change, no schema/RPC/provider/filesystem/security touch, <50 LOC, direct-to-main. In that case, the user request counts as approval, but the agent must still provide a short plan, self-verify, and record why branch/spec review/QA bank were skipped.

## Branch & PR Workflow

### Risk Classification (closed at the end of brainstorm)

| Field | Value |
|---|---|
| Layers | UI / contract / Go / SQL / docs |
| Breaking change | yes / no |
| Schema/migration | yes / no |
| Est. LOC | <50 / 50-300 / >300 |
| Workflow | direct-to-main / branch + PR |

### Decision Rule

| Condition | Workflow |
|---|---|
| Schema/migration: yes **OR** Breaking change: yes | **MUST** branch + PR |
| Layers >= 3 **OR** Est. LOC > 300 | **SHOULD** branch + PR |
| Independent multi-slice parallel work | worktree per slice |
| UI-only / docs-only, < 50 LOC | OK direct-to-main |

- Branch naming: `<type>/<kebab-slug>` (for example `feat/dashboard-plugins-metric`) or a compatible provider/tooling prefix (for example `codex/<type>-<slug>`). PR target is always `main`.
- **Do not combine create + merge.** Push -> `gh pr create` -> PR review -> fix loop -> merge.
- Review must leave a real trace on the PR. `reviews: []` after merge means the gate was skipped.

> **Same-owner gotcha:** GitHub blocks `--approve` / `--request-changes` on PRs owned by the same account. The Reviewer posts the verdict with `gh pr comment` and clearly writes **APPROVE / BLOCK + file:line**. `reviewDecision` will be empty even after review; use `gh pr checks` + `mergeStateStatus=CLEAN` as merge conditions.

## Review & Smoke

| Review type | Target |
|---|---|
| Code review | diff / commit |
| PR review | full PR scope |
| Spec/design review | architecture, risk, missing cases |
| Security review | filesystem, network, auth, data exposure, injection, data loss |

**Verdict: the Reviewer returns exactly one of:** `APPROVE` / `BLOCK` / `NEEDS_DISCUSSION`. For `BLOCK` or `NEEDS_DISCUSSION`, include:

```text
Severity: P0 | P1 | P2 | P3
File/area:
Issue:
Why it matters:
Required fix:
Evidence:
Docs impact: none | required | missing
```

If a concept changed and docs are missing, the verdict is `BLOCK`.

**Review depth scales with risk:**

- Docs/test/low-risk small changes: light review or skip with a recorded reason.
- UI flow change: code review + smoke evidence/screenshot when useful.
- Cross-layer: spec/plan review **before** implementation, code review **after**.
- Schema / RPC / provider / filesystem write: deep review, consider security + smoke.

**Smoke principles:**

- **Smoke scenarios are designed in the Spec phase**, not during execution. The Implementer proposes them; user/Reviewer approves them; Reviewer executes and reports pass/fail with evidence. Gaps found during execution are logged back into the spec.
- Smoke verifies **end-to-end** behavior (UI, CLI, API, IPC, data flow), not just unit behavior.
- If smoke belongs to delta/smoke/release/regression QA, use the `astraler-qa` skill and QA bank: select cases/tags, create a run folder, write `run-plan.yaml`, append `results.jsonl`, write `report.md`, and store evidence under `docs/qa/runs/<run>/evidence/`.
- Drive the running `pnpm dev` Electron app through CDP. Read [`agent-browser-smoke.md`](agent-browser-smoke.md). **Do not** launch a second instance.
- When the Reviewer finds an issue, they report the verdict (`BLOCK` + `file:line`) and **stop**. They do not self-poll for fixes, self-drive the loop, or edit production files. One review = one verdict.
- "No verdict" (no inspection) means rerun from the beginning. Docs drift found during review must be fixed before close.

### QA Scope Mapping

| Change / risk | QA expectation |
|---|---|
| Schema, DB/filesystem consistency, destructive path, install/remove/switch behavior | Run impacted T0 cases + out-of-band DB/filesystem/screenshot evidence. |
| Core user journey or cross-screen truth | Run impacted T1 cases and related invariants. |
| Bug fix | Add or select a regression case, then run related cases. |
| Release readiness | Run release QA per [`../qa/README.md`](../qa/README.md). |
| Docs-only/test-only/tiny UI polish | QA bank can be skipped if behavior is unchanged; record the reason. |

## Docs & ADR

Concept changes require updating canonical docs **in the same slice and before review**. See the full map in [`documentation.md`](documentation.md). Check this trigger list:

> schema/migration; RPC method / notification; domain object; provider adapter; UI screen; user flow; edge/UX state; repeated implementation pattern; architecture/process boundary.

Use an **ADR** for major architecture, domain, tech stack, or process/workflow decisions. Do **not** create an ADR for local refactors, typos, formatting, small config changes, or tests-only changes.

**Commit trailer** when landing into `main` and touching a documented concept:

```text
DOC-VERIFIED: <reason>
```

## Quality Bar

- **Plan first**: do not code blind; have a plan/spec before editing.
- **Respect phase gates**: stop at the correct phase; do not collapse spec -> code -> PR while skipping review or user approval.
- **Self-verify**: build/test without errors; show evidence (diff, log, screenshot, smoke result) before declaring done.
- **No placeholders**: no fake TODOs, open stubs, or unfinished "implement later" code.
- **Stay on scope**: no unrelated refactors; include clear **MUST** and **MUST NOT** constraints.
- **Measurable success**: criteria must be verifiable, not "implement feature X".
- Follow code conventions and architecture hard rules ([`10-technical-architecture.md`](../10-technical-architecture.md)). Code must pass review. Include `DOC-VERIFIED` when docs are touched.

## Ownership

- **One actor per file at a time.**
- Reviewer does not edit files unless the user explicitly asks.
- If an agent fails or gets stuck, recover the agent first (clear, restart, split task, switch model). If still stuck, ask the user. Do not overstep the assigned role.

## Maintenance

Each governance failure should add one rule here. **Principles over recipes, references over duplication.** If a rule applies only once, do not add it.

## Related Operational Playbooks

Agent operation, handoff, runtime, tmux, or `/goal` playbooks must comply with this governance document.
