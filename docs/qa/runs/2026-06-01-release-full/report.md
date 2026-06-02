# QA Run Report: 2026-06-01-release-full

- Date: 2026-06-01
- Scope: release
- Profile: release-full
- App mode: dev-electron + packaged DMG smoke
- Target version/commit: 0.1.0 / 1a623d7
- Operator: codex-agent

## Verdict

NO-GO

Post-merge update: the two app behavior blockers found in this run were fixed
by PR #17 and verified on `main` at `a69149f` in
`docs/qa/runs/2026-06-01-post-merge-pr17-smoke/`. The original verdict above is
kept as the historical result for this release-full run at commit `1a623d7`.
Current release readiness is improved, but not a clean GO until remaining
T0/T1 BLOCKED cases are triaged or explicitly accepted, and signed/notarized
release verification is completed or deferred by owner decision.

## Summary

| Tier | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|
| T0 | 9 | 3 | 12 | 0 | 0 |
| T1 | 20 | 1 | 13 | 1 | 0 |
| T2 | 3 | 0 | 6 | 0 | 0 |

## Blocking Findings

- Fixed after this run: Reset All Data no longer relaunches into a blank window.
  PR #17 merged to `main` as `a69149f`; post-merge smoke passed
  `TC-SETTINGS-002`, `TC-DASH-003`, and `TC-DB-003`.
- Fixed after this run: Missing active Skill Host Folder no longer shows as an
  active healthy host. PR #17 merged to `main` as `a69149f`; post-merge smoke
  passed `TC-SETUP-002`.
- Several release-full cases are blocked because they require dedicated harness/product surface: switch install mode, chmod/read-only FS, stale operation restart, corrupt/migrated DB launch, accessibility keyboard pass, signed/notarized artifact verification.
- Signing/notarization credentials are absent; ad-hoc verification passes, but customer-ready release verification needs human-provided Apple credentials.

Post-merge verification for PR #17:

- Report: `docs/qa/runs/2026-06-01-post-merge-pr17-smoke/report.md`
- Results: `docs/qa/runs/2026-06-01-post-merge-pr17-smoke/results.jsonl`
- Gates passed: contract drift, targeted Go services/RPC handlers, renderer reset hook test.
- Cases passed on merged `main`: `TC-SETUP-002`, `TC-SETTINGS-002`,
  `TC-DASH-003`, `TC-DB-003`.

Automated gates passed before UI QA:

- Go tests: `evidence/gate-go-test.txt`
- Desktop typecheck: `evidence/gate-desktop-typecheck.txt`
- Desktop tests: `evidence/gate-desktop-test.txt`
- Contract drift: `evidence/gate-contracts-drift.txt`

Packaged DMG smoke passed with `dist/astraler-skillbox-0.1.0-arm64.dmg`.

## Cases Run

