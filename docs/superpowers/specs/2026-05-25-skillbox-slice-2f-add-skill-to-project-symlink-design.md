# Slice 2F: Add Skill To Project (Symlink MVP) — Design

Date: 2026-05-25
Status: draft
Scope: install one or more active Skill Host skills into a project provider as symlinks, with conflict validation and a post-install rescan.

## Purpose

Slices 2A–2E gave Skillbox read-only project scanning, provider detection
(`generic_agents`, `claude`), and read-only install-target metadata. Slice 2F is
the first **write** slice: it lets a user pick skills from the active Skill Host
Folder and install them into a project's provider skills path as symlinks.

This is the minimum viable install. It establishes the install write path,
operation handling, conflict validation, and the filesystem-gateway write surface
that later slices (rsync/copy, remove, relink, switch mode, global installs) build
on. Symlink is the only mode in this slice.

## Decisions

- New command `install.skill` (long-running, returns `operationId`).
- One call may install multiple skills (`skillIds: []`), matching the Add Skill
  wireframe. The install is **fail-fast and atomic on conflict**: if any selected
  skill's target path is already occupied, the whole install aborts before any
  symlink is created.
- Install target is resolved from the read-only `InstallTarget` metadata added in
  Slice 2E: `generic_agents → .agents/skills`, `claude → .claude/skills`. The
  target skills path is derived deterministically as
  `join(project.Path, InstallTarget.RelativeSkillsPath)`.
- Symlink target is the **absolute** path to the source skill inside the active
  Skill Host Folder (`skills.absolute_path`).
- After symlinks are created, the install operation runs an **inline rescan** of
  the project (reusing the existing scan/classify logic). The `installs` table is
  reconciled by that rescan — install records are *derived by scan classification*,
  not hand-written by the install service. This reuses the same tested
  classification the rest of the app relies on and avoids divergent write logic.
- The operation's target is `project:<projectId>`, so an install and a scan on the
  same project mutually exclude through the existing per-target operation lock.
- All filesystem writes go through `filesystem.Gateway`. No service calls
  `os.Symlink`/`os.MkdirAll` directly.

## In Scope

- `install.skill` JSON-RPC command + handler + contract schema.
- `InstallService` (or an install method group on a service) that validates,
  creates directories/symlinks via the gateway, then triggers an inline rescan.
- Filesystem gateway write methods: `LstatExists`, `EnsureDir`, `CreateSymlink`.
- Provider-target resolution and validation for `generic_agents` and `claude`,
  reusing Slice 2E `InstallTarget` metadata and `provider_definitions`
  capabilities (`status`, `can_create_structure`).
- Conflict validation against existing entries (lstat-based, so broken symlinks
  and real directories both count as conflicts).
- Minimal Add Skill UI wizard: select skills, (auto-)select provider, confirm,
  invoke `install.skill`, surface success/error.
- Tests: Go service + gateway, contract drift, minimal renderer invoke test.

## Out Of Scope

- `rsync_copy` and `direct` install modes.
- Remove, relink, switch-install-mode, sync, update, fetch.
- Global / user-level installs.
- Replace / skip / overwrite on conflict (conflict is a hard block this slice).
- Renaming or repurposing provider keys.
- Persisting `InstallTarget` IDs as provider identity.
- New providers beyond the seeded `generic_agents` and `claude`.
- Any RPC method beyond `install.skill`.
- Recording `installed_version` / `installed_commit` / `installed_checksum`
  (left null for symlink MVP; populated by future fetch/update slices).

## Architecture And Flow

The command is a write operation that ends in a read reconciliation:

```text
install.skill(projectId, providerKey, skillIds)
  -> validate request shape (non-empty skillIds, known providerKey)
  -> load project; must exist and be status=active
  -> resolve project_provider row for (projectId, providerKey)
       must be present with detection_status in {detected, configured}
       provider_definitions.status in {supported, experimental}
  -> resolve target skills path:
       skillsPath = NormalizeAbs(join(project.Path, InstallTarget.RelativeSkillsPath))
       must canonicalize to a path inside the project root
  -> load each requested skill by id from the ACTIVE host
       skill must exist, belong to the active host, status=available
       source = skill.absolute_path
  -> conflict validation (fail-fast):
       for each skill: linkPath = join(skillsPath, skill.Name)
       if gateway.LstatExists(linkPath) -> collect conflict
       if any conflicts -> abort with conflict_error (no writes done)
  -> ensure skills path exists:
       if missing and provider can_create_structure=1 -> gateway.EnsureDir(skillsPath)
       if missing and can_create_structure=0 -> provider_error (no scaffold)
  -> create symlinks:
       for each skill: gateway.CreateSymlink(source=skill.absolute_path, link=linkPath)
       stop on first filesystem error (best-effort; already-created links remain,
       rescan will reconcile)
  -> inline rescan:
       call scanProjectInternal(project) directly (already inside the operation,
       no new lock) to refresh project_providers + installs from filesystem truth
  -> return summary metadata { installed, requested, providerKey }
```

