# Slice 2D: Claude Provider Detection — Design

Date: 2026-05-25
Status: approved
Scope: add read-only Claude project provider detection before any provider install flow.

## Purpose

Skill install targets depend on provider conventions. Before symlink install, Skillbox needs a multi-provider project model that can show Claude beside the existing `.agents` provider without writing files or guessing unverified provider paths.

## In Scope

- Add a distinct `claude` provider definition and path candidates for `.claude` and `.claude/skills`.
- Add a read-only `ClaudeAdapter` that detects project-level Claude folder structure.
- Register Claude in the provider registry and seed it through migration `000003`.
- Show detected Claude providers in the existing project provider list/detail surfaces.
- Fix project warning aggregation so `no_provider_detected` is emitted only when no provider is detected.
- Add guard tests for registry/seed drift, seed/adapter path drift, and contract/domain status drift.

## Out Of Scope

- Symlink, rsync, copy, mkdir, or provider structure creation.
- Codex, Antigravity, opencode, custom providers, and global provider scan.
- Runtime use of `provider_path_candidates` by adapters.
- Contract enum expansion for `configured`, `unsupported`, or `format_unknown`.
- Skill format conversion or install-state classification changes.
- Any behavior based on Claude `has_global_level`; the flag is inert metadata in this slice.

## Technical Approach

Claude mirrors `GenericAgentsAdapter` with provider-specific paths. `.claude` missing produces no provider row. `.claude` as a file or unreadable directory produces `invalid_structure`. A valid `.claude` with no `.claude/skills` is detected with zero entries. Existing skill-entry scanning is reused when `.claude/skills` exists.

Provider path candidates remain seeded metadata in this slice, while adapters keep hardcoded paths. Drift tests enforce that seeded candidate rows match adapter constants until a later refactor wires candidates into adapter resolution.

The multi-provider warning behavior changes from per-adapter `no_provider_detected` to aggregate project-level warning behavior: only emit `no_provider_detected` after all adapters fail to detect a provider.

## Acceptance Criteria

- A project with `.claude/skills` and `.agents/skills` shows both providers.
- A project with only `.agents` does not warn that Claude is missing.
- A project with only `.claude` does not warn that `.agents` is missing.
- A project with no provider still reports `no_provider_detected`.
- Claude invalid and missing structures match the documented semantics.
- No filesystem writes occur.
- Go tests, desktop typecheck/tests/build, contract generation check, and `git diff --check` pass.
