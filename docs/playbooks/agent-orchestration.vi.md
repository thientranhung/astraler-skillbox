# Playbook Điều Phối Agent

## Vai Trò

| Tên | tmux pane | Vai trò |
|-----|-----------|---------|
| **Tom** | `agent-tech-skillbox` | Lập trình viên chính. Brainstorm, spec, plan, triển khai code. |
| **Larry** | `agent-lead-skillbox` | Reviewer & QA. Review code, review spec, smoke test. KHÔNG sửa file. |
| **Orchestrator** | (session hiện tại) | PM & điều phối. Chỉ quyết định **ai làm** và **khi nào**, không bao giờ quyết định **làm gì** hay **làm thế nào**. Mọi ý kiến kỹ thuật đều route cho Tom hoặc Larry. |

Orchestrator chỉ được phép trực tiếp sửa: playbook này, tài liệu quy trình, sửa doc nhỏ, hoặc ngoại lệ do user cho phép.

## Template `/goal`

Trước khi dùng `/goal`, đọc và tuân theo template này:

```
/Users/tranthien/Library/Mobile Documents/iCloud~md~obsidian/Documents/Obsidian/40-Collection/References/Slash goal prompt template.md
```

Goal lớn → viết file mô tả trong `.scratch/` rồi gửi `/goal` ngắn tham chiếu tới file đó.

## Quy Trình Giai Đoạn

Thứ tự khuyến nghị cho công việc lớn. Có thể nén lại cho slice nhỏ, nhưng không bao giờ được bỏ bước user phê duyệt spec.

> **Slice** là một lát công việc end-to-end mỏng (UI → service → data, hoặc bất kỳ cắt ngang layer nào) — lớn hơn 1 edit đơn lẻ, nhỏ hơn 1 feature.

