<!--
Template — file-backed /goal cho Astraler Skillbox.
Lưu ở `.scratch/goal-<slice>-<phase>.md`, gửi tmux 1 dòng:
  /goal đọc `.scratch/goal-<slice>-<phase>.md` rồi tiến hành.

Nguồn gốc: template `/goal` của user (Obsidian References).
Adapt cho Tom/Larry phase-gate workflow của repo này.

Đọc anatomy + when/anti-pattern ở docs/playbooks/agent-orchestration.md § /goal.
-->

/goal [THE FINAL OUTCOME — what "done" looks like in one line]

Bạn là **Tom** (hoặc **Larry**). Phase = <Brainstorm | Spec | Plan | Implement | Review | Fix>.

── CONTEXT ──

- Project: Astraler Skillbox
- Slice / Feature: <tên slice>
- Stack: <chỉ phần liên quan — Electron + React renderer, Go core, SQLite, …>
- Current state: <slice đang ở đâu — chưa có gì / đã có brainstorm / đã có spec>
- Working dir: <branch + relevant subpath, vd `feat/dashboard-plugins-metric` · `apps/desktop/renderer/src/screens`>
- Inputs:
  - Spec: <path nếu có>
  - Plan: <path nếu có>
  - Brainstorm note: <path nếu có>
  - Doc liên quan: <docs/...>
- Constraints:
  - PHẢI: <bullets — việc bắt buộc>
  - KHÔNG: <negative scope — đừng đụng module X, đừng refactor Y, đừng đổi contract>
- Audience: <user product / team / Larry review / orchestrator handoff>

── SUCCESS CRITERIA (ALL MUST BE TRUE) ──

1. <Specific measurable outcome>
2. <Specific measurable outcome>
3. <Specific measurable outcome>
4. Final deliverable runs/builds without errors (tests pass nếu trong scope)
5. Bằng chứng được show rõ ràng (diff, log, screenshot, scenario pass/fail)
6. Docs cập nhật theo `docs/playbooks/documentation.md` nếu thay đổi concept

── OPERATING RULES — NON-NEGOTIABLE ──

1. PLAN FIRST. Output task list / outline trước khi viết code hay spec body.
2. RESPECT PHASE GATE. Nếu phase = Spec → DỪNG sau khi spec xong, KHÔNG implement. Nếu phase = Plan → DỪNG sau plan, KHÔNG code. Nếu phase = Implement → dừng trước PR/merge để chờ user duyệt.
3. SELF-VERIFY. Sau mỗi bước: chạy test/typecheck/lint liên quan, capture output, confirm pass.
4. DEBUG YOURSELF. Fail → diagnose + fix trong cùng phase. Không hand lại Orchestrator trừ khi blocked thực sự.
5. USE EVERY TOOL. Đọc code, grep, run, MCP, fixtures — không đoán mò.
6. NO PLACEHOLDERS. Không TODO trong commit, không stub, không "// implement later".
7. PROGRESS LOG. Track completed / in-flight / decisions / blockers trong response cuối phase.
8. STAY ON SCOPE. Phát hiện off-spec → ghi note, không tự ý mở rộng. Báo Orchestrator nếu critical.
9. IF BLOCKED. Log rào cản cụ thể, tiếp tục mọi nhánh parallelizable, không idle.
10. CHECK SUCCESS BEFORE STOPPING. Re-read SUCCESS CRITERIA, xác nhận từng dòng đã đạt; nếu không, fix trước khi báo done.

── QUALITY BAR ──

- Code: clean, typed, theo conventions (`AGENTS.md` § Conventions).
- Architecture: tôn trọng hard rules trong `docs/10-technical-architecture.md` (renderer / main / Go boundaries).
- Output: vượt được Larry code review + smoke tests.
- Docs: concept mới / RPC mới / migration mới → update doc canonical theo `documentation.md` map.
- Commit: short imperative, kèm trailer `DOC-VERIFIED: <reason>` nếu push range chạm `main`.

── FINAL DELIVERABLE ──

- [OK] Confirm từng Success Criterion đã đạt.
- [FILE] List mọi file created/modified (path tương đối).
- [RUN] Lệnh cụ thể để verify (test cmd, smoke scenario, screenshot path).
- [PROOF] Diff / log output / screenshot / scenario result.
- [LOG] Decisions made, trade-offs đã chọn, thứ cần biết.
- [WARN] Known limitations, follow-up tasks, scope hoãn lại.
- [STOP] Phase tiếp theo là gì + ai làm (vd "Larry review spec next").
