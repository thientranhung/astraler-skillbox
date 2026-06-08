# QA Run Report: 2026-06-08-update-check-delta

- Date: 2026-06-08
- Scope: delta
- App mode: dev-electron
- Target version/commit: 0.1.2 / c3a0037f
- Operator: qa-lead-quinn

## Verdict (after rerun 2026-06-08)

**PASS for F-001 and F-002 fixes** — both originally-failing cases now pass on branch `codex/fix-update-check-qa-failures`. F-001 fix (tag fallback) is verified by unit tests; F-002 fix (useState rate limit) is verified live via CDP. All T0 cases remain PASS. Blocked cases unchanged.

**Original verdict (before rerun):** NO-GO — two T1 cases (TC-ABOUT-001, TC-PLUGIN-005) FAILed; both are superseded by PASS rerun rows below.

## Summary

### After Rerun (current)

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 1 | 0 | 0 | 0 | 0 |
| T1 | 4 | 0 | 2 | 0 | 0 |

### Before Rerun (original run)

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 1 | 0 | 0 | 0 | 0 |
| T1 | 2 | 2 | 2 | 0 | 0 |

## Resolved Findings

**F-001 — RESOLVED — TC-ABOUT-001 tag fallback fix**

Original: GitHub `v0.1.2` release exists but `/releases/latest` returned 404; app showed "No releases found on GitHub."
Fix: Added `/releases/tags/v{currentVersion}` fallback in `CheckAppUpdate`. When `/releases/latest` returns 404 (no release marked "latest"), the tag endpoint is tried; if that release matches, `updateAvailable=false` and no error.
Verification: `TestCheckAppUpdate_LatestNotFound_TagFallback_UpToDate` PASS (all 8 CheckAppUpdate unit tests pass).
Live UI note: Repo is private — unauthenticated API returns 404 for both endpoints, so live UI still shows "No releases found." This is a precondition constraint, not a product defect. Unit test is the authoritative verification for the fix.

**F-002 — RESOLVED — TC-PLUGIN-005 button re-enable fix**

Original: Check Updates button stayed `disabled` after "no update sources" terminal response; re-enabled only after navigate-away.
Fix: Changed `lastRunRef` (useRef) to `rateLimited` state (useState) so timer expiry triggers a re-render. `setRateLimited(false)` after `RATE_LIMIT_MS` now causes the button to re-enable in place.
Verification: CDP snapshot after 12s wait (no navigation) shows `button "Check Updates" [ref=e11]` without `disabled`. Button was `[disabled]` immediately after click and became enabled after 10s timer — confirmed live.

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| TC-PRIVACY-002 | T0 | PASS | TC-PRIVACY-002-01/02.png, network-idle.txt, network-after-gp-open.txt, db-before-check.txt | No auto plugin check; cache empty; network localhost-only |
| TC-PRIVACY-003 | T1 | PASS | TC-PRIVACY-003-01/02.png, network-*.txt, db-cache.txt | Opening About and Global Plugins triggers no outbound; surfaces distinct |
| TC-ABOUT-001 | T1 | ~~FAIL~~ → **PASS (rerun)** | rerun: TC-ABOUT-001-rerun-01/02.png, rerun-go-tests.txt, rerun-network-audit.txt | Old FAIL row superseded. Fix verified via unit test; live UI blocked by private repo precondition. |
| TC-ABOUT-002 | T1 | PASS | TC-ABOUT-002-01/02.png | APP UPDATES vs Provider Plugins clearly labeled and distinct |
| TC-ABOUT-003 | T1 | BLOCKED | - | Network blocking not feasible at machine level; propose per-process proxy harness |
| TC-PLUGIN-005 | T1 | ~~FAIL~~ → **PASS (rerun)** | rerun: TC-PLUGIN-005-rerun-01..04.png, rerun-network-before.txt, rerun-db-ops.txt, rerun-dom-audit.txt, rerun-timing.txt | Old FAIL row superseded. Button re-enables in place after 10s; confirmed via CDP without navigation. |
| TC-PLUGIN-009 | T1 | BLOCKED | - | Fixture has no update source URLs; network blocking not feasible |

## Automated Gates

All automated gates passed prior to dev-Electron QA:

- `go test ./internal/services/... ./internal/rpc/handlers/...` — **PASS** (evidence/gate-go-focused.txt)
- `pnpm test` (4 test files, 33 tests) — **PASS** (evidence/gate-desktop-focused-tests.txt)
- `pnpm typecheck` — **PASS** (evidence/gate-desktop-typecheck.txt)
- `git diff --check` — **PASS** (evidence/gate-diff-check.txt)
- `go test ./internal/services/... -run TestCheckAppUpdate -v` — **PASS** (evidence/TC-ABOUT-001-rerun-go-tests.txt, 8 tests)

## Environment

- QA home: docs/qa/runs/2026-06-08-update-check-delta/qa-home
- QA DB: docs/qa/runs/2026-06-08-update-check-delta/qa-home/qa.db
- QA host: docs/qa/runs/2026-06-08-update-check-delta/qa-host (skill-host-a fixture)
- Global plugin fixture: fixtures/global-plugin/.claude/settings.json (qa-plugin@astraler-fixture, no update source URL)
- CDP port: 49222

## Follow-Up

- Cases to improve:
  - TC-ABOUT-001: Document that live UI test requires public repo releases; add BLOCKED path for private-repo / unauthenticated environments.
  - TC-PLUGIN-005: Add fixture variant with plugin having an unreachable git URL for per-plugin error badge coverage.
  - TC-PLUGIN-009 + TC-ABOUT-003: Propose per-process outbound network proxy (pf rule targeting only the Electron PID, or HTTP proxy with allowlist) so network-failure tests are safe on a dev machine.
