# Provider Model Follow-Up Review Result

## Reviewer

- Agent/model: Claude Sonnet 4.6 (claude-sonnet-4-6)
- Review date: 2026-05-22
- Context used: docs/01-product-brief.md, docs/02-product-notes.md,
  docs/03-information-architecture.md, docs/04-user-flows.md,
  docs/05-edge-cases-and-ux-states.md, docs/06-data-model.md,
  docs/07-schema-dictionary.md, docs/08-provider-model.md,
  docs/archive/review-results/provider-model-review.md
- Browsing used: no
- Prior review: provider-model-review.md (same date, same session)

---

## Decision

**Approved.**

All six critical issues from the prior review are resolved. The model is ready
for implementation with the non-blocking notes below.

---

## Critical Issue Verification

### 1. Detection flow loads unsupported providers for detection while blocking writes

**Prior state**: "Load provider_definitions đang enabled/supported/experimental"
excluded `unsupported` definitions.

**Current state**: "Load provider_definitions có status khác disabled" — correctly
includes `supported`, `experimental`, and `unsupported`.

The subsequent clarification is explicit and correctly separates the two
concerns:

> "Detection được phép nhận diện `unsupported` providers để UI báo rõ cho user.
> Install target resolution mới là nơi chặn write vào provider chưa support."

**Status: Resolved. ✓**

---

### 2. rsync/copy detection rule is now defined

**Prior state**: "entry là folder thường có Skillbox metadata" — undefined.

**Current state**: The rule is now unambiguous:

> "Nếu entry là folder thường và có `installs` DB record cho path đó với
> `install_mode = rsync_copy`, mode là `rsync_copy`."

The choice of DB record over filesystem marker is explicitly documented, and the
consequence on DB loss is stated:

> "Nếu database bị mất và app scan lại từ đầu, các rsync/copy installs cũ có
> thể bị phân loại thành `direct`; user cần sync lại bằng Skillbox nếu muốn
> đưa chúng về managed state."

**Status: Resolved. ✓**

---

### 3. Adapter output contract is specified

**Prior state**: `warnings` and `entries` listed but untyped.

**Current state**: A "Minimum output contract" block with field-level types is
present:

```text
warnings: list of {
  code: text
  severity: info | warning | error | blocking
  message: text
  action_key: text | null
}
entries: list of {
  name: text
  path: absolute path
  entry_type: symlink | directory | unknown
  symlink_target: path | null
}
```

This is sufficient for Phase 1. Adapters can now be written consistently.

**Status: Resolved. ✓**

---

### 4. `priority` field direction is specified

**Prior state**: "Priority thấp/cao theo convention implementation chọn."

**Current state**:

> "Priority thấp hơn thắng. Adapter kiểm tra `priority = 1` trước `priority = 10`."

Equal-priority tie-breaking is also defined:

> "Nếu nhiều candidate cùng purpose có cùng priority, adapter chọn theo thứ tự
> path alphabet để Phase 1 không cần thêm UI chọn path."

**Status: Resolved. ✓**

---

### 5. Filesystem write boundary is clear

**Prior state**: Adapter responsibilities listed "Tạo provider folder structure
nếu adapter được phép" with no clarity on who performs IO.

**Current state**:

> "Adapter cũng không tự thực hiện filesystem writes. Các thao tác như `mkdir`,
> symlink creation, rsync/copy, delete, relink đều do core Skillbox logic thực
> hiện sau khi adapter trả về path và capability metadata."

Core logic responsibility also updated: "Thực hiện filesystem writes sau khi
validate output của adapter."

The adapter responsibility list is now corrected: "Báo provider folder structure
có thể tạo được không" — adapter reports capability; core performs IO.

**Status: Resolved. ✓**

---

### 6. Install target path safety is addressed

**Prior state**: No check that resolved `skills_path` stays within project root.

**Current state**: Validity condition added:

> "`skills_path` nằm trong project root sau khi canonicalize/normalize path."

**Status: Resolved. ✓**

---

## Additional Improvements Verified

| Improvement from prior review | Applied? |
|---|---|
| `commands`/`config` purposes deferred for Phase 1 | Yes — "Phase 1 adapter chỉ bắt buộc cần `detect` và `skills`." |
| `configured` status trigger clarified | Yes — "future/manual setup state. Phase 1 chưa cần flow riêng." |
| `key` vs `provider_type` distinction documented | Yes — "key là stable identifier... provider_type là enum/category để app dispatch adapter." |
| `experimental` UI behavior defined | Yes — "badge/tooltip nhẹ." `disabled` behavior also defined. |
| Claude implementation blocked until convention confirmed | Yes — "Không implement Claude scan/install cho tới khi convention này được xác minh." |
| Provider path moved/missing reconciliation documented | Yes — "Khi rescan thấy provider path cũ đã missing, `project_providers.detection_status` nên chuyển thành `missing`, và các installs thuộc provider đó nên được đánh dấu `install_status = missing`." |
| Format detection added to adapter responsibilities | Yes — "Detect skill format trong provider scope khi Phase 2 conversion bắt đầu." |
| `can_create_structure` attributed to core logic | Yes — "core Skillbox logic scaffold folder/path." |

