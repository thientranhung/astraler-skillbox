# Delta QA Report: PR #17 Reset + Missing Host Fix

Run id: `2026-06-01-delta-pr17`
Date: 2026-06-01
Target: `fix/reset-blank-missing-host` at `21c8dba`
Scope: regression/delta for release-full blockers found in `2026-06-01-release-full`

## Verdict

GO for PR #17 delta scope.

The two release-full blockers targeted by PR #17 are fixed in this delta run:

- Reset All Data no longer leaves a blank Electron window.
- Missing active Skill Host Folder no longer appears as an active healthy host.

## Selected Cases

- `TC-SETTINGS-002`
- `TC-DASH-003`
- `TC-DB-003`
- `TC-SETUP-002`
- `TC-SETUP-001`
- `TC-DASH-001`
- `TC-SKILL-001`

## Automated Gates

- PASS: `go test ./...`
- PASS: `pnpm typecheck`
- PASS: `pnpm test` (`54` files, `523` tests)
- PASS: `pnpm check:contracts-drift`

## Findings

- `TC-SETTINGS-002`: PASS. Reset was executed through the Settings Danger Zone UI, metadata was cleared, and QA fixture folders remained on disk.
- `TC-DASH-003`: PASS. After reset the UI showed first-run Setup, not a blank window; Dashboard core output returned zero counts and no active host.
- `TC-DB-003`: PASS. Metadata tables were cleared and `schema_migrations` remained at version `23`, `dirty=0`.
- `TC-SETUP-002`: PASS. After the configured host folder was moved, Dashboard and Settings showed `status: missing` with visible recovery guidance.
- `TC-SETUP-001`, `TC-DASH-001`, `TC-SKILL-001`: PASS. Nearby setup/dashboard/skills smoke stayed green before destructive reset.

Residual note: Skills Library still displays last-scan cached skill rows under the missing-host warning. This delta accepts that behavior because the UI clearly labels the host as missing and provides recovery guidance; if the product intent is to suppress cached rows entirely, add a separate T2/T3 UX case.

## Evidence

Evidence is stored under `evidence/`.

Key files:

- `evidence/reset-after.png`
- `evidence/reset-after-ui-text.txt`
- `evidence/reset-after-db.txt`
- `evidence/missing-host-dashboard.png`
- `evidence/missing-host-settings.png`
- `evidence/missing-host-core.json`
