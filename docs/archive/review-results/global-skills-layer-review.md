# Global Skills Layer Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: README.md, docs/index.md, docs/01-product-brief.md,
  docs/02-product-notes.md, docs/03-information-architecture.md,
  docs/04-user-flows.md, docs/05-edge-cases-and-ux-states.md,
  docs/06-data-model.md, docs/07-schema-dictionary.md,
  docs/08-provider-model.md, docs/09-ui-wireframes.md,
  docs/review-results/ (prior reviews for context)
- Browsing used: no

---

## Executive Summary

The Global Skills layer is well-modeled at the data layer (docs/06) and well
represented in the UI wireframes (docs/09). The `global_provider_locations` and
`global_installs` tables are coherent and use consistent semantics with their
project-level counterparts. However, the layer has three blockers before
implementation: (1) the global detection flow in docs/08 loads providers with
"global support" but this criterion has no backing field or mechanism in the
schema; (2) no global adapter output contract is defined, making it impossible
to write a consistent global scan adapter; and (3) docs/07 schema dictionary is
out of sync with docs/06 on two enum values (`scan_results.target_type` and
`operations.operation_type`) and the `priority` direction fix. Beyond the
blockers, docs/01–05 make no mention of global skills, leaving the foundational
product documents inconsistent with the three-layer model now described in the
data model and wireframes.

---

## Decision

**Not approved.** Three blockers must be resolved before implementation.

---

## Critical Issues

### 1. Global Detection Flow loads "global support" — undefined criterion

**File**: docs/08-provider-model.md, section "Global Detection Flow"

The flow begins:

> "Load provider_definitions có global support hoặc configured global paths"

Neither `provider_definitions` nor `provider_path_candidates` has a
`has_global_level` boolean, a `global_path` field, or any other mechanism to
indicate that a provider supports a global location. The schema has no way for
the app to distinguish providers that have a global layer (e.g., Generic Agents
with `~/.agents/skills`) from those that don't.

Two approaches to fix this:

**Option A — Add a boolean to `provider_definitions`**: Add
`has_global_level INTEGER NOT NULL DEFAULT 0` to `provider_definitions`. Global
detection loads only providers where `has_global_level = 1`. Simple and explicit.

**Option B — Derive from `global_provider_locations`**: A provider "has global
support" if a `global_provider_locations` row exists for its
`provider_definition_id`. Global detection loads providers with existing (or
configured) `global_provider_locations` rows. This means the app seeds one
`global_provider_locations` row per provider that can have a global level, with
`status = not_configured` until the user provides a path.

Option A is cleaner — it keeps the definition in `provider_definitions` where
all other adapter capabilities live (alongside `can_create_structure`). Fix the
docs before writing global scan code.

---

### 2. No global adapter output contract defined

**File**: docs/08-provider-model.md, section "Provider Adapter Boundary"

The existing adapter output contract covers project-level scan:

```text
provider_key, detected_path, skills_path, detection_status, warnings, entries
```

A global scan adapter produces different output — it resolves a global path
(e.g., `~/.agents/skills`), scans entries there, and reports a global-level
status. But no global output contract is defined. Two adapters written by two
developers will produce incompatible global results.

Minimum global output contract needed alongside the project contract:

```text
global_output:
  provider_key: text
  global_path: absolute path | null
  global_skills_path: absolute path | null
  global_status: active | not_configured | missing | unreadable | invalid_structure | empty | disabled
  warnings: list of { code, severity, message, action_key }
  entries: list of { name, path, entry_type, symlink_target }
```

Add this as a "Global Adapter Output Contract" subsection under "Provider
Adapter Boundary" in docs/08.

---

### 3. docs/07 schema dictionary is out of sync with docs/06 on three points

**File**: docs/07-schema-dictionary.md

**3a — `scan_results.target_type` missing `global_provider_location`**

docs/07 `scan_results.target_type` allowed values: `skill_host_folder`,
`project`, `project_provider`.

docs/06 `scan_results.target_type` allowed values: `skill_host_folder`,
`project`, `project_provider`, **`global_provider_location`**.

