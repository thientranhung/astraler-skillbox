# Slice C (light) — "Global Plugins" row on Dashboard — Design Spec

**Author:** Tom · **Date:** 2026-05-28 · **Status:** awaiting Larry review

**Source brainstorm:** [`.scratch/brainstorm-naming-ui-fixes.md`](../../../.scratch/brainstorm-naming-ui-fixes.md) (Slice C, rescoped)

**Scope rescope (user, 2026-05-28):** mirror y chang pattern "Global Skills" hiện có trên Dashboard, click → `/plugins`. KHÔNG per-provider breakdown, KHÔNG count, KHÔNG invent thêm.

---

## §0 Pattern reference — "Global Skills" current row

Đọc `apps/desktop/renderer/src/screens/dashboard-screen.tsx:120–127`:

```tsx
<button
  type="button"
  onClick={() => navigate({ to: "/global" })}
  className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
>
  <span className="text-sm font-medium text-zinc-700">Global Skills</span>
  <span className="text-xs text-blue-600 group-hover:text-blue-700 group-hover:underline">Open global view</span>
</button>
```

**Observations:**
- KHÔNG có backend field tương ứng trong `dashboard.get` summary (verified `shared/api-contracts/methods/dashboard.get.json` + `core-go/internal/services/dashboard_service.go`).
- Value bên phải là static text `"Open global view"`, không phải count.
- Pattern = pure navigation row (button + 2 span + onClick navigate).

---

## §0.1 Flag for reviewer — Approach A reconciliation

User trước approve **Approach A (backend extend `summary.globalPlugins: integer`)**, sau đó rescope "mirror y chang Global Skills, KHÔNG invent thêm gì". Vì Global Skills hiện KHÔNG có backend count, **mirror = pure UI row, không cần đụng contract / Go / SQL.**

Spec này theo rescope (UI-only). Nếu Larry/user muốn giữ Approach A (add count field), cần raise lại vì sẽ phá nguyên tắc "mirror y chang" (Global Plugins sẽ có count còn Global Skills thì không → asymmetric).

**Decision needed before plan:** confirm UI-only mirror. Mặc định spec = UI-only.

---

## §1 Mapping cụ thể

### 1.1 Contract field — N/A

KHÔNG thay đổi `shared/api-contracts/methods/dashboard.get.json`. KHÔNG regenerate types.

### 1.2 Go count — N/A

KHÔNG thay đổi `core-go/internal/services/dashboard_service.go`, `core-go/internal/rpc/handlers/dashboard_get.go`, `core-go/internal/repositories/provider_plugin_repo.go`.

### 1.3 UI row — `apps/desktop/renderer/src/screens/dashboard-screen.tsx`

**Insert** ngay sau row "Global Skills" (sau line 127, trước row "Updates" hiện tại tại line 128):

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/plugins" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Global Plugins</span>
            <span className="text-xs text-blue-600 group-hover:text-blue-700 group-hover:underline">Open plugins view</span>
          </button>
