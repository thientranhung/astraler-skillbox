# Add Skill Wizard — Provider Tabs (Spec)

- **Date**: 2026-05-29
- **Branch**: `fix/add-skill-provider-tabs`
- **Brainstorm**: `.scratch/brainstorm-add-skill-provider-grouping-note.md`
- **Scope**: UI-only refactor của Add Skill wizard trong Project Detail.

## 1. Problem

Wizard hiện (`apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx`) đặt provider thành radio nhỏ phía trên một flat skill list dùng chung. Hệ quả:

- Provider không phải trục chính → user không thấy rõ "skill này sẽ install vào provider nào / path nào".
- Không phân biệt được skill đã installed ở provider A vs còn available ở provider B (dù `project.get.providers[].entries[]` đã có đủ dữ liệu).
- Vi phạm guideline `docs/08-provider-model.md` ("installed skills nên được group hoặc filter theo provider").

Write path (Go `install.skill` → `skills_path` của providerKey) đã đúng; không sửa.

## 2. Decision summary

| Quyết định | Giá trị |
|---|---|
| UX | Tabs per provider (Option A) |
| Tab header | Display name + badge `skillsPath` folder name + tooltip full path; provider `experimental` thêm badge "experimental" |
| Submit | 1 submit = 1 provider (active tab) — tránh `conflict_error 1005` |
| Empty state (0 installable provider) | Card + CTA "Scan project" gọi flow scan hiện có |
| Single-provider | Vẫn render tab strip 1 tab (consistent với multi) |
| Workflow | Branch + PR |

## 3. Component design

### 3.1 `add-skill-wizard.tsx`

**Props** (thêm `entries` required so với hiện tại):

```ts
interface AddSkillWizardProps {
  projectId: number;
  providers: ProjectGetProvider[];
  skills: SkillListSkill[];
  entries: ProjectGetEntry[]; // required — dùng để compute installed-state per provider
  onClose: () => void;
}
```

`entries` là top-level array từ `project.get` (entries không nhúng trong `ProjectGetProvider`); parent (`project-detail-screen.tsx`) truyền xuống. Không default `[]` — caller phải pass tường minh để tránh silently mất installed-state.

**State**:

```ts
const [activeProviderKey, setActiveProviderKey] = useState<string>("");
const [selectedSkillIds, setSelectedSkillIds] = useState<Set<number>>(new Set());
```

`selectedSkillIds` reset khi đổi tab (vì state per-active-tab khác nhau).

**Derived**:

- `installableProviders` = filter `supported|experimental` + `detected|configured` (giữ logic hiện tại, file:22-30).
- `installedSkillIdsByProvider: Map<providerKey, Set<skillId>>` = build từ `entries` (lọc `entry.skillId != null` group theo `entry.providerKey`).
- `activeProvider` = `installableProviders.find(p => p.providerKey === activeProviderKey)`.
- `installedForActive` = `installedSkillIdsByProvider.get(activeProviderKey) ?? new Set()`.

**Render branches**:

| Trạng thái | Render |
|---|---|
| `installableProviders.length === 0` | EmptyState (xem 3.2) |
| `>= 1` | TabStrip + TabBody + Footer |

**TabStrip**:

- Mỗi tab = button: `<ProviderIcon> + displayName + <code>{folderName(skillsPath)}</code>` + tooltip = full `skillsPath`.
- Active tab có border-bottom + text-zinc-900; inactive xám.
- Nếu `providerStatus === "experimental"` → render small "experimental" badge.
- Helper `folderName(path)`: trả basename, hoặc đoạn cuối 2 segment (`/.agents/skills` → `.agents/skills`); chốt trong implementation, suggest `path.split('/').slice(-2).join('/')`.

**TabBody** — skill list:

- Render `availableSkills` (status `available`) như hiện tại.
- Mỗi row: checkbox + name + `relativePath`.
- Nếu skill.id ∈ `installedForActive` → checkbox disabled, row dim opacity, append badge "Installed" thay cho path bên phải.
- Empty: "No available skills" (giữ message hiện tại).

**Footer** (top → bottom order):