### Why install records come from rescan

`scanProjectInternal` already classifies a symlink entry into an `installs` row:
`install_mode=symlink`, `install_status=current` when it points into the active
host, plus `source_skill_path`, `symlink_target_path`, and
`installed_from_host_folder_id`. Re-running it after creating the symlinks means
the install path produces exactly the same records as a manual scan would. The
install service therefore performs filesystem writes only; metadata persistence is
the rescan's job. This keeps a single source of truth for install classification
and means an install is "correct" iff a scan would classify it as `current`.

### Operation lifecycle

`install.skill` uses the existing `operations.Runner.Start` with
`Target{Type: "project", ID: projectId}` and a new operation type
`OperationTypeInstallSkill = "install_skill"`. Because scan uses the same target,
the runner's per-target lock prevents a concurrent scan and install on one
project, returning `conflict_error` to whichever loses. Progress phases:
`validating`, `creating_symlinks`, then the rescan phases (`detecting_providers`,
`classifying_entries`, `done`).

## Filesystem Gateway Additions

All new methods live on `filesystem.Gateway` and delegate to package functions, in
the existing style. They are the only new write surface in this slice.

```text
LstatExists(path string) (bool, error)
  // os.Lstat-based; true if any entry exists at path WITHOUT following symlinks,
  // so a broken or external symlink still counts as occupying the path.

EnsureDir(path string) error
  // os.MkdirAll(path, 0o755). Caller (service) is responsible for the
  // can_create_structure gate; the gateway just creates the directory.

CreateSymlink(source, linkPath string) error
  // os.Symlink(source, linkPath). source is the absolute host skill path,
  // linkPath is the entry inside the provider skills path.
```

Path-safety validation (canonicalize and confirm the link's parent resolves
inside the project root) lives in the install service using existing
`NormalizeAbs` / `Realpath`, not in the gateway.

## Provider Target Resolution

Reuse Slice 2E `providers.InstallTargetByProviderKey(providerKey)` to map a
provider key to its relative skills path and display name. Validation combines
that with the persisted `project_providers` row and `provider_definitions`
capabilities:

- `generic_agents`: `status=supported`, `can_create_structure=1` → may scaffold
  `.agents/skills` if absent.
- `claude`: `status=experimental`, `can_create_structure=0` → install only when
  `.claude/skills` already exists. If absent, return `provider_error`
  ("Claude skills folder does not exist and cannot be created automatically").
  This is an accepted limitation of this slice, not a bug.

A provider target is installable only when all hold: project provider row exists;
`detection_status ∈ {detected, configured}`; `provider_definitions.status ∈
{supported, experimental}`; resolved skills path is inside the project root; and
either the skills path exists or the provider may scaffold it.

## JSON-RPC Contract

Method: `install.skill` (command).

Request:

```json
{
  "projectId": 12,
  "providerKey": "generic_agents",
  "skillIds": [3, 7]
}
```

- `projectId`: integer, required.
- `providerKey`: string, required, one of `generic_agents` | `claude`.
- `skillIds`: array of integers, required, min length 1, unique.

Response (immediate, operation queued):

```json
{ "operationId": 45 }
```

Terminal result is delivered through the existing operation result/metadata
channel; `metadata_json` carries:

```json
{ "installed": 2, "requested": 2, "providerKey": "generic_agents" }
```

Add the schema at `shared/api-contracts/methods/install.skill.json` and register
the method in the Electron main allowlist and `app.New` handler map. Renderer
core-client gains an `installSkill(req)` wrapper mirroring `projectScan`.

## Error Handling

| Condition | Code |
|---|---|
| Missing/invalid params, empty or non-unique `skillIds`, unknown `providerKey` | `validation_error` |
| Project not found or not `active` | `validation_error` |
| Provider not present in project, or `detection_status` not installable | `validation_error` |
| Skill id not found, not on the active host, or not `available` | `validation_error` |
| Provider `status` not in {supported, experimental} | `provider_error` |
| Skills path absent and `can_create_structure=0` (Claude) | `provider_error` |
| Resolved target path escapes project root | `validation_error` |
| One or more target paths already occupied | `conflict_error` (lists colliding names) |
| Another operation already running on this project | `conflict_error` (from runner) |
| `EnsureDir` / `CreateSymlink` failure | `filesystem_error` |

