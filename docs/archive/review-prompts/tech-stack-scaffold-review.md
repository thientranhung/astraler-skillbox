# Tech Stack And Scaffold Review Prompt

You are reviewing the tech stack and scaffold decisions for Astraler Skillbox.

Astraler Skillbox is a GUI-first desktop app for managing AI agent skills across
Skill Host Folder, Global Skills, and Project Installs.

Already decided:

```text
Desktop = Electron
UI = React
Core runtime = Golang
Database = SQLite
Electron main <-> Go core transport = stdio JSON-RPC 2.0 for Phase 1
Go core lifecycle = sidecar process managed by Electron main for Phase 1
```

Review target:

```text
docs/11-tech-stack-and-scaffold-decisions.md
```

Context docs to read:

```text
README.md
docs/index.md
docs/06-data-model.md
docs/08-provider-model.md
docs/09-ui-wireframes.md
docs/10-technical-architecture.md
docs/11-tech-stack-and-scaffold-decisions.md
docs/review-results/technical-architecture-brainstorm.md
docs/review-results/transport-decision-brainstorm.md
```

Write your review result to:

```text
docs/review-results/tech-stack-scaffold-review.md
```

## Review Goal

Review whether the recommended scaffold and tech stack choices are coherent,
safe, and practical before implementation begins.

This is not a general product review. Focus on technical scaffold decisions,
dependency choices, build/dev/test workflow, and whether any major gap exists
before code scaffolding.

## Questions To Answer

1. Is the proposed project structure appropriate for Electron + React + Go
   sidecar?
2. Are the boundaries between renderer, Electron main, preload, Go core,
   JSON-RPC, SQLite, filesystem gateway, and provider adapters clear enough?
3. Are the recommended UI stack choices sensible for this type of operational
   desktop app?
4. Is `shadcn/ui + Radix + Tailwind + lucide-react` a reasonable choice, or is
   there a better UI direction?
5. Are TanStack Router, TanStack Query, TanStack Table, React Hook Form, and Zod
   justified, or does the stack become too heavy too early?
6. Is `electron-builder` the right packaging direction given the bundled Go
   binary?
7. Is `pnpm workspace` appropriate, or should the repo stay simpler?
8. Is `modernc.org/sqlite` the right default, or should `mattn/go-sqlite3` be
   preferred despite CGO?
9. Should migrations use `golang-migrate`, a custom embedded SQL runner, or
   another option?
10. Is the JSON-RPC protocol section missing any important scaffold-level
    requirement?
11. Are there security gaps in Electron/preload/renderer/Go process boundaries?
12. Are testing recommendations sufficient before scaffold?
13. Are any dependencies unnecessary for Phase 1?
14. Are any missing dependencies likely to become painful quickly?
15. What should be decided before scaffold, and what can safely be deferred?

## Required Output Format

```md
# Tech Stack And Scaffold Review Result

## Reviewer

- Agent/model:
- Review date:
- Context used:
- Browsing used:

## Decision

Approved / Approved With Changes / Not Approved

Short rationale.

## Critical Issues

Issues that must be fixed before scaffold.

For each issue:

- Severity:
- File/section:
- Problem:
- Why it matters:
- Recommended fix:

## Non-Blocking Suggestions

Useful improvements that can be handled before or during scaffold.

## Decision-by-Decision Assessment

Assess each area:

- Project structure
- Boilerplate direction
- Vite/build config
- Package manager
- Electron packaging
- UI component stack
- Router
- Query/state
- Forms/validation
- Tables
- JSON-RPC details
- API contracts
- SQLite/migrations
- Keychain
- Testing
- Security

## Missing Decisions

List decisions that still need user/Codex confirmation before scaffold.

## Overengineering Risks

List choices that may be too heavy for Phase 1.

## Underengineering Risks

List missing choices that may cause rework later.

## Recommended Scaffold Set

Give your final recommended Phase 1 scaffold stack.
```
