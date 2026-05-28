# Naming Alignment (Slice A)

## Problem

UI labels và docs hiện dùng nhiều thuật ngữ không nhất quán với mental model "global vs project":

- Sidebar dùng `Plugins` (ý là plugin ở user/global layer) — dễ hiểu nhầm là quản lý plugin cấp app.
- Sidebar dùng `Skills` cho Skill Host Folder library — trùng tên với khái niệm "skill" nói chung, không phân biệt được với "Global Skills".
- Button trong Plugins screen ghi `Disable globally` / `Enable globally` — sau khi đã ở trong context "Global Plugins", chữ "globally" thừa.
- `project-detail-screen` đặt cột plugin tên `User` — trùng với label code (`layer: 'user'`) nhưng confusing với end-user vì layer này áp dụng cho toàn máy (global).
- Docs (`03-information-architecture`, `02-product-notes`) trộn `Skills Library` / `Skills` / `Plugins` không đồng bộ với UI.

User feedback yêu cầu chuẩn hóa **chỉ ở tầng UI labels + docs văn nói**, KHÔNG đụng code identifier, JSON-RPC contract, Go const, hay SQL data.

## Scope

- **In scope:** Rename hiển thị (string literals trong renderer) và mô tả trong các doc văn nói.
- **In scope:** Cập nhật test strings để khớp label mới.
- **Out of scope:** JSON-RPC contract field `layer: 'user' | 'project' | 'local'` — giữ nguyên.
- **Out of scope:** Go domain const `PluginLayerUser = "user"` — giữ nguyên.
- **Out of scope:** SQL column data `settings_layer = 'user'` — giữ nguyên.
- **Out of scope:** Generated TS types trong `shared/generated/` — giữ nguyên (auto-gen từ JSON Schema).
- **Out of scope:** Dashboard plugin metric, hyperlink color, cursor pointer audit — thuộc Slice B/C.

## Quyết định đầu vào (từ brainstorm 2026-05-28)

| # | Quyết định |
|---|------------|
| Q1 | Rename UI + docs only, không chạm code/contract/DB. |
| Q4 | Screen title "Skills Library" giữ nguyên. Chỉ sidebar đổi `Skills` → `Host Skills`. |
| Q5 | Chữ "Global" trong các nút phụ (vd. `Scan Global`) giữ nguyên. |
| Q6 | Cột `User` trong project-detail-screen đổi label thành `Global` (UI only). |

## §1 — Rename Mapping (Before → After)

### Sidebar

| File | Line | Before | After |
|------|------|--------|-------|
| `apps/desktop/renderer/src/components/sidebar.tsx` | 7 | `label: "Skills"` | `label: "Host Skills"` |
| `apps/desktop/renderer/src/components/sidebar.tsx` | 10 | `label: "Plugins"` | `label: "Global Plugins"` |

`label: "Global Skills"` (line 8) giữ nguyên — đã đúng.

### Plugins screen

| File | Line | Before | After |
|------|------|--------|-------|
| `apps/desktop/renderer/src/screens/plugins-screen.tsx` | 77 | `"Disable globally" : "Enable globally"` | `"Disable" : "Enable"` |

Tiêu đề `Provider Plugins` (line 232) và nút `Scan Global` (line 243) **giữ nguyên** (Q5).

### Project Detail screen

| File | Line | Before | After |
|------|------|--------|-------|
| `apps/desktop/renderer/src/screens/project-detail-screen.tsx` | 366 | `<th>User</th>` | `<th>Global</th>` |
| `apps/desktop/renderer/src/screens/project-detail-screen.tsx` | 433-434 | tooltip `"Disable globally"` / `"Enable globally"` | tooltip `"Disable"` / `"Enable"` |

Tooltip line 431 (`"Project layer overrides this setting"`) giữ nguyên — nội dung này nói về precedence, không thuộc nhóm "globally".

### Dashboard screen

Không có rename ở đây trong Slice A. Lý do:

- Row `Skills` (line 99) trỏ tới `/skills` — nội dung là Skills Library; label "Skills" trên dashboard hợp với screen title "Skills Library", không cần đổi thành "Host Skills" vì dashboard là tóm tắt và row đã có context.
- Row `Global Skills` (line 125) đã đúng.

(Slice B sẽ thêm plugin metric vào dashboard.)

### Docs

