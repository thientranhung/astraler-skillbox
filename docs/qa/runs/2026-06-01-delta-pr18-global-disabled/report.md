# Delta QA Report: PR #18 Global Disabled Provider

Run id: `2026-06-01-delta-pr18-global-disabled`
Target: `fix/global-scan-respects-disabled-provider` at `576c52b`

## Verdict

GO for PR #18 delta.

## Findings

- PASS: `TC-GLOBAL-004` — after disabling Claude, `global.scan` persisted the
  Claude global location as `disabled`, cleared stale Claude global install
  rows, and `global.list` returned zero Claude entries. Shared Agent Skills
  stayed `active` with `global-generic` still `current`.
- PASS: `TC-PROVIDER-004` adjacency check — disabling Claude did not delete the
  QA HOME `.claude` folder or the QA project `.claude` folder.
- SKIPPED: `TC-GLOBAL-002` — selected as adjacent global-skills coverage, but not
  executed in this targeted PR #18 delta.

## Evidence

- `evidence/TC-GLOBAL-004-cdp-result.json`
- `evidence/TC-GLOBAL-004-disabled-global-list.json`
- `evidence/TC-GLOBAL-004-disabled-ui.png`
- `evidence/TC-GLOBAL-004-disabled-ui.txt`
- `evidence/TC-GLOBAL-004-db-disabled.txt`
- `evidence/TC-GLOBAL-004-fs-after.txt`

## Automated Gates

- PASS: `(cd core-go && go test ./...)`
- PASS: `(cd apps/desktop && pnpm check:contracts-drift)`
