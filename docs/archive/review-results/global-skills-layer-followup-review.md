# Global Skills Layer Follow-Up Review

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: README.md, docs/index.md, docs/01-product-brief.md,
  docs/02-product-notes.md, docs/03-information-architecture.md,
  docs/04-user-flows.md, docs/05-edge-cases-and-ux-states.md,
  docs/06-data-model.md, docs/07-schema-dictionary.md,
  docs/08-provider-model.md, docs/09-ui-wireframes.md,
  docs/archive/review-results/global-skills-layer-review.md
- Browsing used: no

---

## Decision

**Approved.** All three critical blockers from the prior review are resolved.
Four non-blocking suggestions remain but none affects implementation safety.

---

## Critical Issue Status

### 1. `provider_definitions.has_global_level` — RESOLVED

**docs/06**: Field present in `provider_definitions` entity with note "Cho biết
provider có global/user-level location mà Skillbox có thể scan hoặc cấu hình."

**docs/07**: Row added to `provider_definitions` schema dictionary:
`has_global_level | integer | no | Boolean 0/1. Cho biết provider có
global/user-level location mà Skillbox có thể scan hoặc cấu hình.`

**docs/08**: Field listed in "Provider Definitions" key fields section alongside
`can_create_structure`. Description is accurate.

All three docs are consistent. The field is well-positioned alongside
`can_create_structure` where other per-provider capability flags live.

---

### 2. Global detection flow criterion — RESOLVED

**docs/08** "Global Detection Flow" now reads:

> "Load provider_definitions có `has_global_level = 1` hoặc configured global
> paths"

The criterion is now explicit. The `has_global_level = 1` case handles providers
the app knows have a global layer at seed time. The "configured global paths"
fallback handles the case where a user manually configured a path for a provider
not originally marked, which is the correct safety net.

---

### 3. Global adapter output contract — RESOLVED

**docs/08** "Provider Adapter Boundary" now has a full "Global adapter output
contract" section:

```text
provider_key: text
global_path: absolute path | null
global_skills_path: absolute path | null
global_status: active | not_configured | missing | unreadable | invalid_structure | empty | disabled
warnings: list of { code, severity, message, action_key }
entries: list of { name, path, entry_type, symlink_target }
```

The contract mirrors the project adapter contract structure, is typed, and uses
the same `warnings` and `entries` shapes. A developer can implement a global
adapter without ambiguity.

The `global_status` enum matches the `global_provider_locations.status` enum in
docs/06 and docs/07. Consistent.

---

### 4. docs/07 enum sync — RESOLVED (all three gaps)

**`scan_results.target_type`**: docs/07 now includes `global_provider_location`.
Matches docs/06.

**`operations.operation_type`**: docs/07 now includes `scan_global_skills`.
Matches docs/06.

**`provider_path_candidates.priority`**: docs/07 now reads "Lower value wins.
Priority `1` is checked before priority `10`." Matches docs/08.

---

### 5. Global rsync/copy detection rule — RESOLVED

**docs/08** "Global Detection Flow" now explicitly states:

> "Global scan dùng cùng rule với project rsync/copy detection: nếu entry là
> folder thường và có `global_installs` DB record cho path đó với
> `install_mode = rsync_copy`, mode là `rsync_copy`; nếu không có record thì
> mode là `direct`."

The rule is unambiguous and references the correct table (`global_installs` not
`installs`).

---

### 6. Global path resolution — RESOLVED

**docs/08** "Global Detection Flow" now explicitly states:

> "Global provider paths không dùng `provider_path_candidates.relative_path` vì
> field đó là project-root relative path. Global paths được resolve bởi adapter
> từ user/machine conventions hoặc từ `global_provider_locations.path` đã được
> user cấu hình trong Settings."

Two resolution mechanisms are clearly distinguished:
- Adapter knows machine conventions at implementation time (e.g., `~/.agents/skills`).
- User can override via Settings, stored in `global_provider_locations.path`.

This eliminates the ambiguity flagged in the prior review.

---

### 7. Phase 1 scope boundary — RESOLVED

**docs/04** "Scan Global Skills" (flow 4) now has an explicit Phase 1 block:

```text
Phase 1:
- Global Skills là scan/visibility/remediation surface.
- Chưa có flow `Install Skill To Global Location`.
- Add Skill flow chỉ target project providers.
```

**docs/09** "Global Skills" wireframe rules section now includes:

> "Phase 1 does not include an Add Skill to Global Location flow. Global Skills
> focuses on scan, visibility, and remediation actions."

Both the flow doc and the wireframe doc are consistent. Developers reading either
will not implement "Install to Global" in Phase 1.

---

### 8. Updates UI includes affected global installs — RESOLVED

**docs/09** "Updates" wireframe now has an "Affected Global Installs" table
alongside "Affected Projects":

```text
Affected Global Installs
  Location              Provider          Mode        Result after host update
  User Global           Generic Agents    symlink     updates immediately
  Claude Global         Claude            rsync/copy  needs sync
```

