# Data Model Follow-Up Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: docs/01-product-brief.md, docs/02-product-notes.md,
  docs/03-information-architecture.md, docs/04-user-flows.md,
  docs/05-edge-cases-and-ux-states.md, docs/06-data-model.md,
  docs/review-results/data-model-review.md
- Prior review: data-model-review.md (same date, same session)

---

## Decision

**Approved.**

All four critical blockers from the prior review are resolved. The suggested
improvements are substantially addressed. The model is ready for implementation
to begin.

---

## Critical Blocker Verification

### 1. `install_mode` no longer mixes intent with filesystem anomaly state

**Prior state**: `install_mode` included `external_symlink` and `unknown`.

**Current state**: `install_mode` is now `symlink`, `rsync_copy`, `direct` only.
`external_symlink` has moved to `install_status`. `unknown` is removed entirely.

The Edge Case mapping for External Symlink now correctly reads:
```
installs.install_mode = symlink
installs.install_status = external_symlink
```

The model note explicitly states: "`install_mode` chỉ lưu cơ chế quản lý/install
intent, không lưu detected filesystem anomaly."

**Status: Resolved.**

---

### 2. `projects.status` no longer uses `has_warnings` as a lifecycle state

**Prior state**: `projects.status` had `has_warnings` and `no_provider_detected`.

**Current state**: `projects.status` is now `active`, `missing`, `unreadable`,
`removed` only. The model note explicitly says: "Warning presence và
`no_provider_detected` là derived state từ bảng `warnings`, không nằm trong
`projects.status`."

The `No Provider Detected` edge case mapping is now:
```
warnings.code = no_provider_detected
warnings.scope_type = project
```

**Status: Resolved.**

---

### 3. `fetch_results` relationship is `source_id`-first and no longer ambiguous

**Prior state**: `fetch_results` linked independently to both `skill_id` and
`source_id`.

**Current state**: `fetch_results` has `source_id` as the primary FK only. The
relationship overview shows `skill_sources.id -> fetch_results.source_id` and
nothing else. The model note explicitly addresses the old ambiguity: "Nếu cần
query nhanh theo skill trong implementation, có thể thêm helper denormalized
`skill_id`, nhưng không nên coi nó là FK độc lập."

Additionally, the field rename was applied: `current_version` →
`host_version_at_fetch`, `latest_version` → `upstream_version_at_fetch`, with
equivalent commit fields. This eliminates the point-in-time naming ambiguity.

**Status: Resolved.**

---

### 4. Provider multi-path detection is represented

**Prior state**: `provider_definitions` had a single `default_relative_skills_path`
text field.

**Current state**: A new `provider_path_candidates` table exists with
`provider_definition_id`, `relative_path`, `purpose`, `priority`, `description`.
`provider_definitions` no longer has `default_relative_skills_path`. A dedicated
edge case mapping for "Multi-Path Provider Detection" is present.

The `purpose` enum (`detect`, `skills`, `commands`, `config`) is a useful
addition that allows adapters to reason about which paths are for detection
vs install vs configuration reading.

**Status: Resolved.**

---

## Suggested Improvement Verification

| Improvement from prior review | Applied? | Notes |
|---|---|---|
| `current_checksum` in `skills` | Yes | Field present; notes explain use for local_modified detection and rsync/copy drift. Update Skill Host Folder flow mapping now includes `skills.current_checksum`. |
| Remove redundant `project_id` from `installs` | Yes | Field gone. Model note explains why: `project_provider_id` already implies project. |
| `source_operation_id` in `warnings` | Yes | Field present, nullable, with relationship overview entry `operations.id -> warnings.source_operation_id`. |
| `api_credentials` table | Yes | Full table added as entity #2 with `provider_key`, `credential_type`, `storage_type`, `credential_ref`, `value_encrypted`, `status`. |
| `scan_results` / `operations` overlap acknowledged | Yes | Note in `scan_results` now explicitly says implementation may merge into `operations.metadata_json`. Not forced, left as implementation decision. |
| `active_skill_host_folder_id = NULL` nullable | Yes | Explicitly documented in `app_settings` notes. |
| Hard delete decision for `installs` | Yes | Explicitly documented: "Phase 1 dùng hard delete cho install khi user remove skill bằng Skillbox." No `removed` in `install_status`, which is correct and consistent. |
| Rename `fetch_results` version fields | Yes | See blocker 3 above. |
| `last_successful_fetch_at` in `skill_sources` | Yes | Field present; notes clarify distinction from `last_fetched_at` (includes failed attempts). |
| `not_fetchable` in `skill_sources.last_fetch_status` | Yes | Value present in enum. Notes say "Local/manual source có thể dùng `not_fetchable`." |
| `can_create_structure` in `provider_definitions` | Yes | Field added with explanatory note. |

All suggested improvements are applied or explicitly deferred with documented
rationale.

---

## Remaining Blockers

None.

---

## Non-Blocking Suggestions

### 1. `provider_path_candidates.purpose` relationship to `project_providers` fields is undocumented

The `provider_path_candidates` table introduces a `purpose` enum: `detect`,
`skills`, `commands`, `config`. But `project_providers` has `detected_path`
and `skills_path` as separate fields. It is not documented which `purpose`
value maps to which `project_providers` field after a scan resolves the
candidates.

