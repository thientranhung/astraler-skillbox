# Slice B — Dashboard UI polish (hyperlink color + cursor audit) — Design Spec

**Author:** Tom · **Date:** 2026-05-28 · **Status:** awaiting Larry review

**Source brainstorm:** [`.scratch/brainstorm-naming-ui-fixes.md`](../../../.scratch/brainstorm-naming-ui-fixes.md) (§"Slice B", Q3 chốt blue tone)

**Slice A status:** đã merge (commits `b71b9c3`..`ae7eccc`).

## Mục tiêu

1. Dashboard: số liệu clickable hiển thị affordance màu link (`text-blue-600 hover:text-blue-700 hover:underline`) thay cho `text-zinc-500` hiện tại.
2. Audit các text/icon có function ngoài Dashboard (skills-library, global-skills, plugins, project-detail) — bổ sung `cursor-pointer` cho phần tử clickable đang thiếu.
3. Quy ước: clickable value text dùng cùng class set; ghi 1 dòng trong implementation notes của plan (không cần ADR).

---

## §1 Mapping cụ thể (before → after)

### 1.1 `apps/desktop/renderer/src/screens/dashboard-screen.tsx` — Dashboard hyperlink color

Đổi class của **value span bên phải** trong các row Summary clickable. Container `<button>` đã có `cursor-pointer` + `hover:bg-zinc-50` — giữ nguyên.

| Line | Element | Before | After |
|------|---------|--------|-------|
| 100  | `Skills` value span — `{data.summary.skills}` | `text-sm text-zinc-500` | `text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline` |
| 108  | `Projects` value span — `{data.summary.projects}` | `text-sm text-zinc-500` | `text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline` |
| 117  | `Attention needed` value span — `{data.summary.warnings}` | `text-sm text-zinc-500` | `text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline` |
| 126  | `Global Skills` value span — `"Open global view"` | `text-xs text-zinc-500` | `text-xs text-blue-600 group-hover:text-blue-700 group-hover:underline` |

**Lý do dùng `group-hover:`**: hover của user là trên cả button (parent), không chỉ span. Phải gắn `group` lên `<button>` parent để `group-hover:` trên span kích hoạt. Implementation chỉ thêm `group` vào 4 button class strings hiện có (lines 97, 105, 114, 123).

### 1.2 `apps/desktop/renderer/src/screens/global-skills-screen.tsx` — cursor-pointer audit

| Line | Element | Before (subset) | After (subset) |
|------|---------|-----------------|----------------|
| 51–58 | "Scan Global" button | `flex items-center gap-1.5 rounded border ... hover:bg-zinc-50 disabled:opacity-50` | thêm `cursor-pointer` |
| 104–111 | "Open Folder" (per-location) button | `flex items-center gap-1 rounded border ... hover:bg-zinc-50` | thêm `cursor-pointer` |
| 143–150 | "Open" (per-entry) button | `flex items-center gap-1 rounded border ... hover:bg-zinc-50` | thêm `cursor-pointer` |

> Note: line numbers ở các mapping là chỉ dẫn; implementer verify lại bằng grep trước khi edit.

### 1.3 `apps/desktop/renderer/src/screens/plugins-screen.tsx` — cursor-pointer audit

| Line | Element | Action |
|------|---------|--------|
| 72–79 | `PluginToggleButton` (`Disable` / `Enable`) | thêm `cursor-pointer` (giữ `disabled:cursor-not-allowed`) |
| 237–244 | Header "Scan Global" button | thêm `cursor-pointer` |

### 1.4 `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — cursor-pointer audit

| Line | Element | Action |
|------|---------|--------|
| 121–129 | `PathCell` Copy button | thêm `cursor-pointer` |
| 188–195 | `EntryRow` Remove button | thêm `cursor-pointer` (giữ `disabled:cursor-not-allowed`) |
| 389–414 | Project layer cycle button | thêm `cursor-pointer` (giữ `disabled:cursor-not-allowed`) |
| 426–443 | User layer toggle button | thêm `cursor-pointer` (giữ `disabled:cursor-not-allowed`) |
| 532–538 | "← Projects" back button | thêm `cursor-pointer` |
| 559–567 | Header "Scan" button | thêm `cursor-pointer` |
| 568–576 | Header "Open Folder" button | thêm `cursor-pointer` |
| 577–585 | Header "Terminal" button | thêm `cursor-pointer` |
| 586–593 | Header "Add Skill" button | thêm `cursor-pointer` |
| 594–602 | Header "Remove" project button | thêm `cursor-pointer` |
| 676–687 | "All providers" filter pill | thêm `cursor-pointer` |
| 689–703 | Per-provider filter pill | thêm `cursor-pointer` |

