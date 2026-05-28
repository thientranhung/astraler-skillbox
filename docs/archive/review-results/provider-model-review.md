# Provider Model Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: README.md, docs/index.md, docs/01-product-brief.md,
  docs/02-product-notes.md, docs/03-information-architecture.md,
  docs/04-user-flows.md, docs/05-edge-cases-and-ux-states.md,
  docs/06-data-model.md, docs/07-schema-dictionary.md,
  docs/08-provider-model.md
- Browsing used: no

---

## Executive Summary

The Provider Model is well-structured and solves the right problems: adapters
are isolated from core logic, `provider_path_candidates` is flexible enough for
multi-path providers, and the unsupported provider write policy is correctly
strict. However, four issues must be resolved before implementation: (1) the
detection flow filter excludes `unsupported` providers from being detected at
all, contradicting the stated purpose of that status; (2) the term "Skillbox
metadata" used to distinguish `rsync_copy` from `direct` installs during scan
is undefined — the implementation cannot proceed without knowing whether this is
a DB record check or a filesystem marker; (3) the adapter output structure
(`warnings` and `entries`) is not specified, making it impossible to write
consistent adapters; and (4) the `priority` field direction is unspecified.
Claude's path candidates are intentionally deferred and correctly flagged, but
must be researched before any Claude scanning or install can work.

---

## Decision

**Not approved.** Four critical issues must be resolved before implementation.

---

## Critical Issues

### 1. Detection flow filter contradicts `unsupported` provider_definition semantics

**Section**: "Detection Flow"

The detection flow says: "Load provider_definitions đang enabled/supported/experimental."

`provider_definitions.status` has four values:
- `supported`, `experimental`: adapter can detect AND operate.
- `unsupported`: Skillbox **can detect** but **cannot safely operate**.
- `disabled`: not loaded at all.

