# T0 Hardening Rerun - main 770592c

Date: 2026-06-01
Run: 2026-06-01-t0-hardening-main-770592c
Parent run: 2026-06-01-release-full-rerun-main-770592c

## Verdict

NO-GO for release until the new T0 filesystem escape failure is fixed. This rerun converted the prior T0 blockers into concrete outcomes: 5 PASS, 2 FAIL, 1 BLOCKED.

## Critical Findings

1. TC-FS-001 FAIL: host external symlink escape can be installed. The host scan exposes `evil-host-symlink` as `available`; after creating the target provider folder, `install.skill` succeeds. The project symlink readlink points at the host symlink, but realpath resolves outside the active host folder. Evidence: `evidence/TC-FS-001-rerun-with-target-dir.json`.

2. TC-A11Y-002 FAIL: destructive confirmations exist and cancel/confirm behavior is safe in the tested fixture, but the custom Remove Skill confirmation has `dialogRoleCount: 0`. It should expose dialog semantics such as `role="dialog"`, `aria-modal="true"`, and an accessible label/description. Evidence: `evidence/a11y-remove-skill-dialog-dom.json`.

3. TC-SKILL-004 BLOCKED: copy/rsync is not implemented in the current install surface. `install.skill` accepts only `projectId/providerKey/skillIds`, and the install service writes symlinks only. If copy/rsync is a release requirement, this should become a product FAIL; otherwise re-tier/update the QA bank as future scope.

## Passed

- TC-FS-002: remove refuses externally-mutated project symlink.
- TC-GLOBAL-003: global external symlink is classified and warned, read-only.
- TC-MIGRATE-002: invalid QA DB does not fall back to real app data.
- TC-MIGRATE-003 and TC-OPS-004: stale running/queued operations are failed on restart.

## Evidence

Primary evidence lives under `docs/qa/runs/2026-06-01-t0-hardening-main-770592c/evidence/`.
