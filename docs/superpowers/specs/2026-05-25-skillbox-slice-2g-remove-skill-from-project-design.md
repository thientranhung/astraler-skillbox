# Slice 2G: Remove Skill From Project (Symlink MVP) — Design

Date: 2026-05-25
Status: draft
Scope: remove a single project-level **symlink** install whose target resolves
inside a known Skill Host Folder, by deleting the symlink and reconciling the DB.
Never deletes Skill Host Folder source content.

## Purpose

Slice 2F gave Skillbox its first write path: install active-host skills into a
project provider as symlinks (`install.skill`), ending in an authoritative
rescan. Slice 2G is the inverse: let a user remove one installed skill from a
project/provider.

This is the **minimum safe** remove. It only deletes a symlink that is verified
(at execution time, not from stale DB state) to point inside the **active** Skill
Host Folder. That guarantee — "we only ever unlink a pointer into managed host
content" — is what keeps the slice safe: it can never delete real files. Removing
`old_host`, `direct` installs (real folders), `external_symlink`, or
`broken_symlink` entries is deferred, because those either hold real content,
point into an inactive host, or point somewhere we cannot prove is
managed.

## Key Difference From Slice 2F

2F treats install records as **purely rescan-derived**: it writes symlinks, then
re-runs `scanProjectInternal`, and the rescan reconciles the `installs` table.

Remove cannot use that model alone. `CommitProjectScan` (the scan commit) never
hard-deletes install rows — it **tombstones** entries absent from disk as
`install_status='missing'` (see `markAbsentInstallsMissing`). The project-detail
renderer shows `missing` as a visible red "Missing" badge. So a pure-rescan
remove would leave the removed skill on screen as a "Missing" ghost, not gone.

Therefore remove keeps the rescan as the authority for everything else (provider
detection, sibling installs, entry counts, warnings) and adds **one targeted
hard-delete** of the selected install row, by id, after the rescan. That targeted
delete is the only behavioral addition beyond the 2F pattern, and the only place
installs are ever hard-deleted.

## Decisions

- New command `remove.skill` (long-running, returns `operationId`).
- Request identifies the install by **`installId`** (plus `projectId`), not by
  `(providerKey, skillName)`. The install row's id is stable across rescans
  (`upsertInstall` is `ON CONFLICT(project_provider_id, project_skill_path) DO
  UPDATE`, preserving the id), is unambiguous when the same skill name exists
  under two providers, and is already exposed to the renderer by `project.get`
  (`projectGetEntry.ID`).
- **Removable set (MVP):** `install_mode = symlink` AND `install_status =
  current`. This is exactly a symlink whose target resolves inside the **active**
  Skill Host Folder. Everything else is **not** auto-removed this slice:
  `old_host` (symlink into an inactive/old host — deferred, see Out Of Scope),
  `direct` (real content), `external_symlink` (resolves outside all known hosts),
  `broken_symlink`, `error`.
- Remove **re-verifies on disk** before unlinking and does not trust the stored
  classification (disk may have changed since the last scan). See "On-disk
  re-verification".
- The symlink is deleted with `os.Remove` via the gateway (`RemoveSymlink`),
  which unlinks the symlink **without following it** and refuses to recurse into
  a real directory. This is the only new write surface this slice.
- After the unlink, remove runs an **inline authoritative rescan**
  (`scanProjectInternal`) to reconcile providers/installs/warnings/counts from
  disk truth (the removed path becomes `missing`), then **hard-deletes the one
  install row by id** so the skill leaves the installed list.
- Operation target is `project:<projectId>`, so remove, scan, and install on the
  same project mutually exclude through the existing per-target operation lock.
- All filesystem and DB writes go through the gateway / repositories; no service
  calls `os.Remove` directly.

## In Scope

- `remove.skill` JSON-RPC command + handler + contract schema.
- Remove method on the project service (alongside `InstallSkills`): validate →
  re-verify on disk → `RemoveSymlink` → inline rescan → targeted row delete.
- Filesystem gateway write method: `RemoveSymlink(path string) error`
  (`os.Remove`).
- Repository method: `InstallRepo.DeleteByID(ctx, installID)` — single-row hard
  delete (the only hard delete of an install row in the app).
- On-disk re-verification (lstat + resolve, reusing existing `isWithin` /
  resolve helpers and host summaries) so removal only ever targets a symlink
  resolving inside the active host.
