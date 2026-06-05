# Provider Truth & Scan UX Implementation Plan

Status: DONE on branch `codex/provider-truth-scan-ux`. Implemented by commits `db90f10b`, `ef113660`, and `740ca72b`; verification reported green for Go tests, renderer tests, typecheck, contracts drift, and diff check.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining provider/scan truth UX gaps on branch `codex/provider-truth-scan-ux`, specifically the "never scanned" vs "no providers" ambiguity in Project Detail and missing renderer tests for Host Skills empty states.

**Architecture:** Three-task batch: (1) fix Project Detail to distinguish "never scanned" from "scanned but no providers found", (2) add Host Skills empty-state tests to the existing screen test file, (3) add Project Detail "never scanned" state test. No Go changes needed — the backend reconciliation and its tests are already complete.

**Tech Stack:** React + TypeScript, Vitest + @testing-library/react, happy-dom

---

## Audit Summary (pre-verified, do not re-audit)

Already implemented and tested — do not rework:
- Host Skills: auto-scan on `lastScanAt == null`, "Not yet scanned" empty state, "No skills found" state
- Global Skills: provider tabs with counts, all provider status values, auto-scan
- Global Plugins: provider tabs, "never scanned" badge, visual distinction from Global Skills
- Go: `CommitProjectScan` reconciles deleted provider folder → `detection_status='missing'` + cascade installs → `install_status='missing'`; `TestProjectScanRepo_CommitProjectScan_AbsentProviderInstallsBecomeMissing` covers this
- Settings: host folder read-only, no DB version, correct column order

Real gaps (only these need work):
1. `project-detail-screen.tsx` line 769: `providers.length === 0` shows folder-creation guidance for BOTH "never scanned" and "scanned+empty" — should show a "Scan to detect" prompt when `lastScannedAt == null`
2. `skills-library-screen.test.tsx`: no tests for the "Not yet scanned", "No skills found", or scanning empty states
3. `project-detail-screen.test.tsx`: no test for the new "never scanned" variant

---

## Task 1: Fix Project Detail "never scanned" vs "no providers" distinction

