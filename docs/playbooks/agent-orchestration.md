# Agent Orchestration Playbook

> Khi bí, đọc TL;DR. Khi cần tra cứu, jump bằng tiêu đề.

## TL;DR

- **Tom** code, **Larry** review, **Orchestrator** chỉ điều phối (không quyết định kỹ thuật).
- Plan dài / nhiều bước / cross-layer / hand-off cross-phase → **dùng `/goal`** để khoá context. Việc ngắn, scope rõ → message thường là đủ.
- Phase chuẩn: Brainstorm → Branch? → Spec → Spec review → User approve → Plan → Implement → Review → Smoke → PR → Docs.
- Branch + PR là **bắt buộc** khi có schema migration hoặc breaking change. Phần còn lại xem bảng.
- Mỗi lỗi orchestration → thêm 1 rule vào playbook này. Không lặp lại 2 lần.

## Roles

| Tên | tmux pane | Vai trò |
|------|-----------|------|
| **Tom** | `agent-tech-skillbox` | Senior dev. Brainstorm, spec, plan, implement. |
| **Larry** | `agent-lead-skillbox` | Reviewer & QA. Code/spec/security review, smoke. **Không edit file**. |
| **Orchestrator** | (session này) | PM & coordinator. Quyết **ai** và **khi**, không quyết **gì** hay **làm sao**. |

Orchestrator chỉ được sửa trực tiếp: playbook này, process docs, doc fix tí hon, hoặc khi user explicit cho phép.

## `/goal` — công cụ cho việc dài

`/goal` là cách Orchestrator đóng gói một việc lớn cho Tom/Larry khi context dễ trôi. **Không phải đưa task trần; là khoá role + phase + expected output**. Một goal yếu = agent đoán mò → context drift → phải redo. Với plan dài, đầu tư 2 phút viết goal tốt tiết kiệm 30 phút sửa hậu kỳ. Đây là tiện ích để dùng đúng lúc, không phải cổng bắt buộc cho mọi việc.

### Khi nào nên dùng `/goal`

- Plan dài / nhiều bước, slice cross-layer, hoặc >1 commit → nên dùng (context dễ trôi).
- Hand-off cross-phase (Spec → Plan → Implement), cần gắn output phase trước làm input → nên dùng.
- Single-edit, follow-up nhỏ, scope cực rõ → message thường là đủ, **không cần** `/goal`.

### Anatomy — 5 blocks (cho file-backed goal)

Một file-backed `/goal` đầy đủ gồm 5 block dưới đây. Inline goal dùng bản rút gọn (xem "Hai flavor") — không cần đủ cả 5 khối. Cấu trúc kế thừa template `/goal` gốc của user, adapt cho phase-gate workflow.

1. **CONTEXT** — Project, slice, stack liên quan, current state, working dir/branch, inputs (path tới spec/plan/brainstorm), constraints (PHẢI + KHÔNG), audience. Khoá role + phase ngay đầu block (`Bạn là Tom. Phase = Spec.`).
2. **SUCCESS CRITERIA** — list đánh số, mỗi dòng measurable. Bắt buộc có 1 dòng "build/test pass without errors" và 1 dòng "evidence shown" (diff, log, screenshot, smoke result). Nếu thay đổi concept → thêm 1 dòng docs cập nhật theo `documentation.md`.
3. **OPERATING RULES** — 10 rules non-negotiable (Plan first, Respect phase gate, Self-verify, Debug yourself, Use every tool, No placeholders, Progress log, Stay on scope, If blocked, Check success before stopping). Adapt từ template gốc: rule #2 thay "WORK AUTONOMOUSLY" bằng "**RESPECT PHASE GATE**" — dừng đúng phase, không chạy gộp spec → code.
4. **QUALITY BAR** — code conventions, architecture hard rules (`docs/10`), Larry-review-passable, docs cập nhật, commit trailer `DOC-VERIFIED`.
5. **FINAL DELIVERABLE** — `[OK]` từng criterion, `[FILE]` paths, `[RUN]` cmd verify, `[PROOF]` evidence, `[LOG]` decisions, `[WARN]` limitations, `[STOP]` phase tiếp theo + ai làm.

### Hai flavor

