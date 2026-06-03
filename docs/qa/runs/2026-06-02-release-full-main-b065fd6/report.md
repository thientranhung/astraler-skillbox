# QA Run Report: 2026-06-02-release-full-main-b065fd6

- Date: 2026-06-02
- Scope: release
- Profile: release-full
- App mode: dev Electron first, packaged smoke after artifact/human gates
- Target version/commit: 0.1.2 / b065fd6ba7976d3db0e6de03abb276bdaf1b35a5
- Operator: agent
- Status: PARTIAL RUN — dev-fixture T0 batch complete; P0/A11Y delta verified 2026-06-03; unsigned macOS packaged smoke complete

## Verdict

CONDITIONAL GO for the unsigned macOS DMG release candidate, with owner-accepted
waivers for Apple notarization/signing and Phase 2 copy-mode workflows.

Gates: ALL PASS. T0 dev-fixture batch complete; original 8 FAIL + 2 BLOCKED results. **P0 delta rerun 2026-06-03: 5/5 delta cases PASS** (TC-MIGRATE-002, TC-MIGRATE-005, TC-FS-003, TC-SETUP-003, TC-DASH-003). **A11Y destructive-confirmation rerun 2026-06-03: 2/2 delta cases PASS** (TC-A11Y-002, TC-A11Y-005). **Unsigned macOS packaged smoke 2026-06-03: 8 PASS + 1 NEEDS_HUMAN** (TC-PACKAGE-001, TC-PACKAGE-002, TC-PACKAGE-003, TC-RELEASE-001, TC-RELEASE-002, TC-RELEASE-003, TC-RELEASE-004, TC-RELEASE-006 PASS; TC-RELEASE-005 NEEDS_HUMAN/waived). Phase 2 waivers (TC-SKILL-004, TC-SKILL-011, TC-SKILL-012) and TC-OPS-007 remain unchanged.

## Original Run Scope Before Delta

| Tier | Selected | Passed | Failed | Blocked | Needs Human | Skipped |
|---|---:|---:|---:|---:|---:|---:|
| T0 | 38 | 24 | 8 | 2 | 0 | 0 |
| T1 | 58 | 2 | 0 | 0 | 0 | 0 |
| T2 | 9 | 0 | 0 | 0 | 0 | 0 |
| T3 | 0 | 0 | 0 | 0 | 0 | 0 |

Original cases executed before delta: 36 of 105. Delta reruns appended after P0/A11Y fixes and harness hardening: 7 PASS entries. Packaged/release smoke appended: 8 PASS + 1 NEEDS_HUMAN. `results.jsonl` now contains 52 rows total.

## Required Gates

- `(cd core-go && go test ./...)` — **PASS**
- `(cd apps/desktop && pnpm typecheck)` — **PASS**
- `(cd apps/desktop && pnpm test)` (525 tests) — **PASS**
- `(cd apps/desktop && pnpm check:contracts-drift)` (38 files) — **PASS**

Gate evidence: `evidence/gates/{go-test,typecheck,vitest,contracts-drift}.txt`

---

## Delta QA — P0 Fix Verification (2026-06-03)

**Operator:** Tom (agent)
**Target commit:** uncommitted P0 fixes on `main` branch (reviewed by Larry)
**Delta gates:** go test (fresh, repositories), typecheck, vitest 525/525, contracts-drift — **ALL PASS**
**Evidence:** `evidence/delta-gates.txt`, `evidence/delta-ui-session.log`

### Delta Case Results