- Minimal Project Detail UI: wire the existing per-row `[Remove]` action **only**
  for removable symlink entries; a confirmation dialog showing provider, skill,
  and exact path; toast on success/failure; query invalidation on terminal
  operation result.
- Tests: Go service + gateway + repo, contract drift, renderer hook/confirm test.

## Out Of Scope

- Removing `old_host` installs (symlinks into an inactive/old Skill Host Folder).
  Unlinking one is equally safe as a `current` removal, but it is deferred to keep
  the MVP's removable set to a single, unambiguous state — a symlink into the
  **active** host. Old-host remediation pairs naturally with relink/change-host
  flows and is a follow-up.
- Removing `rsync_copy` / `direct` installs (these delete real folder content;
  a future slice with a stronger "delete N files" confirmation).
- Force-deleting a real directory or file at the install path.
- Removing `external_symlink` or `broken_symlink` entries (the wireframe
  `[Relink]/[Remove]` remediation actions) — future remediation slice.
- Relink.
- Global / user-level remove (Global Skills `[Remove]`).
- Bulk / multi-select remove.
- Switch install mode, sync, update, fetch.
- Cleaning up externally-`missing` install rows (rows already tombstoned by a
  prior scan because they vanished outside Skillbox) — separate remediation;
  remove only targets a user-selected removable symlink.
- Any RPC method beyond `remove.skill`.
- Deleting Skill Host Folder source content — an invariant, never a feature.

## Architecture And Flow

```text
remove.skill(projectId, installId)
  -> validate params (projectId > 0, installId > 0)            -> validation_error
  -> load project; must exist and status=active                -> validation_error
  -> load install by id; must exist AND belong to project      -> validation_error
       (join installs -> project_providers -> project)
  -> removable precheck (DB-level fast reject):                -> validation_error
       install_mode == symlink AND install_status == current
  -> resolve path = install.project_skill_path
       must be inside the project root (NormalizeAbs/Realpath)  -> validation_error
  -> start operation: Target{project, projectId}, type remove_skill
       (per-target lock: concurrent scan/install -> conflict_error)

  [validating]  on-disk re-verification (do NOT trust DB):
       lstat(path):
         - missing            -> idempotent no-op: skip unlink, set alreadyAbsent=true
         - not a symlink      -> conflict_error (entry changed on disk; rescan & retry)
         - symlink:
             resolve target; if it resolves inside the ACTIVE host -> proceed
             else (broken / inactive host / outside all hosts now) -> conflict_error (stale)

  [removing_symlink]  if not alreadyAbsent: gateway.RemoveSymlink(path)
       on failure (permission, etc.) -> operation FAILED, filesystem_error;
       NO rescan, NO row delete (nothing changed; state preserved)

  [rescan phases]  scanProjectInternal(project)  (authoritative reconcile;
       reuses the operation, no new lock; removed path becomes install_status=missing)

  [deleting_record]  installRepo.DeleteByID(installId)
       removes the now-missing tombstone so the skill leaves the installed list

  [done]  complete SUCCESS with metadata
       { projectId, providerKey, skillName, removedPath, alreadyAbsent }
```

### On-disk re-verification

The stored classification can be stale: between the last scan and the remove, a
process outside Skillbox could replace the symlink with a real directory, delete
it, or repoint it outside the host. Removing based on stale DB state could delete
real content. So at execution, before any unlink, the service re-`lstat`s the
path and:

- **missing** → idempotent: the desired end state (not installed) already holds;
  skip the unlink and continue to rescan + row delete. `alreadyAbsent=true`.
- **not a symlink** (real dir/file now) → `conflict_error`; never unlink/delete
  real content. The user rescans and re-decides.
- **symlink resolving inside the active host** → the only state that proceeds
  (this is what the scan classifier records as `current`).
- **symlink that is broken, resolves into an inactive/old host, or resolves
  outside all known hosts now** → `conflict_error` (it is no longer a
  `current` install — outside MVP scope).

This reuses the same resolve / `isWithin(activeHostSkillsPath, resolved)` logic
the scan classifier uses to produce `current`, evaluated against the current
active host, so "resolves inside the active host" means the same thing at remove
time as at scan time.

### Why rescan, then delete by id

