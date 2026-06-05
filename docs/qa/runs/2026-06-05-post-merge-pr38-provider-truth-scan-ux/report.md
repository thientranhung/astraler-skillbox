# QA Run Report: 2026-06-05-post-merge-pr38-provider-truth-scan-ux

- Date: 2026-06-05
- Scope: delta
- App mode: dev-electron automated checks
- Target version/commit: 0.1.2+main-43f4ecdc
- Operator: Codex + astraler-qa

## Verdict

CAUTION for the post-merge PR #38 delta.

The merged behavior has automated coverage and post-merge gates passed. This run
does not replace a dev Electron/CDP exploratory pass for the broader provider
tabs and scan-state surfaces.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| gate | 1 | 0 | 0 | 0 | 0 |
| T1 | 2 | 0 | 0 | 0 | 0 |
| T2 | 0 | 0 | 0 | 0 | 2 |

## Gate Results

- `(cd core-go && go test ./...)`: PASS
- `(cd apps/desktop && pnpm test --run src/screens/__tests__/project-detail-screen.test.tsx src/screens/__tests__/skills-library-screen.test.tsx)`: PASS, 2 files / 50 tests
- `(cd apps/desktop && pnpm typecheck)`: PASS
- `(cd apps/desktop && pnpm check:contracts-drift)`: PASS
- `git status --short --branch`: clean on `main...origin/main`

Evidence: `evidence/post-merge-gates.txt`

## Blocking Findings

- None.

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| GATE-AUTOMATED | gate | PASS | `evidence/post-merge-gates.txt` | Post-merge checks passed on `main` commit `43f4ecdc`. |
| TC-SETUP-006 | T1 | PASS | `evidence/post-merge-gates.txt` | Automated renderer coverage verifies Host Skills not-yet-scanned and no-skills-found states. |
| TC-PROJ-009 | T1 | PASS | `evidence/post-merge-gates.txt` | Automated renderer coverage verifies Project Detail never-scanned vs scanned-no-provider distinction. |
| TC-GLOBAL-008 | T2 | SKIPPED | — | Provider tabs were not changed by PR #38; keep for next provider-tabs exploratory pass. |
| TC-PLUGIN-010 | T2 | SKIPPED | — | Provider tabs were not changed by PR #38; keep for next provider-tabs exploratory pass. |

## Environment

- QA home: not created for automated-only post-merge artifact
- QA DB: not created for automated-only post-merge artifact
- QA host: not created for automated-only post-merge artifact
- QA project: not created for automated-only post-merge artifact
- CDP port: not used

## Follow-Up

- Run a dev Electron/CDP exploratory pass for provider tab behavior before treating `TC-GLOBAL-008` and `TC-PLUGIN-010` as current PASS.
- Start the next product batch on update-check UX and manual network terminal states.