If the detection pass only loads `supported` and `experimental` definitions, then
`unsupported` providers are never loaded, never detected, and
`project_providers.detection_status = unsupported` can never be set. This
contradicts both the status description in `docs/06-data-model.md` and the edge
case in `docs/05-edge-cases-and-ux-states.md` ("Provider convention chưa được
support").

Fix: The detection pass should load ALL provider definitions except `disabled`.
Install target resolution (separate step) should only allow `supported` and
`experimental`. Concretely:

```text
Detection pass: status IN (supported, experimental, unsupported)
Install target filter: status IN (supported, experimental)
```

This allows `unsupported` providers to be detected and reported in the UI
without ever allowing Skillbox to write into their paths.

---

### 2. "Skillbox metadata" for rsync/copy detection is undefined

**Section**: "Scan Installed Skills"

The scan rule says: "Nếu entry là folder thường có Skillbox metadata, mode là
`rsync_copy`."

This is implementation-blocking. There are two possible interpretations:

**Option A — Database record**: The scan checks whether an `installs` row exists
for this path with `install_mode = rsync_copy`. If yes, classify as `rsync_copy`.
If no `installs` row exists, classify as `direct`. This is consistent with the
design principle "Filesystem là trạng thái thật" being primary, and DB records
as secondary context. The risk: after DB loss or full rescan from scratch,
all rsync/copy installs are reclassified as `direct`.

**Option B — Filesystem marker**: The adapter writes a hidden file or sidecar
(e.g., `.skillbox-install`) into the skill folder at install time. Scan checks
for this marker to distinguish `rsync_copy` from `direct`. More resilient to DB
loss, but writes an extra file into the project, which may be unexpected.

Pick one option, document it explicitly in this section. Option A is simpler and
consistent with Phase 1 design principles. The DB loss risk is acceptable if the
data model doc says "scan after DB recovery reclassifies rsync/copy as direct —
user must resync."

---

### 3. Adapter output structure is unspecified

**Section**: "Provider Adapter Boundary"

The adapter output is listed as:

```text
provider_key
detected_path
skills_path
detection_status
warnings
entries
```

But `warnings` and `entries` are opaque. An implementer writing a Claude adapter
and a Codex adapter will produce incompatible outputs unless the field shapes
are defined.

Minimum required specification before implementation:

```text
warnings: list of { code, severity, message, action_key }
entries: list of {
  name,             -- skill folder name
  path,             -- absolute path in project
  entry_type,       -- symlink | directory | unknown
  symlink_target    -- if symlink, the raw target path
}
```

Even a rough type contract in the doc is sufficient. Without it, the adapter
boundary guarantee ("adapters trả về structured result, không tự mutate
database") is unenforceable.

---

### 4. `priority` field direction is unspecified

**Section**: "Provider Path Candidates"

`docs/07-schema-dictionary.md` describes `priority` as "Priority thấp/cao theo
convention implementation chọn." This delegates the decision to the implementer,
which will produce inconsistent adapter code.

Fix: Specify the convention in `docs/08-provider-model.md`. Recommended: lower
integer = higher priority (e.g., priority 1 beats priority 10). This matches
common OS and framework conventions (e.g., CSS z-index inverse, routing
specificity). Add one line: "Lower priority value wins. Adapter checks priority
1 before priority 10."

---

## Suggested Improvements

### 1. Defer `purpose = commands` and `purpose = config` explicitly

**Section**: "Provider Path Candidates"

None of the twelve user flows in `docs/04-user-flows.md` involve reading a
provider's command files or config files. Including `commands` and `config`
purposes without a defined Phase 1 use case adds schema complexity without
value. Implementers may try to populate these candidates and build logic around
them prematurely.

Suggestion: Add a note in this section: "`commands` and `config` purposes are
reserved for future phases. Phase 1 adapters only need `detect` and `skills`
candidates."

---

### 2. `configured` detection_status needs a defined trigger

**Section**: "Project Provider"

The status description says: "user hoặc app đã cấu hình provider target, kể cả
khi path chưa tự detect được."

But no user flow in `docs/04-user-flows.md` includes a "configure provider"
step. There is no UI described in `docs/03-information-architecture.md` for
manually specifying a provider target. It's unclear:

- When does `configured` get set?
- Who sets it — the adapter or the user?
- Can a project have `detection_status = configured` for a provider that
  `detection_status = missing` on the next scan?

If `configured` is purely a future-facing placeholder, say so. If it represents
a real Phase 1 path (e.g., user clicking "set up Claude" in Settings), add the
user flow.

---

### 3. Adapter filesystem responsibility boundary is ambiguous

**Section**: "Provider Adapter Boundary"

The boundary correctly states adapters should not mutate the database. But it is
silent on filesystem operations. Two of the adapter responsibilities listed are:
"Resolve skill install path" and "Tạo provider folder structure nếu adapter
được phép."

Does the adapter perform the `mkdir` when `can_create_structure = true`, or does
it return the paths to create and core logic performs the IO?

Recommendation: Be explicit. State that all filesystem writes (mkdir, symlink
creation, rsync, delete) are performed by core Skillbox logic. The adapter
provides paths and permissions metadata only. This makes adapters pure functions
with no filesystem side effects — easier to test and safer to extend.

---

### 4. Claude path candidates must be researched before any Claude work

**Section**: "Suggested Initial Provider Definitions — Claude"

The document correctly defers Claude's path candidates: "Path candidates should
be finalized after provider convention research."

This is honest and appropriate. However, it means Claude detection and install
cannot be implemented at Phase 1 launch without this research. Calling Claude's
adapter `status = experimental` with `can_create_structure = false` is
conservative and correct, but there are no `provider_path_candidates` rows
defined, so the adapter has nothing to query.

Action required (not a doc change, but a pre-implementation task): Research
Claude Code's actual file convention. Based on this reviewer's knowledge (no
browsing, training data only — treat as assumption): Claude Code uses `.claude/`
at project root, potentially with sub-paths like `.claude/commands/` for slash
commands and possibly `.claude/skills/` or a top-level convention. These paths
should be verified against Claude's documentation before being seeded as
`provider_path_candidates` rows.

---

### 5. Document what happens when multiple `skills` candidates match simultaneously

**Section**: "Provider Path Candidates — Resolution rules"

The current rule says: "Nếu provider có nhiều skills path hợp lệ, adapter phải
chọn một path chính hoặc báo state cần user chọn."

But `project_providers.detection_status` has no value for "awaiting user
selection." Without a defined status, the UI cannot represent this state and the
adapter has no clean way to signal it. Options:

- Add `ambiguous_path` to `detection_status`.
- Or rule that the highest-priority candidate always wins automatically, removing
  user-selection ambiguity for Phase 1.

Pick one and document it. For Phase 1, automatic priority-based selection is
simpler and removes the need for a new UI state.

---

### 6. `provider_type` vs `key` distinction is undocumented

**Section**: "Provider Definitions" (and `docs/07-schema-dictionary.md`)

`provider_definitions` has both `key` (e.g., `claude`) and `provider_type` (e.g.,
`claude`). For all initial providers, these values are identical. The doc does
not explain why both exist.

Likely intent: `key` is a stable string ID for external config/code references;
`provider_type` is an enum for adapter dispatch (switch/case on provider_type to
instantiate the right adapter class). If so, state this explicitly so implementers
know when to use `key` vs `provider_type`.

---

### 7. No scan state for "provider path moved within project"

**Section**: "Detection Flow"

If a provider's path moves within a project (e.g., `.agents/skills` renamed to
`.agents/capabilities`), a rescan would produce `detection_status = missing` for
the old provider record and create a new `project_providers` row for the new
location (if the new path matches a candidate). The old `installs` rows would
remain linked to the now-missing `project_providers` row.

Suggestion: Document the reconciliation behavior: on rescan, if a provider
transitions to `missing`, its associated `installs` should receive
`install_status = missing`. This is implied by the data model but not stated
in the Provider Model.

---

## Missing Concepts Or Sections

- **Rsync/copy marker definition**: Whether and what Skillbox writes to disk to
  tag rsync/copy installs (see Critical Issue 2).

- **Adapter output contract**: Full type definition for adapter output (see
  Critical Issue 3).

- **Adapter lifecycle**: No documentation of when adapters are initialized, how
  they are registered, or whether they are loaded lazily vs eagerly at startup.

- **Provider path change/move reconciliation**: Not covered (see Suggested
  Improvement 7).

- **`configured` status trigger**: Not covered (see Suggested Improvement 2).

- **Skill format detection for Phase 2**: The Phase 2 section mentions
  `skills.detected_format` but does not specify whether the adapter or core
  logic is responsible for detecting format during scan. A brief note that
  "adapter is responsible for format detection within its provider scope" would
  close this gap.

---

## Over-Modeled Or Risky Areas

### `purpose = commands` and `purpose = config` in Phase 1

These purpose values have no Phase 1 use cases. They are likely placeholder
design for Claude's `.claude/commands/` path. Until a user flow references them,
they add noise. Defer explicitly.

### `configured` detection_status without a trigger

`configured` is a status with no defined trigger in Phase 1. It risks becoming
a catch-all that different parts of the code set inconsistently. Either remove
it from Phase 1 or define its exact trigger.

### Generic `.agents` as a catch-all vs a real convention

The generic_agents provider with `status = supported` may cause over-detection.
Any project with a `.agents` folder — even one unrelated to AI agents — would
trigger provider detection. For a GUI app targeting non-developers, this could
produce confusing "Generic Agents" badges on unrelated projects. This is
acceptable for Phase 1 but should be noted as a known UX risk.

---

## Adapter Boundary Review

The boundary statement is correct in principle: adapters return structured
results; core logic writes to DB. This is the right architecture.

Gaps:

1. Filesystem IO responsibility is not assigned (see Suggested Improvement 3).
   Without this, adapter implementations may diverge: some adapters doing their
   own `mkdir`, others delegating to core.

2. The adapter output type contract is missing (see Critical Issue 3).

3. The boundary document doesn't state whether an adapter can be stateless (a
   set of pure functions given a project path) or whether it carries instance
   state (e.g., caches the candidate list after load). Stateless is strongly
   preferable and simpler to test.

