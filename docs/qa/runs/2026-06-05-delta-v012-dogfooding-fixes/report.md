# QA Run Report: 2026-06-05-delta-v012-dogfooding-fixes

- Date: 2026-06-05
- Scope: delta
- App mode: dev-electron
- Target version/commit: 0.1.2+main-4800924d
- Operator: Codex + astraler-qa

## Verdict

CAUTION for release retag from this delta pass.

The three blocking product findings from the first delta pass were fixed and
rerun to PASS. This is not a full release GO because native-dialog/manual-network
cases in this delta scope remain `NEEDS_HUMAN` or `SKIPPED`.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 2 | 0 | 0 | 2 | 4 |
| T1 | 8 | 0 | 0 | 8 | 1 |
| T2 | 5 | 0 | 0 | 0 | 0 |

## Gate Results

- `go test ./...`: PASS
- `pnpm test --run`: PASS, 56 files / 570 tests
- `pnpm typecheck`: PASS
- `pnpm check:contracts-drift`: PASS
- YAML parse for selected QA cases and invariants: PASS
- JSONL parse and `git diff --check`: PASS

Evidence: `evidence/gates.txt`, `evidence/delta-fix-rerun.txt`

## Blocking Findings

None for the three current-scope delta product fixes after rerun.

## Resolved Findings