### 1.4b `apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx` — cursor-pointer audit

Modal dialog mở khi click "Add Skill" trên project detail. Goal user nêu là "tất cả text/function clickable cần cursor pointer" → in scope.

| Line | Element | Action |
|------|---------|--------|
| 78–84 | Close (X) icon button | thêm `cursor-pointer` |
| 139–143 | "Cancel" button | thêm `cursor-pointer` |
| 145–151 | "Install" button | thêm `cursor-pointer` **và** `disabled:cursor-not-allowed` (button có `disabled` prop nhưng class hiện chỉ có `disabled:opacity-50`) |

### 1.5 `apps/desktop/renderer/src/features/skills-library/skill-row.tsx` — không đụng

Row đã có `cursor-pointer` (line 19). Skip.

### 1.6 `apps/desktop/renderer/src/screens/skills-library-screen.tsx` — không đụng

Các button (Open Folder, Terminal, Scan, filter pills "All skills" / "Shared Agent Skills") đã có `cursor-pointer`. Skip.

---

## §2 Edge cases

1. **KHÔNG đụng các button đã có `cursor-pointer`** — chỉ bổ sung khi thiếu. Implementer phải verify class string hiện tại trước khi thêm.
2. **KHÔNG đè màu blue lên text non-clickable**. Cụ thể trong `dashboard-screen.tsx`:
   - Host block rows (lines 73, 79, 84): static, **không** đổi màu.
   - Installs by Mode rows (lines 141, 145, 149): static, **không** đổi màu.
   - "Updates / Not in this slice" row (line 130): static, **không** đổi màu.
3. **Disabled state**: nút có `disabled:cursor-not-allowed` vẫn giữ — `cursor-pointer` thêm vào không lấn vì Tailwind generate `disabled:` variant với specificity cao hơn class thường.
4. **Group hover trên dashboard**: thêm `group` vào `<button>` parent, KHÔNG đụng `hover:bg-zinc-50` đã có. Nếu lỡ quên `group`, `group-hover:` trên span sẽ không kích hoạt → underline + blue-700 mất → smoke scenario S1 phát hiện.
5. **Conditional render "Attention needed" row** (line 110–119): chỉ render khi `warnings > 0`. Class change áp dụng vào span con bất kể số. Test hiện đã set up data có warnings, sẽ render đúng.
6. **`PathCell` Copy icon button** (`project-detail-screen.tsx` line 121): icon-only, không có border / bg. Vẫn cần `cursor-pointer` để consistency. KHÔNG đổi màu icon.

---

## §3 Test impact

### 3.1 Test query class hiện có

- `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx:161–163`: query `getByRole("button", ...).className).toContain("cursor-pointer")` — verify trên **button parent**, không phải span. Việc thêm `group` vào button class **không** break test (className vẫn chứa `cursor-pointer`).
- Các test khác (global-skills, plugins, project-detail) **không** query `cursor-pointer` theo class → không break.

### 3.2 Test mới cần thêm

**Recommended:** thêm assertion trong `dashboard-screen.test.tsx` verify value span có `text-blue-600` cho 3 row Skills / Projects / Attention needed. Lý do: nếu implementer quên `group` trên button parent, snapshot/visual check không catch nhưng group-hover effect mất → user thấy underline không hiện khi hover. Test class trên span là cheap guard.

**Lưu ý S1 smoke (§4):** PHẢI hover thật bằng chuột (không chỉ chụp screenshot tĩnh) để verify `group-hover:text-blue-700` và `group-hover:underline` activate. Static visual chỉ catch được màu base `text-blue-600`.

### 3.3 Snapshot tests

Không có snapshot test trong repo (đã grep `.snap` → 0 match dự kiến). Không có rủi ro snapshot churn.

### 3.4 Smoke check toàn bộ

```bash
pnpm --filter desktop test
```

Kỳ vọng: pass full suite. Nếu fail, root cause phải là test mới-thêm hoặc miss `group` parent — KHÔNG được sửa test cũ để pass.

---

## §4 Smoke scenarios (Larry execute)

