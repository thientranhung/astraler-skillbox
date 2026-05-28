# Naming Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename UI labels (sidebar, plugin buttons, project-detail plugin column) và đồng bộ docs để khớp mental model "global vs project", không đụng code identifier / contract / DB.

**Architecture:** Pure cosmetic — chỉ edit string literals trong renderer + descriptive text trong docs. Mỗi task chạm 1 file, kèm verification command. Test strings cập nhật song hành với label edit để Vitest pass liên tục.

**Tech Stack:** TypeScript/React (renderer), Vitest (tests), Markdown (docs). Không đụng Go, JSON-RPC contract, hoặc SQLite.

**Reference spec:** [`docs/superpowers/specs/2026-05-28-naming-alignment-design.md`](../specs/2026-05-28-naming-alignment-design.md)

---

## File Structure

Files modified (8 total):

**Renderer (3):**
- `apps/desktop/renderer/src/components/sidebar.tsx` — sidebar nav labels
- `apps/desktop/renderer/src/screens/plugins-screen.tsx` — Action button label
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — plugin column header + tooltip

**Tests (3):**
- `apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx`
- `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx`
- `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx`

**Docs (2):**
- `docs/03-information-architecture.md`
- `docs/02-product-notes.md`

---

## Task 1: Rename sidebar labels

**Files:**
- Modify: `apps/desktop/renderer/src/components/sidebar.tsx:7,10`
- Test: `apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx:13,24`

- [ ] **Step 1: Update sidebar.tsx**

In `apps/desktop/renderer/src/components/sidebar.tsx` lines 5-12, replace the NAV_ITEMS array:

```ts
export const NAV_ITEMS = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/skills", label: "Host Skills", icon: Library },
  { to: "/global", label: "Global Skills", icon: Globe },
  { to: "/projects", label: "Projects", icon: FolderGit2 },
  { to: "/plugins", label: "Global Plugins", icon: Puzzle },
  { to: "/settings", label: "Settings", icon: Settings },
] as const;
```

Only `Skills` → `Host Skills` (line 7) and `Plugins` → `Global Plugins` (line 10) change. Routes (`to:` values) stay.

- [ ] **Step 2: Update sidebar.test.tsx**

In `apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx`:

Line 13: `const skillsIdx = labels.indexOf("Skills");` → `const skillsIdx = labels.indexOf("Host Skills");`

Line 24: `const pluginsIdx = labels.indexOf("Plugins");` → `const pluginsIdx = labels.indexOf("Global Plugins");`

- [ ] **Step 3: Run sidebar test to verify pass**

Run: `(cd apps/desktop && pnpm test -- sidebar.test)`

Expected: all sidebar tests PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/desktop/renderer/src/components/sidebar.tsx apps/desktop/renderer/src/components/__tests__/sidebar.test.tsx
git commit -m "Rename sidebar labels: Skills → Host Skills, Plugins → Global Plugins"
```

---

## Task 2: Rename Plugins screen Action button

**Files:**
- Modify: `apps/desktop/renderer/src/screens/plugins-screen.tsx:77`
- Test: `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx:177,178,189,200,213,232`

- [ ] **Step 1: Update plugins-screen.tsx**

In `apps/desktop/renderer/src/screens/plugins-screen.tsx` line 77, replace:

```tsx
{isEnabled ? "Disable globally" : "Enable globally"}
```

with:

```tsx
{isEnabled ? "Disable" : "Enable"}
```

Tiêu đề `Provider Plugins` (line 232) và nút `Scan Global` (line 243) giữ nguyên.

- [ ] **Step 2: Update plugins-screen.test.tsx — 6 occurrences**

In `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx`, replace:
- Line 177: `name: "Disable globally"` → `name: "Disable"`
- Line 178: `name: "Enable globally"` → `name: "Enable"`
- Line 189: `name: "Enable globally"` → `name: "Enable"`
- Line 200: `name: "Disable globally"` → `name: "Disable"`
- Line 213: `name: "Disable globally"` → `name: "Disable"`
- Line 232: `name: "Disable globally"` → `name: "Disable"`

Verify với grep sau khi sửa:

```bash
grep -n "Disable globally\|Enable globally" apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx
```

Expected: no match.

- [ ] **Step 3: Run plugins-screen test to verify pass**

Run: `(cd apps/desktop && pnpm test -- plugins-screen.test)`

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/desktop/renderer/src/screens/plugins-screen.tsx apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx
git commit -m "Rename plugin toggle button: Disable/Enable globally → Disable/Enable"
```