**Inline** — nội dung < 500 ký tự, gửi thẳng qua tmux. Dùng cho follow-up nhỏ trong cùng phase, single-edit, scope cực rõ. Chỉ giữ phần cốt lõi (outcome + context + success + stop), không cần đủ 5 block.

**File-backed** — viết `.scratch/goal-<slice>-<phase>.md` rồi gửi `/goal` 1 dòng trỏ tới file. Dùng khi cần context dài, nhiều file path, snippet, hand-off cross-phase. Naming: `goal-<slice>-<phase>.md` (vd `goal-slice-b.md`, `goal-naming-ui-fixes.md`). Tham khảo `.scratch/goal-*.md` đã có để xem ví dụ thực tế trong dự án.

### Templates

Copy từ `docs/playbooks/templates/`:

- [`goal-inline.md`](templates/goal-inline.md) — template ngắn.
- [`goal-file.md`](templates/goal-file.md) — template đầy đủ.

### Anti-patterns

- Việc lớn (cross-layer, nhiều bước) gửi trần kiểu "Tom, implement skill provider tabs" — thiếu CONTEXT/SUCCESS/RULES, agent tự brainstorm + spec + implement gộp → mất kiểm soát phase. (Việc nhỏ scope rõ thì gửi trần là ổn — đây chỉ là vấn đề với plan dài.)
- Paste lại 50 dòng context vào tmux thay vì link file — dễ hỏng input area, khó re-run.
- Goal không có khối **KHÔNG** trong CONTEXT.constraints — agent dễ vượt scope sang refactor không liên quan.
- Bỏ rule **RESPECT PHASE GATE** vì copy nguyên template gốc — agent sẽ chạy autonomous hết spec → code → PR, bỏ qua Larry review và user approval.
- SUCCESS CRITERIA viết "implement feature X" thay vì measurable outcome — không có cách verify done.

## Phase Gates

1. Brainstorm & scope → Tom. Output kèm **Risk Classification** (xem bảng dưới).
2. Branch decision → Orchestrator áp rule, tạo branch nếu cần, **trước** Spec.
3. Spec → Tom.
4. Spec review → Larry.
5. User approval.
6. Implementation plan → Tom.
7. Implement → Tom; review → Larry; smoke test.
8. Tom `gh pr create` (nếu trên branch) → user duyệt → merge.
9. Docs / source-of-truth update — Tom update doc canonical **trong cùng slice** (không nợ sang sau); chi tiết doc nào xem `documentation.md`.

> Slice = thin cross-layer cut (UI → service → data). Compress phase cho slice nhỏ, nhưng **không skip user approval ở Spec**.

## Branch & PR Workflow

### Risk Classification (Tom đóng ở cuối brainstorm note)

| Field | Value |
|-------|-------|
| Layers | UI / contract / Go / SQL / docs |
| Breaking change | yes / no |
| Schema/migration | yes / no |
| Est. LOC | <50 / 50–300 / >300 |
| Workflow | direct-to-main / branch + PR |

### Decision Rule (Orchestrator áp trước Spec)

| Điều kiện | Workflow |
|---|---|
| Schema/migration: yes **HOẶC** Breaking change: yes | **MUST** branch + PR |
| Layers ≥ 3 **HOẶC** Est. LOC > 300 | **SHOULD** branch + PR |
| Multi-slice độc lập song song | worktree per slice |
| UI-only / docs-only, < 50 LOC | OK direct-to-main |

Branch naming: `<type>/<kebab-slug>` (vd `feat/dashboard-plugins-metric`). PR target luôn là `main`. Tom tạo PR ngay sau Larry approve commit cuối.