Fix: Add `global_provider_location` to `scan_results.target_type` in docs/07.

**3b — `operations.operation_type` missing `scan_global_skills`**

docs/07 `operations.operation_type` allowed values: `scan`, `fetch`,
`update_host_skill`, `sync_install`, `install_skill`, `remove_install`,
`switch_install_mode`, `change_skill_host_folder`.

docs/06 adds `scan_global_skills`.

Fix: Add `scan_global_skills` to `operations.operation_type` in docs/07.

**3c — `provider_path_candidates.priority` direction still unspecified in docs/07**

docs/07: "Priority thấp/cao theo convention implementation chọn."
docs/08: "Priority thấp hơn thắng. Adapter kiểm tra `priority = 1` trước
`priority = 10`."

Fix: Update docs/07 `provider_path_candidates.priority` description to: "Lower
value wins. Priority 1 is checked before priority 10."

---

## Suggested Improvements

### 1. docs/01 and docs/02 don't mention Global Skills

The product brief (docs/01) and product notes (docs/02) describe a two-layer
model (Skill Host Folder → Project Installs). The three-layer model that now
exists in the data model and wireframes is invisible to anyone reading from the
top.

Minimum fix: Add one section to docs/01 under "Product Scope" or "Model" naming
the three layers:

```text
Skill Host Folder — source of truth for skill content on the machine.
Global Skills — provider global-level skill state (e.g., ~/.agents/skills).
Project Installs — skills installed into a specific project/provider.
```

---

### 2. docs/03 Information Architecture doesn't list Global Skills

docs/03 "Main App Areas" lists: Dashboard, Skills Library, Projects, Project
Detail, Skill Detail, Updates, Settings. "Global Skills" is absent.

docs/09 sidebar shows: Dashboard, Skills Library, **Global Skills**, Projects,
Updates, Settings.

Fix: Add a "Global Skills" section to docs/03 describing its purpose, what it
displays, and its primary actions (Scan Global, Configure Location, Open Folder).

---

### 3. docs/04 User Flows has no global skills flows

The authoritative user flows document covers 12 flows. None of them involves
global skills. But docs/06 "Mapping From User Flows" includes "Scan Global
Skills" as a documented flow with specific DB writes.

A developer using docs/04 as their guide will not find the global scan flow.

Minimum fix: Add at least "Scan Global Skills" as user flow 13, following the
same format as the other flows:

```text
Mục tiêu: đọc và reconcile provider global locations trên máy.
Flow: User → Scan Global → adapter resolves global paths → scan entries →
      update global_provider_locations / global_installs / warnings
Kết quả: Global Skills screen phản ánh trạng thái thật của global locations.
```

---

### 4. docs/05 Edge Cases has no global skills section

docs/05 has categories for Skill Host Folder, Project, Install, Fetch, Provider,
Database, and UI states. No "Global Skills States" category exists.

Relevant states to document:
- Global location `not_configured` (no path set yet).
- Global location `missing` (path set but no longer exists).
- Global direct install (entry exists but not installed by Skillbox).
- Global broken symlink.
- Global/project skill overlap (same skill name in both scopes).
- Global rsync/copy outdated.
- Global location `unreadable` (permission error).

---

### 5. Updates wireframe doesn't surface global installs

docs/09 "Updates" wireframe shows "Affected Projects" per skill update but does
not show global installs as an additional affected scope. docs/06 "Data Needed
By Main Views — Updates" lists `global_installs` and `global_provider_locations`
as needed tables, implying global installs with `needs_sync` status should also
appear here.

Fix: Add a row to the Updates wireframe impact section, e.g.:

```text
Global installs needing sync after update:
  Generic Agents Global
```

---

### 6. No "Install Skill to Global Location" flow defined anywhere

The Skill Detail wireframe shows "Global Usage" with `symlink` and `rsync/copy`
modes alongside `direct`. This implies users can install skills from the Skill
Host Folder into global locations as managed installs. But no user flow, no
operation type beyond the generic `install_skill`, and no UI flow screen covers
this action.

