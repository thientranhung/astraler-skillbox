# Add Skill Wizard — Provider Tabs (Implementation Plan)

- **Date**: 2026-05-29
- **Branch**: `fix/add-skill-provider-tabs`
- **Spec**: `docs/superpowers/specs/2026-05-29-add-skill-provider-tabs-spec.md`
- **Brainstorm**: `.scratch/brainstorm-add-skill-provider-grouping-note.md`

Plan này biên dịch spec thành tasks độc lập, có verification cụ thể. Renderer only — không touch Go / preload / contracts.

## Pre-flight findings (verify trước khi code)

Đọc `apps/desktop/renderer/src/features/projects/use-install-skill.ts` và `use-scan-project.ts`:

1. **`useInstallSkill` đã invalidate `queryKeys.projects.detail(projectId)` + `list()` trong `onSuccess` của mutation**. Wizard chỉ cần KHÔNG đóng modal ngay sau `mutate()` để Error row có cơ hội render — đóng sau khi mutation success thay vì trước.
2. **`useScanProject` KHÔNG khai báo `mutationKey`** → 2 instance (screen + wizard) sẽ độc lập, `isPending` không share. → Q4 plan-TODO trigger: cần xử lý (xem Task 5).
3. `useInstallSkill.mutate` đang được gọi xong rồi `onClose()` (file `add-skill-wizard.tsx:64-72`) — wizard đóng trước khi mutation finish → mất Error row. Đổi sang `mutate(req, { onSuccess: onClose })`.

## Task breakdown

| # | Task | Files | Verification |
|---|------|-------|--------------|
| 1 | Tách helpers compute installed-state | `add-skill-wizard.tsx` (mới: nội module) | `pnpm typecheck` pass |
| 2 | Rewrite wizard layout (tab strip + body + footer) | `add-skill-wizard.tsx` | `pnpm typecheck`; visual qua dev |
| 3 | Empty state branch + Scan CTA | `add-skill-wizard.tsx` | Vitest T7; manual smoke S3+S10 |
| 4 | Wire entries prop từ parent | `project-detail-screen.tsx` (chỗ render `<AddSkillWizard>` ~line 759-770) | `pnpm typecheck` |
| 5 | Scope `useScanProject` mutation key theo projectId | `use-scan-project.ts` | `pnpm test -- use-scan-project` (nếu có); manual: 2 click Scan ở 2 chỗ cùng project chỉ trigger 1 op |
| 6 | Đổi install-submit thành `mutate(req, { onSuccess: onClose })`; render Error row khi `isError` | `add-skill-wizard.tsx` | Vitest T9 + new test "Error row hiển thị khi conflict_error" |
| 7 | Rewrite vitest fixtures + 9 test cases | `apps/desktop/renderer/src/features/projects/__tests__/add-skill-wizard.test.tsx` | `pnpm test -- add-skill-wizard` xanh |
| 8 | Docs update | `docs/03-information-architecture.md`, `docs/04-user-flows.md`, `docs/05-edge-cases-and-ux-states.md`, (optional) `docs/08-provider-model.md` | `pnpm check:contracts-drift` (không liên quan, sanity); diff review |
| 9 | Manual smoke run (10 scenarios) | n/a | Larry chạy `pnpm dev` theo §"Smoke run sequence" |
| 10 | PR opening | n/a | `gh pr create` với screenshots |

### Task 1 — Helpers

- Thêm 2 helper trong file:
  - `buildInstalledMap(entries, ACTIVE_DISABLE_STATUSES)`: `Map<providerKey, Set<skillId>>`. Lọc `entry.skillId != null` và `entry.status ∈ {current, outdated, needs_sync, conflict}` (Q2 default).
  - `shortSkillsPath(absPath)`: trả 2 segment cuối (`path.split('/').slice(-2).join('/')`).
- Const `ACTIVE_DISABLE_STATUSES = ['current','outdated','needs_sync','conflict'] as const`.
- Verify: `pnpm typecheck`.

### Task 2 — Layout rewrite

- State: `activeProviderKey` (string), `selectedSkillIds` (Set).
- `useEffect` set default `activeProviderKey = installableProviders[0]?.providerKey ?? ''` khi list thay đổi (Q1 default).
- Khi `activeProviderKey` thay đổi → `setSelectedSkillIds(new Set())` (Q5).
- TabStrip render danh sách button. Active style: border-bottom blue.
- TabBody render checkbox list, disable + dim opacity nếu `installedForActive.has(skill.id)`, badge "Installed" thay path bên phải.
- Footer hint (truncate giữa với CSS `truncate` + tooltip title=full path).
- Verify: `pnpm typecheck`; chạy `pnpm dev`, mở 1 project, mở wizard, check render.