**Files:**
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx` (lines ~769–780)

Current behavior: when `providers.length === 0`, always shows "No provider folders detected" + folder-creation guidance, regardless of whether the project has ever been scanned.

Required: when `data.project.lastScannedAt == null` and `providers.length === 0`, show a "Scan this project" prompt. When `lastScannedAt != null` and `providers.length === 0`, keep the existing folder-creation guidance.

- [x] **Step 1: Read the current providers section**

Read `apps/desktop/renderer/src/screens/project-detail-screen.tsx` lines 762–800 to confirm the current JSX structure before editing.

- [x] **Step 2: Replace the providers empty state with two-branch logic**

Find this block (approximately line 769):

```tsx
{data.providers.length === 0 ? (
  <div className="rounded border border-zinc-200 bg-zinc-50 px-3 py-3">
    <p className="text-sm font-medium text-zinc-700">No provider folders detected</p>
    <p className="mt-1 text-xs text-zinc-500">
      To make this project install-ready, create a provider folder manually inside the project, for example:
    </p>
    <ul className="mt-1.5 list-inside list-disc text-xs text-zinc-500">
      <li><code className="font-mono">.agents/skills/</code> — Shared Agent Skills</li>
      <li><code className="font-mono">.claude/skills/</code> — Claude Code</li>
    </ul>
    <p className="mt-1.5 text-xs text-zinc-400">After creating the folder, scan the project again to detect providers.</p>
  </div>
) : (
```

Replace with:

```tsx
{data.providers.length === 0 ? (
  data.project.lastScannedAt == null ? (
    <div className="rounded border border-zinc-200 bg-zinc-50 px-3 py-3">
      <p className="text-sm font-medium text-zinc-700">Not yet scanned</p>
      <p className="mt-1 text-xs text-zinc-500">
        Scan this project to detect provider folders and skill entries.
      </p>
    </div>
  ) : (
    <div className="rounded border border-zinc-200 bg-zinc-50 px-3 py-3">
      <p className="text-sm font-medium text-zinc-700">No provider folders detected</p>
      <p className="mt-1 text-xs text-zinc-500">
        To make this project install-ready, create a provider folder manually inside the project, for example:
      </p>
      <ul className="mt-1.5 list-inside list-disc text-xs text-zinc-500">
        <li><code className="font-mono">.agents/skills/</code> — Shared Agent Skills</li>
        <li><code className="font-mono">.claude/skills/</code> — Claude Code</li>
      </ul>
      <p className="mt-1.5 text-xs text-zinc-400">After creating the folder, scan the project again to detect providers.</p>
    </div>
  )
) : (
```

- [x] **Step 3: Run typecheck to confirm no type errors**

```bash
(cd apps/desktop && pnpm typecheck)
```

Expected: exits 0 with no errors.

- [x] **Step 4: Commit the UI fix**

```bash
git add apps/desktop/renderer/src/screens/project-detail-screen.tsx
git commit -m "fix: distinguish never-scanned vs scanned-no-providers in Project Detail providers section"
```

---

## Task 2: Add Host Skills empty-state tests

**Files:**
- Modify: `apps/desktop/renderer/src/screens/__tests__/skills-library-screen.test.tsx`

The existing test file has no coverage for the three conditional empty states in `SkillsLibraryScreen`:
- `isPending` → spinning state (if rendered)
- `skills.length === 0 && lastScanAt == null` → "Not yet scanned"
- `skills.length === 0 && lastScanAt != null` → "No skills found"

- [x] **Step 1: Read the end of the existing test file to find the insertion point**

Read `apps/desktop/renderer/src/screens/__tests__/skills-library-screen.test.tsx` lines 125–148 (end of file) to confirm the `describe` block ends there.

- [x] **Step 2: Add three new tests inside the describe block**

Append these tests inside the `describe("SkillsLibraryScreen", () => {` block (before its closing `}`):

```tsx
  it("shows 'Not yet scanned' when skills empty and lastScanAt is null", () => {
    mockUseSkillsList.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, skills: [], totals: { available: 0, missing: 0, unreadable: 0, local_modified: 0, unknown: 0 }, lastScanAt: null },
    });

    render(<SkillsLibraryScreen />);
    expect(screen.getByText("Not yet scanned")).toBeTruthy();
    expect(screen.queryByText("No skills found")).toBeNull();
  });

  it("shows 'No skills found' when skills empty and lastScanAt is set", () => {
    mockUseSkillsList.mockReturnValue({
      isPending: false,
      isError: false,
      data: { ...baseData, skills: [], totals: { available: 0, missing: 0, unreadable: 0, local_modified: 0, unknown: 0 }, lastScanAt: "2026-06-05T10:00:00Z" },
    });

    render(<SkillsLibraryScreen />);
    expect(screen.getByText("No skills found")).toBeTruthy();
    expect(screen.queryByText("Not yet scanned")).toBeNull();
  });

  it("does not show empty state messages when skills are present", () => {
    mockUseSkillsList.mockReturnValue({ isPending: false, isError: false, data: baseData });

    render(<SkillsLibraryScreen />);
    expect(screen.queryByText("Not yet scanned")).toBeNull();
    expect(screen.queryByText("No skills found")).toBeNull();
  });
```

- [x] **Step 3: Run the Host Skills tests to confirm all pass**

```bash
(cd apps/desktop && pnpm test --run renderer/src/screens/__tests__/skills-library-screen.test.tsx)
```

Expected: all tests pass (the three new ones plus the 8 existing ones).

- [x] **Step 4: Commit**

```bash
git add apps/desktop/renderer/src/screens/__tests__/skills-library-screen.test.tsx
git commit -m "test: add Host Skills empty-state tests for not-yet-scanned and no-skills-found"
```

---

## Task 3: Add Project Detail "never scanned" test

**Files:**
- Modify: `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx`

After the Task 1 UI fix, the providers section has two distinct empty states. The existing test at line ~769 (`TC-PROJ-009`) covers the "scanned, no providers" case (it uses `lastScannedAt: null` in the project fixture — but now that will show "Not yet scanned" instead). That test needs to be updated AND a new test for the "no providers after scan" case must be added.

- [x] **Step 1: Read the TC-PROJ-009 test to confirm its current fixture**

Read `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx` lines 768–782.

The current test data fixture (line 75) has `lastScannedAt: null` on the project. After the Task 1 fix, `providers.length === 0 + lastScannedAt == null` shows "Not yet scanned", so TC-PROJ-009 will now verify a different path. Update it and add a complementary test.

- [x] **Step 2: Update TC-PROJ-009 and add TC-PROJ-010**

Find this block (approximately line 768):

```tsx
  // TC-PROJ-009: no-provider project must show guidance, not an empty section, and must not create folders
  it("shows guidance when no providers are detected", () => {
    mockUseProjectDetail.mockReturnValue({
      data: { ...projectDetail, providers: [], entries: [] },
      isPending: false,
      isError: false,
      error: null,
    });

    render(<ProjectDetailScreen />);
    expect(screen.getByText("No provider folders detected")).toBeTruthy();
    expect(screen.getByText(/create a provider folder manually/i)).toBeTruthy();
  });
```

Replace with:

```tsx
  // TC-PROJ-009: project not yet scanned shows scan prompt, not folder-creation guidance
  it("shows scan prompt when project has never been scanned (no providers, no lastScannedAt)", () => {
    mockUseProjectDetail.mockReturnValue({
      data: {
        ...projectDetail,
        project: { ...projectDetail.project, lastScannedAt: null },
        providers: [],
        entries: [],
      },
      isPending: false,
      isError: false,
      error: null,
    });

    render(<ProjectDetailScreen />);
    expect(screen.getByText("Not yet scanned")).toBeTruthy();
    expect(screen.queryByText("No provider folders detected")).toBeNull();
    expect(screen.queryByText(/create a provider folder manually/i)).toBeNull();
  });

  // TC-PROJ-010: scanned project with no providers shows folder-creation guidance
  it("shows folder-creation guidance when project has been scanned but no providers detected", () => {
    mockUseProjectDetail.mockReturnValue({
      data: {
        ...projectDetail,
        project: { ...projectDetail.project, lastScannedAt: "2026-06-05T10:00:00Z" },
        providers: [],
        entries: [],
      },
      isPending: false,
      isError: false,
      error: null,
    });

    render(<ProjectDetailScreen />);
    expect(screen.getByText("No provider folders detected")).toBeTruthy();
    expect(screen.getByText(/create a provider folder manually/i)).toBeTruthy();
    expect(screen.queryByText("Not yet scanned")).toBeNull();
  });
```

- [x] **Step 3: Run the project detail tests to confirm all pass**

```bash
(cd apps/desktop && pnpm test --run renderer/src/screens/__tests__/project-detail-screen.test.tsx)
```

Expected: all tests pass.

- [x] **Step 4: Commit**

```bash
git add apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx
git commit -m "test: add TC-PROJ-009/010 to cover never-scanned vs scanned-no-providers distinction"
```

---

## Task 4: Full verification pass

- [x] **Step 1: Run Go tests**

```bash
go test ./...
```

Expected: all pass (no Go changes in this plan, so this is a regression guard).

- [x] **Step 2: Run all renderer tests**

```bash
(cd apps/desktop && pnpm test --run)
```

Expected: all pass.

- [x] **Step 3: Run typecheck**

```bash
(cd apps/desktop && pnpm typecheck)
```

Expected: exits 0.

- [x] **Step 4: Run contract drift check**

```bash
(cd apps/desktop && pnpm check:contracts-drift)
```

Expected: exits 0.

- [x] **Step 5: Check for whitespace/merge-marker issues**

```bash
git diff --check
```

Expected: no output.

---

## Self-Review Against Spec

| Requirement | Status | Evidence |
|---|---|---|
| Host Skills auto-scan / not-scanned state | Pre-existing | `skills-library-screen.tsx` lines 24–35, 208–213 |
| Host Skills "no skills found" after scan | Pre-existing | `skills-library-screen.tsx` lines 215–220 |
| Host Skills tests for these states | **Added in Task 2** | New tests in `skills-library-screen.test.tsx` |
| Global Skills all providers covered | Pre-existing | `WithProviderRegistryLister` in `global_skills_service.go` |
| Global Skills provider tabs/counts | Pre-existing + tested | `global-skills-screen.test.tsx` existing tests |
| Project provider folder deleted → missing on scan | Pre-existing | `markAbsentProvidersMissing` + `cascadeInstallsMissingForAbsentProviders` |
| Go test for above | Pre-existing | `TestProjectScanRepo_CommitProjectScan_AbsentProviderInstallsBecomeMissing` |
| Project Detail guidance when no provider | **Fixed in Task 1** | Two-branch logic by `lastScannedAt` |
| Project Detail tests for never-scanned vs no-providers | **Added in Task 3** | TC-PROJ-009 + TC-PROJ-010 |
| Global Skills / Global Plugins visual separation | Pre-existing | Different screens, different data models |
| Settings: host folder read-only | Pre-existing | `settings-screen.tsx` lines 363–389 |
| Settings: no DB version | Pre-existing | Not rendered anywhere |
| Settings: column order | Pre-existing | Provider → Key → Detection → GlobalConfig → ProjectConfig → GlobalSkills → ProjectSkills |
