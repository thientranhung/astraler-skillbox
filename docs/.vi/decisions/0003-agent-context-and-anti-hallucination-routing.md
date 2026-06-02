# ADR-0003: Agent Context And Anti-Hallucination Routing

- **Status:** accepted
- **Date:** 2026-06-02
- **Deciders:** owner + Codex
- **Tags:** process, documentation, governance, qa

## Context

Agents thường bắt đầu task từ fresh context. Việc manual load toàn bộ project
governance playbook ở đầu mỗi session vừa tốn context vừa dễ bị skip, nhưng nếu
đưa toàn bộ process rules vào `AGENTS.md` thì always-loaded context sẽ quá lớn.
Repo đã có governance, QA, docs, và agent-orchestration playbooks chi tiết; phần
còn thiếu là routing ngắn gọn cộng một guardrail chống hallucination đủ nhỏ để
hoạt động trong fresh context.

## Decision

Giữ `AGENTS.md` ngắn gọn và chỉ thêm task routing cùng pre-edit/pre-verdict
guardrail bắt buộc dạng ngắn.

Thêm `docs/context-map.md` làm compact discovery map cho code, docs, contracts,
và QA paths.

Giữ full process rules trong các playbook hiện có:

- `docs/playbooks/governance-project.md` cho phase gates, review, PR, ownership,
  và full implementer/reviewer anti-hallucination checklist.
- `docs/qa/governance.md` cho QA evidence, result semantics, clean GO, và QA
  anti-hallucination checklist.
- `docs/playbooks/documentation.md` cho source-of-truth và docs drift mapping.
- `docs/playbooks/agent-orchestration.md` cho tmux/agent handoff operations.

Không thêm Plan/Act mode mới. Superpowers, `/goal`, và phase gates hiện tại đã
bao phủ planning và approval.

## Alternatives Considered

- **Đưa toàn bộ governance playbook vào `AGENTS.md`** — loại vì tăng
  always-loaded context và duplicate source of truth.
- **Tạo cây `.memory/`, `.rules/`, và `.generators/` riêng** — tạm loại vì repo
  đã có canonical docs, playbooks, `.scratch/`, `/goal`, và QA runs bao phủ các
  vai trò đó mà không cần thêm một hierarchy source-of-truth nữa.
- **Tiếp tục dựa vào manual playbook loading** — loại vì fresh agent contexts có
  thể bỏ lỡ routing và anti-hallucination checks bắt buộc.
- **Thêm Plan/Act policy mới** — loại vì duplicate Superpowers và phase gates
  hiện tại.

## Consequences

**Positive:**

- Fresh agent contexts có đủ routing để bắt đầu an toàn.
- Broad repository discovery nên tốn ít token hơn vì bắt đầu từ map.
- Full governance và QA rules vẫn nằm ở source-of-truth hiện có.
- Anti-hallucination checks áp dụng trước cả code edits lẫn review/QA verdicts.

**Negative / chi phí:**

- `docs/context-map.md` phải được giữ current khi major folders hoặc
  source-of-truth docs đổi chỗ.
- `AGENTS.md` tăng một ít always-loaded text.

**Neutral / cần theo dõi:**

- Nếu repeated exceptions hoặc task workflows tích tụ, thêm chúng vào bề mặt
  playbook nhỏ nhất hiện có trước khi cân nhắc một cây rules/generators mới.

## Implementation Notes

- Add `docs/context-map.md`.
- Update `AGENTS.md` với routing và guardrail ngắn.
- Update `docs/playbooks/governance-project.md` với full implementer/reviewer
  checklist.
- Update `docs/qa/governance.md` với QA verdict checklist.
- Update `docs/index.md` và `docs/decisions/index.md`.

## Verification

- Confirm toàn bộ linked docs tồn tại.
- Confirm không có process rule mới duplicate full governance hoặc QA playbooks.
- Với code hoặc QA task tương lai, agents nên start từ `AGENTS.md`, rồi chỉ load
  deeper playbook liên quan và `docs/context-map.md` khi cần discovery.

## References

- `AGENTS.md`
- `docs/context-map.md`
- `docs/playbooks/governance-project.md`
- `docs/qa/governance.md`