### Task 3 — Empty state + Scan CTA

- Branch `installableProviders.length === 0`.
- Gọi `const scan = useScanProject()` trong wizard (option a từ spec §3.2).
- CTA button: `onClick={() => { scan.mutate(projectId); onClose(); }}`. Label đổi "Scanning…" + disabled khi `scan.isPending`.
- Phụ thuộc Task 5 để `isPending` đúng nếu screen cũng đang scan.
- Verify: Vitest T7; smoke S3 + S10 manual.

### Task 4 — Parent prop wiring

- `project-detail-screen.tsx` chỗ render `<AddSkillWizard>`: thêm `entries={data.entries}` (top-level field từ `project.get` response).
- Spec §F5: parent ĐÃ guard `data != null` (`{wizardOpen && validId != null && data != null && ...}` line 759) → wizard không cần tự fetch. Confirm guard giữ nguyên.
- Verify: `pnpm typecheck`.

### Task 5 — Scope `useScanProject` mutation key

- Hiện tại `useMutation({ mutationFn: ... })` không có `mutationKey`.
- Đổi sang: signature truyền projectId vào hook hoặc dùng `mutationKey: ['scan-project', projectId]` (gán động trước mutation — nhưng useMutation key static; chọn pattern: chuyển hook thành `useScanProject(projectId: number)` và đặt `mutationKey: ['scan-project', projectId]`).
- Cập nhật call sites:
  - `project-detail-screen.tsx:487` `const scan = useScanProject()` → `useScanProject(validId!)`. Cần đảm bảo `validId` non-null khi gọi (guard hiện đã có ở chỗ render Scan button line 561).
  - Wizard gọi `useScanProject(projectId)`.
- Trong `mutationFn` giờ không cần nhận projectId nữa (capture từ closure). Cẩn thận: nếu sites khác dùng (grep!) — verify trước khi đổi signature.
- Verify: `grep -rn "useScanProject" apps/desktop/renderer/src` để tìm hết call sites. `pnpm test` của file đó pass. Smoke: 2 click Scan từ Scan button + wizard CTA cho cùng project → label "Scanning…" share state.
- **Alternative nếu rủi ro lan rộng**: giữ signature cũ, dùng `useIsMutating({ mutationKey: ['scan-project', projectId] })` trong wizard sau khi gán key tĩnh — nhưng vẫn cần đổi `useMutation` để có key. Recommend đổi signature một lần.

### Task 6 — Install submit + Error row

- `handleInstall`:
  ```
  installSkill.mutate(req, { onSuccess: () => onClose() });
  ```
  Bỏ `onClose()` đứng riêng sau `mutate()`.
- `installSkill.error` có thể là `Error` từ JSON-RPC client; extract message (`String(error.message ?? error)`).
- Render Error row giữa hint và button row khi `installSkill.isError`. Reset error khi user đổi tab hoặc đổi selection? → chốt: reset bằng `installSkill.reset()` khi `activeProviderKey` đổi (cùng effect reset selection).
- Verify: vitest mock RPC trả conflict_error → wizard hiển thị message, không tự đóng.

### Task 7 — Tests

File: `apps/desktop/renderer/src/features/projects/__tests__/add-skill-wizard.test.tsx` (rewrite full).

Fixtures (in-file):

- `mkProvider(key, displayName, status, detectionStatus, skillsPath)`.
- `mkSkill(id, name)`.
- `mkEntry(skillId, providerKey, status)`.

9 test cases (mirror spec §6.1 T1–T9) + 1 thêm cho Error row:

| # | Tên |
|---|-----|
| T1 | renders one tab per installable provider |
| T2 | active tab disables installed skills (status current) |
| T3 | switching tab resets selected skills |
| T4 | install submits providerKey of active tab |
| T5 | footer hint shows active skillsPath |
| T6 | experimental provider shows badge |
| T7 | empty state renders Scan CTA and triggers scan mutation |
| T8 | single provider still renders tab strip |
| T9 | install button disabled while pending |
| T10 | (new) Error row renders when mutation fails (e.g. conflict_error 1005) and wizard stays open |

Mocks:
- `useInstallSkill` mock trả `{ mutate, isPending, isError, error, reset }` controllable.
- `useScanProject` mock tương tự.
- Wrap với `QueryClientProvider` nếu hook đụng tới (tránh phải mock toàn bộ thì mock hook trực tiếp).

Verify: `(cd apps/desktop && pnpm test -- add-skill-wizard)` xanh.

