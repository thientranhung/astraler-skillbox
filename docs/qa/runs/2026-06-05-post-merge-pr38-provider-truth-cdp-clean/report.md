# QA Run Report: 2026-06-05-post-merge-pr38-provider-truth-cdp-clean

- **Date:** 2026-06-05
- **Scope:** delta - post-merge PR #38 provider truth / scan UX
- **App mode:** dev-electron (CDP, port 49222)
- **Target version/commit:** 0.1.2 + main-43f4ecdc
- **Operator:** claude-sonnet-4-6 + astraler-qa

## Verdict

**GO**

All T1/T2 cases in scope pass. One NEEDS_HUMAN sub-state (scan-in-progress spinner) is a
harness-speed limitation, not a product defect. No T0 cases were in scope for this delta.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| gate | 1 | 0 | 0 | 0 | 0 |
| T1 | 2 | 0 | 0 | 1 | 0 |
| T2 | 2 | 0 | 0 | 0 | 0 |

## Blocking Findings

None.

## Cases Run

| Case | Tier | Status | Key Evidence | Notes |
|---|---|---|---|---|
| GATE-AUTOMATED | gate | PASS | evidence/automated-gates.txt | go test + typecheck clean on 43f4ecdc |
| TC-SETUP-006 | T1 | PASS | TC-SETUP-006-E-dashboard-never-scanned.png, TC-SETUP-006-H-empty-host-skills-library.png | 3/4 sub-states confirmed via CDP; scanning sub-state NEEDS_HUMAN (see below) |
| TC-SETUP-006-scanning-substate | T1 | NEEDS_HUMAN | - | Scan-in-progress spinner sub-second on local fixture; harness cannot capture |
| TC-PROJ-009 | T1 | PASS | TC-PROJ-009-B-project-detail-never-scanned.png, TC-PROJ-009-filesystem-after-detail-view.txt | No-provider guidance correct; no dirs created (filesystem verified) |
| TC-GLOBAL-008 | T2 | PASS | TC-GLOBAL-008-A/B/C/D screenshots, TC-GLOBAL-008-db-global-locations.txt | All tab counts correct; empty state (Claude missing) confirmed |
| TC-PLUGIN-010 | T2 | PASS | TC-PLUGIN-010-A/B/C/D/E screenshots, TC-PLUGIN-010-db-plugin-scans.txt | All tab counts correct; empty state (Codex not configured) confirmed |

## Key Observations

### TC-SETUP-006 - Host Skills empty states

The app distinguishes three observable states:

1. **Not-yet-configured (pre-host):** Welcome screen shown - "Choose the folder that contains your agent skills." No Skills Library accessible until host is set.
2. **Not-yet-scanned (post-host-choose, pre-scan):** Dashboard shows "Last Scan: Never" in the Skill Host Folder card. Skills Library auto-scans on mount, making the not-yet-scanned state sub-second in the Skills Library screen itself. Dashboard is the product's persistent not-yet-scanned indicator.
3. **No-skills-found (scanned, empty host):** Skills Library shows "No skills found - The host folder scan ran but found no skills. Add skill folders to your Skill Host Folder and scan again." Distinct message, not a generic empty state.
4. **Scan-in-progress (transient):** NEEDS_HUMAN - completes in <100ms on local fixture, not capturable via CDP agent-browser. Harness recommendation: add a slow-path fixture or a test-delay flag.

### TC-PROJ-009 - Project Detail no-provider guidance

- After scanning a no-provider project, PROVIDERS section shows: "No provider folders detected - To make this project install-ready, create a provider folder manually inside the project, for example: .agents/skills/ - Shared Agent Skills, .claude/skills/ - Claude Code. After creating the folder, scan the project again to detect providers."
- No install/create-folder action buttons present. PASS
- Filesystem confirmed: only README.md under project fixture after detail view. PASS
- PROVIDER PLUGINS section correctly shows "not configured" for each layer.
- Auto-scan fires on Project Detail mount so the "never-scanned" state on the detail screen is sub-second. The post-scan no-provider guidance is the verifiable state.

### TC-GLOBAL-008 - Global Skills provider tabs

Tab bar shows all registered providers with accurate counts:
`All(2)`, `Antigravity CLI(0)`, `Claude(0)`, `Codex(0)`, `Shared Agent Skills(2)`, `OpenCode(0)`, `Pi(0)`.

- Shared Agent Skills tab: qa-global-skill-1 and qa-global-skill-2 displayed with mode=direct, status=current. PASS
- Claude tab (0 entries): shows "Claude - missing" badge with path `~/.claude/skills` and "No global skills found." PASS
- All tab: restores full multi-provider view. PASS
- DB: `generic_agents` active, `claude` missing, all others disabled. PASS

### TC-PLUGIN-010 - Global Plugins provider tabs

Tab bar shows:
`All(2)`, `Claude(2)`, `Codex(0)`, `Antigravity CLI(0)`.

- Claude tab (2 plugins): qa-plugin (enabled, 1.0.0) and qa-disabled (disabled, 0.2.0) from fixture. PASS
- Codex tab (0 plugins): shows "Codex - not configured" badge with path `~/.codex/config.toml` and "Some settings in this file are managed outside Skillbox." PASS
- All tab: restores both providers. PASS
- DB: `claude` scan_status=ok with 2 entries; `codex` and `antigravity_cli` missing. PASS

## Run Notes

- **Prior polluted run** (`2026-06-05-post-merge-pr38-provider-truth-cdp`): abandoned due to native-dialog noise during host selection attempt. Not used for verdicts.
- **Native path picker:** not exercised in this delta run; out-of-scope per PM direction. Not counted as a FAIL.
- **All fixture seeding** done via `window.core.invoke` RPC through CDP (no osascript path selection).
- **QA HOME** (`qa-home/`): contained `.agents/skills/qa-global-skill-{1,2}` for global skills and `.claude/settings.json` + `.claude/plugins/installed_plugins.json` from plugin-home fixture.
- Dev Electron stopped cleanly after all evidence collection.

## Environment

- QA home: `docs/qa/runs/2026-06-05-post-merge-pr38-provider-truth-cdp-clean/qa-home`
- QA DB: `qa-home/qa.db`
- QA host (populated): `fixtures/host` (3 skills: alpha-skill, beta-skill, nested-gamma)
- QA host (empty): `fixtures/empty-host` (.agents/skills/ empty)
- QA project (no-provider): `fixtures/projects/no-provider-project`
- CDP port: 49222

## Follow-Up

- **Add regression case:** TC-SETUP-006 scan-in-progress sub-state - recommend a slow-scan fixture (e.g. large directory) or a `SKILLBOX_SCAN_DELAY_MS` test flag for the harness.
- **TC-PROJ-009 never-scanned instant prompt:** If the product ever adds a deliberate "never-scanned" state to Project Detail (before auto-scan), add an explicit CDP test for it.
- **No bugs filed:** all observed behaviors match product intent for PR #38 scope.