What is well designed: the principle of "adapter provides scope, core scan logic
classifies install state" correctly places install classification (broken symlink,
old_host, external_symlink, etc.) in core — not in adapters. Each adapter would
need to replicate this logic otherwise.

---

## Provider Path Candidate Review

The `provider_path_candidates` table with `purpose` and `priority` is the right
abstraction. It avoids the single-path limitation from the previous data model
version and is flexible enough to represent Claude's multi-path structure.

Issues:

- `priority` direction unspecified (Critical Issue 4).
- `purpose = commands` and `purpose = config` are Phase 1 noise (Suggested
  Improvement 1).
- No defined behavior when multiple same-purpose candidates exist with equal
  priority.

The schema dictionary (`docs/07-schema-dictionary.md`) correctly describes
`detected_path` as "resolve từ candidate `purpose = detect`" and `skills_path`
as "resolve từ candidate `purpose = skills`." This closes the gap flagged in the
prior data model follow-up review.

---

## Detection Flow Coverage

| State | Covered? | Notes |
|---|---|---|
| No provider detected | Yes | `projects.status = active`, warning `no_provider_detected`. |
| One provider detected | Yes | Single `project_providers` row. |
| Multiple providers detected | Yes | Multiple `project_providers` rows. |
| Unsupported provider | Broken | Detection flow filter excludes `unsupported` providers. See Critical Issue 1. |
| Missing provider path | Yes | `detection_status = missing`, warning expected. |
| Invalid structure | Yes | `detection_status = invalid_structure`. |
| Format unknown | Yes | `detection_status = format_unknown`. |
| Provider path changed or moved | Partially | Implied by rescan → `missing`. Not documented explicitly. |

---

## Install Target Resolution Review

