# Dashboard "Global Plugins" Row Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 1 navigation row "Global Plugins" trên Dashboard Summary, mirror y chang Global Skills pattern, click → `/plugins`.

**Architecture:** Pure UI insertion (7-dòng button block) trong renderer; KHÔNG đụng contract / Go / SQL. Reuse class string từ Global Skills row, đổi `to`, label, value text.

**Tech Stack:** React 19 + TypeScript + Tailwind CSS 4, Vitest + @testing-library/react.

**Spec:** [`../specs/2026-05-28-dashboard-global-plugins-row-design.md`](../specs/2026-05-28-dashboard-global-plugins-row-design.md)

**Larry F1 (apply):** Test `"uses pointer cursor for clickable summary rows"` mở rộng từ 3 → 5 rows (Skills, Projects, Attention needed, Global Skills, Global Plugins).

---

## Task 1: Insert row + update tests

**Files:**
- Modify: `apps/desktop/renderer/src/screens/dashboard-screen.tsx` (insert 7 dòng sau line 127)
- Modify: `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx` (extend cursor test + add navigation test)

### Steps

- [ ] **1.1 — Extend cursor-pointer test (F1)**

Tìm test `"uses pointer cursor for clickable summary rows"` (~line 152). Thay 3 assertion bằng 5:

```tsx
    expect(screen.getByRole("button", { name: /^Skills 5$/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /^Projects 3$/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /^Attention needed 2$/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /Global Skills Open global view/i }).className).toContain("cursor-pointer");
    expect(screen.getByRole("button", { name: /Global Plugins Open plugins view/i }).className).toContain("cursor-pointer");
```

- [ ] **1.2 — Add navigation test for /plugins**

Thêm test mới ngay sau test "navigates to global view ..." (gần line 133):

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

- [ ] **1.3 — Run tests verify FAIL**

Run: `pnpm test -- dashboard-screen.test.tsx`
Expected: 2 test mới fail (button "Global Plugins ..." not found).

- [ ] **1.4 — Insert row trong dashboard-screen.tsx**

Tìm Global Skills row (lines 120–127):

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

Insert ngay sau `</button>` của Global Skills, TRƯỚC `<div>` row Updates:

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

- [ ] **1.5 — Run tests verify PASS**

Run: `pnpm test`
Expected: full suite PASS.

- [ ] **1.6 — Typecheck**

Run: `pnpm typecheck`
Expected: PASS.

- [ ] **1.7 — Commit**

```bash
git add apps/desktop/renderer/src/screens/dashboard-screen.tsx \
        apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx
git commit -m "ui(dashboard): add Global Plugins navigation row mirroring Global Skills"
```
