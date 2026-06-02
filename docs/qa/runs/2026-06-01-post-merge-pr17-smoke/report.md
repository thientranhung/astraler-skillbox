# Post-Merge Smoke Report: PR #17

Run id: `2026-06-01-post-merge-pr17-smoke`
Date: 2026-06-01
Target: `main` at `a69149f`
Scope: post-merge regression smoke for PR #17

## Verdict

GO for post-merge smoke scope.

PR #17 was merged into `main` as `a69149f`. The targeted reset and
missing-host regressions remain fixed on merged `main`.

## Selected Cases

- `TC-SETTINGS-002`
- `TC-DASH-003`
- `TC-DB-003`
- `TC-SETUP-002`

## Automated Gates

- PASS: `pnpm check:contracts-drift`
- PASS: `go test ./internal/services/... ./internal/rpc/handlers/...`
- PASS: `pnpm test -- use-reset-all` (`54` files, `523` tests)

## Findings

- `TC-SETUP-002`: PASS. Moving the configured QA host path away made Dashboard
  and Settings report `status: missing` with visible recovery guidance.
- `TC-SETTINGS-002`: PASS. Reset All Data was executed through the Settings
  Danger Zone UI; the app returned to first-run Setup, metadata was cleared,
  and QA host/project fixtures were preserved.
- `TC-DASH-003`: PASS. After reset, the renderer showed Setup instead of a
  blank window; dashboard core output returned zero counts and no active host.
- `TC-DB-003`: PASS. Metadata tables were cleared and `schema_migrations`
  remained at version `23`, `dirty=0`.

## Evidence

Evidence is stored under `evidence/`.

Key files:

- `evidence/missing-host-dashboard.png`
- `evidence/missing-host-settings.png`
- `evidence/reset-after.png`
- `evidence/reset-after-db.txt`
- `evidence/agent-browser-final-snapshot.txt`
