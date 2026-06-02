# QA Governance

This document defines the durable rules for Astraler Skillbox QA. It is a policy
document, not a run log. Specific dates, PRs, case outcomes, waivers, and
evidence belong in `docs/qa/runs/<run-id>/`.

## Principles

- QA verifies product safety, not just test execution.
- T0 cases are release-critical because they cover data integrity, filesystem
  safety, destructive behavior, and DB/filesystem consistency.
- Product scope determines whether a case is a blocker. QA records that
  decision; it does not silently redefine the product.
- Real user project folders, provider folders, plugin folders, and host folders
  are never destructive QA targets unless a case explicitly allows opt-in and
  the owner approved the exact target.
- Source fixtures are immutable templates. A run may mutate only run-local
  fixture copies or explicit temporary paths.
- Full QA establishes release confidence; delta QA protects that confidence
  after each change.

## Result Semantics

Use only the statuses defined in `schema.md`.

| Status | Meaning |
|---|---|
| `PASS` | The expected behavior was verified with sufficient evidence for the case tier. |
| `FAIL` | The product violated the expected behavior, invariant, or safety rule. |
| `BLOCKED` | The case cannot be completed because the harness, fixture, setup, or product state prevents valid execution. |
| `NEEDS_HUMAN` | The next step requires human judgment, credentials, a real artifact, or explicit approval of an opt-in target. |
| `SKIPPED` | The case is outside the approved run scope or current product phase, and the reason is recorded. |

A waiver is not a separate status. Record the result status plus waiver metadata
in `results.jsonl` and explain the risk in `report.md`. A waiver must include:

- owner approval;
- scope of the waiver;
- why the residual risk is acceptable;
- mitigation, documentation, or follow-up tracking;
- whether the waiver applies only to the current run or to a documented phase.

## Scope And Phase

Every release run must distinguish current release scope from future scope.

- Current-scope T0 failures block release unless explicitly waived by the owner.
- Future-phase cases should be `SKIPPED` with a phase/defer reason, not counted
  as current-release blockers.
- Manual-only cases may be `NEEDS_HUMAN` until the required artifact, credential,
  platform, or human approval exists.
- If product scope is unclear, stop and clarify the source-of-truth docs before
  forcing a result.

## T0 Handling

T0 cases must not be closed early just because automation is difficult. Before a
T0 case can be marked `BLOCKED`, the executor must try the available safe paths:

- automated gate or unit/contract test;
- RPC or sidecar harness;
- dev Electron via CDP;
- out-of-band DB/filesystem inspection;
- source or contract inspection when UI automation cannot reach the state;
- human/owner input when the case explicitly requires it.

If the case remains blocked, the report must say what prevented execution and
what QA bank, fixture, harness, or product change would unblock it.

## Evidence Standard

Evidence depth scales with tier and risk.

| Tier | Minimum evidence |
|---|---|
| T0 | Independent evidence such as DB query, filesystem state, RPC output, source/contract check, and screenshot when UI is involved. |
| T1 | Screenshot or UI snapshot plus at least one independent persisted-state or RPC check when state changes. |
| T2/T3 | Screenshot, log, or concise note sufficient to reproduce the observation. |

Evidence files are collected under the active run's `evidence/` folder. They are
local run artifacts by default and are ignored by git to prevent large screenshots,
logs, caches, and transient outputs from entering source control. The committed
`report.md` and `results.jsonl` must summarize the decisive evidence clearly
enough for reviewers to understand the result. Commit raw evidence only when it
is small, intentionally curated, and needed as durable review material.

Generated fixture copies, temporary homes, caches, module downloads, and
outside-target sandboxes are disposable run artifacts, not canonical evidence.

## Anti-Hallucination Checklist

Before issuing a QA result or clean GO verdict, verify:

- The selected case IDs, tags, tier, and run scope match the change under test.
- The run folder exists and contains the expected `run-plan.yaml`,
  `results.jsonl`, `report.md`, and evidence paths.
- Each `PASS` has enough evidence for its tier; T0/T1 state changes include an
  independent check when available.
- `FAIL`, `BLOCKED`, `NEEDS_HUMAN`, and `SKIPPED` use the status meanings in
  this document and `schema.md`.
- Destructive or filesystem-writing cases used run-local fixture copies unless
  the case explicitly allowed opt-in real targets and the owner approved the
  exact target.
- The report separates gate results, case results, waivers, skipped/future
  scope, residual risk, and final GO/NO-GO.
- Claims in the QA verdict are grounded in case output, screenshots, logs, DB/RPC
  checks, filesystem checks, source inspection, or contract inspection.

## Full, Delta, And Clean GO

- First release and release-candidate validation use the `release-full` profile.
- Changes after a full run use delta QA selected by touched screens, tags,
  invariants, and risk.
- Bug fixes must select or add a regression case before closing.
- A release, T0, filesystem, schema/RPC, or cross-layer fix is not clean GO until
  the required delta or release QA passes on the merge commit.
- The final report must separate gate results, case results, owner waivers,
  skipped future scope, residual risk, and the final GO/NO-GO verdict.

## QA Bank Maintenance

When execution finds ambiguity, unsafe setup, missing evidence, or missing
coverage, update the QA bank instead of encoding that lesson only in a run
report.

Update the smallest canonical surface:

- `cases/` for executable behavior and expected results;
- `invariants.yaml` for safety and consistency rules that recur;
- `schema.md` for result/run-plan metadata;
- `profiles/` for selection policy;
- this file for durable QA operating rules.
- `runs/<run-id>/run-plan.yaml`, `results.jsonl`, and `report.md` for durable run
  summaries when a run should be preserved in git.

Do not add one-off historical notes here. Keep run-specific details in the run
folder.