1. Hint (text-xs): `Sẽ ghi vào: <activeProvider.skillsPath>` (truncate giữa với ellipsis + tooltip full path on hover).
2. Error row: khi `installMutation.isError`, render error message (`text-red-600`, `text-sm`) giữa hint và button row. Display raw message từ RPC error (đặc biệt `conflict_error 1005` hiển thị nguyên message để user biết project đang có active operation). Ẩn khi không error.
3. Button row: Cancel + Install.
   - Install disabled khi `selectedSkillIds.size === 0` hoặc `installSkill.isPending`.
   - Submit: `install.skill { projectId, providerKey: activeProviderKey, skillIds: [...selectedSkillIds] }`.

### 3.2 Empty state branch

Khi `installableProviders.length === 0`:

```
┌─────────────────────────────────────────────┐
│ No provider is ready for install.           │
│                                             │
│ Run a scan to detect providers in this      │
│ project, or open Settings to configure one. │
│                                             │
│ [ Scan project ]   [ Cancel ]               │
└─────────────────────────────────────────────┘
```

CTA "Scan project":

- Đích: gọi cùng mutation `useScanProject` đã có ở `project-detail-screen.tsx:487` (`scan.mutate(projectId)`), sau đó `onClose()`.
- Cách wiring (chọn 1 trong implementation):
  - **(a)** wizard tự gọi `useScanProject()` (đơn giản nhất, isolate).
  - **(b)** parent truyền callback `onTriggerScan?: () => void` để giữ scan ownership ở screen.
- Recommend **(a)** vì wizard tự đóng sau khi trigger, không cần parent share state.

### 3.3 `project-detail-screen.tsx`

Edit nhỏ ở chỗ render `<AddSkillWizard>` (line 759-770):

- Pass thêm `entries={data.entries}` (hoặc filter theo project providers nếu cần).
- Không đổi gì khác.

## 4. Data flow

```
project.get ──► providers[]      ─┐
            └─► entries[]        ─┤
                                  ├─► AddSkillWizard
skill.list  ──► skills[]         ─┘   (compute installed-state, render tabs)

User picks tab+skills+Install
       └─► install.skill {projectId, providerKey, skillIds}  (unchanged)
```

- `entries[i].skillId` link 1 entry với 1 host skill; `entries[i].providerKey` xác định installed tại provider nào.
- Filter `entries` để loại entries có `skillId == null` hoặc `status === "missing"` (skill đó coi như chưa available để tick install lại). Final rule: chỉ disable nếu có entry `skillId === skill.id && providerKey === active && status ∈ {current, outdated, needs_sync, conflict}` — tinh chỉnh trong implement.

## 5. Boundaries

| Layer | Đổi? |
|---|---|
| Renderer (React) | ✅ `add-skill-wizard.tsx` rewrite + nhỏ ở `project-detail-screen.tsx` |
| Preload bridge | ❌ |
| `shared/api-contracts/*` | ❌ (không đổi schema) |
| `shared/generated/*` | ❌ |
| Electron main | ❌ |
| Go core | ❌ |
| SQLite migration | ❌ |

## 6. Tests

### 6.1 Vitest (renderer)

File mục tiêu: `apps/desktop/renderer/src/features/projects/__tests__/add-skill-wizard.test.tsx` (đã có; rewrite).

Fixture cần:

- Project có 2 installable providers (`generic_agents`, `claude`), 1 unsupported provider → expect chỉ 2 tab.
- `skills`: 3 host skills (S1, S2, S3) tất cả `available`.
- `entries`:
  - S1 installed ở `generic_agents`, status `current`.
  - S2 installed ở `claude`, status `current`.
  - S3 chưa cài ở đâu cả.

Test cases:

| # | Scenario | Assertion |
|---|----------|-----------|
| T1 | Render multi-provider | 2 tab; tab đầu active mặc định |
| T2 | Tab A (generic) active | S1 disabled + badge "Installed"; S2, S3 tickable |
| T3 | Switch tab → claude | S2 disabled; S1, S3 tickable; `selectedSkillIds` reset |
| T4 | Tick S3 → Install | mutation gọi với `{providerKey: <activeKey>, skillIds: [S3.id]}` |
| T5 | Footer hint hiển thị `skillsPath` của active tab | text contains absolute path |
| T6 | Provider `experimental` badge | badge "experimental" render |
| T7 | Empty state | 0 installable → render CTA; click "Scan project" gọi scan mutation với projectId; wizard close |
| T8 | Single provider | TabStrip render 1 tab; vẫn hoạt động bình thường |
| T9 | Install pending | button Install disabled + label "Installing…" |