**S1 — Dashboard hyperlink affordance**
1. Launch desktop app, mở `/` (DashboardScreen).
2. Quan sát các value `Skills`, `Projects`, `Global Skills` ở Summary section.
3. **Expected:** value text màu blue (`text-blue-600`).
4. Hover qua row → text chuyển blue đậm hơn + underline.
5. **Expected:** background row vẫn highlight `bg-zinc-50` (giữ behavior cũ).

**S2 — "Attention needed" row (cần data có warnings)**
1. Trong host folder, rename `documentation-writer/` → `documentation-writer_bak/` (giả lập missing skill).
2. App → `/skills` → click "Scan" để re-scan host.
3. Mở `/` (Dashboard) → verify row "Attention needed" xuất hiện với count ≥ 1.
4. Value số warnings hiển thị màu blue; hover row → text chuyển blue-700 + underline.
5. **Restore:** rename `documentation-writer_bak/` → `documentation-writer/`, re-scan để clear warning.

**S3 — Installs by Mode KHÔNG đổi màu**
1. Trên dashboard, quan sát section "Installs by Mode" (Symlink / Rsync-copy / Direct).
2. **Expected:** value vẫn `text-zinc-500`, KHÔNG blue, KHÔNG hover affordance.

**S4 — Cursor-pointer audit (off-Dashboard)**
1. Mở `/global` → hover lên buttons "Scan Global", per-location "Open Folder", per-entry "Open".
2. **Expected:** cursor = pointer trên cả ba.
3. Mở `/plugins` → hover lên "Scan Global" header và per-plugin "Enable"/"Disable".
4. **Expected:** cursor = pointer.
5. Mở project detail (vd. `/projects/1`) → hover lên: back button, "Scan", "Open Folder", "Terminal", "Add Skill", "Remove", các provider filter pills, Copy path icon, Remove skill button, project/user toggle buttons trong Provider Plugins table.
6. **Expected:** tất cả cursor = pointer.

**S5 — Disabled state**
1. Trên dashboard error retry hoặc bất kỳ button nào có disabled state — click khi đang `Scanning…`.
2. **Expected:** cursor = `not-allowed` (giữ behavior cũ qua `disabled:cursor-not-allowed`).

---

## §5 Files touched / KHÔNG đụng

### Touched
- `apps/desktop/renderer/src/screens/dashboard-screen.tsx`
- `apps/desktop/renderer/src/screens/global-skills-screen.tsx`
- `apps/desktop/renderer/src/screens/plugins-screen.tsx`
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx`
- `apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx`
- `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx` (assertion class span — §3.2 Recommended)

### KHÔNG đụng
- `apps/desktop/renderer/src/screens/skills-library-screen.tsx` (đã có cursor-pointer khắp nơi)
- `apps/desktop/renderer/src/features/skills-library/skill-row.tsx` (đã có cursor-pointer)
- `apps/desktop/renderer/src/features/skills-library/skill-status-badge.tsx` (badge static)
- Backend / contract / Go / SQL: không liên quan slice này
- Tất cả test files (không sửa test cũ; optional add mới theo §3.2)
- `apps/desktop/renderer/src/screens/settings-screen.tsx`, `setup-screen.tsx`, `projects-screen.tsx`, `skill-detail-screen.tsx`: ngoài scope brief

---

## §6 Non-goals

- **Slice C — Per-provider plugins metric trên Dashboard**: thêm field `pluginsByProvider` vào contract, render group theo provider. ĐỂ LẠI cho slice sau.
- Đổi typography / spacing toàn cục — chỉ class color/cursor.
- Refactor `<button>` thành `<a>` (semantic link) — quyết định để slice riêng nếu cần.
- Thêm focus-visible ring chuẩn hóa — out of scope.
- Audit `settings-screen.tsx` / `setup-screen.tsx` / `projects-screen.tsx` / `skill-detail-screen.tsx` — brief giới hạn 4 screens.
- ADR riêng cho convention — quy ước ghi inline trong plan implementation notes.

---

## Convention note (dành cho plan)

> Clickable value text (số liệu link-style): dùng `text-blue-600 hover:text-blue-700 hover:underline`. Khi value nằm trong `<button>` parent, dùng `group` trên parent + `group-hover:text-blue-700 group-hover:underline` trên span.

---

## Next step

Đợi Larry review spec. Sau khi pass → viết plan tại `docs/superpowers/plans/2026-05-28-dashboard-ui-polish-plan.md`.