| Case | Old Status | New Status | Root Fix |
|------|-----------|-----------|---------|
| TC-MIGRATE-002 | FAIL | **PASS** | `manager.ts` no longer calls `fatal()` on pre-ready timeout; `index.ts` stores error and creates window; `router.tsx` gates child routes, queries `getStartupError()`, navigates to `/startup-error`; `StartupErrorScreen` renders error with Go stderr detail |
| TC-MIGRATE-005 | FAIL | **PASS** | Same root fix as TC-MIGRATE-002; dirty migration message (`Dirty database version 23`) visible on startup error screen |
| TC-FS-003 | FAIL | **PASS** | `use-install-skill.ts` tracks `lastOperationError` state; `add-skill-wizard.tsx` renders inline error row for both RPC errors and async operation failures |
| TC-SETUP-003 | FAIL | **PASS** | `skill_host_folder_repo.go` `UpsertAndActivate` immediately marks old-host installs as `old_host`; UI Project Detail shows "Linked to old host" (not "Linked to active host") |
| TC-DASH-003 | FAIL | **PASS** | Original failure was a **QA harness artifact**: stage-2 confirmation requires typing "RESET" then clicking "Xác nhận Reset"; original test only clicked stage-1 "Tiếp tục" twice without completing stage-2. Full two-stage flow completes correctly: DB cleared (projects=0, installs=0), app navigates to Setup. The `use-reset-all.ts` `onError` fix correctly surfaces genuine RPC failures. |
| TC-A11Y-002 | BLOCKED | **PASS** | Native/macOS sheet handling moved outside CDP with PID-scoped System Events; cancel paths for Remove Skill, Remove Project, and Reset All Data preserve DB/FS state |
| TC-A11Y-005 | not previously run | **PASS** | Accept paths mutate only managed run-local QA targets: symlink/install removed, project folder preserved, reset clears user metadata and returns to Setup |

### TC-MIGRATE-002 Delta — PASS

StartupErrorScreen renders with error detail:
```
server.ready timeout

time=... level=ERROR msg="failed to open database" err="pragma \"PRAGMA journal_mode=WAL\": file is not a database (26)"
exit status 1
```
Real app-data path (`~/.local/share/astraler-skillbox/`) does not exist. No files written outside run-local fixture.

Evidence: `evidence/TC-MIGRATE-002-delta-error-screen.png`, `evidence/TC-MIGRATE-002-delta-launch.log`, `evidence/TC-MIGRATE-002-delta-fs.txt`

### TC-MIGRATE-005 Delta — PASS

StartupErrorScreen renders with dirty migration error:
```
server.ready timeout

time=... level=ERROR msg="failed to open database" err="migrate up: Dirty database version 23. Fix and force version."
exit status 1
```
Dirty DB remains version=23 dirty=1 (unchanged). Real app-data path does not exist.

Evidence: `evidence/TC-MIGRATE-005-delta-error-screen.png`, `evidence/TC-MIGRATE-005-delta-launch.log`, `evidence/TC-MIGRATE-005-delta-fs.txt`

### TC-FS-003 Delta — PASS

Install of beta-skill into chmod-555 provider skills folder fails with operation status=failed. Wizard shows inline error row:
> "Skill install failed: 0/1 installed. [filesystem_error] Could not create skill symlink: failed to link skill "beta-skill": symlink ...: permission denied"

DB: 0 active install records for beta-skill. No partial target on filesystem. Skills folder permissions confirmed 0555 during test.

Evidence: `evidence/TC-FS-003-delta-wizard-before.png`, `evidence/TC-FS-003-delta-wizard-after.png`, `evidence/TC-FS-003-delta-db-fs.txt`

### TC-SETUP-003 Delta — PASS

After `host.choose` with host-b while alpha-skill installs from host-a exist:
- DB: `installs.install_status = old_host` for both installs (ids 6, 8) — immediate, no scan required
- UI Project Detail: "Linked to old host" label on both Shared Agent Skills and Codex alpha-skill entries
- Both fixture folders (host-a, host-b) intact on disk
- Symlink still points to host-a (not silently rewritten)

Evidence: `evidence/TC-SETUP-003-delta-settings-after-repoint.png`, `evidence/TC-SETUP-003-delta-project-detail-after.png`, `evidence/TC-SETUP-003-delta-db.txt`, `evidence/TC-SETUP-003-delta-fs.txt`

### TC-DASH-003 Delta — PASS (QA Artifact Resolved)

Two-stage reset confirmation executed correctly:
1. Click "Reset All Data" → stage-1 modal
2. Click "Tiếp tục" → stage-2 RESET textbox
3. Fill "RESET" → "Xác nhận Reset" enabled
4. Click "Xác nhận Reset" → reset executes

