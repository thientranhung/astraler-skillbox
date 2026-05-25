# Repository Guidelines

## Project Structure & Module Organization

This repository is currently a planning and architecture repository for Astraler Skillbox, a local-first desktop app for managing agent skills. The source of truth is `README.md` plus the numbered documents in `docs/`; read `docs/index.md` first for the intended order. Review prompts live in `docs/review-prompts/`, and completed review notes live in `docs/review-results/`.

The planned implementation structure is documented in `docs/11-tech-stack-and-scaffold-decisions.md`: Electron + React under `apps/desktop/`, Go core under `core-go/`, shared API contracts under `shared/api-contracts/`, helper scripts under `scripts/`, and test fixtures under `fixtures/`. Do not create alternate top-level layouts without updating the architecture docs.

## Build, Test, and Development Commands

No application scaffold exists yet, so there are no build or test commands to run today. For documentation-only changes, verify Markdown by reading the changed files and checking links manually:

```sh
rg "TODO|GAP|Open" docs README.md
git diff --check
```

When the scaffold is added, prefer the documented stack: `pnpm` for `apps/desktop`, `electron-vite` for Electron targets, and `go test ./...` for `core-go`.

## Coding Style & Naming Conventions

Keep documentation concise, specific, and ordered by decision flow. Existing docs use numbered filenames such as `01-product-brief.md`; continue this pattern for major source-of-truth documents. Use lowercase kebab-case for new Markdown filenames, for example `13-release-plan.md`.

For future code, preserve the architecture boundaries: React renderer must not access filesystem or SQLite directly; Electron main owns lifecycle and preload bridge concerns; Go core owns domain logic, repositories, provider adapters, filesystem gateway, operations, migrations, and JSON-RPC handlers.

## Testing Guidelines

Current changes are documentation-only. Validate consistency against `README.md`, `docs/index.md`, and the relevant architecture document before committing. Once code exists, add focused tests near the affected layer: Vitest for renderer TypeScript, Playwright for desktop/e2e flows, and Go unit/integration tests for `core-go`.

## Commit & Pull Request Guidelines

Recent commits use short imperative messages such as `Add implementation patterns document` and `Remove obsolete reviewer prompt`. Follow that style: one clear action, present tense, no trailing period.

Pull requests should include a brief summary, affected docs or modules, verification performed, and linked issue or decision document when applicable. For UI work, include screenshots or short recordings; for architecture changes, update the relevant numbered doc and `docs/index.md`.

## Agent-Specific Instructions

Before implementing, read the relevant docs in order from `docs/index.md`. Keep edits scoped, do not rewrite unrelated Vietnamese project text unless requested, and preserve documented stack decisions unless explicitly changing the decision record.

For multi-agent coordination through `agent-tech-skillbox` and `agent-lead-skillbox`, read `docs/agent-orchestration-playbook.md` first. Use it as the hardening checklist for tmux hygiene, long-prompt delivery, phase gates, review loops, and recovery from stale TUI input.