If Skillbox supports installing to global locations (not just scanning
pre-existing ones), this flow needs to be defined. If Skillbox only scans global
locations and never installs to them (global installs are always `direct`), that
should be stated explicitly.

This is a product decision with implementation consequences — it determines
whether `global_installs` ever has `install_mode = symlink` or `rsync_copy` for
managed entries vs always `direct`.

---

### 7. Global scan loading state not in wireframes Loading States section

docs/09 "Loading States" lists scanning Skill Host Folder, scanning project,
fetching updates, etc. It does not list "Scanning global locations." If the
global scan is a separate operation triggerable from Dashboard ("Scan Global"
action) or Global Skills screen, the loading state should be documented.

---

## Concept Separation Review

### Skill Host Folder

Clearly distinct in docs/06 and docs/09. The Skill Host Folder is the source
of truth for skill content, not a location where skills are "used." It is
managed explicitly by the user and referenced by both installs and global
installs via `source_skill_path` and `installed_from_host_folder_id`.

**Status: Clear in docs/06 and docs/09. Absent in docs/01–05.**

---

### Global Skills

The concept is defined clearly in docs/06 ("global_provider_locations lưu
provider global locations ở cấp user/máy") and in docs/08 ("Global provider
location là provider scope ở cấp user/máy, không thuộc một project cụ thể").
The wireframe (docs/09) shows it as a distinct screen.

The definition is consistent: global skills are skills/entries that exist in a
provider's machine-level location (e.g., `~/.agents/skills`), not inside any
specific project.

**Status: Clear in docs/06, docs/08, docs/09. Absent in docs/01–05.**

---

### Project Installs

Clearly distinct and well-established from prior doc versions. `installs` is
scoped via `project_provider_id`; `global_installs` is scoped via
`global_provider_location_id`. The two are never mixed.

**Status: Clear and consistent across all docs.**

---

## Data Model Review

### `global_provider_locations`

Structure is sound:
- FK to `provider_definitions` — correct.
- `path` and `skills_path` both nullable for `not_configured` case — correct.
- `status` enum covers `active`, `not_configured`, `missing`, `unreadable`,
  `invalid_structure`, `empty`, `disabled` — comprehensive.
- `last_scanned_at` — correct.

One naming inconsistency: `project_providers.detected_path` corresponds
functionally to `global_provider_locations.path`. The field names differ but
the semantics are the same. This is minor but worth noting for developers
reading both tables: `path` in global locations = `detected_path` in project
providers.

No blocker. Acceptable for Phase 1.

---

### `global_installs`

Structurally mirrors `installs` with the FK swapped from `project_provider_id`
to `global_provider_location_id`. All the same install modes, statuses, and
tracking fields (`symlink_target_path`, `installed_from_host_folder_id`,
checksums, version/commit fields) are present.

The warning scope types `global_provider_location` and `global_install` in the
`warnings` table cover global-scoped warnings.

One gap: `global_installs` has no `rsync_copy` detection rule stated. The
project-level scan rules (docs/08) define "folder + DB record with
`install_mode = rsync_copy` = rsync_copy; folder without DB record = direct."
This same rule should apply to global scan, but it is not stated for global
scope. Implementation can infer this, but explicit documentation would prevent
divergent behavior.

---

### Relationship with `skills`

Both `installs.skill_id` and `global_installs.skill_id` are nullable FKs to
`skills.id`. This correctly allows global entries that don't map to any skill in
the Skill Host Folder (unmanaged/direct entries). The Skill Detail "Global
Usage" section in docs/09 shows this relationship from the other direction.

The relationship overview in docs/06 includes:
```text
skills.id -> global_installs.skill_id
```

---

### Relationship with `provider_definitions`

`global_provider_locations.provider_definition_id` → `provider_definitions.id`.
This correctly scopes each global location to a known provider.

---

### Warning scopes

`warnings.scope_type` includes `global_provider_location` and `global_install`.
The edge case mapping in docs/06 shows these in use:

```text
global_provider_locations.status = missing
warnings.code = global_provider_location_missing
warnings.scope_type = global_provider_location
```

This is correct and consistent with how project warnings work.

---

### Separate `global_installs` vs polymorphic installs

The duplication is acceptable for Phase 1. The separate table approach is
simpler to query, clearer in schema intent, and avoids polymorphic FK complexity.
The maintenance risk — enum changes must be applied to both tables — is real but
manageable at this scale.

If the product grows to add a third install scope (e.g., workspace-level
installs), revisit the model. For Phase 1, keep the tables separate.

---

## Schema Dictionary Review

docs/07-schema-dictionary.md covers `global_provider_locations` and
`global_installs` with field-level descriptions. The descriptions are clear and
AI/developer friendly.

Three out-of-sync issues (covered as Critical Issue 3):
- `scan_results.target_type` missing `global_provider_location`.
- `operations.operation_type` missing `scan_global_skills`.
- `provider_path_candidates.priority` description not updated with direction rule.

All other global-layer fields in docs/07 are consistent with docs/06.

---

## User Flow Coverage

### Scan Global Skills

Documented in docs/06 "Mapping From User Flows" with DB writes:
- `global_provider_locations`
- `global_installs`
- `warnings`
- `scan_results`

**Not documented in docs/04.** This is the primary missing flow.

---

### Missing flows not documented anywhere

| Flow | Where needed | Impact |
|---|---|---|
| Scan Global Skills | docs/04 | High — core global operation |
| Install Skill to Global Location | docs/04, docs/08 | Medium — unclear if supported |
| Remove Global Install | docs/04 | Medium — remove action exists in wireframe |
| Configure Global Provider Location | docs/04, docs/09 Settings | Medium — "Configure" button in wireframe |
| Rescan Global on App Startup | docs/04, docs/12 App Startup | Low — implied but not stated |

---

## Edge Case Coverage

### Covered in docs/06 edge case mappings

- Missing Global Provider Location → `status = missing`, `warnings.code = global_provider_location_missing` ✓
- Global Direct Install → `install_mode = direct`, `install_status = current`, `skill_id = null` ✓
- Global Skill Overlap → `warnings.code = global_project_skill_overlap` ✓

### Not covered (gaps in docs/05 and docs/06)

- `not_configured` global location initial state — no edge case or UI guidance
  for the "Claude Global — not configured" state shown in the wireframe. The
  status exists in the enum but no edge case defines how the app gets there and
  what the UI shows.
- Global broken symlink — implied by `install_status = broken_symlink` reuse,
  but no explicit edge case.
- Global location `unreadable` / `invalid_structure` — statuses exist, no edge
  case guidance.
- Global rsync/copy outdated — implied by `needs_sync` status reuse, no edge case.

These are documentation gaps, not model gaps. The enums cover the states; the
edge case descriptions are missing.

---

## Provider Model Coverage

### What is covered

- "Global Provider Location" concept section: clear definition, separation from
  project scope. ✓
- Adapter responsibilities: includes global detection and global path resolution. ✓
- "Global Detection Flow": structure exists. ✓
- Constraint that global scan must not be mixed with project scan. ✓

### What is missing

1. **"Global support" criterion is undefined** (Critical Issue 1 — see above).

2. **No global adapter output contract** (Critical Issue 2 — see above).

3. **Global rsync/copy detection rule not stated for global scope.** The
   "Scan Installed Skills" section defines the DB record rule for project-level
   scan. It is silent on whether the same rule applies when scanning
   `global_installs`. Explicitly state: "The same rsync/copy detection rule
   (DB record check) applies to global scan with `global_installs` as the
   reference table."

4. **Global scan boundary with provider path candidates.** The project detection
   flow uses `provider_path_candidates` with paths relative to the project root.
   Global paths (e.g., `~/.agents/skills`) are absolute and machine-level.
   `provider_path_candidates.relative_path` cannot represent `~/.agents/skills`
   meaningfully as a relative path from a project root. The doc is silent on
   how global candidate paths are stored or resolved. Either:
   - Global path candidates need a separate mechanism (a new table or a
     `global_path` field on `provider_definitions`).
   - Or the global path is derived from system conventions known by the adapter
     implementation and stored directly into `global_provider_locations.path`
     without using `provider_path_candidates` at all.

   This ambiguity must be resolved before implementing the global adapter.

---

## UI/UX Coverage

### What is covered

- Global Skills screen in sidebar navigation ✓
- Global Locations table: provider, path, status, entries count ✓
- Global Entries table: provider, entry name, mode, status, actions ✓
- Per-entry actions: Open, Relink, Remove ✓
- Global warnings: broken symlink, missing location, overlap info ✓
- Skill Detail "Global Usage" section ✓
- Dashboard "Global skills: N" summary count ✓
- Dashboard "Scan Global" primary action ✓
- Settings "Global Provider Locations" with per-provider path and actions ✓
- Empty state "No Global Skills" with Scan Global and Configure actions ✓
- Rules section in Global Skills wireframe is well-stated ✓

### What is missing or incomplete

1. **Updates wireframe doesn't show global installs as affected** (see
   Suggested Improvement 5).