This correctly shows that global symlink installs update immediately (same as
project symlinks) and global rsync/copy installs need explicit sync.

**docs/06** "Data Needed By Main Views — Updates" already listed `global_installs`
and `global_provider_locations` as required tables. The wireframe and data model
are now aligned.

---

### 9. No new contradictions — CONFIRMED

Cross-doc consistency check:

- docs/01: "Global Skills" is now listed under "Skillbox quản lý" and in
  "Quyết Định Thiết Kế Đã Chốt" as a scan/observe surface. ✓
- docs/02: SQLite Database section now mentions "global installs." ✓
- docs/03: "Global Skills" now listed as a main app area; "Global Provider
  Location" and "Global Install" defined in Core Concepts. ✓
- docs/04: Flow 4 "Scan Global Skills" is present; Flow 13 "App Startup"
  explicitly checks global provider locations. ✓
- docs/05: Section 3 "Global Skill States" added with 5 edge cases. ✓
- docs/06 ↔ docs/07: All enum values consistent. ✓
- docs/06 ↔ docs/08: `has_global_level` consistent; global output contract in
  docs/08 matches `global_provider_locations.status` enum in docs/06. ✓
- docs/08 ↔ docs/09: Phase 1 boundary consistent in both docs. ✓

No new contradictions found.

---

## Non-Blocking Suggestions

### 1. docs/08 Initial Provider Definitions are missing `has_global_level` values

The "Suggested Initial Provider Definitions" section in docs/08 lists per-provider
fields but does not include `has_global_level` for any provider. A developer
seeding the database needs to know which providers get `has_global_level = 1`.

Suggested addition per provider definition block:

```text
Generic Agents:  has_global_level = true  (has ~/.agents/skills convention)
Claude:          has_global_level = true  (has ~/.claude/... convention)
Codex:           has_global_level = false (uses generic_agents for global, no distinct global)
opencode:        has_global_level = false
Antigravity CLI: has_global_level = false
```

This is not blocking — the field exists and the developer can determine values
from product intent — but documenting expected seed values eliminates a judgment
call at implementation time.

---

### 2. docs/05 "Loading/scanning state" list doesn't include global scan

docs/05 section 8 "Loading/scanning state" lists:

> "Scan Skill Host Folder. Scan project. Fetch update. Sync rsync/copy."

docs/09 "Loading States" now correctly includes "Scanning global locations."

The gap is in docs/05. Suggested fix: add "Scan global locations" to the loading
state list in docs/05 section 8.

---

### 3. Impact Preview example doesn't include global installs

docs/09 "Impact Preview" example for "Update adr-helper" shows symlink projects,
rsync/copy projects, and direct installs — but not global installs. The Updates
wireframe now shows global installs correctly. The Impact Preview example should
also demonstrate global impact.

Suggested addition to the example:

```text
Global symlink installs updated immediately:
  User Global (Generic Agents)

Global rsync/copy installs needing sync:
  Claude Global
```

---

### 4. docs/03 Global Skills section doesn't state Phase 1 scope boundary

docs/03 "Global Skills" section lists actions including "Remove global entry" and
"Relink hoặc sync nếu entry được Skillbox quản lý." It does not mention that
Phase 1 does not include installing to global locations.

Both docs/04 and docs/09 have the Phase 1 boundary statement. docs/03 is the IA
source of truth read before the other docs, so adding a note there would prevent
confusion for any developer starting from the IA document.

Suggested addition to docs/03 Global Skills section:

```text
Phase 1 scope: scan, visibility, and remediation. Install Skill To Global
Location is not a Phase 1 flow. Add Skill flow targets project providers only.
```

---

## What Changed Since Prior Review (Summary)

| Prior Blocker | Status |
|---|---|
| `has_global_level` missing from schema | Fixed in docs/06, docs/07, docs/08 |
| Global detection criterion undefined | Fixed in docs/08 |
| No global adapter output contract | Fixed in docs/08 |
| docs/07 `scan_results.target_type` missing `global_provider_location` | Fixed |
| docs/07 `operations.operation_type` missing `scan_global_skills` | Fixed |
| docs/07 `priority` direction not specified | Fixed |
| Global rsync/copy detection rule not stated | Fixed in docs/08 |
| Global path resolution ambiguous | Fixed in docs/08 |
| Phase 1 scope not stated | Fixed in docs/04 and docs/09 |
| Updates wireframe missing global installs | Fixed in docs/09 |
| docs/01–05 missing global coverage | Fixed: all five docs updated |

All prior suggestions that were addressed:
- docs/03 now lists Global Skills as a main app area ✓
- docs/04 now has Scan Global Skills as flow 4 ✓
- docs/05 now has Global Skill States edge cases ✓
- docs/09 Loading States now includes global scan ✓
- docs/09 Updates wireframe now shows affected global installs ✓
