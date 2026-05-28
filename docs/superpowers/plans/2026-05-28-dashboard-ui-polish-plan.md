# Dashboard UI Polish (Slice B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dashboard summary values dùng blue link affordance (`text-blue-600` + hover-700 + underline) và bổ sung `cursor-pointer` cho mọi text/icon clickable đang thiếu ở global-skills, plugins, project-detail, add-skill-wizard screens.

**Architecture:** Pure cosmetic Tailwind class changes ở renderer layer — không đụng JSON-RPC contract, không đụng Go backend, không đụng SQL. Dashboard dùng pattern `group` trên `<button>` + `group-hover:*` trên `<span>` để hover affordance trải dài toàn row. Các screen khác chỉ bổ sung `cursor-pointer` (và `disabled:cursor-not-allowed` khi có disabled prop).

**Tech Stack:** React 19 + TypeScript + Tailwind CSS 4, Vitest + @testing-library/react, pnpm workspace.

**Spec:** [`docs/superpowers/specs/2026-05-28-dashboard-ui-polish-design.md`](../specs/2026-05-28-dashboard-ui-polish-design.md)

**Convention:** Clickable value text dùng `text-blue-600 hover:text-blue-700 hover:underline`. Khi value nằm trong `<button>` parent, dùng `group` trên parent + `group-hover:text-blue-700 group-hover:underline` trên span.

---

## Pre-flight

- [ ] **Verify baseline tests pass before starting**

Run: `pnpm --filter desktop test`
Expected: full suite PASS. Nếu fail trước khi sửa, dừng và báo orchestrator.

- [ ] **Verify typecheck baseline**

Run: `pnpm --filter desktop typecheck`
Expected: PASS.

---

## Task 1: Dashboard hyperlink color (4 summary rows)

**Files:**
- Modify: `apps/desktop/renderer/src/screens/dashboard-screen.tsx` (4 button + span pairs trong Summary section)
- Test: `apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx` (thêm 1 test mới)

### Step 1.1: Write failing test cho blue value class

- [ ] **Step 1.1 — Add test assertion**

Thêm test mới ngay sau test `"uses pointer cursor for clickable summary rows"` (line ~164):

```tsx
  it("uses blue link style for clickable summary values", () => {
    mockUseDashboard.mockReturnValue({
      isPending: false,
      isError: false,
      data: baseData,
      refetch: vi.fn(),
    });

    render(<DashboardScreen />);
    const skillsBtn = screen.getByRole("button", { name: /^Skills 5$/i });
    const projectsBtn = screen.getByRole("button", { name: /^Projects 3$/i });
    const attentionBtn = screen.getByRole("button", { name: /^Attention needed 2$/i });
    const globalBtn = screen.getByRole("button", { name: /Global Skills Open global view/i });

    // Parent buttons must opt-in to group-hover
    expect(skillsBtn.className).toContain("group");
    expect(projectsBtn.className).toContain("group");
    expect(attentionBtn.className).toContain("group");
    expect(globalBtn.className).toContain("group");

    // Value spans (last child) must use blue link class
    for (const btn of [skillsBtn, projectsBtn, attentionBtn, globalBtn]) {
      const valueSpan = btn.querySelector("span:last-child");
      expect(valueSpan).not.toBeNull();
      expect(valueSpan!.className).toContain("text-blue-600");
      expect(valueSpan!.className).toContain("group-hover:text-blue-700");
      expect(valueSpan!.className).toContain("group-hover:underline");
    }
  });
```

- [ ] **Step 1.2 — Run test to verify FAIL**

Run: `pnpm --filter desktop test -- dashboard-screen.test.tsx`
Expected: new test FAIL với `Expected "..." to contain "group"`.

### Step 1.3: Apply class changes to dashboard-screen.tsx

- [ ] **Step 1.3 — Edit Skills row (button + span)**

Tìm block (~lines 94–101):

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/skills" })}
            className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Skills</span>
            <span className="text-sm text-zinc-500">{data.summary.skills}</span>
          </button>
```

Đổi thành:

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/skills" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Skills</span>
            <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.skills}</span>
          </button>
```

- [ ] **Step 1.4 — Edit Projects row**

Tìm block (~lines 102–109):

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/projects" })}
            className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Projects</span>
            <span className="text-sm text-zinc-500">{data.summary.projects}</span>
          </button>
```

Đổi thành:

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/projects" })}
            className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Projects</span>
            <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.projects}</span>
          </button>
```

- [ ] **Step 1.5 — Edit Attention needed row**

Tìm block (~lines 110–119):

