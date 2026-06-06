# QA Run Report: 2026-06-06-tc-setup-006-scan-progress

- Date: 2026-06-06
- Scope: delta
- App mode: dev-electron
- Target version/commit: post-PR39 (5339ff80)
- Operator: agent (QA Lead Quinn)

## Verdict

**GO** - TC-SETUP-006 scanning sub-state now verifiable via CDP. Prior `NEEDS_HUMAN`
replaced with `PASS`.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 0 | 0 | 0 | 0 | 0 |
| T1 | 1 | 0 | 0 | 0 | 0 |

## Blocking Findings

- None.

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| TC-SETUP-006-scanning-substate | T1 | PASS | evidence/TC-SETUP-006-C-scanning-progress-T400ms.png, evidence/TC-SETUP-006-C-scanning-snapshot-T400ms.txt | Spinner captured at T+400ms during SKILLBOX_SCAN_DELAY_MS=1000 window |

## Automated Gate

| Gate | Result |
|---|---|
| core-go `./internal/testhooks/...` | PASS (8/8 tests) |
| core-go `./internal/services/...` | PASS (all tests including TestValidateSkillSegment, TestIsWithin) |

## Key Evidence

### Scan-in-progress indicator confirmed (T+400ms DOM snapshot)

```
button "Scanning..." [disabled, ref=e13]
listitem "Scanning skills..." [level=1, ref=e21] focusable
```

- Scan button text changes from "Scan" to "Scanning..." and becomes disabled during
  the scan operation.
- Toast notification "Scanning skills..." appears during the delay window.
- After completion: button reverts to "Scan" (enabled), toast updates to "Skills
  scanned".

### Env var verification

`SKILLBOX_SCAN_DELAY_MS=1000` confirmed present in Go sidecar process environment
via `ps eww -p 57289`. Sidecar inherits env from Electron main process as expected
(no explicit `env:` override in `spawn()` call in `manager.ts`).

### Capture methodology

Auto-scan uses `sessionAutoScanRegistry` to prevent re-triggering within a session.
Manual **Scan** button (bypasses registry guard) was used to reliably trigger a new
scan in the same session. Screenshots taken at T+150ms, T+400ms, T+700ms, T+950ms;
spinner confirmed at T+400ms and T+700ms.

## Environment

- QA home: `docs/qa/runs/2026-06-06-tc-setup-006-scan-progress/qa-home/`
- QA DB: `qa-home/qa.db`
- QA host: `qa-host/` (fixture copy of `fixtures/qa/skill-host-a`)
- CDP port: 49222
- SKILLBOX_SCAN_DELAY_MS: 1000

## Follow-Up

- Add regression cases: None required - TC-SETUP-006-scanning-substate is now
  the canonical test for this sub-state.
- Case update: Update `docs/qa/cases/setup-and-settings.yaml` TC-SETUP-006 notes
  to document that the manual Scan button must be used (not auto-scan) when
  retesting the scanning sub-state in the same app session.
- Observation: `sessionAutoScanRegistry` means auto-scan only fires once per host
  per app session. QA harness must either restart the app or use the manual Scan
  button for repeated scan-state captures.