| File | Phạm vi cần update |
|------|--------------------|
| `docs/03-information-architecture.md` | Giữ "Skills Library" trong list `Main App Areas` (line 95 — đó là tên screen, Q4). Trong mục `Global Plugins` (line 165-191): (a) line 167 đổi `"plugin ở user layer"` → `"plugin ở global (user) layer"`; (b) line 188 đổi `"Chỉ user layer được hiển thị ở Global Plugins"` → `"Chỉ global (user) layer được hiển thị ở Global Plugins"`; (c) thêm 1 note ngắn ở cuối section giải thích "UI hiển thị label `Global` cho layer mà code/contract dùng identifier `user`". |
| `docs/02-product-notes.md` | Section `Plugins và Marketplaces`: line 91 `"Toggle enable/disable globally (user layer)..."` → `"Toggle enable/disable ở global (user) layer..."` cho khớp với button đã rename `Disable globally` → `Disable`. Rà các đề cập "user layer" khác trong section này, thay bằng "global (user) layer". Giữ code reference `layer: "user"` nếu có. |
| `docs/superpowers/specs/2026-05-28-plugin-layer-toggle-clarity-design.md` | Spec landed gần nhất đề cập "Disable globally"/"Enable globally" ở §3. Thêm 1 dòng cập nhật: "Slice A (naming-alignment) đã đổi label `Disable globally` → `Disable` sau khi sidebar đã ghi rõ `Global Plugins`." Không rewrite lịch sử. |

### Tests (cập nhật strings)

| File | Update |
|------|--------|
| `apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx` | **Required.** Line 13: `labels.indexOf("Skills")` → `labels.indexOf("Host Skills")`. Line 24: `labels.indexOf("Plugins")` → `labels.indexOf("Global Plugins")`. |
| `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx` | **Required.** Đổi mọi `getByRole("button", { name: "Disable globally" })` / `"Enable globally"` → `"Disable"` / `"Enable"`. **6 chỗ**: line 177, 178, 189, 200, 213, 232. |
| `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx` | **Required.** Assertions FAIL nếu không đổi: line 370, 382 (`getByRole("columnheader", { name: "User" })` → `{ name: "Global" }`). Description (test names — không fail nhưng phải đồng bộ): line 360, 373 (`"shows Project and User columns..."` → `"shows Project and Global columns..."`); line 461 (`"local override shows 'overridden' text in both Project and User columns"` → `"...Project and Global columns"`). |
| `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx` | Không đổi (dashboard không rename trong slice này). |

## §2 — Edge Cases

1. **`projects-screen.tsx:70`** có column header `Plugins` (đếm plugin per-project trong projects table). **KHÔNG rename** — đây là per-project count, không phải global plugins. Spec ghi rõ để tránh "find/replace toàn bộ Plugins".
2. **Generated TS types** chứa string `'user'` trong union types. **KHÔNG đụng** — auto-gen từ JSON Schema, sẽ revert khi regenerate.
3. **`contract_test.go` và Go test** dùng literal `"user"` cho `settings_layer`. **KHÔNG đụng** — data layer.
4. **Search-and-replace mù** sẽ phá test snapshots và contract assertions. Tom phải đổi từng chỗ theo mapping bảng §1, không dùng `sed -i` toàn cây.
5. **Old plan/spec docs trong `docs/superpowers/plans/` và `docs/superpowers/specs/`** (history) — không rewrite. Chỉ note ở spec mới nhất (`2026-05-28-plugin-layer-toggle-clarity-design.md`) như §1 đã nêu.
6. **Translation hai chiều**: sau Slice A, người đọc UI thấy `Global`, người đọc code thấy `user`. Cần 1 dòng note trong `docs/03-information-architecture.md` giải thích mapping này.

## §3 — Test Impact

### Unit / component tests (Vitest)
- 3 file `__tests__` cần cập nhật string (xem bảng §1).
- Không thêm test mới (slice là cosmetic rename).

### Contract drift
- Không chạy `pnpm check:contracts-drift` vì không đụng contracts. Nhưng vẫn nên chạy 1 lần xác nhận pass (sanity).

### Type check
- `pnpm typecheck` phải pass — không có type change.

### Go tests
- Không chạy bắt buộc; Slice A không đụng `core-go/`. Nếu CI mặc định chạy, vẫn pass.

## §4 — Smoke Test Scenarios

Larry sẽ execute các kịch bản sau **sau implementation**, dựa trên packaged dev build (`pnpm dev`). Báo cáo pass/fail kèm screenshot mỗi scenario.

### S1. Sidebar labels
**Steps:**
1. `cd apps/desktop && pnpm dev` → app mở dashboard.
2. Quan sát sidebar.

