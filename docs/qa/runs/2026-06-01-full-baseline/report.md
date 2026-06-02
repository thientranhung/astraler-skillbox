# QA Run Report: 2026-06-01-full-baseline

- Date: 2026-06-01
- Scope: release
- App mode: dev-electron
- Target version/commit: 0.1.0 / 1a623d7
- Operator: codex-agent

## Verdict

NO-GO

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 3 | 1 | 1 | 0 | 0 |
| T1 | 11 | 1 | 1 | 0 | 0 |

## Blocking Findings

- `TC-SETTINGS-002` FAIL: Reset All Data clears metadata and preserves fixture folders, but relaunches into a detached blank Electron window instead of returning to setup/first-run.
- `TC-SETUP-002` FAIL: Missing configured Skill Host Folder is not surfaced; Dashboard continues to show status `active` and stale counts with no recovery warning.
- `TC-SKILL-004` BLOCKED: current UI/RPC surface has no switch install mode action or install mode parameter.
- `TC-PACKAGE-001` BLOCKED: packaged artifact smoke cannot run in a dev Electron baseline without a packaged app artifact and isolated app-data setup.

Automated gates all passed before UI QA:

- `(cd core-go && go test ./...)`
- `(cd apps/desktop && pnpm typecheck)`
- `(cd apps/desktop && pnpm test)` - 54 files / 523 tests passed
- `(cd apps/desktop && pnpm check:contracts-drift)`

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| TC-SKILL-002 | T0 | PASS | `evidence/TC-SKILL-002-project-detail-after-reload.png`, `evidence/TC-SKILL-002-readlink.txt`, `evidence/TC-SKILL-002-db.txt` | Symlink install agrees across UI/DB/FS. |
| TC-SKILL-003 | T0 | PASS | `evidence/TC-SKILL-003-after-remove.png`, `evidence/TC-SKILL-003-fs.txt`, `evidence/TC-SKILL-003-db.txt` | Project target removed; host source preserved. |
| TC-SKILL-004 | T0 | BLOCKED | `evidence/TC-SKILL-002-project-detail-after-reload.png` | No switch-mode product surface exists. |
| TC-SKILL-005 | T0 | PASS | `evidence/TC-SKILL-005-conflict.png`, `evidence/TC-SKILL-005-fs.txt`, `evidence/TC-SKILL-005-db.txt` | Conflict fails safely without a current managed install. |
| TC-SETTINGS-002 | T0 | FAIL | `evidence/TC-SETTINGS-002-before-reset-settings.png`, `evidence/TC-SETTINGS-002-after-reset.png`, `evidence/TC-SETTINGS-002-after-db.txt` | Reset relaunch leaves blank detached Electron window. |
| TC-SETUP-001 | T1 | PASS | `evidence/TC-SETUP-001-settings.png`, `evidence/TC-SETUP-001-db.txt`, `evidence/TC-SETUP-001-host-find.txt` | Folder choice used renderer RPC bridge because native dialog is not CDP-drivable. |
| TC-SETUP-002 | T1 | FAIL | `evidence/TC-SETUP-002-missing-host.png`, `evidence/TC-SETUP-002-ui-text.txt`, `evidence/TC-SETUP-002-fs.txt` | Missing host not detected in UI after reload. |
| TC-SETTINGS-001 | T1 | PASS | `evidence/TC-SETTINGS-001-host-b-skills.png`, `evidence/TC-SETTINGS-001-db.txt`, `evidence/TC-SETTINGS-001-fs.txt` | Active host changed to host B; host folders preserved. |
| TC-PROJ-001 | T1 | PASS | `evidence/TC-PROJ-001-projects.png`, `evidence/TC-PROJ-001-detail.png`, `evidence/TC-PROJ-001-db.txt` | Project/provider facts agree. |
| TC-PROJ-002 | T1 | PASS | `evidence/TC-PROJ-002-after-manual-scan.png`, `evidence/TC-PROJ-002-ops-db.txt` | Manual scan completed and timestamp updated. |
| TC-SKILL-001 | T1 | PASS | `evidence/TC-SKILL-001-library.png`, `evidence/TC-SKILL-001-db.txt`, `evidence/TC-SKILL-001-fs.txt` | Host skills match fixture contents. |
| TC-SKILL-006 | T1 | PASS | `evidence/TC-SKILL-006-broken.png`, `evidence/TC-SKILL-006-readlink.txt`, `evidence/TC-SKILL-006-db.txt` | Broken symlink warning surfaced. |
| TC-PLUGIN-001 | T1 | PASS | `evidence/TC-PLUGIN-001-global-rescanned.png`, `evidence/TC-PLUGIN-001-db-rescanned.txt` | Global plugin fixture scanned with versions. |
| TC-PLUGIN-002 | T1 | PASS | `evidence/TC-PLUGIN-002-before-settings.json`, `evidence/TC-PLUGIN-002-after-settings.json`, `evidence/TC-PLUGIN-002-roundtrip-diff.txt` | Toggle round trip stayed within QA settings. |
| TC-PLUGIN-003 | T1 | PASS | `evidence/TC-PLUGIN-003-project-detail.png`, `evidence/TC-PLUGIN-003-project-ui-text.txt`, `evidence/TC-PLUGIN-003-db.txt` | Project override/effective state displayed. |
| TC-PLUGIN-004 | T1 | PASS | `evidence/TC-PLUGIN-004-malformed.png`, `evidence/TC-PLUGIN-004-sha-diff.txt`, `evidence/TC-PLUGIN-004-db.txt` | Malformed fixture reported and unchanged. |
| TC-PLUGIN-005 | T1 | PASS | `evidence/TC-PLUGIN-005-no-manual-check.png`, `evidence/TC-PLUGIN-005-db.txt`, `evidence/TC-PLUGIN-005-lsof.txt` | No automatic update-check observed before manual action. |
| TC-PACKAGE-001 | T1 | BLOCKED | None | Requires packaged artifact smoke setup. |

## Environment

- QA home: `<repo-root>/docs/qa/runs/2026-06-01-full-baseline/qa-home`
- QA DB: `<repo-root>/docs/qa/runs/2026-06-01-full-baseline/qa-home/qa.db`
- QA host: `<repo-root>/docs/qa/runs/2026-06-01-full-baseline/qa-host-a`
- QA project: `<repo-root>/docs/qa/runs/2026-06-01-full-baseline/qa-project-claude`
- CDP port: 49222

## Follow-Up

- Add regression cases: reset relaunch blank window; missing configured host remains active/stale.
- Bugs/tasks to create: fix `app.resetAll` dev relaunch behavior; validate/refresh active host status on app load/dashboard/settings.
- Cases to improve: split dev baseline from packaged smoke or mark packaged case out-of-scope for dev runs; update `TC-SKILL-004` to match current install-mode product surface; document Claude plugin fixture shape (`enabledPlugins`) in QA setup guidance.