Rescan first so all sibling installs, provider detection, entry counts, and
warnings are reconciled from disk truth in one authoritative pass; the removed
path is tombstoned to `missing` by `markAbsentInstallsMissing`. Then
`DeleteByID(installId)` removes that single tombstone. `DeleteByID` is
delete-by-primary-key and harmless if the row is already gone. The rescan stays
the authority for everything except the one row the user asked to remove.

### Operation lifecycle and partial-failure semantics

`remove.skill` uses the existing `operations.Runner.Start` with
`Target{Type: "project", ID: projectId}` and a new
`OperationTypeRemoveSkill = "remove_skill"`. Because scan and install share the
same target, the per-target lock serializes them; the loser gets `conflict_error`.

Failure points and resulting state:

- **Unlink fails** (e.g. permission): operation FAILED with `filesystem_error`;
  the rescan and row delete are skipped. Nothing changed on disk or in the DB —
  the entry stays exactly as it was. Safe to retry.
- **Rescan fails** after a successful unlink (DB error): operation FAILED with
  `database_error`. The symlink is gone but the install row was not deleted, so
  it lingers (as `current`) until the next manual scan tombstones it.
  Rare; accepted and documented. Recovery: rescan the project.
- **Row delete fails** after a successful unlink + rescan (DB error): operation
  FAILED with `database_error`. The row is currently `missing` (tombstoned by the
  rescan) and shows as a "Missing" badge until the next scan or a retried remove
  clears it. Rare; accepted and documented.

The terminal operation result (success or failed) is emitted **only after** the
rescan and targeted delete complete — both run inside the same work function — so
the renderer's invalidate-on-terminal-result never reads a half-reconciled state.

Progress phases: `validating`, `removing_symlink`, then the rescan phases
(`detecting_providers`, `classifying_entries`), `deleting_record`, `done`.

## Filesystem Gateway Addition

One new method on `filesystem.Gateway`, delegating to a package function in the
existing style (mirrors the 2F additions `LstatExists` / `EnsureDir` /
`CreateSymlink`). It is the only new write surface this slice.

```text
RemoveSymlink(path string) error
  // os.Remove(path). On a symlink this unlinks the link itself WITHOUT
  // following it (the target is untouched). On a real non-empty directory
  // os.Remove returns an error rather than recursing — defense in depth so a
  // regression in the caller's checks can never destroy real content.
```

The service only calls `RemoveSymlink` after re-verifying the on-disk entry is a
symlink resolving inside the active host. Path-safety (confirm the path is inside
the project root) lives in the service using existing `NormalizeAbs` / `Realpath`,
not in the gateway.

## Repository Addition

```text
InstallRepo.DeleteByID(ctx, installID int64) (rowsAffected int64, error)
  // DELETE FROM installs WHERE id = ?. The only hard delete of an install row
  // in the app. Idempotent: deleting an already-absent row affects 0 rows and
  // is not an error.
```

## JSON-RPC Contract

Method: `remove.skill` (command).

Request:

```json
{ "projectId": 12, "installId": 88 }
```

- `projectId`: integer, required.
- `installId`: integer, required (the `id` of the installed-skill row from
  `project.get`).

Response (immediate, operation queued):

```json
{ "operationId": 51 }
```

Terminal result is delivered through the existing operation result/metadata
channel, **after** the rescan and row delete complete:

```json
{
  "projectId": 12,
  "providerKey": "generic_agents",
  "skillName": "documentation-writer",
  "removedPath": "/repo/content-lab/.agents/skills/documentation-writer",
  "alreadyAbsent": false
}
```

`alreadyAbsent: true` means the on-disk entry was already gone at remove time and
only the DB row was cleaned up (still a SUCCESS).

Add the schema at `shared/api-contracts/methods/remove.skill.json`, register the
method in the Electron main allowlist and the `app.New` handler map (mirroring
`install.skill`), and add a renderer core-client `removeSkill(req)` wrapper
mirroring `installSkill`.

## Error Handling

| Condition | Code |
|---|---|
| Missing/invalid params (`projectId`/`installId` absent or ≤ 0) | `validation_error` |
| Project not found or not `active` | `validation_error` |
| Install id not found, or not belonging to this project | `validation_error` |
| Install not removable (`mode ≠ symlink`, or `status ≠ current`) | `validation_error` |
| Resolved path escapes the project root | `validation_error` |
| On-disk entry changed: now a real dir/file, or a symlink no longer resolving inside the active host | `conflict_error` (stale; rescan & retry) |
| Another operation already running on this project | `conflict_error` (from runner) |
| `RemoveSymlink` failure (permission, etc.) | `filesystem_error` (operation FAILED; no rescan, no row delete) |
| Rescan or row-delete DB failure after a successful unlink | `database_error` (operation FAILED; documented lingering-state) |