---

## Task 3: Rename Project Detail plugin column User → Global

**Files:**
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx:366,433,434`
- Test: `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx:360,370,373,382,461`

- [ ] **Step 1: Update column header in project-detail-screen.tsx**

In `apps/desktop/renderer/src/screens/project-detail-screen.tsx` line 366, replace:

```tsx
<th className="px-3 py-1.5 text-xs font-medium text-zinc-500">User</th>
```

with:

```tsx
<th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Global</th>
```

Header line 363 (`Project`) and line 368 (`Effective`) stay.

- [ ] **Step 2: Update tooltip strings**

In `apps/desktop/renderer/src/screens/project-detail-screen.tsx` lines 432-434, replace:

```tsx
                                    : isUserEnabled
                                      ? "Disable globally"
                                      : "Enable globally"
```

with:

```tsx
                                    : isUserEnabled
                                      ? "Disable"
                                      : "Enable"
```

Line 431 tooltip `"Project layer overrides this setting"` stays.

- [ ] **Step 3: Update project-detail-screen.test.tsx — assertions (failing if not updated)**

Line 370: `getByRole("columnheader", { name: "User" })` → `getByRole("columnheader", { name: "Global" })`

Line 382: `getByRole("columnheader", { name: "User" })` → `getByRole("columnheader", { name: "Global" })`

- [ ] **Step 4: Update project-detail-screen.test.tsx — test description strings (cosmetic)**

Line 360: `it("shows Project and User columns for claude project plugins"` → `it("shows Project and Global columns for claude project plugins"`

Line 373: `it("shows Project and User columns for antigravity_cli project plugins"` → `it("shows Project and Global columns for antigravity_cli project plugins"`

Line 461: `it("local override shows 'overridden' text in both Project and User columns"` → `it("local override shows 'overridden' text in both Project and Global columns"`

- [ ] **Step 5: Run project-detail-screen test to verify pass**

Run: `(cd apps/desktop && pnpm test -- project-detail-screen.test)`

Expected: all tests PASS, descriptions in output read "Project and Global".

- [ ] **Step 6: Commit**

```bash
git add apps/desktop/renderer/src/screens/project-detail-screen.tsx apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx
git commit -m "Rename project detail plugin column: User → Global (UI only)"
```

---

## Task 4: Sanity check — no stray "globally" or sidebar-label remnants in renderer

**Files:** none (verification only)

- [ ] **Step 1: Grep for stray "globally" in renderer**

Run:

```bash
grep -rn "globally" apps/desktop/renderer/src --include="*.tsx" --include="*.ts"
```

Expected: only matches in `project-detail-screen.tsx:431` (`"Project layer overrides this setting"` — không có "globally"; nếu vẫn còn match nào liên quan plugin button/tooltip, chưa fix xong Task 2/3).

Acceptable matches: code comments mentioning "globally" trong context khác (vd. business rule comment). Hiện chưa thấy. Nếu có, đánh giá từng case.

- [ ] **Step 2: Grep for old sidebar labels**

Run:

```bash
grep -rn "\"Skills\"\|\"Plugins\"" apps/desktop/renderer/src --include="*.tsx" --include="*.ts"
```

Expected matches (acceptable, KHÔNG đụng):
- `apps/desktop/renderer/src/screens/projects-screen.tsx:70` — column header `Plugins` (per-project count, giữ nguyên theo spec §2 edge case 1).
- Bất kỳ code identifier nào (vd. import, type).

Nếu thấy match nào là label hiển thị mà chưa rename → fix về Task 1.

- [ ] **Step 3: Run full renderer test suite + typecheck**

Run:

```bash
(cd apps/desktop && pnpm typecheck)
(cd apps/desktop && pnpm test)
```

Expected: both PASS.

- [ ] **Step 4: Commit sanity (chỉ nếu Task 1-3 phát hiện thiếu gì)**

Nếu không có thay đổi: skip commit.

---

## Task 5: Update docs/03-information-architecture.md

**Files:**
- Modify: `docs/03-information-architecture.md:167,188` + thêm note cuối Global Plugins section.

- [ ] **Step 1: Edit line 167**

Replace:

```
Global Plugins là nơi xem và quản lý plugin ở user layer cho các provider hỗ
```

with:

```
Global Plugins là nơi xem và quản lý plugin ở global (user) layer cho các provider hỗ
```

- [ ] **Step 2: Edit line 188**

Replace:

```
- Chỉ user layer được hiển thị ở Global Plugins. Project layer và effective
```

with:

```
- Chỉ global (user) layer được hiển thị ở Global Plugins. Project layer và effective
```

- [ ] **Step 3: Append UI/code mapping note**

Sau bullet `- Managed settings (enterprise config) là out-of-scope.` (line 191) thêm block mới (cách 1 dòng trắng):

```markdown
> **Naming note:** UI hiển thị label `Global` cho layer mà code/contract dùng
> identifier `user` (`layer: "user"`, `PluginLayerUser`, SQL `settings_layer =
> 'user'`). End-user terminology favors `Global`; code/data terminology giữ
> `user` để không phá contract và DB.
```

- [ ] **Step 4: Verify**

Run:

```bash
grep -n "user layer\|global (user) layer\|Naming note" docs/03-information-architecture.md
```

Expected: thấy `global (user) layer` ở line 167 và 188, `Naming note` ở cuối Global Plugins section.

- [ ] **Step 5: Commit**

```bash
git add docs/03-information-architecture.md
git commit -m "Doc: align Global Plugins terminology with renamed UI labels"
```

---

## Task 6: Update docs/02-product-notes.md

**Files:**
- Modify: `docs/02-product-notes.md:91`

- [ ] **Step 1: Edit line 91**

Locate the bullet (under `Phase 1 scope:` của section `Plugins và Marketplaces`):

```
- Toggle enable/disable globally (user layer) hoặc per-project (project
  layer, 3-state cycle: inherit → enabled → disabled).
```

Replace với:

```
- Toggle enable/disable ở global (user) layer hoặc per-project (project
  layer, 3-state cycle: inherit → enabled → disabled).
```

- [ ] **Step 2: Rà các đề cập "user layer" khác trong section**

Run:

```bash
grep -n "user layer" docs/02-product-notes.md
```

Mỗi match: nếu là văn mô tả → đổi sang `global (user) layer`. Nếu là code reference trong backtick (`layer: "user"`) → giữ nguyên.

- [ ] **Step 3: Verify**

Run:

```bash
grep -n "disable globally\|enable globally" docs/02-product-notes.md
```

Expected: no match (đã thay).

- [ ] **Step 4: Commit**

```bash
git add docs/02-product-notes.md
git commit -m "Doc: sync product notes with global/user layer naming"
```

---

## Task 7: Final verification + smoke handoff

**Files:** none (verification only)

- [ ] **Step 1: Full typecheck + test**

Run:

```bash
(cd apps/desktop && pnpm typecheck)
(cd apps/desktop && pnpm test)
```

Expected: both PASS.

- [ ] **Step 2: Contract drift sanity (no expected change)**

Run:

```bash
(cd apps/desktop && pnpm check:contracts-drift)
```

Expected: PASS (slice không đụng contracts).

- [ ] **Step 3: Visual confirm trong dev**

Run: `(cd apps/desktop && pnpm dev)` (background). Mở app, click qua: Dashboard → Host Skills (sidebar) → Global Plugins (sidebar) → 1 project có plugin → Project Detail.

Quick visual checks:
- Sidebar đọc `Host Skills`, `Global Plugins`.
- Global Plugins screen Action button đọc `Disable`/`Enable`.
- Project Detail plugin table cột header đọc `Global`.

Nếu OK, đóng `pnpm dev`.

- [ ] **Step 4: Hand-off cho Larry smoke test**

Báo orchestrator: implementation done. Larry execute 7 smoke scenarios S1-S7 trong spec §4. Tom đợi report.

---

## Spec coverage check

| Spec section | Plan task |
|--------------|-----------|
| §1 Sidebar | Task 1 |
| §1 Plugins screen | Task 2 |
| §1 Project Detail screen | Task 3 |
| §1 Dashboard (no-op) | — (intentional skip) |
| §1 Docs/03 | Task 5 |
| §1 Docs/02 | Task 6 |
| §1 Tests sidebar | Task 1 step 2 |
| §1 Tests plugins-screen (6 chỗ) | Task 2 step 2 |
| §1 Tests project-detail (assertions + descriptions) | Task 3 steps 3-4 |
| §2 Edge cases (projects-screen "Plugins" giữ) | Task 4 step 2 |
| §3 Test impact | Task 4 step 3, Task 7 step 1 |
| §4 Smoke scenarios | Task 7 step 4 (Larry executes) |

Không có gap.