**PR gate — KHÔNG gộp create + merge (bài học PR #2–#8):**

- **Tạo PR và merge là HAI gate riêng, KHÔNG được làm liền trong một bước.** Cấm pattern "Tom `gh pr create` rồi `gh pr merge` ngay" — làm vậy PR thành thủ tục trống, không ai gate được.
- **One actor không tự create + tự merge.** Người mở PR ≠ người bấm merge.
- **Review phải nằm TRÊN PR**, không chỉ trong tmux. Larry post verdict lên PR (`gh pr review --approve/--request-changes` kèm evidence) để PR có dấu vết review thật (`reviews: []` = chưa được gate).
- **Merge là gate của user** (playbook §Phase Gates: "user duyệt → merge"). Flow đúng: Tom `gh pr create` → **DỪNG** → Larry `gh pr review` trên PR → Orchestrator đưa link PR + tóm tắt cho user → **user (hoặc Orchestrator khi user ủy quyền PR cụ thể) merge**. Orchestrator KHÔNG tự ý chỉ thị Tom merge thay user trừ khi user explicit ủy quyền merge cho PR đó.
- Nếu user đã ủy quyền autonomous: vẫn giữ tách 2 gate + review-on-PR; Orchestrator đóng vai merge-gate thay user, nhưng người merge vẫn ≠ người tạo PR (Orchestrator/`gh` chứ không phải để Tom tự merge commit mình vừa push).

## tmux Handoff Contract

### Before every handoff

```sh
tmux capture-pane -t <pane> -p | tail -80
git status --short
```

Verify: TUI đang chạy (không phải shell), input area trống, không có text cũ.

### Sending a prompt

Clear input cũ → send prompt → send Enter **riêng** để submit → capture lại để xác nhận đã vào transcript. Enter đầu chỉ confirm multi-line; phải 2 Enter.

### Selection prompt = nguy hiểm

Nếu capture cho thấy agent đang ở `Enter to select · ↑/↓ to navigate · Esc to cancel`:

- **KHÔNG** gửi text thẳng — nó sẽ register thành selection (thường là default/Recommended) và corrupt câu trả lời.
- Đúng: gửi `Escape` → verify input clean → gửi text.
- Relay user choice vào selection: gửi đúng `Down`/`Up` × N rồi `Enter` → capture ngay để verify ghi nhận đúng.

### Context & model switching

- `/clear` trước task không liên quan hoặc khi đổi phase. **Không clear giữa goal.**
- Map model theo task: opus/strong cho brainstorm/scope/plan, sonnet/fast cho implement và fix loop.

### Waiting for an agent

Auto-poll bằng background bash, không bắt user poll tay — **nhưng poll phải có checkpoint, không treo mù**.

**Nguyên tắc**: chia poll thành cửa sổ ngắn (~3 phút), hết mỗi cửa sổ thì quay lại Orchestrator để inspect, thay vì 1 lần block 20–30 phút. Loop thoát theo **2 điều kiện**: (a) agent idle, HOẶC (b) phát hiện **stuck-state** (cần can thiệp), HOẶC (c) hết cửa sổ (quay lại check rồi quyết có poll tiếp không).

```sh
# 1 cửa sổ poll = ~3 phút. exit_reason cho Orchestrator biết phải làm gì.
stable=0; iters=0; max=36   # 36*5s = 3 phút
exit_reason=window_elapsed
while [ $iters -lt $max ]; do
  out=$(tmux capture-pane -t <pane> -p 2>/dev/null)
  # stuck-state: selection/permission prompt, shell leak, lỗi rõ → dừng ngay
  if echo "$out" | grep -qE 'Enter to select|Do you want to proceed|❯ 1\.|Yes, |No, go back|\$ $|command not found|Error:'; then
    exit_reason=needs_attention; break
  fi
  # busy khi còn spinner; idle khi mất spinner ≥40s (8 vòng)
  # Pattern PHẢI bắt cả format elapsed "Verb… (3m 0s · ↓ 1.2k tokens)" — dùng "… *\(" + token counters.
  # KHÔNG dùng glyph ✻/✶ làm tín hiệu busy: chúng PERSIST trên dòng summary đã xong ("✻ Churned for 4m") → false-busy treo loop.
  if echo "$out" | grep -qE '… *\([0-9]+[smh]|esc to interrupt|◎ /goal active|↓ [0-9]|↑ [0-9]|· [0-9.]+k? tokens'; then stable=0; else stable=$((stable+1)); fi
  if [ $stable -ge 8 ]; then exit_reason=idle; break; fi
  iters=$((iters+1)); sleep 5
done
echo "EXIT=$exit_reason after ~$((iters*5))s"; tmux capture-pane -t <pane> -p | grep -n '[^[:space:]]' | tail -40
```

- `run_in_background: true`. Mỗi cửa sổ ~3 phút; khi notify thì đọc `EXIT`:
  - `idle` → agent xong, đọc report.
  - `needs_attention` → có selection/permission/shell-leak/lỗi → inspect + recover (xem mục Recovery), KHÔNG poll tiếp mù.
  - `window_elapsed` → còn busy → ghi nhận tiến độ rồi phóng cửa sổ mới (lặp lại tối đa N lần).
- **Wall-clock budget cứng**: đặt trần tổng (vd 6 cửa sổ ≈ 18 phút cho task lớn). Hết budget mà vẫn busy → capture pane, đánh giá có thật sự tiến triển không (token/spinner đổi), nếu nghi treo → `C-c` + inspect, không để chạy vô hạn.
- Pattern `… \([0-9]+[smh]` khớp spinner Claude Code; runtime khác cần adapt. Grep stuck-state cũng theo TUI cụ thể.

### Audit

```sh
tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} cmd=#{pane_current_command} cwd=#{pane_current_path}' | rg 'agent-tech|agent-lead'
```

## Review & Smoke

Larry chọn loại review theo target. Orchestrator chỉ nêu target + intent, không chỉ định tool.

| Loại | Target |
|---|---|
| Code review | diff/commit |
| PR review | full PR scope |
| Spec/design review | architecture, risk, missing case |
| Security review | auth, data exposure, injection |

**Smoke scenarios thiết kế trong phase Spec**, không phải lúc execute. Tom propose scenarios kèm spec → user/Larry duyệt → Larry execute và report pass/fail kèm evidence. Scenario gap phát hiện khi execute → log lại vào spec cho lần sau. Smoke verify end-to-end (UI, CLI, API, IPC, data flow — không chỉ UI), không phải unit.

**Driving the Electron app**: drive instance `pnpm dev` đang chạy qua CDP — đọc [`agent-browser-smoke.md`](agent-browser-smoke.md) trước (connect port 49222, **không** launch instance thứ 2).

**Nguyên tắc chung cho mọi review/smoke**:

- Scope cụ thể (commit, PR, file, spec).
- Larry verdict: approve / block / needs-discussion. **Không edit file.**
- Lead finding → Tom fix scoped commit → Larry chỉ re-review follow-up đó.
- "No verdict" (Larry không inspect) → rerun từ đầu.
- Drift docs phát hiện trong review → fix trước khi close.
- **Reviewer KHÔNG tự-poll chờ fix, KHÔNG tự drive vòng lặp.** Khi review ra finding → report verdict (BLOCK + mô tả file:line) rồi **DỪNG**. Orchestrator kiểm soát handoff: nhận finding → dispatch Tom fix → verify fix → re-engage reviewer để re-smoke. (Bài học G3c: Larry tự mở vòng `while` chờ Tom commit → context cạn + overstep sang **tự sửa file production** (revert lại sau, branch không hỏng nhưng vi phạm role). Reviewer self-drive = nguy cơ drift vai trò + đốt context. Một review = một verdict, một lần.)

## Ownership

- One actor per file at a time.
- Larry không edit file trừ khi user explicit yêu cầu.
- Orchestrator không implement product code. Agent fail → restore agent trước (clear, restart, split task, switch model) → nếu vẫn kẹt thì hỏi user.

## Recovery

**Stale/wrong prompt**: `C-c` → capture → nếu còn hỏng, `C-c C-c` thoát TUI → restart → verify input trống.

**Tom degraded** (stale behavior, wrong scope, context corrupt):

1. `C-c`, capture pane.
2. `/clear` hoặc restart TUI.
3. Re-send task nhỏ hơn kèm stop condition rõ.
4. Vẫn lặp → hỏi user. **Không tự implement.**

**Shell leak**: agent rớt về shell → restart TUI với flag uninterrupted để permission prompt không stall:

- Claude Code: `claude --dangerously-skip-permissions`
- Codex: `codex --yolo`
- OpenCode / agy: `agy --dangerously-skip-permissions`

Không trust process name — phải inspect input area thực.

## Maintenance

Mỗi orchestration failure → thêm 1 rule vào đây. Nguyên tắc playbook: **principles over recipes, references over duplication**. Nếu một rule chỉ áp dụng 1 lần, không thêm vào.