**Expected:**
- Thấy 6 mục theo thứ tự: `Dashboard`, `Host Skills`, `Global Skills`, `Projects`, `Global Plugins`, `Settings`.
- Không còn label `Skills` (đứng một mình) hay `Plugins` (đứng một mình).

### S2. Global Plugins screen — button labels
**Steps:**
1. Trong app, click sidebar `Global Plugins`.
2. Nếu chưa có data plugin: click `Scan Global` (nút giữ nguyên tên).
3. Khi có ít nhất 1 plugin row có Action column hiển thị, quan sát button.

**Expected:**
- Header vẫn ghi `Provider Plugins` và `Scan Global`.
- Action button hiển thị `Disable` (nếu plugin enabled) hoặc `Enable` (nếu disabled).
- Không còn chữ "globally" trên button.

### S3. Project Detail — Plugin table column header
**Steps:**
1. Vào `Projects` → mở 1 project có provider `claude`/`codex`/`antigravity_cli`.
2. Cuộn xuống Plugins section.

**Expected:**
- Header table thấy: `Plugin | Marketplace | Project | Global | Effective` (Project và Global chỉ hiện khi `canToggle`).
- Cột `User` cũ không còn xuất hiện.

### S4. Project Detail — Plugin tooltip
**Steps:**
1. Trong Plugins section của 1 project, hover button trạng thái ở cột `Global` của 1 plugin.

**Expected:**
- Tooltip ghi `Disable` (khi enabled) hoặc `Enable` (khi disabled).
- Khi cột Project có giá trị (enabled/disabled), tooltip vẫn là `Project layer overrides this setting` (không đổi).

### S5. Toggle vẫn hoạt động sau rename
**Steps:**
1. Trong Plugins screen (Global Plugins), click `Disable` trên 1 plugin enabled.
2. Chờ operation done; refresh hoặc đợi UI cập nhật.
3. Mở settings file user-layer (vd. `~/.claude/settings.json`) bằng terminal hoặc Open Folder.

**Expected:**
- Button đổi từ `Disable` sang `Enable`.
- Settings JSON cho thấy plugin chuyển sang disabled (chứng minh rename không vô tình đụng handler logic).

### S6. Docs alignment (manual)
**Steps:**
1. `grep -n "Plugins\b\|Skills\b" docs/03-information-architecture.md`.
2. Đọc các section đã đụng.

**Expected:**
- `Main App Areas` list các tên screen khớp với sidebar (`Host Skills`, `Global Skills`, `Global Plugins`).
- Có 1 note ngắn giải thích "UI hiển thị `Global` cho layer mà code/contract dùng `user`".
- Không còn câu mô tả layer là "user layer" mà thiếu chú thích "global" trong văn mô tả.

### S7. Tests pass
**Steps:**
1. `(cd apps/desktop && pnpm typecheck)`.
2. `(cd apps/desktop && pnpm test)`.

**Expected:**
- Cả 2 lệnh pass. Test strings đã cập nhật khớp label mới.

## §5 — Files Touched

### Renderer (UI rename)
- `apps/desktop/renderer/src/components/sidebar.tsx`
- `apps/desktop/renderer/src/screens/plugins-screen.tsx`
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx`

### Tests
- `apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx`
- `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx`
- `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx`

### Docs
- `docs/03-information-architecture.md`
- `docs/02-product-notes.md`
- `docs/superpowers/specs/2026-05-28-plugin-layer-toggle-clarity-design.md` *(1 dòng note cuối §3, không rewrite)*

### KHÔNG đụng
- `shared/api-contracts/**`, `shared/generated/**`
- `core-go/**`
- `apps/desktop/electron/**`
- `apps/desktop/renderer/src/screens/projects-screen.tsx` (column header `Plugins` per-project — giữ)
- `apps/desktop/renderer/src/screens/dashboard-screen.tsx` (slice B)

## §6 — Non-Goals

- Không đổi JSON-RPC contract, Go const, SQL data — Q1 đã chốt.
- Không thêm/đổi behavior, không thêm tooltip mới ngoài rename tooltip cũ.
- Không refactor cấu trúc component.
- Không động chạm Dashboard (slice B), không thêm plugin metric (slice C).
- Không sửa lịch sử docs/specs/plans đã landed (chỉ thêm note ở spec gần nhất nếu cần).

## §7 — Build Order & Hand-off

1. Tom viết spec này → Larry review spec.
2. User duyệt spec.
3. Tom invoke `superpowers:writing-plans` → tạo plan implementation.
4. Tom implement theo plan.
5. Larry execute smoke scenarios §4, report pass/fail.
6. Tom fix nếu fail; Larry re-run scenarios fail.
7. Merge.