```tsx
          {data.summary.warnings > 0 && (
            <button
              type="button"
              onClick={() => navigateToAttention(data.warnings)}
              className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
            >
              <span className="text-sm font-medium text-zinc-700">Attention needed</span>
              <span className="text-sm text-zinc-500">{data.summary.warnings}</span>
            </button>
          )}
```

Đổi thành:

```tsx
          {data.summary.warnings > 0 && (
            <button
              type="button"
              onClick={() => navigateToAttention(data.warnings)}
              className="group flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
            >
              <span className="text-sm font-medium text-zinc-700">Attention needed</span>
              <span className="text-sm text-blue-600 group-hover:text-blue-700 group-hover:underline">{data.summary.warnings}</span>
            </button>
          )}
```

- [ ] **Step 1.6 — Edit Global Skills row**

Tìm block (~lines 120–127):

```tsx
          <button
            type="button"
            onClick={() => navigate({ to: "/global" })}
            className="flex w-full cursor-pointer items-center justify-between px-4 py-3 text-left hover:bg-zinc-50"
          >
            <span className="text-sm font-medium text-zinc-700">Global Skills</span>
            <span className="text-xs text-zinc-500">Open global view</span>
          </button>
```

Đổi thành:

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

- [ ] **Step 1.7 — Run dashboard tests verify PASS**

Run: `pnpm --filter desktop test -- dashboard-screen.test.tsx`
Expected: all PASS including new test.

- [ ] **Step 1.8 — Commit**

```bash
git add apps/desktop/renderer/src/screens/dashboard-screen.tsx \
        apps/desktop/renderer/src/screens/__tests__/dashboard-screen.test.tsx
git commit -m "ui(dashboard): blue link affordance for clickable summary values"
```

---

## Task 2: cursor-pointer audit — global-skills-screen.tsx

**Files:**
- Modify: `apps/desktop/renderer/src/screens/global-skills-screen.tsx` (3 buttons)

Không cần test mới — change cơ học, không ảnh hưởng logic.

- [ ] **Step 2.1 — Add cursor-pointer to "Scan Global" header button**

Tìm (~lines 51–58):

```tsx
        <button
          onClick={() => scanMutation.mutate()}
          disabled={isScanning || scanMutation.isPending}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
```

Đổi `className` thành:

```tsx
          className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
```

(Thêm `cursor-pointer` và `disabled:cursor-not-allowed` để consistent với pattern ở skills-library-screen.)

- [ ] **Step 2.2 — Add cursor-pointer to per-location "Open Folder"**

Tìm (~lines 104–110):

```tsx
                      <button
                        onClick={() => handleOpenFolder((loc.skillsPath ?? loc.path)!)}
                        className="flex items-center gap-1 rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50"
                      >
```

Đổi `className` thành:

```tsx
                        className="flex cursor-pointer items-center gap-1 rounded border border-zinc-300 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50"
```

- [ ] **Step 2.3 — Add cursor-pointer to per-entry "Open" button**

Tìm (~lines 143–149):

```tsx
                            <button
                              onClick={() => handleOpenFolder(entry.globalSkillPath)}
                              className="flex items-center gap-1 rounded border border-zinc-200 px-2 py-0.5 text-xs text-zinc-500 hover:bg-zinc-50"
                            >
```

Đổi `className` thành:

```tsx
                              className="flex cursor-pointer items-center gap-1 rounded border border-zinc-200 px-2 py-0.5 text-xs text-zinc-500 hover:bg-zinc-50"
```

- [ ] **Step 2.4 — Run tests verify PASS**

Run: `pnpm --filter desktop test -- global-skills-screen.test.tsx`
Expected: PASS.

- [ ] **Step 2.5 — Commit**

```bash
git add apps/desktop/renderer/src/screens/global-skills-screen.tsx
git commit -m "ui(global-skills): add cursor-pointer to scan/open buttons"
```

---

## Task 3: cursor-pointer audit — plugins-screen.tsx

**Files:**
- Modify: `apps/desktop/renderer/src/screens/plugins-screen.tsx` (2 buttons)

- [ ] **Step 3.1 — Add cursor-pointer to `PluginToggleButton`**

Tìm (~lines 72–79):

```tsx
    <button
      onClick={() => onToggle(plugin.pluginName, plugin.marketplaceName, !isEnabled)}
      disabled={disabled}
      className="rounded border border-zinc-200 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-40"
    >
```

Đổi `className` thành:

```tsx
      className="cursor-pointer rounded border border-zinc-200 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-40"
```

- [ ] **Step 3.2 — Add cursor-pointer to header "Scan Global" button**

Tìm (~lines 237–244):