2. **No "Install to Global" flow screen.** The Skill Detail shows `symlink`
   mode in "Global Usage" implying managed global installs can be created, but
   no screen or action for creating them is shown anywhere in the wireframes.

3. **"Scan Global" loading state** not in the Loading States section. If the
   Dashboard has a "Scan Global" action, its loading state should be documented.

4. **Global location `not_configured` UI treatment.** The wireframe shows
   "Claude / not configured / 0 / [Configure]" but no guidance on what the
   "Configure" action does (opens a path picker? links to Settings?).

5. **Global/project overlap treatment.** The warning says "[info] Global skill
   also exists in 3 projects." but the wireframe doesn't explain what action
   (if any) the user takes on this. If it's purely informational, state that
   no action is required.

---

## Contradictions Across Docs

### A. docs/07 `scan_results.target_type` vs docs/06

docs/07: `skill_host_folder | project | project_provider`
docs/06: adds `global_provider_location`

**Fix needed in docs/07.**

---

### B. docs/07 `operations.operation_type` vs docs/06

docs/07 does not include `scan_global_skills`.
docs/06 does.

**Fix needed in docs/07.**

---

### C. docs/07 `provider_path_candidates.priority` direction vs docs/08

docs/07: "Priority thấp/cao theo convention implementation chọn."
docs/08: "Priority thấp hơn thắng."