Suggestion: Add a note to `project_providers` clarifying that `detected_path`
comes from the candidate with the highest-priority `detect`-purpose path that
exists on disk, and `skills_path` comes from the resolved `skills`-purpose
candidate. This will matter when writing the first provider adapter.

---

### 2. No `unknown` classification for scan-discovered items that cannot be categorized

The previous model had `install_mode = unknown` which was correctly removed.
But now there is no way to represent an install-like filesystem entry that
Skillbox genuinely cannot classify. The closest available values are
`install_status = error` (which implies a failure, not ambiguity) or
`install_mode = direct` (which asserts "regular folder"). If a scan finds a
broken non-symlink entry that is neither a normal folder nor a valid symlink,
`error` status may be the only option.

Suggestion: Accept `install_status = error` as the catch-all for truly
unclassifiable cases. Add a note to that effect. No new enum value needed.

---

### 3. `installed_from_host_folder_id` missing from relationship overview

`installs.installed_from_host_folder_id` is a field in `installs` that
references `skill_host_folders.id`. It is used for "old host symlink" detection
(comparing against `symlink_target_path`). But the relationship overview does not
include this link:

```
skill_host_folders.id
  -> installs.installed_from_host_folder_id  ← missing
```

Suggestion: Add this line to the Relationship Overview section to make the
reference explicit and visible to future schema implementers.

---

### 4. `install_mode = symlink` for scan-discovered external symlinks vs user-installed symlinks

For externally created symlinks (not installed by Skillbox), `install_mode =
symlink` is used. For Skillbox-installed symlinks, `install_mode = symlink` is
also used. The two cases are indistinguishable from `install_mode` alone.
`install_status = external_symlink` disambiguates them at runtime, but a scan
of a brand-new project with a user-created symlink would need to determine
`install_mode = symlink` solely from the filesystem type. This is a valid
pragmatic choice (a symlink is a symlink), but the intent differs.

This is not a problem today because the distinction is observable
(`install_status` carries the semantics), but document the rule explicitly:
"Scan sets `install_mode = symlink` whenever a symlink is found on disk,
regardless of whether Skillbox installed it. `install_status` distinguishes
the management state."

---

### 5. `fetch_results` retention policy is undefined

`fetch_results` accumulates one row per fetch attempt per source. Without a
retention policy, this table grows unboundedly. For Phase 1, the Updates view
only needs the most recent fetch result per source.

Suggestion: Document a Phase 1 retention policy — for example, keep only the
last N rows per `source_id`, or only keep the latest row and hard-delete older
ones when a new fetch completes. This prevents unbounded growth without needing
a complex archival system.

---

### 6. `warnings.scope_type = source` is present but fetch failure mapping omits scope

The "Fetch Failure" edge case mapping shows:
```
fetch_results.status = failed | auth_required | not_found | network_error
warnings.code = fetch_failed
```

But it does not specify `warnings.scope_type` for this warning. Logically it
should be `source` (scoped to the `skill_source` record), but leaving it
implicit means implementers may make inconsistent choices.

Suggestion: Add `warnings.scope_type = source` to the Fetch Failure mapping, and
verify all edge case warning mappings specify their `scope_type` explicitly.

---

## Contradictions Between `docs/06-data-model.md` and Other Docs

### None found.

The following potential contradictions were checked and cleared:

**`docs/05` says "Phân loại là `direct`" for project manual skills.**
Model maps direct installs as `install_mode = direct`, `install_status = current`.
Consistent.

**`docs/05` says "Phân loại là symlink nhưng đánh dấu `old host`" for old host
symlinks.**
Model maps as `install_status = old_host`. The `install_mode = symlink` is
implied but unspecified in the edge case mapping. Not a contradiction — just an
incomplete mapping (see non-blocking suggestion 3 above for the related note).

**`docs/05` says "Phân loại là `external symlink`" for symlinks outside the
current host.**
Model maps as `install_mode = symlink` + `install_status = external_symlink`.
The classification term matches. Consistent.

**`docs/04` user flow 8 (Switch Install Mode) says "User chọn symlink hoặc
rsync/copy."**
Model has exactly those two modes as valid switch targets (excluding `direct`
which is scan-discovered, not user-chosen). Consistent.

**`docs/03` IA lists install modes as "symlink, rsync/copy, direct."**
Matches `install_mode` enum. Consistent.

**`docs/04` fetch flow writes `skill_sources.last_fetched_at` and
`skill_sources.last_fetch_status`.**
Both fields exist in the updated `skill_sources`. The flow mapping also still
references these fields correctly. Consistent.

**`docs/04` Update Skill Host Folder flow does not mention `skills.current_checksum`.**
The model's flow mapping adds `skills.current_checksum` to the Update Skill Host
Folder writes. This is an additive improvement, not a contradiction. The user
flow doc (04) describes the user-visible behavior, not every DB write. No issue.

---

## Overall Assessment

The updated `docs/06-data-model.md` is internally consistent, addresses all
prior critical feedback, and is coherent with the product direction, user flows,
and edge cases described in the other docs. The non-blocking suggestions above
are documentation gaps and edge-case clarifications that can be resolved during
implementation without revisiting the schema design.

The model is ready for implementation to begin.