The validity check is well-specified:

- `detection_status` = `detected` or `configured`
- `provider_definitions.status` = `supported` or `experimental`
- `skills_path` resolved
- If `skills_path` does not exist: `can_create_structure = 1`

The write block for `unsupported` providers is explicit and correct: "Nếu
provider là `unsupported`, Skillbox không được tự ghi file vào provider path."

One gap: there is no check that the resolved `skills_path` stays within the
project root. For a local desktop app this is low-risk, but an adapter with an
incorrect `relative_path` (e.g., starting with `../`) could resolve a path
outside the project. Recommendation: core install logic should verify that the
resolved `skills_path` is a subpath of the project root before writing.

---

## Initial Provider Assumptions Review

### Generic `.agents`

- Detection path `.agents` at project root — reasonable.
- Skills path `.agents/skills` — reasonable.
- `can_create_structure = true` — appropriate.
- `status = supported` — appropriate as the primary convention.
- Risk: over-detects any project with a `.agents` folder (noted above).
- **Assumption confidence: High.** This convention is explicitly described in
  README.md and docs throughout the project.

---

### Claude

- `status = experimental, can_create_structure = false` — correctly conservative.
- No path candidates defined — correctly deferred.
- **Assumption (no browsing):** Claude Code uses `.claude/` at project root, with
  `.claude/commands/` for slash commands. Skill-equivalent content may live at
  `.claude/` with CLAUDE.md and a `skills/` subfolder, or it may be a
  different structure entirely. This must be verified. Do not implement the
  Claude adapter without confirming the actual convention.
- The `can_create_structure = false` is the right choice until the convention
  is confirmed and stable.

---

### Codex

- Sharing generic `.agents` convention until a distinct adapter is needed —
  pragmatic and acceptable.
- **Assumption (no browsing):** OpenAI Codex does not have a publicly documented
  "skills" folder convention comparable to Claude Code. If Codex uses a project
  workspace convention, it may differ from `.agents`. Treat as unverified.
- Starting with `generic_agents` path candidates is low-risk because
  `experimental` status means the adapter is not guaranteed stable.
- `can_create_structure = true` — acceptable as a starting point, but if Codex
  has no `.agents` convention, scaffolding `.agents/skills` may confuse Codex
  users.

---

### opencode

- Sharing generic `.agents` convention — acceptable.
- **Assumption (no browsing):** opencode is an open-source AI coding assistant.
  If it follows a `.agents`-style convention, the shared path works. If not,
  this assumption will produce false positive detections.
- `can_create_structure = true` — same risk as Codex above.

---

### Antigravity CLI

- Sharing generic `.agents` convention — acceptable.
- **Note:** Antigravity CLI is likely an internal/Astraler product. The product
  owner should know the convention. If Antigravity CLI IS the primary user of
  `.agents`, the generic_agents adapter effectively serves as the Antigravity
  adapter for Phase 1, which is pragmatic.
- `can_create_structure = true` — appropriate if this is the primary convention
  for this product.

---

## UI/UX Review

The UI representation section covers the essential elements:

- Provider badge/icon via `icon_key` — correct.
- Support state display (`supported`, `experimental`, `unsupported`, `disabled`)
  — correct and sufficient for Phase 1.
- Detection status per project — correct.
- Skill count per provider — correct.
- Warning surfacing for missing/unsupported/invalid — correct.

One gap: no guidance on how UI should represent an `experimental` provider
differently from a `supported` one. Should the UI show a warning badge?
A tooltip? An asterisk? The data is there (`status = experimental`), but the
UX behavior is unspecified. This is not a blocker but should be decided before
UI design begins.

A second gap: the UI representation section does not address the `disabled`
status. A provider that is disabled — should it be hidden entirely, or shown as
unavailable? This matters if there's a Settings screen where users can enable or
disable providers.

---

## Phase 2 Conversion Readiness

The Phase 2 section is appropriately brief and does not over-engineer. The key
conditions for Phase 2 are already satisfied:

- Provider is an entity (not hardcoded).
- Installs are scoped per `project_provider`.
- Path candidates are decoupled from provider definition.
- `operations` can accommodate `convert_skill`.

Two remaining gaps to note for the future:

1. **Format detection ownership**: When Phase 2 is approached, the adapter will
   need a `detect_format()` responsibility. The current boundary document doesn't
   mention format detection. Add it to the adapter responsibilities list now
   (even if it returns `unknown` for all Phase 1 adapters) to set the expectation.