- `TC-PROJ-009` / `TC-PROJ-001`: Project Detail no-provider guidance is present in DOM text, including manual provider-folder creation guidance. Filesystem stayed unchanged.
- `TC-PROJ-001`: project provider scan now ignores `.gitkeep` marker entries and does not create phantom install rows.
- `TC-DISPLAY-001`: Settings provider path cells now trim display-only trailing slashes for each configured path, including Antigravity Global skills.

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| GATE-AUTOMATED | gate | PASS | `evidence/gates.txt`, `evidence/delta-fix-rerun.txt` | Automated gates passed after fixes. |
| TC-PRIVACY-001 | T0 | PASS | `evidence/privacy-idle-network.txt` | Only localhost dev/CDP traffic observed for the QA instance. |
| TC-PRIVACY-002 | T0 | PASS | `evidence/global-plugins-db.txt`, `evidence/privacy-idle-network.txt` | No plugin update check was triggered automatically. |
| TC-PRIVACY-003 | T1 | PASS | `evidence/about-diagnostics.png`, `evidence/global-plugins-after-scan.png` | App update surface remains distinct from plugin update surface. |
| TC-PRIVACY-005 | T1 | NEEDS_HUMAN | `evidence/privacy-idle-network.txt` | Short idle sample only; full extended idle was not completed after blockers were found. |
| TC-SETUP-001 | T1 | NEEDS_HUMAN | `evidence/host-scan-db.txt`, `evidence/dashboard-after-host-scan.png` | Native folder picker was bypassed through RPC; auto-scan through the actual picker remains unverified. |
| TC-SETUP-005 | T1 | NEEDS_HUMAN | `evidence/host-scan-db.txt`, `evidence/dashboard-after-host-scan.png` | Explicit host scan found 3 skills; picker-triggered auto-scan path needs human/osascript harness. |
| TC-SETUP-006 | T1 | SKIPPED | — | Not executed after current-scope NO-GO blockers were found. |
| TC-FS-006 | T0 | SKIPPED | — | Remove-project destructive coverage deferred until after fixes. |
| TC-PROVIDER-004 | T0 | SKIPPED | — | Not executed after current-scope NO-GO blockers were found. |
| TC-PROVIDER-006 | T0 | SKIPPED | — | Not executed after current-scope NO-GO blockers were found. |
| TC-SKILL-007 | T0 | SKIPPED | — | Not executed after current-scope NO-GO blockers were found. |
| TC-A11Y-002 | T0 | NEEDS_HUMAN | — | Native destructive confirmation cancel path requires osascript/human handling. |
| TC-A11Y-005 | T0 | NEEDS_HUMAN | — | Native destructive confirmation accept path requires osascript and run-local fixture confirmation. |
| TC-GLOBAL-001 | T1 | PASS | `evidence/global-skills-after-scan.png`, `evidence/global-skills-db.txt` | Provider registry rows have explicit global states. |
| TC-GLOBAL-007 | T1 | PASS | `evidence/global-skills-after-scan.png`, `evidence/global-skills-db.txt` | All Settings/provider registry providers accounted for. |
| TC-GLOBAL-008 | T2 | PASS | `evidence/global-skills-after-scan.png` | Provider tabs and counts visible. |
| TC-PLUGIN-010 | T2 | PASS | `evidence/global-plugins-after-scan.png`, `evidence/global-plugins-db.txt` | Provider tabs and plugin counts visible. |
| TC-PLUGIN-005 | T1 | NEEDS_HUMAN | `evidence/global-plugins-after-scan.png`, `evidence/global-plugins-db.txt` | Update-check network action not run in this pass. |
| TC-PLUGIN-009 | T1 | NEEDS_HUMAN | `evidence/global-plugins-after-scan.png` | Needs network-failure/update-check harness. |
| TC-ABOUT-002 | T1 | PASS | `evidence/about-diagnostics.png`, `evidence/about-after-copy-snapshot.txt` | App Updates and Diagnostics are distinct. |
| TC-ABOUT-001 | T1 | NEEDS_HUMAN | `evidence/about-diagnostics.png` | Manual GitHub update check not run. |
| TC-ABOUT-003 | T1 | NEEDS_HUMAN | `evidence/about-diagnostics.png` | Needs offline/proxy harness or manual network approval. |
| TC-DIAG-001 | T1 | NEEDS_HUMAN | `evidence/about-diagnostics.png`, `evidence/diagnostics-clipboard.txt` | Copy passed and redaction passed; native export save dialog remains human/harness. |
| TC-SETTINGS-005 | T2 | PASS | `evidence/settings-provider-columns-display.png`, `evidence/settings-provider-columns-display-snapshot.txt` | Column order verified. |
| TC-SETTINGS-006 | T2 | PASS | `evidence/settings-provider-columns-display.png`, `evidence/settings-provider-columns-display-snapshot.txt` | Host folder is read-only with Host Skills link. |
| TC-DISPLAY-001 | T2 | PASS | `evidence/delta-fix-rerun.txt` | Rerun after fix confirmed Antigravity Global skills path is display-trimmed. Earlier failure kept in JSONL history. |
| TC-PROJ-001 | T1 | PASS | `evidence/delta-fix-rerun.txt` | Rerun after fix confirmed no-provider guidance and no `.gitkeep` phantom install rows. Earlier failure kept in JSONL history. |
| TC-PROJ-009 | T1 | PASS | `evidence/delta-fix-rerun.txt` | Rerun after fix confirmed actionable no-provider guidance with filesystem unchanged. Earlier failure kept in JSONL history. |
| TC-PROJ-010 | T1 | PASS | `evidence/project-detail-effective-global-skills.png`, `evidence/project-detail-effective-global-skills-db.txt` | Global Skills section is distinct and read-only. |
| TC-PROJ-011 | T1 | PASS | `evidence/global-skills-after-scan.png`, `evidence/project-detail-effective-global-skills.png` | Global Skills and Project Detail agree for detected provider. |

## Environment

- QA home: `<repo>/docs/qa/runs/2026-06-05-delta-v012-dogfooding-fixes/qa-home`
- QA DB: `<repo>/docs/qa/runs/2026-06-05-delta-v012-dogfooding-fixes/qa-home/qa.db`
- QA host: `<repo>/docs/qa/runs/2026-06-05-delta-v012-dogfooding-fixes/fixtures/skill-host-a`
- QA project: `<repo>/docs/qa/runs/2026-06-05-delta-v012-dogfooding-fixes/fixtures/projects/generic-agents-project`
- CDP port: 49222

## Follow-Up

- Complete or explicitly waive the remaining native-dialog/manual-network cases before using this run as a full release GO.
- Native dialogs should be driven by `osascript` or marked `NEEDS_HUMAN` per governance.
- Keep the `.gitkeep` regression and Settings display-path regression in the automated suite.
