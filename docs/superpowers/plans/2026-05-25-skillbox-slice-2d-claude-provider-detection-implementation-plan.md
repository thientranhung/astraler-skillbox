# Slice 2D Claude Provider Detection Implementation Plan

> **For agentic workers:** This is a cross-layer implementation with a clear validation loop. Use `/goal` only when executing the full slice end to end; do not use `/goal` for follow-up findings or narrow fixes.

**Goal:** Add read-only Claude provider detection while preserving correct multi-provider warning behavior and keeping install writes out of scope.

**Architecture:** Go core owns provider adapters, migrations, scan aggregation, and project provider data. Renderer only displays detected provider state and read-only target preview.

**Tech Stack:** Go provider adapters/repositories/services, SQLite migrations, JSON Schema contracts, generated TypeScript, React renderer, Vitest/Go tests.

---

## Tasks

### Task 1: Provider Seed And Registry

**Files:** `core-go/migrations/**`, `core-go/migrations/migrations.go`, `core-go/cmd/skillbox-core/main.go`, provider seed tests

- [ ] Add migration `000003` to seed `claude` provider definition as `experimental`, `can_create_structure=0`, `has_global_level=1`.
- [ ] Treat `has_global_level=1` as inert metadata in this slice; do not add global scan or global UI behavior.
- [ ] Seed path candidates: `.claude` with purpose `detect`, `.claude/skills` with purpose `skills`.
- [ ] Add down migration that removes only the seeded Claude rows.
- [ ] Register `NewClaudeAdapter()` beside `NewGenericAgentsAdapter()`.
- [ ] Add registry-vs-seed guard test.

### Task 2: Claude Adapter

**Files:** `core-go/internal/providers/claude.go`, `core-go/internal/providers/claude_test.go`

- [ ] Add constants for Claude key and relative detect/skills paths.
- [ ] Implement read-only detection mirroring Generic Agents semantics.
- [ ] Test missing `.claude`, file/unreadable `.claude`, missing `.claude/skills`, populated skills, symlink entries, and unreadable skills.
- [ ] Add seed-vs-adapter path drift tests for Claude and Generic Agents.

### Task 3: Warning Aggregation

**Files:** `core-go/internal/services/project_service.go`, project scan tests

- [ ] Suppress per-adapter `no_provider_detected` warnings during multi-adapter scan.
- [ ] Emit one project-level `no_provider_detected` warning only when no provider is detected after all adapters run.
- [ ] Test `.agents`-only project has no Claude-missing warning.
- [ ] Test `.claude`-only project has no Generic Agents missing warning.
- [ ] Test no-provider project still reports `no_provider_detected`.

### Task 4: Contracts And UI Preview

**Files:** `shared/api-contracts/**`, `shared/generated/**`, renderer project components/tests as needed

- [ ] Keep exposed detection statuses limited to existing contract values.
- [ ] Add contract/domain drift test proving registered adapters emit only contract-allowed statuses.
- [ ] Ensure Project Detail/List render multiple providers, including experimental Claude, as read-only provider targets.
- [ ] Do not add install/create/sync actions.

### Task 5: Verification

- [ ] `(cd core-go && go test ./...)`
- [ ] `(cd apps/desktop && pnpm generate:contracts && pnpm check:contracts-drift)`
- [ ] `(cd apps/desktop && pnpm typecheck && pnpm test && pnpm build)`
- [ ] `git diff --check`