2. **`skills.detected_format`**: The open question in `docs/06-data-model.md`
   asks whether to add this field in Phase 1. The Provider Model does not add a
   recommendation. Recommendation: add `detected_format TEXT NULLABLE` to
   `skills` in Phase 1, seeded as `null`. This is a zero-cost additive field that
   prevents a migration-required schema change when Phase 2 begins.

---

## Recommended Changes

Changes required before implementation begins:

1. **Fix detection flow**: Change "Load provider_definitions đang
   enabled/supported/experimental" to "Load provider_definitions với status !=
   `disabled`." Add a separate note: "Install target resolution chỉ chấp nhận
   `supported` và `experimental`."

2. **Define rsync/copy scan rule**: Choose and document whether "Skillbox
   metadata" for rsync/copy detection is (a) the presence of an `installs` DB
   record with `install_mode = rsync_copy`, or (b) a filesystem marker file.
   Add a sentence to the "Scan Installed Skills" section stating the rule and
   the consequence when the DB record is absent.

3. **Specify adapter output contract**: Add a "Adapter Output Contract" subsection
   under "Provider Adapter Boundary" with field-level descriptions for `warnings`
   (list of `{ code, severity, message, action_key }`) and `entries` (list of
   `{ name, path, entry_type, symlink_target }`).

4. **Specify priority direction**: Add one line to "Provider Path Candidates":
   "Lower `priority` value wins. Adapter checks priority 1 before priority 10."

Non-blocking changes (do before first adapter is written):

5. Add a note deferring `purpose = commands` and `purpose = config` to future
   phases.

6. Clarify the `configured` detection_status trigger, or remove it from Phase 1
   scope.

7. State explicitly that filesystem IO (mkdir, symlink, rsync, delete) is
   performed by core Skillbox logic, not by adapter code.

8. Document that Claude path candidates must be researched and seeded before the
   Claude adapter can function.

9. Add automatic priority-based resolution for the case where multiple
   same-purpose candidates exist simultaneously, to avoid needing a new
   `detection_status` value.

10. Document that the `skills_path` resolved by core install logic must be
    verified to be a subpath of the project root before any write.

---

## Open Questions For The Product Owner

1. **What is Claude Code's actual skill convention?** Without this, the Claude
   adapter cannot be implemented. Verify `.claude/`, `.claude/commands/`, and
   any other paths before implementation starts.

2. **Do Codex, opencode, and Antigravity CLI have their own skill conventions?**
   If they use `.agents`, the generic adapter covers them. If not, separate path
   candidates are needed. This determines whether Phase 1 needs distinct adapter
   seeds for each.

3. **Is "Skillbox metadata" a DB record or a filesystem marker for rsync/copy
   detection?** This is Critical Issue 2. A product decision is needed.

4. **What does the UI show for `experimental` providers?** Badge, tooltip,
   warning? Define before UI design begins.

5. **What does the UI show for `disabled` providers?** Hidden or shown as
   unavailable?

6. **Should users be able to configure a provider manually (set
   `detection_status = configured`) in Phase 1?** If not, remove `configured`
   from Phase 1 scope.

7. **When multiple `skills` candidates match with equal priority, should the
   app auto-select or prompt the user?** For Phase 1, auto-select by highest
   priority candidate is recommended.

8. **Should `skills.detected_format` be added in Phase 1 as a nullable
   placeholder?** Low cost, removes a future migration. Recommended: yes.

---

## What Looks Solid

- **Adapter responsibility list** is clear and appropriately scoped: detect,
  resolve, scan, classify, scaffold (if permitted), report. No business policy
  logic in adapters.

- **Unsupported provider write policy** is correctly strict: "Nếu provider là
  `unsupported`, Skillbox không được tự ghi file vào provider path." This is
  the right security posture for a tool that writes to user's project files.

- **Scan install state rules** ("Scan Installed Skills" section) are
  comprehensive and correctly placed in core logic, not adapters. The symlink
  detection rules (valid/broken/old_host/external) are consistent with the data
  model.

- **Install target validity conditions** are correctly defined and comprehensive.
  The guard against writing into a path with `detection_status = unsupported` is
  explicitly stated.

- **Provider absence is not a blocking error**: "Provider absence không phải lỗi
  blocking" is the right product decision. A project with no providers should
  still open cleanly in the UI.

- **`can_create_structure` on `provider_definitions`** cleanly separates
  "detection-only" adapters from "setup-capable" adapters. Claude being
  `can_create_structure = false` is correctly conservative.

- **Phase 2 readiness** is handled lightly and correctly: no premature Phase 2
  tables, but the model is designed additively.

- **Generic `.agents` as a seed provider** is a practical starting point that
  ensures Skillbox is functional on day 1 without requiring all provider
  conventions to be researched simultaneously.