### 6.2 Smoke scenarios (e2e/manual)

Mirror brainstorm note §6 + bổ sung empty state:

| # | Setup | Action | Expected |
|---|-------|--------|----------|
| S1 | 2 providers detected | Mở wizard | 2 tab, tab đầu active |
| S2 | 1 provider detected | Mở wizard | 1 tab |
| S3 | 0 provider hợp lệ | Mở wizard | Empty state + CTA "Scan project" |
| S4 | Tick S1+S2 ở tab generic → Install | RPC | `providerKey: generic_agents`; file ghi vào `<P>/.agents/skills/` |
| S5 | S1 đã installed ở generic | Tab generic | S1 disabled badge "Installed" |
| S6 | S1 đã installed ở generic, switch claude | Tab claude | S1 tickable; install ghi vào `<P>/.claude/.../` |
| S7 | Provider claude experimental | Tab claude | badge "experimental" |
| S8 | Active operation cho project | Bấm Install | RPC trả `conflict_error 1005`; UI hiển thị inline error (toast/text) |
| S9 | Path override cho skillsPath | Tab → footer hint | hint show đúng path đã override (từ `project.get.providers[].skillsPath`) |
| S10 | Empty state CTA | Click "Scan project" | scan mutation fire với projectId; wizard close |

### 6.3 Test files

- Rewrite: `apps/desktop/renderer/src/features/projects/__tests__/add-skill-wizard.test.tsx`.
- Touch (nếu cần): `apps/desktop/renderer/src/features/projects/__tests__/use-install-skill.test.tsx` (chỉ nếu signature mutate đổi — không dự kiến đổi).

## 7. Docs touch

Theo `docs/playbooks/documentation.md`:

| Doc | Sửa gì |
|---|---|
| `docs/03-information-architecture.md` | Nếu có mô tả Add Skill modal → update mô tả tabs |
| `docs/04-user-flows.md` | Update flow "Add skill to project": provider chọn bằng tab; empty state CTA |
| `docs/05-edge-cases-and-ux-states.md` | Add wizard empty state (0 installable provider), installed-badge per provider, reset-on-tab-switch behavior, experimental badge behavior |
| `docs/08-provider-model.md` | Không bắt buộc; có thể thêm 1 câu xác nhận "Add Skill wizard nhóm theo provider tab" trong mục "UI Representation" |
| ADR | Không cần (UI refactor, không thay đổi quyết định kiến trúc) |

Implement plan sẽ confirm các file trên có nội dung cần sửa hay không (Tom check trước khi mở section).

## 8. Risk Classification

| Field | Value |
|---|---|
| Layers | UI only (renderer) |
| Breaking change | no |
| Schema/migration | no |
| Contract change | no |
| Est. LOC | 50–300 (rewrite wizard + tests) |
| Workflow | branch + PR (UI visible → screenshot trong PR theo `AGENTS.md`) |

Không phát hiện rủi ro mới so với brainstorm.

## 9. Resolved decisions (Larry defaults)

Các open questions ở vòng spec trước đã được Larry chốt:

1. **Active tab default**: tab đầu trong `installableProviders` (theo thứ tự backend trả). Không có "most installable" logic.
2. **`installedForActive` disable rule**: chỉ disable khi entry status ∈ `{current, outdated, needs_sync, conflict}`. Status `missing` / `broken_symlink` cho phép re-install (skill được coi là chưa chiếm path).
3. **Footer path display**: truncate giữa với ellipsis + tooltip hiển thị full path on hover.
4. **Empty state Scan CTA**: share `isPending` qua `useScanProject` để CTA disabled + label "Scanning…" khi scan đang chạy ở screen ngoài. **Plan-phase TODO**: verify `useScanProject` mutation key có scope theo `projectId` để 2 component cùng dùng share state đúng project; nếu không, plan phải nâng key scope.
5. **Reset selection on tab switch**: reset `selectedSkillIds` về empty Set khi user switch tab (không giữ per-tab Map).

Không còn open question nào ở spec level. Mọi câu hỏi mới sẽ raise trong plan phase.

---

Stop condition: dừng sau spec. Plan + implement sẽ làm ở phase sau khi Larry approve.
