# QA Run Report: 2026-06-02-release-full-main-b065fd6

- Date: 2026-06-02
- Scope: release
- Profile: release-full
- App mode: dev Electron first, packaged smoke after artifact/human gates
- Target version/commit: 0.1.2 / b065fd6ba7976d3db0e6de03abb276bdaf1b35a5
- Operator: agent
- Status: PREPARED, not executed

## Verdict

NOT RUN

This folder prepares the next release-full QA execution after the P0/T0 and T1
case-bank expansion PRs were merged. It is not a GO/NO-GO verdict yet.

## Planned Scope

| Tier | Selected | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|---:|
| T0 | 38 | 0 | 0 | 0 | 0 | 0 |
| T1 | 58 | 0 | 0 | 0 | 0 | 0 |
| T2 | 9 | 0 | 0 | 0 | 0 | 0 |
| T3 | 0 | 0 | 0 | 0 | 0 | 0 |

Total selected cases: 105.

## Required Gates

- `(cd core-go && go test ./...)`
- `(cd apps/desktop && pnpm typecheck)`
- `(cd apps/desktop && pnpm test)`
- `(cd apps/desktop && pnpm check:contracts-drift)`

## Blocking Findings

- None yet. No QA cases have been executed in this run.

## Known Human Gates

- Signed/notarized release verification remains `NEEDS_HUMAN` until Apple
  credentials are available.
- Packaged artifact checks require the exact release artifact path before
  execution.
- Outbound network observations or manual update checks require explicit owner
  approval at execution time.
- `TC-SKILL-004` has an owner-accepted Phase 2 waiver for current release
  confidence; execute and record the actual status/residual risk rather than
  silently treating it as covered.

## Cases Run

No cases have been run yet. Append one result object per case to
`results.jsonl` during execution.

## Environment

- QA home: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/qa-home`
- QA DB: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/qa-home/qa.db`
- Fixture copy: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/fixtures`
- Evidence: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/evidence`
- CDP port: `49222`
- Real environment allowed: `false`

## Follow-Up

- Create `evidence/`, `fixtures/`, and `qa-home/` locally before execution.
- Copy `fixtures/qa/` into the run-local `fixtures/` folder before mutating any fixture state.
- Run automated gates first, then T0, then T1, then release-full T2, then packaged smoke when artifact inputs are ready.
- If a case is unclear or unsafe, mark it `BLOCKED` or `NEEDS_HUMAN` and update the QA bank with the smallest durable fix.
