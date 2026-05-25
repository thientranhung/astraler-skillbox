# Slice 2E: Provider Terminology And Install Targets — Design

Date: 2026-05-25
Status: approved
Scope: clarify `.agents/skills` terminology and introduce read-only install target metadata before symlink install.

## Purpose

`.agents/skills` is not an unknown or generic-only convention. It is a shared agent skills path that can be used by Codex, Antigravity, and compatible agents. Claude remains the special case with `.claude/skills`.

Before adding symlink install, Skillbox needs user-facing wording and target selection concepts that match this model without renaming stable provider keys or adding write behavior.

## Decisions

- Keep the internal provider key `generic_agents` unchanged in this slice.
- Present `generic_agents` to users as `Shared Agent Skills (.agents)`.
- Present `claude` as `Claude (.claude)`.
- Treat provider keys as detected filesystem conventions, not user intent.
- Introduce install target terminology as read-only metadata for the next install slice:
  - `shared_agents` resolves to `.agents/skills`.
  - `claude` resolves to `.claude/skills`.
- Do not persist install target IDs as replacements for provider keys yet.

## In Scope

- Update docs and UI copy that currently says `Generic Agents` where the user sees it.
- Add a small read-only core install target model for future install flow groundwork.
- Ensure list/detail surfaces still expose `providerKey = generic_agents` while displaying `Shared Agent Skills (.agents)`.
- Add tests that verify display labels and target metadata without changing provider persistence.

## Out Of Scope

- Renaming `generic_agents` to `shared_agents` in migrations, database rows, contracts, or persisted scan data.
- Adding Codex, Antigravity, or provider aliases as detected providers.
- Filesystem writes, symlink creation, mkdir, repair, or install execution.
- Exposing install target metadata through JSON-RPC or renderer UI.
- Persisting install target selections.
- Runtime provider-path candidate resolution.

## UI Wording

Use `Shared Agent Skills (.agents)` for `.agents/skills`. Supporting text may say it is compatible with Codex, Antigravity, and other agents that load `.agents/skills`, but the UI must not claim Codex or Antigravity were detected as separate providers.

Use `Claude (.claude)` for `.claude/skills`.

## Risks

Renaming internal keys now would create migration and compatibility risk without improving install behavior. The safer path is presentation-layer correction now, then a real install-write contract later.

The main UX risk is implying product-specific detection for Codex or Antigravity. Slice 2E avoids that by labeling `.agents/skills` as a shared target, not a Codex provider.

The display-name migration must only update known seeded display names so a future customized label is not overwritten by a terminology migration.

## Acceptance Criteria

- No user-facing project list/detail label says `Generic Agents`.
- `providerKey` remains `generic_agents` in contracts and responses.
- The shared target path is `.agents/skills`; the Claude target path is `.claude/skills`.
- Install target metadata is core-only groundwork and is not exposed through JSON-RPC in this slice.
- No filesystem write paths are introduced.
- Existing provider detection behavior remains unchanged.
- Tests cover list/detail labels, core target metadata, unchanged provider keys, and no lingering renderer-visible `Generic Agents` text.
