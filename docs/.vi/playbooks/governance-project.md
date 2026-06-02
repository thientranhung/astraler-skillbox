# Governance Project

> Quy chế hoạt động cho mọi agent làm việc trong repo này. Đọc trước khi code, review, hay mở PR. Tài liệu này định nghĩa **luật chơi**: phase nào, gate nào, ai sở hữu gì, chất lượng tới đâu.

## TL;DR

- Phase chuẩn: **Brainstorm → Branch? → Spec → Spec review → User approve → Plan → Implement + Docs → PR → Review + Smoke/QA → Merge.**
- **Không skip user approval ở Spec** với feature, cross-layer, behavior, schema, RPC, provider, filesystem, security, hoặc process change. Tiny low-risk work được nén phase theo rule bên dưới.
- Branch + PR **bắt buộc** khi có schema/migration hoặc breaking change. Còn lại xem bảng Decision Rule.
- **One actor per file.** Implementer code, Reviewer chỉ review — không edit file.
- Docs canonical cập nhật **trong cùng slice, trước review**, không nợ sang sau.

## Roles

| Vai trò | Trách nhiệm |
|---------|-------------|
| **Implementer** | Brainstorm, spec, plan, implement, mở PR, merge sau khi được approve, cập nhật docs. |
| **Reviewer** | Code / spec / security review, smoke test. Verdict: approve / block / needs-discussion. **Không edit file production.** |

> Slice = lát cắt mỏng cross-layer (UI → service → data). Một slice đi trọn từ Spec tới Docs/QA evidence.

## `/goal` — đóng gói việc dài

`/goal` là capsule hand-off cho việc dài/stateful, không phải lệnh giao việc chung. Chỉ dùng khi outcome, scope, negative scope, success criteria, stop condition, và phase boundary đã rõ.

Không dùng `/goal` cho subagent/background thread/same-session worker prompt, brainstorm ban đầu, exploratory reading, review comment, fix nhỏ, hoặc task chưa có success criteria.

Khi cần viết prompt dài cho hand-off, dùng template:

- [`templates/goal-file.md`](templates/goal-file.md) — cho hand-off dài, nhiều context, nhiều path, hoặc cross-phase.
- [`templates/goal-inline.md`](templates/goal-inline.md) — cho follow-up nhỏ, cùng phase, scope rất rõ.

File-backed `/goal` phải được viết dưới `.scratch/` và tuân thủ convention ở mục bên dưới.

## `.scratch/` — workspace nháp và hand-off

`.scratch/` là workspace tạm, gitignored, dùng cho:

- prompt dài/file-backed `/goal` trước khi gửi cho agent tmux;
- brainstorm note, draft spec/plan, hoặc phác thảo AI giữa user và agent;
- run note tạm, context pack, checklist, hoặc hand-off capsule chưa phải source-of-truth.

Không dùng `.scratch/` làm docs canonical, spec đã duyệt, ADR, QA report chính thức, hoặc nơi lưu quyết định cuối cùng. Khi nội dung trong `.scratch/` được duyệt, digest/sync sang tài liệu canonical phù hợp trong `docs/`.

**Naming bắt buộc:** mọi file trong `.scratch/` phải có prefix ngày để sort và trace được:

```text
YYYY-MM-DD-<topic>.md
YYYY-MM-DD-<topic>-<phase>.md
YYYY-MM-DD-goal-<slice>-<phase>.md
```

Ví dụ:

```text
.scratch/2026-06-01-governance-project-review.md
.scratch/2026-06-01-goal-plugin-settings-spec.md
.scratch/2026-06-01-dashboard-slice-plan.md
```

Tên file dùng lowercase kebab-case. Nội dung nên ghi rõ phase, owner, input paths, constraints, success criteria, và stop condition nếu dùng cho hand-off.

## Workflow Skills / Superpowers

Governance không phụ thuộc vào một workflow engine cụ thể, nhưng agent **phải khai thác workflow skills có sẵn** khi task khớp phase. Nếu môi trường có Superpowers, dùng Superpowers làm workflow engine mặc định; nếu không có, agent phải mô phỏng cùng output/gate bằng prompt thường hoặc playbook tương ứng.

Mapping tối thiểu:

| Nhu cầu | Workflow skill ưu tiên |
|---|---|
| Làm rõ intent, explore approach, viết spec/design | `brainstorming` |
| Chuyển spec đã duyệt thành implementation plan | `writing-plans` |
| Thực thi plan có task độc lập, ownership rõ | `subagent-driven-development` |
| Thực thi plan tightly-coupled trong một context | `executing-plans` |
| Request/reconcile review loop | `requesting-code-review`, `receiving-code-review` |
| Verify trước khi báo done | `verification-before-completion` |

Workflow skills không được bỏ qua governance: user approval gate, ownership, docs/ADR, review, PR, QA, và `.scratch/` convention vẫn áp dụng.