Post-reset: projects=0, installs=0, skill_host_folders=0, active_host=null. App navigates to "Welcome to Astraler Skillbox" Setup screen. QA fixture folders preserved on disk (alpha-skill symlink intact in project skills folder).

**Root cause of original FAIL**: The previous harness clicked "Tiếp tục" (stage-1 Continue) twice without completing the stage-2 textbox requirement. The reset itself was always functional.

Evidence: `evidence/TC-DASH-003-delta-stage1-visible.png`, `evidence/TC-DASH-003-delta-stage2-dialog.png`, `evidence/TC-DASH-003-delta-after-confirm.png`, `evidence/TC-DASH-003-delta-setup-screen.png`, `evidence/TC-DASH-003-delta-db.txt`

### TC-A11Y-002 Delta — PASS

Cancel destructive confirmations were rerun on the run-local `qa-home-a11y3`
fixture with a single Claude provider and one installed `alpha-skill`.

- Remove Skill cancel: install remained current, symlink remained intact, host source remained intact.
- Remove Project cancel: native/macOS sheet was handled outside CDP via PID-scoped System Events; project metadata, folder, install symlink, and host skill remained intact.
- Reset All Data stage-2 cancel: confirmation remained gated on the `RESET` token and cancel preserved DB/FS state.

Evidence: `evidence/TC-A11Y-002-delta-02-remove-skill-dialog.png`, `evidence/TC-A11Y-002-delta-05-remove-project-cancelled.png`, `evidence/TC-A11Y-002-delta-10-reset-stage2-textbox.png`, `evidence/TC-A11Y-002-delta-final-verify.txt`

### TC-A11Y-005 Delta — PASS

Accept destructive confirmations were executed only against run-local QA
fixtures.

- Remove Skill accept: removed the managed `.claude/skills/alpha-skill` symlink and active install record; host `alpha-skill` source remained intact.
- Remove Project accept: Skillbox showed `0 projects`; DB marked the project non-active/removed; the project folder on disk remained intact.
- Reset All Data accept: required typing `RESET`, then cleared user metadata tables (`projects=0`, `installs=0`, `project_providers=0`, `skills=0`, `skill_host_folders=0`) and returned the app to Setup. Run-local fixture folders remained on disk.

Evidence: `evidence/TC-A11Y-005-delta-01-remove-skill-dialog.png`, `evidence/TC-A11Y-005-delta-05-after-remove-project.png`, `evidence/TC-A11Y-005-delta-10-after-reset.png`, `evidence/TC-A11Y-005-delta-11-after-reset-db-fs.txt`

## Packaged Smoke — unsigned macOS artifact (2026-06-03)

**Artifact under test:** `apps/desktop/dist/astraler-skillbox-0.1.2-arm64.dmg`

**Build source:** local `main@04950f6` (`Record A11Y QA rerun results`), package version `0.1.2`.

**SHA256:** `1c27c1912c5561608397c085eddfda5e50b10bb99ea8b47aa77581790c2c2d5e`

### Packaged Case Results

| Case | Status | Notes |
|------|--------|-------|
| TC-RELEASE-001 | PASS | Preflight reports missing signing/notarization credentials clearly; all non-credential readiness checks pass |
| TC-PACKAGE-001 | PASS | DMG mounted, copied to temp install root, launched, Go core ready, clean quit |
| TC-PACKAGE-002 | PASS | Fresh packaged launch reached ready without Keychain/Safe Storage prompt; source gates `use-mock-keychain` by default |
| TC-PACKAGE-003 | PASS | No orphaned packaged `skillbox-core` after quit |
| TC-RELEASE-002 | PASS | Unsigned DMG launches with bundled sidecar and isolated smoke DB |
| TC-RELEASE-003 | PASS | No CDP listener exposed on reserved ports `49222-49250` during packaged launch |
| TC-RELEASE-004 | PASS | Manifest generated and `SHA256SUMS` verification passes |
| TC-RELEASE-005 | NEEDS_HUMAN | Unsigned/ad-hoc DMG correctly fails Developer ID, hardened runtime, Gatekeeper, and stapling verification; owner accepted waiver until Apple credentials are available |
| TC-RELEASE-006 | PASS | Packaged app uses temp packaged smoke DB paths, not repo/dev DB |

