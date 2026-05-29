# CLAUDE.md

Claude-Code-specific delta. All shared project knowledge lives in `AGENTS.md`.

> **MUST READ first:** [`AGENTS.md`](./AGENTS.md)

## Claude Code Notes

- **No project-level hooks or slash commands defined** beyond defaults. If you add any under `.claude/`, document them here.
- **Scratchpad**: `.scratch/` (gitignored) for long task briefs that don't fit in tmux input.

## IPC Allowlist

- Mỗi RPC method mới PHẢI thêm vào `apps/desktop/electron/main/core-process/method-allowlist.ts` — bỏ sót → `method_not_allowed` lúc runtime (không bắt được qua unit test).

## PR Flow (Tom + Larry)

- Tom: `gh pr create --base main` → DỪNG (không merge).
- Larry: `gh pr review <n> --approve/--request-changes --body "..."` trực tiếp trên PR.
- Tom merge sau khi Larry approve. GitHub chặn self-approve (cùng account) → Larry post comment verdict thay.
- Chi tiết: `docs/playbooks/agent-orchestration.md`.

## DOC-VERIFIED pre-push gate

- Hook chỉ check nhãn, không tự verify drift. Phải chạy Gap-Find thủ công trước push lên main.
- Playbook: `docs/playbooks/documentation.md` (Gap-Find Procedure).
- Trigger: thêm migration / RPC method / screen / provider adapter mới.

## Khi xóa/đổi feature: trace toàn bộ UI

- Xóa/đổi một feature → grep toàn bộ renderer tìm text/component liên quan, không chỉ sửa file đang đọc. Vd: bỏ network toggle → phải grep "network\|toggle\|disabled by default" trong tất cả screens.

## EPERM trong Claude Code subprocess

- Không cần Full Disk Access. Fix: exit TUI → `q` → "Exit anyway" → restart `claude --dangerously-skip-permissions`.

## Electron dev + Go binary

- `pnpm dev` tự recompile Go qua `go run`. Nếu smoke dùng binary sẵn (`resources/core/`), phải rebuild: `cd core-go && go build -o apps/desktop/resources/core/skillbox-core ./cmd/skillbox-core/`.

## Poll agents (tmux)

- Bounded windows ~4-5 phút. Busy pattern: `… *\([0-9]+[smh]|esc to interrupt|◎ /goal active|↓ [0-9]|↑ [0-9]|· [0-9.]+k? tokens`.
- Glyph `✻` persist trên dòng summary đã xong → KHÔNG dùng làm tín hiệu busy.
- Feedback prompt "How is Claude doing?" → gửi `0` (Escape) để dismiss trước khi poll tiếp.
- Chi tiết: `docs/playbooks/agent-orchestration.md` §Waiting for an agent.

## Release / CI

- CI: `.github/workflows/ci.yml` (push/PR vào main → go test + typecheck + renderer test).
- Release: `.github/workflows/release.yml` (push tag `v*.*.*` → build 4 platform + GitHub Release).
- Chưa release production. Khi sẵn sàng: bump `apps/desktop/package.json` version → `git tag vX.Y.Z && git push origin vX.Y.Z`.
- Cần setup secrets trước: `APPLE_ID`, `APPLE_APP_SPECIFIC_PASSWORD`, `APPLE_TEAM_ID` (cho Mac notarize).