All validation and the on-disk re-verification run **before** the unlink, so a
`validation_error` or `conflict_error` guarantees nothing was deleted.

## UI (Minimal)

Entry point: the existing per-row `[Remove]` action in Project Detail → Installed
Skills. This slice wires it **only** for removable entries (`mode=symlink` and
`status=current`); for `old_host` / `direct` / `external_symlink` /
`broken_symlink` / `error` entries the `[Remove]` action is not wired this slice
(disabled, with a tooltip that removal of that entry type is not yet supported).

Confirmation dialog (the wireframes require confirming before "Remove skill from
project" and showing the object, the filesystem path, the provider, and that the
action changes only the project install — not the Skill Host Folder):

```text
Remove skill from project

Remove  documentation-writer
from    content-lab / Shared Agent Skills (.agents)

This deletes the symlink at:
  /repo/content-lab/.agents/skills/documentation-writer

The skill in your Skill Host Folder is not affected.

[Remove] [Cancel]
```

- On `[Remove]`: invoke `remove.skill`; show progress via the existing operation
  progress channel. Invalidate the project detail query **only on the terminal
  operation result** (success or failed) — never on intermediate progress —
  because the rescan + row delete that produce the final state run before the
  terminal result is emitted.
- On SUCCESS: toast "Removed *documentation-writer* from Shared Agent Skills."
  (On `alreadyAbsent`, the same success toast — the entry is gone either way.)
- On `conflict_error` (stale disk): toast "This entry changed on disk. Rescan the
  project and try again," and surface the project's Scan action.
- On `filesystem_error` / `database_error`: error toast; the row remains.

### Impact preview

The wireframes list "Remove Skill" under impact preview. At this slice's
granularity the impact is exactly one symlink at one path in one provider, so the
confirmation dialog (which shows that path and provider) **is** the impact
preview. A richer cross-project/global impact table applies to host-level
operations (update host copy, change host folder) and is not applicable to a
single project-scoped symlink removal.

Renderer additions: core-client `removeSkill(req)`; a `useRemoveSkill()` hook
mirroring `useInstallSkill` (buffer progress events, subscribe if not already
terminal, invalidate project queries on the terminal result); a small
confirmation dialog component showing skill / provider / path.

## Testing Strategy

Go (`go test ./...`, with `-race` on the write path):

- Happy path: a `current` symlink install is removed → symlink gone from disk,
  rescan runs, install row hard-deleted, project detail no longer lists it,
  operation SUCCESS with metadata (`alreadyAbsent=false`).
- Idempotent already-absent: DB row exists but the on-disk symlink is already
  gone → no unlink, rescan + row delete still run, SUCCESS with
  `alreadyAbsent=true`.
- Not-removable rejections (no writes): `old_host` symlink, `direct` entry,
  `external_symlink`, `broken_symlink`, `error` status → `validation_error`.
- Install not found / install belongs to another project → `validation_error`.
- Project not found / not active → `validation_error`.
- Path escaping project root (crafted `project_skill_path`) → `validation_error`,
  no unlink.