### Task 8 — Docs

Trước khi edit, đọc từng file để chắc nó có nội dung liên quan:

- `docs/03-information-architecture.md` — nếu có entry mô tả Add Skill modal: update mô tả thành "tab strip per installable provider".
- `docs/04-user-flows.md` — flow "Add skill to project": cập nhật bước chọn provider thành chọn tab; thêm branch empty state CTA "Scan project".
- `docs/05-edge-cases-and-ux-states.md` — thêm cases: 0 installable provider (empty state + Scan CTA), per-provider installed badge, reset selection on tab switch, experimental provider badge, error row khi conflict_error 1005.
- `docs/08-provider-model.md` — optional, 1 câu trong "UI Representation".

Verify: `git diff docs/` review tay; không có doc nào cần regenerate.

### Task 9 — Smoke run sequence (Larry runs manually)

Setup: `cd apps/desktop && pnpm dev`. Cần ≥ 1 project trong DB có nhiều provider detected, ≥ 3 skill trong host folder.

| # | Scenario | Cách chạy | Expected |
|---|----------|-----------|----------|
| S1 | Multi-provider | Mở project có generic + claude detected → Add Skill | 2 tab, tab generic active |
| S2 | Single-provider | Mở project chỉ có generic | 1 tab |
| S3 | Empty state | Mở project chưa scan / không có provider hợp lệ | Card empty + CTA Scan |
| S4 | Install generic | Tab generic, tick 2 skill, Install | Toast "Skills installed (2)"; FS `<P>/.agents/skills/` chứa 2 symlink |
| S5 | Installed disabled | S1 đã installed | Row S1 dim + disabled checkbox + badge "Installed" |
| S6 | Switch + reinstall | Tab → claude, tick S1 | Cho phép tick; Install ghi `<P>/.claude/.../` |
| S7 | Experimental badge | Provider claude có status experimental | Tab claude render "experimental" |
| S8 | Conflict error | Đang có operation active, bấm Install | Error row hiển thị message conflict; modal không đóng |
| S9 | Path override | Đã set override skillsPath cho generic | Footer hint hiển thị path override |
| S10 | Scan CTA sharing | Bấm Scan ở screen, mở wizard ngay khi đang scan | CTA Scan trong wizard label "Scanning…" + disabled |

Nếu fixture không reproduce được S3 / S8 / S9 thì ghi rõ trong PR description và skip với note.

### Task 10 — PR

- `gh pr create` với title "Add Skill wizard: provider tabs + empty state".
- Body bullets từ spec §2, screenshots S1/S3/S5/S6/S8 (5 ảnh đủ cover trục).
- Link spec + plan + brainstorm.

## Verify `useScanProject` mutation key (Q4 plan-TODO)

Đã verify trong Pre-flight finding #2: hiện tại không có `mutationKey`. → Task 5 xử lý: đổi `useScanProject` thành nhận `projectId` và đặt `mutationKey: ['scan-project', projectId]`. Đây là breaking change signature cục bộ, phải sweep call sites bằng `grep -rn "useScanProject"`.

## Commit strategy

- **3 commits** (mỗi commit pass `pnpm typecheck` + `pnpm test`):
  1. `refactor: scope useScanProject mutation key by projectId` (Task 5, có thể isolate).
  2. `feat: AddSkillWizard provider tabs + empty state + error row` (Task 1–4 + 6 + 7, include tests).
  3. `docs: update IA / flows / edge cases for provider tab wizard` (Task 8).
- Lý do tách: commit 1 là refactor độc lập revertible. Commit 2 là core feature. Commit 3 docs riêng để review nhanh.
- Không squash khi merge — Larry quyết định.

## Rollback note

- Wizard rewrite tương đối nhỏ (~150–250 LOC). **Không cần giữ wizard cũ song song**.
- Revert: `git revert <commit-2>` đủ khôi phục UI cũ (radio + flat list). Commit 1 (mutation key scope) có thể giữ vì backward-compatible behaviorally.
- Docs revert riêng nếu cần.
- Vì branch + PR, rollback chính thức là `git revert` trên main sau merge. Trước merge, có thể `git reset --hard <pre-feat>` trên branch.

## Out of scope (defer)

- Per-tab giữ selection (Map per provider) — Larry default = reset.
- "Install to all providers" CTA — defer Phase 2.
- Format conversion (skill ↔ provider) — Phase 2 theo provider-model.md.
- Custom provider trong wizard — open question chung của repo, không thuộc slice này.

## Stop

Plan dừng ở đây. Larry review → approve → mới sang implement.