### Packaged Evidence Highlights

- `release:mac:dmg-smoke` output: mounted DMG read-only, copied `Astraler Skillbox.app` to temp, launched bundled sidecar from `Contents/Resources/core/skillbox-core`, opened temp `SKILLBOX_DB_PATH`, reached `[manager] Go core ready`, quit cleanly, no orphaned sidecar, detached DMG.
- CDP probe: no listeners on `49222-49250` before launch, during packaged launch, or after quit.
- DB path probe: active smoke DB opened under `/tmp/skillbox-cdp-smoke-*/skillbox.db` and `/var/folders/.../skillbox-dmgsmoke-ud-*/skillbox.db`, never under the repo.
- Signing verification: `release:mac:verify` correctly fails the unsigned artifact for Developer ID/notarization/Gatekeeper.

Evidence: `evidence/packaged-smoke/TC-PACKAGE-001-003-RELEASE-002-006-dmg-smoke.txt`, `evidence/packaged-smoke/TC-PACKAGE-002-keychain-source-check.txt`, `evidence/packaged-smoke/TC-RELEASE-003-006-cdp-db-path.txt`, `evidence/packaged-smoke/TC-RELEASE-004-manifest-checksum.txt`, `evidence/packaged-smoke/TC-RELEASE-005-unsigned-verify.txt`

### Residual Risk After Delta

| Item | Risk | Mitigation |
|------|------|-----------|
| TC-SKILL-004, TC-SKILL-011, TC-SKILL-012 | Phase 2 waiver; copy mode not implemented | Owner-accepted; defer to Phase 2 |
| TC-A11Y-002 | Native/macOS sheet harness risk | Cleared for this rerun with PID-scoped System Events / `osascript`; keep playbook rule for future native sheets |
| TC-OPS-007 | Blocked by copy mode inoperative (TC-SKILL-011) | Unblocks when Phase 2 implements copy mode |
| TC-RELEASE-005 | Apple Developer ID signing/notarization unavailable | Owner-accepted unsigned DMG waiver with README Gatekeeper bypass instructions; revisit when Apple credentials exist |

---

## Blocking Findings (Original — 2026-06-02)

> **Post-delta status**: TC-MIGRATE-002, TC-MIGRATE-005, TC-FS-003, TC-SETUP-003, TC-DASH-003, TC-A11Y-002, and TC-A11Y-005 are **PASS** as of the 2026-06-03 delta reruns. See Delta QA section above for current status. The original findings are preserved below for historical record.

### TC-MIGRATE-002 — ~~FAIL~~ PASS after P0 fix (T0)

**Original (2026-06-02):** App launched with an invalid-bytes DB path showed a blank renderer window with no
visible user-facing error. The Go core fatally exited with "did not send
server.ready within 10s" after failing to open the invalid DB, leaving the
Electron window blank instead of rendering a crash/error screen.

Safety invariant holds: the real DB at `~/.local/share/astraler-skillbox/skillbox.db`
was not created.

Original evidence: `evidence/TC-MIGRATE-002-log.txt`, `evidence/TC-MIGRATE-002-invalid-db-screen.png`

### TC-SKILL-011 — FAIL (T0)

App ignores `default_install_mode=rsync_copy` in app_settings. Add Skill wizard
has no mode picker — only skill checkboxes and Install. All installs created
install_mode=symlink regardless of DB default. Copy mode install is not functional
in the current release.

Evidence: `evidence/TC-SKILL-011-db.txt`, `evidence/TC-SKILL-011-project-with-provider-after.png`

### TC-SKILL-004 — FAIL (T0) [Phase 2 waiver accepted]

Switch mode action is absent from the Project Detail skill entry Actions column.
Only Remove is available. No UI mechanism exists to switch an install between
symlink and copy mode. Residual risk: copy mode installs cannot be created or
managed. Owner-accepted Phase 2 waiver — recording actual status per run-plan.