```

**Diff = thêm 1 block 7 dòng**, structure copy 100% từ Global Skills row, chỉ đổi:
- `to: "/global"` → `to: "/plugins"`
- label `"Global Skills"` → `"Global Plugins"`
- value text `"Open global view"` → `"Open plugins view"`

### 1.4 Route verify

Verify `/plugins` route exists trong renderer router (đã verify từ Slice B: `plugins-screen.tsx` mount tại `/plugins`).

---

## §2 Edge cases

1. **Empty state (chưa có plugin nào):** row vẫn render và click vẫn đi `/plugins`. `plugins-screen` tự handle empty state. Giống y chang Global Skills khi `/global` chưa có data.
2. **Loading:** Dashboard loading state (spinner toàn screen) đã cover; row chỉ render khi `data.activeHost != null` (same gate với Global Skills row). Vì row KHÔNG phụ thuộc data backend mới, không có loading riêng.
3. **Error backend:** nếu `dashboard.get` fail, `ErrorDisplay` render thay cho toàn screen → row không render. Không có error riêng cho row plugins.
4. **`activeHost == null`:** Dashboard render branch "No Skill Host Folder configured" (line 51–63), KHÔNG render Summary section → row Global Plugins KHÔNG hiển thị. Behavior này khớp Global Skills.
5. **Conditional render:** KHÔNG có condition đặc biệt cho row mới (giống Global Skills). Luôn render khi Summary render.
6. **Cursor + hover:** kế thừa từ class copy → `cursor-pointer`, `group-hover:underline`, `hover:bg-zinc-50` đã đầy đủ. Không cần audit thêm.

---

## §3 Test impact

### 3.1 Test hiện có

- `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx`:
  - Test `"navigates to global view when summary global row is clicked"` (~line 130) dùng matcher `/Global Skills Open global view/i`. Row mới có label `"Global Plugins"` + `"Open plugins view"` → matcher cũ vẫn match chỉ Global Skills row (regex không nuốt sang row mới).
  - Test `"uses pointer cursor for clickable summary rows"` (~line 152) query 3 button (Skills/Projects/Attention) → không bị ảnh hưởng.
  - Test `"uses blue link style for clickable summary values"` (vừa thêm Slice B) query 4 button (Skills/Projects/Attention/Global Skills) bằng tên cụ thể → không bị ảnh hưởng (không query Global Plugins).
  - Test `"navigates to skills and projects from summary rows"` → không bị ảnh hưởng.

### 3.2 Test mới — Recommended

Thêm 1 test vào `dashboard-screen.test.tsx`:

```tsx
it("navigates to plugins view when Global Plugins row is clicked", () => {
  const mockNavigate = vi.fn();
  mockUseNavigate.mockReturnValue(mockNavigate);
  mockUseDashboard.mockReturnValue({
    isPending: false,
    isError: false,
    data: baseData,
    refetch: vi.fn(),
  });

  render(<DashboardScreen />);
  fireEvent.click(screen.getByRole("button", { name: /Global Plugins Open plugins view/i }));
  expect(mockNavigate).toHaveBeenCalledWith({ to: "/plugins" });
});
```

Cover navigation + verify row rendered đúng label.

### 3.3 Test convention reuse (optional)

Có thể mở rộng test `"uses blue link style for clickable summary values"` để include Global Plugins button. Implementer quyết định trong plan.

### 3.4 Snapshot

Không có snapshot test trong repo. Không rủi ro.

### 3.5 Backend tests

KHÔNG thêm/sửa Go test, KHÔNG thêm/sửa contract test (vì không đụng contract / Go).

---

## §4 Smoke scenarios (Larry execute)

**S1 — Render row**
1. Launch desktop app với active host configured.
2. Mở `/` (Dashboard).
3. Trong section "Summary", **giữa** "Global Skills" row và "Updates" row, có row mới label "Global Plugins" với value `"Open plugins view"`.
4. **Expected:** value text màu blue (`text-blue-600`), hover row → background `bg-zinc-50` + text blue-700 + underline.

**S2 — Click navigate**
1. Trên Dashboard, click row "Global Plugins" (click vào bất kỳ chỗ nào trong button).
2. **Expected:** điều hướng đến `/plugins` (PluginsScreen render).

**S3 — Empty state no plugin**
1. Trong môi trường chưa từng scan plugin (hoặc DB trống provider_plugin entries).
2. Mở Dashboard → row "Global Plugins" vẫn hiển thị, value vẫn `"Open plugins view"`.
3. Click → đi `/plugins` → screen render empty state riêng của `plugins-screen.tsx` ("No plugin data. Run Scan Global ..." hoặc tương tự).

**S4 — No active host**
1. Reset/clear active host (Setup screen → unset).
2. Mở Dashboard → branch "No Skill Host Folder configured" hiển thị.
3. **Expected:** row "Global Plugins" KHÔNG render (Summary section không render). Đúng pattern Global Skills.

**S5 — Backward — Global Skills không đổi**
1. Trên Dashboard, click row "Global Skills".
2. **Expected:** đi `/global` (regression check sau khi thêm row mới).

---

## §5 Files touched / KHÔNG đụng

### Touched
- `apps/desktop/renderer/src/screens/dashboard-screen.tsx` (thêm 7 dòng giữa row Global Skills và Updates)
- `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx` (thêm 1 test navigation)

### KHÔNG đụng
- `shared/api-contracts/methods/dashboard.get.json` (không thêm field)
- `shared/generated/methods/dashboard-get.ts` (không regenerate)
- `core-go/internal/services/dashboard_service.go` (không count plugin)
- `core-go/internal/rpc/handlers/dashboard_get.go` (không thêm field response)
- `core-go/internal/repositories/provider_plugin_repo.go` (không thêm CountEnabled)
- Mọi test Go (dashboard, contract, repo)
- `plugins-screen.tsx` (đã có route + logic)
- Mọi screen / feature khác

---

## §6 Non-goals

- **Per-provider plugins breakdown** (Claude X enabled, Codex Y enabled, ...) — Slice C-full nếu user muốn sau này.
- **Count tổng enabled plugins** (số nguyên kế bên label) — đã rescope OUT.
- Backend extension cho `dashboard.get` (`summary.globalPlugins` field) — moot theo rescope; cần raise lại nếu user đổi ý.
- Disable row khi plugin scan chưa xảy ra — không cần (mirror Global Skills cũng không có gate này).
- Thêm badge "X plugins" hoặc status indicator — out of scope.
- Reorder Summary rows — giữ thứ tự hiện có, chỉ insert giữa Global Skills và Updates.
- Đổi text "Open plugins view" thành thứ khác — giữ mirror y chang pattern "Open global view".

---

## Next step

Đợi Larry review. Sau khi pass + user confirm decision tại §0.1 → viết plan `docs/superpowers/plans/2026-05-28-dashboard-global-plugins-row-plan.md`.