Khi giao việc cho subagent/worker, prompt phải bounded và có tối thiểu:

```text
Objective:
Owned files/modules:
Context:
Constraints:
Verification:
Expected final report:
```

Subagent/worker không tự mở rộng scope, không tự đổi owner file, không tự merge phase. Kết quả cuối nên dùng một trong các trạng thái:

- `DONE`
- `DONE_WITH_CONCERNS`
- `NEEDS_CONTEXT`
- `BLOCKED`

## Anti-Hallucination Checklist

Trước khi edit code, Implementer phải verify:

- Target files và directories có tồn tại.
- Symbols, components, RPC methods, contract files, tables, và provider names
  được nhắc tới có tồn tại trước khi dùng.
- Import paths và package boundaries khớp với code gần đó.
- Đã inspect pattern gần đó trước khi thêm abstraction mới.
- Boundary của layer bị chạm đã rõ: renderer, Electron main, preload, Go core,
  repository/SQL, filesystem gateway, provider adapter, hoặc shared contract.
- Contract/schema changes có generated files và drift checks.
- Docs và QA impact được map qua [`documentation.md`](documentation.md) và bảng
  QA scope bên dưới.

Trước khi ra review verdict, Reviewer phải verify:

- Target đã inspect rõ ràng: diff, commit, PR, spec, hoặc run report.
- File/line references và evidence paths có tồn tại.
- Claims được grounding bằng source, tests, contracts, docs, hoặc QA evidence.
- Verdict `PASS`/`APPROVE` không chỉ dựa vào summary của Implementer.
- Docs impact đã được kiểm tra khi concept thay đổi.

Trước khi search code rộng, đọc [`../context-map.md`](../context-map.md) để chọn
path có khả năng đúng, rồi mới search trong vùng target.

## Phase Gates

1. **Brainstorm & scope** — output kèm **Risk Classification** (bảng dưới).
2. **Branch decision** — áp Decision Rule, tạo branch nếu cần, **trước** Spec.
3. **Spec** — thiết kế kèm smoke scenarios.
4. **Spec review** — Reviewer.
5. **User approval** — gate cứng cho mọi work không thuộc tiny low-risk exception.
6. **Implementation plan**.
7. **Implement + docs** — code/test/docs trong cùng slice; self-verify trước khi mở PR.
8. **PR create** — nếu đang trên branch, push rồi tạo PR; không gộp create + merge.
9. **Review + smoke/QA** — Reviewer review trên PR khi có PR. Lỗi → `BLOCK`/request changes + `file:line`; Implementer fix + push → re-review. Lặp đến sạch → approve.
10. **Merge** — Implementer merge sau khi review/QA gates pass.

Tiny low-risk work có thể nén phase: docs-only/test-only/UI polish nhỏ, không đổi behavior, không đụng schema/RPC/provider/filesystem/security, <50 LOC, direct-to-main. Khi đó user request được xem là approval, nhưng vẫn phải có short plan, self-verify, và ghi rõ vì sao skip branch/spec review/QA bank.

## Workflow Branch & PR

### Risk Classification (đóng ở cuối brainstorm)

| Field | Value |
|-------|-------|
| Layers | UI / contract / Go / SQL / docs |
| Breaking change | yes / no |
| Schema/migration | yes / no |
| Est. LOC | <50 / 50–300 / >300 |
| Workflow | direct-to-main / branch + PR |

### Decision Rule

| Điều kiện | Workflow |
|---|---|
| Schema/migration: yes **HOẶC** Breaking change: yes | **MUST** branch + PR |
| Layers ≥ 3 **HOẶC** Est. LOC > 300 | **SHOULD** branch + PR |
| Multi-slice độc lập song song | worktree per slice |
| UI-only / docs-only, < 50 LOC | OK direct-to-main |

- Branch naming: `<type>/<kebab-slug>` (vd `feat/dashboard-plugins-metric`) hoặc provider/tooling prefix tương thích (vd `codex/<type>-<slug>`). PR target luôn là `main`.
- **KHÔNG gộp create + merge.** Push → `gh pr create` → review trên PR → fix loop → merge.
- Review phải để lại dấu vết thật trên PR. `reviews: []` sau merge = gate bị bỏ qua.

> **Gotcha same-owner:** GitHub chặn `--approve` / `--request-changes` trên PR của chính account đó. Reviewer đăng verdict bằng `gh pr comment` — ghi rõ **APPROVE / BLOCK + file:line**. `reviewDecision` sẽ rỗng dù đã review xong; dùng `gh pr checks` + `mergeStateStatus=CLEAN` làm điều kiện merge.

## Review & Smoke

| Loại review | Target |
|---|---|
| Code review | diff / commit |
| PR review | full PR scope |
| Spec/design review | architecture, risk, missing case |
| Security review | filesystem, network, auth, data exposure, injection, data loss |

