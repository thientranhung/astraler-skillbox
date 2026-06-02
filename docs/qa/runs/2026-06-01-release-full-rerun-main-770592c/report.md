# QA Run Report: 2026-06-01-release-full-rerun-main-770592c

- Date: 2026-06-01
- Scope: release rerun after PR #17 and PR #18
- Profile: release-full
- App mode: dev-electron, packaged smoke where feasible
- Target version/commit: 0.1.0 / 770592c
- Operator: codex-agent

## Verdict

NO-GO for a clean release gate.

No product FAIL was found in this rerun, and the previous PR #17/#18 blockers
now pass on `main` at `770592c`. The release still cannot be called clean GO
because 8 T0 cases remain BLOCKED by missing dedicated harness/product-surface
coverage, and signed/notarized release verification remains NEEDS_HUMAN.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 16 | 0 | 8 | 0 | 0 |
| T1 | 25 | 0 | 9 | 1 | 0 |
| T2 | 4 | 0 | 5 | 0 | 0 |

Automated gates:

- PASS: `core-go go test ./...`
- PASS: `apps/desktop pnpm typecheck`
- PASS: `apps/desktop pnpm test` (54 files, 523 tests)
- PASS: `apps/desktop pnpm check:contracts-drift`

Packaged release smoke:

- PASS: unsigned DMG build
- PASS: mounted-DMG launch smoke with bundled sidecar and temp DB
- PASS: manifest + SHA256SUMS generation/check
- NEEDS_HUMAN: customer-ready signing/notarization; Apple credentials are absent
  and owner has deferred this gate.

## Findings

- Fixed blocker verified: `TC-DASH-003`, `TC-DB-003`, and `TC-SETTINGS-002`
  pass after PR #17. Reset clears metadata, preserves fixture folders, and
  returns to Setup with no blank-window regression.
- Fixed blocker verified: `TC-SETUP-002` passes after PR #17. Missing active
  Skill Host now reports `missing` in Dashboard and Settings instead of stale
  `active`.
- Fixed blocker verified: `TC-GLOBAL-004` passes after PR #18. Disabled Claude
  global provider is persisted as `disabled` with zero entries; generic_agents
  remains active/current.
- Safety-critical filesystem checks passed where exercised: remove project is
  metadata-only; remove skill preserves host source; read-only install fails
  without DB success; conflict install fails without false current state.
- Global Plugins too-large fixture now reports `too_large` and leaves the
  oversized file hash unchanged.
- Remaining BLOCKED T0 cases need purpose-built harness coverage, not product
  bug fixes yet: host path-escape install, mutated managed symlink removal,
  global external symlink, invalid migration DB launch, crash/stale operation
  recovery, operation stale-state restart, switch install mode, and accessible
  destructive confirmations.

## Cases Run

Full machine-readable results are in `results.jsonl`.

Key evidence:

- `evidence/rpc-release-full-rerun-result.json`
- `evidence/rpc-install-remove-readonly-rerun.json`
- `evidence/rpc-second-pass-remove-large-route.json`
- `evidence/rpc-too-large-plugin.json`
- `evidence/db-state.txt`
- `evidence/reset-before-after.json`
- `evidence/reset-after-db.txt`
- `evidence/fs-fixture-tree.txt`
- `evidence/release-mac-dmg-smoke.txt`
- `evidence/release-mac-check.txt`
- `evidence/release-mac-verify-adhoc.txt`

## Environment

- QA home: `<repo-root>/docs/qa/runs/2026-06-01-release-full-rerun-main-770592c/fixtures/homes/plugin-home`
- QA DB: `<repo-root>/docs/qa/runs/2026-06-01-release-full-rerun-main-770592c/fixtures/homes/plugin-home/qa.db`
- QA host A: `<repo-root>/docs/qa/runs/2026-06-01-release-full-rerun-main-770592c/fixtures/skill-host-a`
- QA host B: `<repo-root>/docs/qa/runs/2026-06-01-release-full-rerun-main-770592c/fixtures/skill-host-b`
- CDP port: 49222

## Follow-Up

- Build focused harnesses for the 8 blocked T0 cases, then run a T0-only
  release gate before calling clean GO.
- Keep Apple signing/notarization as owner-deferred until credentials are
  available.
- Do not treat the current NO-GO as a new product regression: this rerun found
  zero FAIL cases, but clean release confidence still requires closing the T0
  harness gaps.