1. Brainstorm & xác định phạm vi → Tom
2. Viết design spec → Tom
3. Review spec → Larry
4. User phê duyệt
5. Viết implementation plan → Tom
6. Triển khai → Tom, review → Larry, test & smoke-test
7. Cập nhật docs / source-of-truth (xem [Docs & Source of Truth](#docs--source-of-truth))

## Quy Tắc tmux

### Trước Mỗi Lần Giao Việc

```sh
tmux capture-pane -t <pane> -p | tail -80
git status --short
```

Xác nhận: TUI đang chạy (không phải shell), ô input trống, không có text cũ.

### Gửi Prompt

Xóa input cũ, gửi prompt, gửi Enter riêng để submit, rồi capture pane để xác nhận prompt đã vào transcript. Enter đầu tiên trong TUI chỉ confirm multi-line input — muốn submit phải gọi Enter lần thứ hai.

**Ngắn vs file:** Mặc định gửi trực tiếp inline. Chỉ viết ra `.scratch/` khi message quá dài (~500+ ký tự). Đặt tên file theo nội dung, ví dụ `.scratch/fix-useeffect-regression.md`, `.scratch/slice-3k-impl-plan.md`.

### Chuyển Context & Model

- `/clear` trước task không liên quan hoặc khi chuyển giai đoạn. Không clear giữa goal đang chạy.
- Khớp sức mạnh model với loại task: model mạnh / deep-thinking cho brainstorm/scope/plan, model nhanh / rẻ hơn cho triển khai, fix, vòng lặp test. Tom (Claude Code): **opus** ↔ **sonnet**. Runtime khác thì map tương đương.

### Lệnh Kiểm Tra

```sh
tmux list-panes -a -F '#{session_name}:#{window_index}.#{pane_index} cmd=#{pane_current_command} cwd=#{pane_current_path}' | rg 'agent-tech|agent-lead'
```

## Review

Larry phụ trách review. Có nhiều loại review — chọn đúng loại và tin tưởng skill/tool của Larry để thực hiện:

- **Code review** — kiểm tra diff/commit về tính đúng, style, regression.
- **PR review** — toàn bộ PR, tính nhất quán giữa các commit, sẵn sàng merge.
- **Spec/design review** — kiến trúc, rủi ro, case thiếu trước khi triển khai.
- **Security review** — auth, lộ dữ liệu, lỗ hổng injection.

Tận dụng mọi công cụ review mà runtime của Larry cung cấp — slash command có sẵn, skill, MCP review server, hay flow review riêng của provider. Larry tự chọn công cụ phù hợp với loại review và codebase; orchestrator chỉ nêu target và mục đích.

Nguyên tắc (áp dụng cho mọi loại review):
- Scope vào target cụ thể (commit, PR, file, spec).
- Findings trước; Larry quyết approve / block / cần thảo luận.
- Larry KHÔNG sửa file trừ khi được yêu cầu rõ ràng.
- Larry phát hiện lỗi → Tom sửa trong commit có scope → Larry review lại đúng commit sửa.
- Larry báo "No verdict" (chưa kiểm tra) → chạy lại review từ đầu.
- Docs/source-of-truth lệch → sửa trước khi đóng task.

## Smoke Test

Smoke test kiểm chứng hành vi end-to-end — UI, CLI, API, data flow, IPC, bất cứ thứ gì slice chạm tới. Không chỉ UI.

Kịch bản test được thiết kế **trước khi triển khai**, trong giai đoạn spec/plan của Tom:

- Tom brainstorm kịch bản smoke ngay khi viết spec.
- Kịch bản lưu cùng spec (để user review và phê duyệt từ đầu).
- Đến giai đoạn triển khai, Larry (hoặc user) chỉ chạy lại kịch bản đã phê duyệt.

Nguyên tắc:
- Kịch bản phủ surface bên ngoài của slice, không phủ unit nội bộ (đó là việc của unit test).
- Larry chạy và báo pass/fail kèm bằng chứng; Larry không tự nghĩ kịch bản mới.
- Fail → Tom sửa. Larry không bao giờ edit file.
- Phát hiện thiếu kịch bản khi chạy → ghi lại vào spec cho iteration kế tiếp.

## Quyền Sở Hữu File

- Mỗi file chỉ 1 người sửa tại 1 thời điểm.
- Larry KHÔNG sửa file trừ khi được yêu cầu rõ ràng.
- Orchestrator KHÔNG triển khai code sản phẩm. Agent lỗi → khôi phục agent trước (clear, restart, chia nhỏ task, đổi model), rồi hỏi user nếu vẫn kẹt.

## Khắc Phục Sự Cố

**Prompt cũ/sai:** `C-c` → capture → nếu vẫn lỗi, `C-c C-c` thoát TUI → khởi động lại TUI → xác nhận input trống.

**Tom bị lỗi (hành vi cũ, sai scope, context nhiễm):**
1. `C-c`, capture pane
2. `/clear` hoặc restart TUI
3. Gửi lại task nhỏ hơn với stop condition rõ ràng
4. Nếu lặp lại → hỏi user, không tự triển khai

**Rơi ra shell:** Nếu agent thoát về shell, restart TUI bằng launch flag "uninterrupted" chuẩn của runtime để permission prompt không chặn công việc. Xác nhận ô input sạch trước khi gửi việc. Lệnh thường dùng:

- Claude Code: `claude --dangerously-skip-permissions`
- Codex: `codex --yolo`
- OpenCode / agy: `agy --dangerously-skip-permissions`

Không tin process name — phải nhìn ô input thực tế.

## Docs & Source of Truth

Mỗi slice kết thúc phải có docs và source-of-truth khớp với implementation. Coi đây là một phần của "done", không phải việc làm sau.

- Tom cập nhật docs liên quan như một phần của commit triển khai (hoặc commit kèm trong cùng slice): docs kiến trúc, `CLAUDE.md`/`AGENTS.md`, schema dictionary, contracts/types, README, changelog.
- Larry kiểm tra docs lệch khi review và block nếu implementation lệch khỏi spec hoặc spec/source-of-truth chưa cập nhật.
- Source-of-truth là chính: nếu code và docs mâu thuẫn, sửa bên sai — không để drift tích lũy âm thầm.
- Khi slice thay đổi public contract, schema, hay convention, cập nhật cả file gốc lẫn ví dụ/quickstart tham chiếu nó.
- Sau khi update docs, chạy search có target để bắt label/path cũ, ví dụ `rg "term cũ|path cũ" docs apps core-go`.

## Maintenance

Mỗi lỗi điều phối → thêm 1 rule vào đây. Giữ playbook gọn: nguyên tắc thay vì công thức, tham chiếu thay vì trùng lặp.