```tsx
        <button
          onClick={() => scanMutation.mutate()}
          disabled={isScanning}
          className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
        >
```

Đổi `className` thành:

```tsx
          className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
```

- [ ] **Step 3.3 — Run tests verify PASS**

Run: `pnpm --filter desktop test -- plugins-screen.test.tsx`
Expected: PASS.

- [ ] **Step 3.4 — Commit**

```bash
git add apps/desktop/renderer/src/screens/plugins-screen.tsx
git commit -m "ui(plugins): add cursor-pointer to scan and toggle buttons"
```

---

## Task 4: cursor-pointer audit — project-detail-screen.tsx

**Files:**
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx` (12 buttons)

Đây là file lớn nhất; làm tuần tự từ trên xuống, cẩn thận chỉ chèn `cursor-pointer` (và `disabled:cursor-not-allowed` khi cần). Không đụng class khác.

- [ ] **Step 4.1 — PathCell Copy button (~lines 121–129)**

Tìm:

```tsx
        <button
          type="button"
          onClick={() => void copyPath()}
          aria-label={`Copy ${label}`}
          title={`Copy ${label}`}
          className="shrink-0 rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
        >
```

Đổi className thành:

```tsx
          className="shrink-0 cursor-pointer rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
```

- [ ] **Step 4.2 — EntryRow Remove button (~lines 188–195)**

Tìm:

```tsx
        <button
          onClick={() => onRemove(entry)}
          disabled={!isRemovable(entry)}
          title={isRemovable(entry) ? "Remove skill from project" : "Only current symlink installs can be removed in this slice"}
          className="rounded border border-zinc-300 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
        >
```

Đổi className thành:

```tsx
          className="cursor-pointer rounded border border-zinc-300 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
```

- [ ] **Step 4.3 — Project layer cycle button (~lines 389–414)**

Tìm:

```tsx
                              <button
                                onClick={() => {
                                  if (projState === "not-set") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, true);
                                  } else if (projState === "enabled") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, false);
                                  } else {
                                    handleRemoveProjectOverride(projectView.providerKey, p.pluginName, p.marketplaceName);
                                  }
                                }}
                                disabled={isOperationInFlight}
                                title={
                                  projState === "not-set"
                                    ? "Click to enable at project level"
                                    : projState === "enabled"
                                      ? "Click to disable at project level"
                                      : "Click to clear project override"
                                }
                                className={`rounded px-1.5 py-0.5 font-medium disabled:cursor-not-allowed disabled:opacity-40 ${
                                  projState === "not-set"
                                    ? "text-zinc-400 hover:bg-zinc-100"
                                    : projectStateBadgeClass(projState) + " hover:opacity-80"
                                }`}
                              >