**Fix needed in docs/07.**

---

### D. docs/03 Main App Areas vs docs/09 sidebar

docs/03 lists: Dashboard, Skills Library, Projects, Project Detail, Skill Detail,
Updates, Settings. No "Global Skills."

docs/09 sidebar: Dashboard, Skills Library, **Global Skills**, Projects, Updates,
Settings.

**Gap in docs/03.** Not a data conflict, but docs/03 is the IA source of truth
and it's missing a whole app section.

---

### E. docs/04 User Flows vs docs/06 Mapping From User Flows

docs/04 has no global skills flows.
docs/06 "Mapping From User Flows" includes "Scan Global Skills."

**Gap in docs/04.** A developer using docs/04 as their flows guide will not
implement global scan.

---

### F. docs/01 and docs/02 describe a two-layer model

The README and docs/01 describe the product as managing Skill Host Folder →
Project Installs. No mention of global skills. The three-layer model in the
data model and wireframes is inconsistent with the foundational docs.

**Gap in docs/01, docs/02, README.md.**

---

## Recommended Changes

### Before implementation begins (required)

1. **docs/08 — Define "global support" criterion** for the global detection flow.
   Recommend: add `has_global_level INTEGER NOT NULL DEFAULT 0` to
   `provider_definitions` in docs/06, docs/07, and docs/08. Update the global
   detection flow to "Load provider_definitions có `has_global_level = 1`."

2. **docs/08 — Add global adapter output contract** under "Provider Adapter
   Boundary" (see Critical Issue 2 for the proposed contract shape).