**Verdict — Reviewer trả về đúng một trong ba:** `APPROVE` / `BLOCK` / `NEEDS_DISCUSSION`. Với `BLOCK` hoặc `NEEDS_DISCUSSION`, kèm:

```text
Severity: P0 | P1 | P2 | P3
File/area:
Issue:
Why it matters:
Required fix:
Evidence:
Docs impact: none | required | missing
```

Concept đổi mà thiếu docs → verdict là `BLOCK`.

**Review depth co giãn theo rủi ro:**

- Docs/test/low-risk nhỏ → review nhẹ hoặc skip (ghi lý do).
- UI flow change → code review + smoke evidence/screenshot khi cần.
- Cross-layer → spec/plan review **trước** implement, code review **sau**.
- Schema / RPC / provider / filesystem write → deep review, cân nhắc security + smoke.

**Nguyên tắc smoke:**

- **Smoke scenarios thiết kế trong phase Spec**, không phải lúc execute. Implementer propose → user/Reviewer duyệt → Reviewer execute, report pass/fail kèm evidence. Gap phát hiện khi execute → log lại vào spec.
- Smoke verify **end-to-end** (UI, CLI, API, IPC, data flow), không phải unit.
- Nếu smoke thuộc delta/smoke/release/regression QA, phải dùng skill `astraler-qa` và QA bank: chọn case/tag, tạo run folder, ghi `run-plan.yaml`, append `results.jsonl`, viết `report.md`, và lưu evidence dưới `docs/qa/runs/<run>/evidence/`.
- Drive Electron app qua CDP instance `pnpm dev` đang chạy — đọc [`agent-browser-smoke.md`](agent-browser-smoke.md), **không** launch instance thứ 2.
- Reviewer ra finding → report verdict (BLOCK + `file:line`) rồi **DỪNG**. Không tự-poll chờ fix, không tự drive vòng lặp, **không tự sửa file production**. Một review = một verdict.
- "No verdict" (không inspect) → rerun từ đầu. Drift docs phát hiện trong review → fix trước khi close.

### QA Scope Mapping

| Thay đổi / rủi ro | Kỳ vọng QA |
|---|---|
| Schema, tính nhất quán DB/filesystem, destructive path, hành vi install/remove/switch | Chạy các case T0 bị ảnh hưởng + bằng chứng out-of-band DB/filesystem/screenshot. |
| Core user journey hoặc cross-screen truth | Chạy các case T1 bị ảnh hưởng và invariants liên quan. |
| Bug fix | Thêm hoặc chọn regression case, rồi chạy các case liên quan. |
| Release readiness | Chạy release QA theo [`../qa/README.md`](../../qa/README.md). |
| Docs-only/test-only/tiny UI polish | Có thể skip QA bank nếu behavior không đổi; ghi rõ lý do. |

## Docs & ADR

Concept đổi → cập nhật doc canonical **trong cùng slice và trước review**. Bản đồ đầy đủ ở [`documentation.md`](documentation.md). Trigger cần check:

> schema/migration · RPC method / notification · domain object · provider adapter · UI screen · user flow · edge/UX state · implementation pattern lặp lại · architecture/process boundary.

**ADR** cho quyết định lớn: thay đổi architecture, domain, tech stack, hoặc process/workflow. **KHÔNG** ADR cho refactor local, typo, format, config vặt, hay thay đổi tests-only.

**Commit trailer** khi land vào `main` (đụng concept có docs):

```text
DOC-VERIFIED: <lý do>
```

## Quality Bar

- **Plan first** — không code mù; có plan/spec trước khi sửa.
- **Respect phase gate** — dừng đúng phase, không chạy gộp spec → code → PR bỏ qua review + user approval.
- **Self-verify** — build/test pass không lỗi; show evidence (diff, log, screenshot, smoke result) trước khi tuyên bố done.
- **No placeholders** — không TODO giả, không stub bỏ ngỏ.
- **Stay on scope** — không refactor ngoài phạm vi slice; có khối **KHÔNG** rõ ràng trong constraints.
- **Measurable success** — criterion viết được cách verify, không phải "implement feature X".
- Tuân code conventions + architecture hard rules ([`10-technical-architecture.md`](../../10-technical-architecture.md)). Code phải qua được review. Commit kèm trailer `DOC-VERIFIED` khi đụng docs.

## Ownership

- **One actor per file at a time.**
- Reviewer không edit file trừ khi user explicit yêu cầu.
- Agent fail/kẹt → khôi phục agent trước (clear, restart, split task, switch model) → vẫn kẹt thì hỏi user. Không tự ý vượt vai trò.

## Maintenance

Mỗi governance failure → thêm 1 rule vào đây. **Principles over recipes, references over duplication.** Rule chỉ áp dụng đúng 1 lần thì không thêm.

## Related Operational Playbooks

Các playbook vận hành agent, hand-off, runtime, tmux, hoặc `/goal` phải tuân thủ tài liệu governance này.