Evidence: `evidence/TC-SKILL-004-no-switch-mode.png`

### TC-SKILL-012 — FAIL (T0)

Same root cause as TC-SKILL-004. Switch mode action absent; symlink↔copy round-trip
cannot be tested. No UI mechanism for mode switching.

### TC-FS-003 — ~~FAIL~~ PASS after P0 fix (T0)

**Original (2026-06-02):** Install into read-only provider skills folder (chmod 555) fails at OS level
(filesystem_error: permission denied in operations table) but the error was not
surfaced to the user in the wizard. Wizard remained in pre-install state; no
error message was shown. Safety invariant holds (no active install, no partial
target). UX invariant violated: user saw no feedback on why install failed.

Original evidence: `evidence/TC-FS-003-wizard-after-fail.png`, `evidence/TC-FS-003-db-and-fs.txt`

### TC-SETUP-003 — ~~FAIL~~ PASS after P0 fix (T0)

**Original (2026-06-02):** After re-pointing host from host-a to host-b, Project Detail showed existing
host-a install (alpha-skill) as "Linked to active host" instead of stale/old-host state.
DB install_status remained `current` instead of being marked stale/missing.

Original evidence: `evidence/TC-SETUP-003-{settings-after,host-skills-after,project-detail-after,dashboard-after}.png`, `evidence/TC-SETUP-003-db.txt`

### TC-DASH-003 — ~~FAIL~~ PASS (QA artifact) (T0)

**Original (2026-06-02):** Reset appeared not to clear DB; however this was a QA harness
artifact — the two-stage confirmation was not completed correctly (stage-2 RESET textbox
was not typed). The reset RPC was not actually called. No product bug.

Original evidence: `evidence/TC-DASH-003-{reset-dialog,after-reset-click,post-reset}.png`, `evidence/TC-DASH-003-post-reset-db.txt`

### TC-MIGRATE-005 — ~~FAIL~~ PASS after P0 fix (T0)

**Original (2026-06-02):** Partial interrupted migration (dirty DB version=23) correctly detected by Go core:
"migrate up: Dirty database version 23. Fix and force version." App exited after FATAL
timeout on both launches silently with no user-visible error screen. Same root cause as TC-MIGRATE-002.

Original evidence: `evidence/TC-MIGRATE-005-launch-log.txt`, `evidence/TC-MIGRATE-005-relaunch-log.txt`

### TC-A11Y-002 — ~~BLOCKED~~ PASS after A11Y rerun (T0)

**Original (2026-06-02):** Partial evidence collected (Reset All Data two-stage confirmation verified, cancel leaves
DB unchanged). Remove Project flow triggered a native Electron/macOS confirmation sheet
(dialog.showMessageBox) which blocked all CDP commands (Runtime.evaluate, snapshot,
screenshot) while open. Sheet dismissed before native-dialog playbook rule was established.
Harness must use osascript to operate native sheets. Test incomplete; no Remove Project
cancel flow or Remove Skill flow executed.

Original evidence: `evidence/TC-A11Y-002-{initial,reset-cancelled,reset-dialog,reset-textbox-required,projects-list}.png`, `evidence/TC-A11Y-002-db-before.txt`

**Delta (2026-06-03):** PASS. The rerun used a clean run-local Claude-only
fixture and PID-scoped `osascript` for native sheets. Remove Skill cancel,
Remove Project cancel, and Reset All Data stage-2 cancel all preserved DB and
filesystem state.

### TC-OPS-007 — BLOCKED (T0)

Blocked by TC-SKILL-004/TC-SKILL-011 FAILs. Copy (rsync) install mode is not available
in the current release: app ignores default_install_mode=rsync_copy; UI has no mode
picker; all installs create symlinks regardless. Symlink installs complete in milliseconds
and cannot be intercepted mid-operation to test restart recovery.

Evidence: `evidence/TC-OPS-007-blocked-evidence.txt`

## Known Human Gates

- Signed/notarized release verification remains `NEEDS_HUMAN` until Apple
  credentials are available.