3. **docs/07 — Sync three enum gaps** with docs/06:
   - `scan_results.target_type`: add `global_provider_location`.
   - `operations.operation_type`: add `scan_global_skills`.
   - `provider_path_candidates.priority`: update description to "Lower value
     wins. Priority 1 is checked before priority 10."

4. **docs/08 — State that global rsync/copy detection uses `global_installs`
   DB records**, mirroring the project-level rule.

5. **docs/08 — Clarify how global provider paths are resolved** — either via a
   new mechanism (recommended: `has_global_level + adapter-known conventions`)
   or explicit storage in `global_provider_locations.path` during first-time
   setup per provider. The current doc leaves this ambiguous.

### Before writing docs/04 or docs/05 based content (recommended)

6. **docs/04 — Add "Scan Global Skills" as user flow 13**, including the trigger,
   steps, and expected database writes.

7. **docs/05 — Add "Global Skills States" edge case section** covering:
   `not_configured`, `missing`, broken symlink, rsync/copy outdated, overlap,
   `unreadable`.

8. **docs/03 — Add Global Skills as a main app area**, describing its purpose,
   displayed data, and actions.

9. **docs/01/docs/02/README — Add the three-layer model** to product description.

10. **Decide and document whether users can install skills to global locations**
    (managed symlink/rsync) or whether global installs are always `direct`
    (scan-only). This changes whether an "Install to Global" UI flow is needed.

---

## Open Questions For The Product Owner

1. **Can users install skills from Skill Host Folder into global provider
   locations (managed symlink/rsync), or are global installs always unmanaged
   (`direct`)?** This is the most consequential global skills product decision.
   Scanning and displaying global entries is clearly supported. Installing into
   them is unclear.

2. **Which providers support global locations in Phase 1?** Generic Agents
   (`~/.agents/skills`) seems clear. Claude global conventions are unconfirmed.
   Codex/opencode/Antigravity CLI global conventions are unknown. Defining
   `has_global_level = 1` for Phase 1 requires knowing this.

3. **Is "global/project overlap" always informational, or can it ever be
   blocking?** The wireframe shows `[info]` for overlap. The warning code exists
   in docs/06. But some providers may have behavior where global and project
   skills conflict at runtime — should Skillbox warn more strongly in those cases?

4. **What does "Configure Global Location" do in the Settings screen?** Is it
   a path picker? Does it auto-detect the path from system conventions? Or does
   the user type a path manually?

5. **What happens to global installs in the Updates flow?** If a skill in the
   Skill Host Folder is updated and a global install references it via symlink,
   does the Updates screen show the global install as "updated immediately"
   (like symlink projects)? The Updates wireframe doesn't mention this.

6. **How does "Scan Global" interact with App Startup?** User flow 12 (App
   Startup) checks Skill Host Folder and project paths. Does it also check global
   provider locations? Should global scan run automatically at startup, or only
   on explicit user action?

---

## What Looks Solid

- **`global_provider_locations` and `global_installs` schema design** is
  internally consistent, follows the same patterns as `project_providers` and
  `installs`, and reuses all the same install_mode/install_status semantics.
  No redesign needed.

- **Warning scope types** `global_provider_location` and `global_install` are
  correctly added to the `warnings` table. The warning code
  `global_provider_location_missing` and `global_project_skill_overlap` are
  appropriately named.

- **Scope separation** between global and project installs is enforced at the
  data model level. There is no ambiguity about which table holds what.

- **Global Skills UI screen** in docs/09 is well-designed: two-panel layout
  (Global Locations + Global Entries), clear warning surfacing, per-entry
  actions, and sensible empty state. The "Scan Global" trigger and Settings
  "Global Provider Locations" configuration panel are correctly placed.

- **Skill Detail "Global Usage" section** in docs/09 correctly shows global
  impact alongside project impact, giving users a complete picture of where a
  skill is referenced on the machine.

- **The policy that global scope is never auto-merged with project scope** is
  explicitly stated in both docs/08 and docs/09, reducing the risk of accidental
  cross-contamination in implementation.

- **`scan_global_skills` operation type** in docs/06 correctly creates a
  trackable operation for the global scan, enabling loading state and audit
  trail in the UI.