---

## Remaining Blockers

None.

---

## Non-Blocking Suggestions

### 1. Stale Open Question about multi-path skills path selection

The "Open Questions" section still asks:

> "Khi một provider có nhiều skills path hợp lệ, app nên auto chọn theo
> priority hay yêu cầu user chọn?"

This question is already answered in the body of the same document:

> "Nếu nhiều candidate cùng purpose cùng tồn tại, adapter chọn candidate có
> priority thấp nhất."

The Open Question should be closed or removed to avoid sending implementers to
search for a decision that has already been made.

---

### 2. Codex, opencode, Antigravity CLI path candidates create a multi-detection risk

The document says these providers "may use the generic `.agents` convention"
until they need distinct adapters. If implemented literally — i.e., seeding their
`provider_path_candidates` rows with `purpose = detect, relative_path = .agents`
— any project with a `.agents` folder would produce **four** `project_providers`
rows: Generic Agents, Codex, opencode, and Antigravity CLI, all pointing to the
same path. The user would see four provider badges for a single filesystem
convention.

Two safe approaches for Phase 1:

- **Recommended**: Seed Codex, opencode, and Antigravity CLI with **no** path
  candidates. They appear in `provider_definitions` as future-ready entries but
  produce no detection hits until given distinct candidates. Users see only
  "Generic Agents" when `.agents` is found.

- **Alternative**: Document a detection de-duplication rule: if multiple
  providers resolve the same `detect` path, only the most specific provider
  (lowest priority value, or alphabetical) creates a `project_providers` row.

The current doc does not specify which approach to take. This should be decided
before seeding provider data.

---

### 3. `entries.path` semantics could be more explicit

The output contract defines `path: absolute path` for entries. It is implied
this is the absolute path to the skill folder within the provider's
`skills_path`, but the contract does not say this explicitly.

Suggestion: add one line to the entries contract:

```text
path: absolute path to the skill entry within the provider's skills_path
```

This prevents an adapter returning the path relative to the project root vs the
full absolute path.

---

## Contradictions Between `docs/08-provider-model.md` and Other Docs

### Contradiction: `can_create_structure` ownership in `docs/06` and `docs/07`

`docs/08-provider-model.md` now correctly states:

> "`can_create_structure` cho biết provider có thể được **core Skillbox logic**
> scaffold folder/path cần thiết..."
>
> "Adapter cũng không tự thực hiện filesystem writes."

However, both `docs/06-data-model.md` and `docs/07-schema-dictionary.md` still
describe `can_create_structure` using "adapter":

**`docs/06-data-model.md`**, section "7. provider_definitions", Notes:

> "`can_create_structure` cho biết **adapter** có thể scaffold provider folder
> hay chỉ được scan/install vào structure đã tồn tại."

**`docs/07-schema-dictionary.md`**, `provider_definitions` table:

> "Cho biết **adapter** có thể scaffold provider folder structure hay không."

A developer reading docs/06 or docs/07 during implementation would conclude that
the adapter performs the scaffold IO — directly contradicting the boundary
established in docs/08. This needs to be fixed in both files before any adapter
is written.

Fix: In docs/06 and docs/07, replace "adapter có thể scaffold" with "core
Skillbox logic có thể scaffold" to match docs/08.

---

### No other contradictions found

The following were checked and are consistent:

- `docs/04` user flows (Scan Project, Add Project, Install Skill) are consistent
  with the detection flow and install target resolution rules in `docs/08`.
- `docs/05` provider states (`unsupported`, `invalid_structure`, `format_unknown`,
  `missing`) all map to values defined in `docs/08`.
- `docs/06` / `docs/07` install mode, install status, and detection_status enums
  match `docs/08` exactly (except the `can_create_structure` issue above).
- The adapter output contract's `detection_status` values match
  `project_providers.detection_status` in `docs/07`.
- The adapter output contract's `severity` values (`info`, `warning`, `error`,
  `blocking`) match `warnings.severity` in `docs/07`.

---

## Overall Assessment

`docs/08-provider-model.md` is internally consistent, addresses all prior
critical feedback, and is coherent with the product, data model, and edge case
docs. The `can_create_structure` language in `docs/06` and `docs/07` must be
updated to match before implementation begins. The multi-detection risk for
Codex/opencode/Antigravity CLI path candidates should be decided before provider
seed data is written. Both issues are small and contained.

The model is ready for implementation.