| Case | Tier | Status | Evidence | Notes |
|---|---|---|---|---|
| TC-A11Y-002 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-DASH-003 | T0 | FAIL | evidence/reset-after.png, evidence/reset-after-ui-text.txt, evidence/reset-after-db.txt, evidence/reset-after-fs.txt | Historical failure at `1a623d7`: reset clears metadata and preserves fixture folders, but the app relaunches into a blank window. Fixed by PR #17 and verified PASS on `main` at `a69149f` in `2026-06-01-post-merge-pr17-smoke`. |
| TC-DB-001 | T0 | PASS | evidence/db-integrity.txt, evidence/db-counts.txt, evidence/dashboard.png | SQLite integrity and foreign-key checks passed during populated release QA state; app loaded Dashboard. |
| TC-DB-002 | T0 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-installs.txt, evidence/fs-conflict-stat.txt | Conflicting install failed with conflict_error and did not create a current managed install for beta-skill. |
| TC-DB-003 | T0 | FAIL | evidence/reset-before-db.txt, evidence/reset-after-db.txt, evidence/reset-after.png, evidence/reset-after-ui-text.txt | Historical failure at `1a623d7`: reset cleared metadata but UI relaunch outcome was blank. Fixed by PR #17 and verified PASS on `main` at `a69149f`; schema version remained `23`, `dirty=0`. |
| TC-FS-001 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-FS-002 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-FS-003 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-FS-006 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-GLOBAL-003 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-MIGRATE-002 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-MIGRATE-003 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-OPS-004 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-OPS-005 | T0 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-operations.txt, evidence/db-installs.txt | Failed conflict operation surfaced as failed operation; Project Detail did not show false current install. |
| TC-PRIVACY-001 | T0 | PASS | evidence/network-lsof-skillbox-core.txt, evidence/cdp-version.json, evidence/dashboard.png | No outbound network sockets were observed for the Go core during idle QA workflow; app remained local-first. |
| TC-PRIVACY-002 | T0 | PASS | evidence/db-counts.txt, evidence/global-plugins.png, evidence/db-plugin-layers.txt | No plugin update-check cache rows/operation were created before explicit manual update action. |
| TC-PROVIDER-002 | T0 | PASS | evidence/rpc-core-workflows-output.json | Project provider override containing .. was rejected with validation_error and not saved. |
| TC-PROVIDER-004 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-RELEASE-003 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-SETTINGS-002 | T0 | FAIL | evidence/reset-before-settings.png, evidence/reset-before-db.txt, evidence/reset-after.png, evidence/reset-after-db.txt, evidence/reset-after-fs.txt | Historical failure at `1a623d7`: reset preserved fixture folders and cleared DB metadata, but app relaunched blank. Fixed by PR #17 and verified PASS on `main` at `a69149f`; app now returns to Setup and fixtures remain on disk. |
| TC-SKILL-002 | T0 | PASS | evidence/rpc-core-workflows-output.json, evidence/project-detail-claude.png, evidence/db-installs.txt | Alpha skill installed by symlink into QA Claude project and was visible in RPC/Project Detail before removal. |
| TC-SKILL-003 | T0 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-installs.txt, evidence/fs-fixture-tree.txt | Remove operation completed; alpha-skill project symlink was removed while host fixture remained. |
| TC-SKILL-004 | T0 | BLOCKED | None | Requires a dedicated destructive/edge harness or product surface not currently available in this run; not safe to infer a pass. |
| TC-SKILL-005 | T0 | PASS | evidence/rpc-core-workflows-output.json, evidence/fs-conflict-stat.txt, evidence/db-installs.txt | Install conflict failed safely; existing unmanaged beta-skill target remained. |
| TC-DASH-001 | T1 | PASS | evidence/dashboard.png, evidence/projects-list.png, evidence/skills-library.png, evidence/db-counts.txt | Dashboard counts were captured and cross-checked against Projects/Skills Library/DB counts. |
| TC-DASH-002 | T1 | PASS | evidence/dashboard.png, evidence/project-detail-broken.png, evidence/db-warnings.txt | Broken symlink warning appeared in Project Detail and Dashboard warning count. |
| TC-DB-004 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-ERROR-002 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-ERROR-003 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-FS-004 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-GLOBAL-001 | T1 | PASS | evidence/global-skills.png, evidence/rpc-core-workflows-output.json, evidence/fs-fixture-tree.txt | Global Skills scan found QA HOME generic and Claude global skills under run-local fixture paths. |
| TC-GLOBAL-002 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-GLOBAL-004 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-MIGRATE-001 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-MIGRATE-004 | T1 | PASS | evidence/dashboard.png, evidence/db-operations.txt, evidence/db-warnings.txt | Completed scan metadata and warning rows persisted in the same QA session before reset. |
| TC-OPS-001 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-OPS-002 | T1 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-operations.txt, evidence/project-detail-claude.png | Manual project scans reached success and updated project state; no stuck scan state observed. |
| TC-OPS-003 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-PACKAGE-001 | T1 | PASS | evidence/release-mac-dmg-smoke.txt | Packaged DMG smoke launched from mounted DMG, used bundled sidecar and temp DB path, and left no orphaned sidecar. |
| TC-PLUGIN-001 | T1 | PASS | evidence/global-plugins.png, evidence/db-plugin-layers.txt, evidence/rpc-core-workflows-output.json | Global Plugins showed QA Claude plugin status/version metadata from run-local settings. |
| TC-PLUGIN-002 | T1 | PASS | evidence/rpc-core-workflows-output.json, evidence/global-plugins.png | QA plugin toggle off/on succeeded and restored enabled state without touching real settings. |
| TC-PLUGIN-003 | T1 | PASS | evidence/project-detail-claude.png, evidence/global-plugins.png, evidence/db-plugin-layers.txt | Project Detail displayed project/user/local plugin layer state; Global Plugins remained user-layer view. |
| TC-PLUGIN-004 | T1 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-plugin-layers.txt | Malformed QA plugin settings path scanned as malformed and was later restored to normal fixture settings. |
| TC-PLUGIN-005 | T1 | PASS | evidence/global-plugins.png, evidence/db-operations.txt | No update-check operation was run before explicit manual action; network action was not executed. |
| TC-PRIVACY-003 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-PRIVACY-004 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-PROJ-001 | T1 | PASS | evidence/projects-list.png, evidence/project-detail-claude.png, evidence/db-counts.txt | QA projects were added/scanned and Project Detail showed detected provider facts. |
| TC-PROJ-002 | T1 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-operations.txt | Manual project scans completed with terminal success and updated scan timestamps. |
| TC-PROVIDER-001 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-PROVIDER-003 | T1 | PASS | evidence/rpc-core-workflows-output.json, evidence/db-plugin-layers.txt | Global config override to malformed QA settings affected Global Plugins, then restore override returned to fixture settings. |
| TC-RELEASE-001 | T1 | PASS | evidence/release-mac-check.txt | Release preflight reported tooling readiness and missing signing/notarization credentials clearly without building. |
| TC-RELEASE-002 | T1 | PASS | evidence/release-mac-dmg-smoke.txt | DMG smoke used mounted artifact, bundled sidecar, temp DB, and clean shutdown. |
| TC-RELEASE-004 | T1 | PASS | evidence/release-mac-manifest.txt, evidence/release-sha256sums-check.txt | Manifest generation succeeded and SHA256SUMS verified from dist. |
| TC-RELEASE-005 | T1 | NEEDS_HUMAN | evidence/release-mac-verify-adhoc.txt, evidence/release-mac-check.txt | Only ad-hoc dry-run verification is available; customer-ready signed/notarized verification needs Apple credentials and release artifact approval. |
| TC-SETTINGS-001 | T1 | BLOCKED | None | Release-full case requires additional setup/harness step not completed before the destructive reset phase; kept explicit for follow-up. |
| TC-SETUP-001 | T1 | PASS | evidence/settings.png, evidence/rpc-core-workflows-output.json, evidence/fixture-list.txt | QA host was configured via renderer bridge/RPC and app reached main screens with run-local host path. |
| TC-SETUP-002 | T1 | FAIL | evidence/missing-host-dashboard.png, evidence/missing-host-ui-text.txt, evidence/missing-host-fs.txt | Historical failure at `1a623d7`: after moving active host away, Dashboard still showed active stale state. Fixed by PR #17 and verified PASS on `main` at `a69149f`; Dashboard and Settings now report `status: missing` with recovery guidance. |
| TC-SKILL-001 | T1 | PASS | evidence/skills-library.png, evidence/db-counts.txt, evidence/fs-fixture-tree.txt | Skills Library/DB reflected active QA host skills. |
| TC-SKILL-006 | T1 | PASS | evidence/project-detail-broken.png, evidence/fs-broken-readlink.txt, evidence/db-warnings.txt | Broken project symlink was classified as broken_symlink with warning evidence. |
| TC-A11Y-001 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-A11Y-003 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-A11Y-004 | T2 | PASS | evidence/projects-list.png, evidence/project-detail-claude.png, evidence/global-plugins.png, evidence/settings.png | Long run-folder paths rendered on major screens during screenshot pass without obvious blank/overlap blocking the workflow. |
| TC-DASH-004 | T2 | PASS | evidence/dashboard.png, evidence/projects-list.png, evidence/skills-library.png, evidence/global-skills.png, evidence/global-plugins.png | Primary Dashboard navigation targets loaded during evidence capture without blank screens. |
| TC-ERROR-001 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-ERROR-004 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-FS-005 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-GLOBAL-005 | T2 | BLOCKED | None | Requires manual keyboard/accessibility/detail-route or specialized UI harness; not executed in this automated release pass. |
| TC-PROVIDER-005 | T2 | PASS | evidence/rpc-core-workflows-output.json | Unknown provider key was rejected with validation_error and no unknown provider row was expected. |

## Environment

- QA home: `<repo-root>/docs/qa/runs/2026-06-01-release-full/fixtures/homes/plugin-home`
- QA DB: `<repo-root>/docs/qa/runs/2026-06-01-release-full/fixtures/homes/plugin-home/qa.db`
- QA host: `<repo-root>/docs/qa/runs/2026-06-01-release-full/fixtures/skill-host-a`
- QA project: `<repo-root>/docs/qa/runs/2026-06-01-release-full/fixtures/projects/claude-project`
- CDP port: 49222

## Follow-Up

- Completed after run: reset blank relaunch and missing host stale active state fixed
  by PR #17, reviewed, merged, and post-merge smoke verified.
- Remaining release-readiness work: triage/execute or explicitly accept the
  remaining T0/T1 BLOCKED cases; decide whether unsigned/ad-hoc Mac release is
  acceptable for this milestone; complete signed/notarized verification when
  Apple credentials are available.
- Cases to improve: add harness support for chmod/read-only FS, stale operation
  restart, corrupt DB launch, accessibility keyboard snapshots, and simultaneous
  duplicate operation attempts.