- On-disk divergence (don't trust DB): DB says `current` but the path is now a
  real directory → `conflict_error`, the real directory is **not** deleted.
- On-disk divergence: symlink now resolves into an inactive/old host or outside
  all known hosts → `conflict_error`, no unlink.
- Unlink failure (injected gateway error) → operation FAILED `filesystem_error`,
  rescan and row delete NOT run, install row and symlink unchanged.
- Operation lock: remove vs concurrent scan/install on the same project → one
  `conflict_error`.

Gateway: `RemoveSymlink` removes a symlink without touching its target; returns
an error (and does not recurse) on a real non-empty directory; idempotent-ish
behavior on a missing path is the caller's concern (service guards via lstat).

Repository: `DeleteByID` deletes one row, returns `rowsAffected`, and is a no-op
(0 rows, no error) for an absent id; does not touch sibling installs.

Contracts: `pnpm check:contracts-drift` covers `remove.skill`; generated TS types
include `removeSkill`.

Renderer (Vitest): core-client `removeSkill` calls `invoke("remove.skill", …)`
with the right params; the confirmation dialog shows skill/provider/path and only
invokes on confirm; the project detail query is invalidated on the terminal
operation result (not on intermediate progress); the `[Remove]` action is
disabled for non-removable entry types.

## Risks

- **Stale classification vs disk.** The DB classification can lag disk. Mitigated
  by re-verifying on disk before unlinking and refusing (`conflict_error`) when
  the entry is no longer a known-host symlink — so remove can never delete real
  content based on stale state. The operation lock serializes Skillbox operations
  but not external processes; a change racing between re-verify and unlink is the
  residual window, and `os.Remove`'s refusal to recurse into a real directory is
  the backstop.
- **Targeted hard delete diverges from the pure-rescan model.** Unlike 2F, remove
  hard-deletes one row. Justified because `CommitProjectScan` only tombstones
  (`missing`) and the UI renders tombstones; the delete is scoped to the single
  user-selected row, by primary key, after the authoritative rescan.
- **Lingering row on post-unlink DB failure.** If the rescan or the row delete
  fails after the symlink is gone, the row lingers (as `current` or `missing`)
  until the next scan. Rare DB-error case; documented; recovered by a rescan.
- **Scope conservatism.** Old-host / direct / external / broken removals are
  common real needs but are deferred to keep the slice's removable set to a single
  unambiguous state (a symlink into the active host) and incapable of deleting
  real content. Surfaced in the UI as a disabled action with an explanatory
  tooltip.

## Open Questions

1. **`old_host` remove follow-up.** MVP restricts the removable set to `current`
   (active host) and defers `old_host` (see Out Of Scope). Should the follow-up
   that adds `old_host` removal land standalone, or be bundled with the
   relink/change-host work it pairs with?
2. **Externally-`missing` rows.** Rows tombstoned `missing` by a prior scan
   (entry vanished outside Skillbox) currently render as a red "Missing" badge
   with no action. Out of scope here, but a future "Clear missing entry" action
   or a UI filter is the natural follow-up. Which?
3. **`broken_symlink` remove.** The wireframe offers `[Remove]` on broken
   symlinks; unlinking a dangling link is safe. Deferred here for conservatism —
   confirm that's acceptable, or fold broken-symlink removal into this slice.
4. **Confirmation UI.** This spec recommends a small custom dialog (so it can show
   the path + provider the wireframe requires); the existing project-remove flow
   uses `window.confirm`. Confirm the custom dialog is preferred.

## Acceptance Criteria

- `remove.skill` removes a `current` **symlink** install (symlink into the active
  host) from a project provider by deleting the symlink, then rescanning the
  project and hard-deleting the targeted install row; the skill disappears from
  Project Detail's installed list.
- Remove never deletes a real directory/file: `old_host`, `direct`,
  `external_symlink`, `broken_symlink`, and `error` entries are rejected with
  `validation_error` and no filesystem write; a path that is no longer a symlink
  into the active host on disk is rejected with `conflict_error` and no delete.
- The Skill Host Folder source is never modified by remove.
- Re-verification happens on disk at execution time; stored classification is not
  trusted for the delete decision.
- An already-absent on-disk entry is removed idempotently (SUCCESS,
  `alreadyAbsent=true`) by cleaning up the DB row.
- A `RemoveSymlink` failure leaves disk and DB unchanged and completes the
  operation FAILED with `filesystem_error`.
- Remove, scan, and install on the same project mutually exclude via the operation
  lock; the loser gets `conflict_error`.
- The terminal operation result is emitted only after the rescan and row delete
  complete, and the UI invalidates project detail on that terminal result.
- All filesystem writes go through `filesystem.Gateway.RemoveSymlink`; the only
  hard delete of an install row goes through `InstallRepo.DeleteByID`.
- No out-of-scope behavior (old_host, rsync/copy or direct delete, external/broken
  removal, relink, switch mode, sync, update, global remove, bulk remove) is
  introduced.
- Tests cover happy path (`current`), idempotent already-absent, not-removable
  rejections (including `old_host`), on-disk divergence (real dir not deleted;
  symlink into inactive host or outside hosts), path-escape rejection, unlink
  failure, lock conflict, gateway `RemoveSymlink`, and repo `DeleteByID`.
