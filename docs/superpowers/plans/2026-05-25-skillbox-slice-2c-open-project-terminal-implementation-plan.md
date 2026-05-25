# Slice 2C Open Project Terminal Implementation Plan

> **For agentic workers:** This is a small implementation plan. Do not use `/goal`; execute with normal prompts or inline work.

**Goal:** Add a safe macOS Open Terminal project action.

**Architecture:** Electron main owns native terminal launch; renderer exposes a narrow UI action. No Go core, DB, or project filesystem writes.

**Tech Stack:** Electron main, React, TanStack Query mutation, JSON Schema contracts, Vitest.

---

## Tasks

### Task 1: Contract

**Files:** `shared/api-contracts/electron/dialog.openTerminal.json`, `shared/api-contracts/index.json`, `shared/generated/**`

- [ ] Add strict request/response schema.
- [ ] Regenerate contracts.
- [ ] Verify `pnpm check:contracts-drift`.

### Task 2: Electron Bridge

**Files:** `apps/desktop/electron/main/core-process/ipc-bridge.ts`, `method-allowlist.ts`

- [ ] Add `dialog.openTerminal` allowlist entry.
- [ ] Launch Terminal with argument-array process execution: `open -a Terminal <path>`.
- [ ] Map launch failures to structured `unknown_error`.

### Task 3: Renderer

**Files:** `methods.ts`, project hooks, `project-row.tsx`, `project-detail-screen.tsx`, tests

- [ ] Add `methods.openTerminal(path)`.
- [ ] Add `useOpenProjectTerminal`.
- [ ] Add Terminal icon buttons in list and detail.
- [ ] Add method/hook/client tests.

### Task 4: Verification

- [ ] `(cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift)`
- [ ] `(cd apps/desktop && pnpm typecheck && pnpm test && pnpm build)`
- [ ] `git diff --check`
