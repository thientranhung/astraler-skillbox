# Clean GO Triage Report

Run id: `2026-06-01-clean-go-triage`
Target: `main` at `a69149f`

## Verdict

NO-GO for clean release gate.

This run reduced several previous BLOCKED cases to PASS, but found one real
remaining release-readiness blocker: Global Skills scan/list ignores disabled
provider state for Claude.

## Findings

- PASS: `TC-FS-006` — project removal is metadata-only; project folder and
  `.claude` folder remain on disk.
- PASS: `TC-PROVIDER-004` — disabling Claude does not delete the QA project
  provider folder.
- PASS: `TC-ERROR-002` — nonexistent project path is rejected with
  `validation_error/path_not_found` and no new project row is added.
- PASS: `TC-SETTINGS-001` — changing Skill Host Folder A to B updates the
  source of truth and preserves both host folders. The existing-install branch
  was not exercised because the fixture project lacked `.claude/skills` before
  install; no relink/delete occurred.
- PASS: `TC-GLOBAL-002` — after a successful global scan, moving Claude global
  skills path reports Claude as `missing` and removes stale Claude current
  entries.
- FAIL: `TC-GLOBAL-004` — `provider.setEnabled` saves Claude
  `isEnabled=false`, but `global.scan` and `global.list` still return Claude as
  `active` with `global-claude` current entry. Global Skills does not respect
  disabled provider state.

## Remaining Clean-GO Issues

- Fix `TC-GLOBAL-004` in product code or revise the invariant if disabled
  providers are intentionally still scanned globally.
- Continue triage for remaining hardening/harness cases that were not executed
  in this run.
- `TC-RELEASE-005` remains owner/human-gated for signed/notarized artifact
  verification.