Conflict validation runs **before** any write, so a `conflict_error` guarantees no
symlink was created. A `filesystem_error` mid-loop may leave earlier symlinks in
place; the inline rescan still runs so the DB reflects whatever exists on disk.

## UI (Minimal Wizard)

Entry point: the existing `[Add Skill]` action on Project Detail. The wizard
follows the wireframe but only the symlink path is live this slice:

```text
Add Skill

Step 1: Select Skills          (multi-select from active host skill.list)
Step 2: Select Provider        (auto-select if one installable provider;
                                otherwise choose among installable providers;
                                unsupported/disabled not selectable)
Step 3: Confirm                (mode is symlink, shown read-only)
  Install N skills into <project> / <provider display name> using symlink.
  Affected paths: <skillsPath>/<name> ...
  [Install] [Cancel]
```

- Mode selection step from the wireframe is collapsed to a fixed "symlink" label
  this slice.
- On `[Install]`: invoke `install.skill`, show progress via the existing
  operation progress channel, then invalidate the project detail query so the new
  installs appear after the rescan.
- On `conflict_error`: surface the colliding skill names and let the user adjust
  the selection (no replace/skip flow this slice).
- On `provider_error` for Claude-without-skills-folder: show the explanatory
  message; offer no auto-create action.

## Testing Strategy

Go (`go test ./...`, with `-race` on the write paths):

- Service happy path: install into existing `.agents/skills`, rescan yields a
  `symlink` / `current` install row pointing at the active host.
- Auto-create: `.agents/skills` absent + `generic_agents` → directory created,
  symlink made, install classified.
- Claude block: `.claude/skills` absent + `claude` → `provider_error`, no writes.
- Conflict abort: pre-existing entry (real dir AND broken symlink cases) → atomic
  `conflict_error`, filesystem unchanged.
- Validation: unknown provider, provider not detected in project, empty/duplicate
  `skillIds`, skill on inactive host, skill not `available`, project not active.
- Within-root enforcement: a crafted relative path cannot escape project root.
- Operation lock: install vs concurrent scan on the same project → one
  `conflict_error`.
- Multi-skill: two skills installed in one call; metadata `installed == requested`.

Gateway: `LstatExists` (missing / dir / file / broken symlink), `EnsureDir`
(idempotent), `CreateSymlink` (success + existing-target error), using temp dirs.

Contracts: `pnpm check:contracts-drift` covers the new method; generated TS types
include `installSkill`.

Renderer (Vitest): core-client `installSkill` calls `invoke("install.skill", …)`
with the right params; wizard confirm step triggers the invoke. No full-stack UI
assertion beyond the invoke contract this slice.

## Risks

- **Rescan-derived records.** Install success depends on scan classification being
  correct. Mitigated by reusing the existing, tested scan path rather than a
  parallel write path; a symlink that scan would not classify as `current` is, by
  definition, not a healthy install.
- **Claude scaffold limitation.** `can_create_structure=0` means Claude installs
  fail when `.claude/skills` is absent. Accepted and surfaced clearly; revisited
  when Claude convention is finalized (see provider-model open questions).
- **Partial multi-skill failure.** A filesystem error after some symlinks are
  created leaves them on disk. The inline rescan reconciles the DB to disk truth;
  the user can re-run install for the remaining skills (already-present ones now
  conflict and are reported).
- **Symlink portability.** Absolute targets break if the host folder moves; this
  is consistent with current scan classification (`old_host` / `broken_symlink`
  handling) and is a Skill-Host-move concern, not an install concern.

## Acceptance Criteria

- `install.skill` installs selected active-host skills into a project's
  `generic_agents` or `claude` skills path as symlinks, then rescans the project.
- After a successful install, project detail shows the new skills as
  `mode=symlink`, `status=current`, grouped under the correct provider.
- Conflict validation is atomic: a `conflict_error` leaves the filesystem
  unchanged and names the colliding skills.
- Installing into `generic_agents` auto-creates `.agents/skills` when absent;
  installing into `claude` without `.claude/skills` returns `provider_error`
  without writing.
- All filesystem writes go through `filesystem.Gateway`; no direct `os.Symlink` /
  `os.MkdirAll` in services.
- Install and scan on the same project mutually exclude via the operation lock.
- No out-of-scope behavior (rsync/copy, remove, relink, switch mode, global,
  replace-on-conflict) is introduced.
- Tests cover happy path, auto-create, Claude block, conflict abort, validation
  failures, within-root enforcement, lock conflict, and gateway writes.