```

Đổi base `className` (string đầu trong template) thành:

```tsx
                                className={`cursor-pointer rounded px-1.5 py-0.5 font-medium disabled:cursor-not-allowed disabled:opacity-40 ${
```

- [ ] **Step 4.4 — User layer toggle button (~lines 426–443)**

Tìm:

```tsx
                                <button
                                  onClick={() => handleToggleUserPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, !isUserEnabled)}
                                  disabled={isOperationInFlight}
                                  title={
                                    projectHasValue
                                      ? "Project layer overrides this setting"
                                      : isUserEnabled
                                        ? "Disable"
                                        : "Enable"
                                  }
                                  className={`rounded px-1.5 py-0.5 font-medium hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40 ${
                                    isUserEnabled
                                      ? "bg-green-100 text-green-700"
                                      : "bg-zinc-100 text-zinc-500"
                                  }`}
                                >
```

Đổi base `className` thành:

```tsx
                                  className={`cursor-pointer rounded px-1.5 py-0.5 font-medium hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40 ${
```

- [ ] **Step 4.5 — Back to Projects link button (~lines 532–538)**

Tìm:

```tsx
          <button
            onClick={() => void navigate({ to: "/projects" })}
            className="flex shrink-0 items-center gap-1 text-xs text-zinc-500 hover:text-zinc-800"
          >
```

Đổi className thành:

```tsx
            className="flex shrink-0 cursor-pointer items-center gap-1 text-xs text-zinc-500 hover:text-zinc-800"
```

- [ ] **Step 4.6 — Header action buttons (~lines 559–602): Scan, Open Folder, Terminal, Add Skill, Remove**

Áp dụng cho 5 button liên tiếp. Cho mỗi button, thêm `cursor-pointer` ngay sau `flex items-center gap-1.5` và `disabled:cursor-not-allowed` ngay trước `disabled:opacity-50` (nếu chưa có).

Scan button (~559):
```tsx
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
```
→
```tsx
              className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
```

Open Folder button (~568): cùng pattern.

Terminal button (~577): cùng pattern.

Add Skill button (~586) — không có `disabled` prop:
```tsx
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50"
```
→
```tsx
              className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50"
```

Remove button (~594):
```tsx
              className="flex items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-500 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
```
→
```tsx
              className="flex cursor-pointer items-center gap-1.5 rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-500 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-50"
```

- [ ] **Step 4.7 — Provider filter pills (~lines 676–703)**

"All providers" pill (~676):
```tsx
                    className={`rounded border px-2 py-1 text-xs font-medium ${
                      selectedProviderId === "all"
                        ? "border-zinc-700 bg-zinc-900 text-white"
                        : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
                    }`}
```
→
```tsx
                    className={`cursor-pointer rounded border px-2 py-1 text-xs font-medium ${
```

Per-provider pill (~689):
```tsx
                      className={`inline-flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium ${
                        selectedProviderId === provider.projectProviderId
                          ? "border-zinc-700 bg-zinc-900 text-white"
                          : "border-zinc-200 text-zinc-600 hover:bg-zinc-50"
                      }`}
```
→
```tsx
                      className={`inline-flex cursor-pointer items-center gap-1 rounded border px-2 py-1 text-xs font-medium ${
```

- [ ] **Step 4.8 — Run tests verify PASS**

Run: `pnpm --filter desktop test -- project-detail-screen.test.tsx`
Expected: PASS.

- [ ] **Step 4.9 — Commit**

```bash
git add apps/desktop/renderer/src/screens/project-detail-screen.tsx
git commit -m "ui(project-detail): add cursor-pointer to all clickable buttons"
```

---

## Task 5: cursor-pointer audit — add-skill-wizard.tsx

**Files:**
- Modify: `apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx` (3 buttons)

- [ ] **Step 5.1 — Close (X) icon button (~lines 78–84)**

Tìm:

```tsx
        <button
          onClick={onClose}
          className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600"
          title="Close"
        >
```

Đổi className thành:

```tsx
          className="cursor-pointer rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600"
```

- [ ] **Step 5.2 — Cancel button (~lines 139–143)**

Tìm:

```tsx
        <button
          onClick={onClose}
          className="rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
        >
          Cancel
        </button>
```

Đổi className thành:

```tsx
          className="cursor-pointer rounded border border-zinc-300 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
```

- [ ] **Step 5.3 — Install button (~lines 145–151)**

Tìm:

```tsx
        <button
          onClick={handleInstall}
          disabled={!canInstall || installSkill.isPending}
          className="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
```

Đổi className thành (thêm cả `disabled:cursor-not-allowed` vì button có `disabled` prop):

```tsx
          className="cursor-pointer rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
```

- [ ] **Step 5.4 — Run any relevant tests**

Run: `pnpm --filter desktop test -- add-skill-wizard`
Expected: PASS hoặc "no test files found" (chấp nhận được — không có test file riêng cho wizard).

- [ ] **Step 5.5 — Commit**

```bash
git add apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx
git commit -m "ui(add-skill-wizard): add cursor-pointer to close/cancel/install buttons"
```

---

## Final verification

- [ ] **Step F.1 — Run full typecheck**

Run: `pnpm --filter desktop typecheck`
Expected: PASS.

- [ ] **Step F.2 — Run full desktop test suite**

Run: `pnpm --filter desktop test`
Expected: full suite PASS, không regression.

- [ ] **Step F.3 — Optional: workspace-wide lint nếu repo có script**

Run: `pnpm --filter desktop lint` (skip nếu lint script không tồn tại).
Expected: PASS hoặc skip.

- [ ] **Step F.4 — Báo orchestrator summary commit + dừng**

Liệt kê 5 commit theo task. KHÔNG tự đẩy Larry — đợi orchestrator route.

---

## Self-review checklist

**Spec coverage:**
- §1.1 Dashboard 4 row → Task 1 ✓
- §1.2 global-skills 3 button → Task 2 ✓
- §1.3 plugins 2 button → Task 3 ✓
- §1.4 project-detail 12 button → Task 4 ✓
- §1.4b add-skill-wizard 3 button → Task 5 ✓
- §1.5–§1.6 skills-library / skill-row "không đụng" → không có task (đúng) ✓
- §3.2 Recommended class test → Task 1.1 ✓
- §6 Non-goals (Slice C, focus-ring, settings/setup screen) → không có task (đúng) ✓

**Placeholder scan:** không có "TBD/TODO/similar to Task N". ✓

**Type consistency:** không định nghĩa type mới. ✓