- Unsigned macOS packaged smoke has executed against
  `apps/desktop/dist/astraler-skillbox-0.1.2-arm64.dmg`.
- Outbound network observations or manual update checks require explicit owner
  approval at execution time.
- `TC-SKILL-004` has an owner-accepted Phase 2 waiver for current release
  confidence; execute and record the actual status/residual risk rather than
  silently treating it as covered.

## Original Cases Run Before Delta

| Case | Tier | Status |
|------|------|--------|
| TC-SETUP-001 | T1 | PASS |
| TC-SKILL-001 | T1 | PASS |
| TC-SKILL-002 | T0 | PASS |
| TC-SKILL-003 | T0 | PASS |
| TC-FS-006 | T0 | PASS |
| TC-SETTINGS-002 | T0 | PASS |
| TC-DB-003 | T0 | PASS |
| TC-DB-005 | T0 | PASS |
| TC-DB-001 | T0 | PASS |
| TC-OPS-004 | T0 | PASS |
| TC-MIGRATE-002 | T0 | **FAIL** |
| TC-PRIVACY-001 | T0 | PASS |
| TC-PRIVACY-002 | T0 | PASS |
| TC-PROVIDER-002 | T0 | PASS |
| TC-PROVIDER-004 | T0 | PASS |
| TC-PROJ-003 | T0 | PASS |
| TC-SKILL-005 | T0 | PASS |
| TC-SKILL-007 | T0 | PASS |
| TC-SKILL-011 | T0 | **FAIL** |
| TC-SKILL-004 | T0 | **FAIL** |
| TC-SKILL-012 | T0 | **FAIL** |
| TC-FS-001 | T0 | PASS |
| TC-FS-002 | T0 | PASS |
| TC-FS-003 | T0 | **FAIL** |
| TC-DB-002 | T0 | PASS |
| TC-OPS-005 | T0 | PASS |
| TC-OPS-006 | T0 | PASS |
| TC-SETUP-003 | T0 | **FAIL** |
| TC-MIGRATE-003 | T0 | PASS |
| TC-DASH-003 | T0 | **FAIL** |
| TC-A11Y-002 | T0 | ~~BLOCKED~~ **PASS after A11Y rerun** |
| TC-GLOBAL-003 | T0 | PASS |
| TC-PROVIDER-006 | T0 | PASS |
| TC-PROVIDER-011 | T0 | PASS |
| TC-MIGRATE-005 | T0 | **FAIL** |
| TC-OPS-007 | T0 | **BLOCKED** |

Append one result object per case to `results.jsonl` during execution.

## Harness Notes (this batch)

- **Native Electron/macOS sheets**: Any destructive action that uses `dialog.showMessageBox` opens a native sheet outside the React DOM. While open, CDP `/json/list` may stay healthy but `Runtime.evaluate`, `snapshot`, `screenshot`, and `click` all hang. Use `osascript` / System Events to inspect or dismiss. See `docs/playbooks/agent-browser-smoke.md` for the canonical workaround.
- **agent-browser snapshot on large trees**: Compact snapshot (`-c -d 3`) also hangs when a native sheet is blocking CDP. Use `agent-browser eval` (which bypasses accessibility tree) for DOM state checks when snapshot is unavailable.

## Environment

- QA home: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/qa-home`
- QA DB: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/qa-home/qa.db`
- Fixture copy: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/fixtures`
- Evidence: `/Users/tranthien/Documents/2.DEV/2.PRIVATE/astraler-skillbox/docs/qa/runs/2026-06-02-release-full-main-b065fd6/evidence`
- CDP port: `49222`
- Real environment allowed: `false`

## Follow-Up

- Create `evidence/`, `fixtures/`, and `qa-home/` locally before execution.
- Copy `fixtures/qa/` into the run-local `fixtures/` folder before mutating any fixture state.
- Run automated gates first, then T0, then T1, then release-full T2, then packaged smoke when artifact inputs are ready.
- If a case is unclear or unsafe, mark it `BLOCKED` or `NEEDS_HUMAN` and update the QA bank with the smallest durable fix.
