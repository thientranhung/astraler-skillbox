# Repository Guidelines

Canonical contributor & agent guide for Astraler Skillbox. Loaded by all AI providers (Claude Code, Codex, OpenCode, etc.). Keep concise: this file orients work; deep specs live under `docs/`.

## Project Overview

Astraler Skillbox is a local-first desktop app for managing agent skills across projects and providers (Claude, Codex, …). It is the local control center, GUI-first.

Core invariants:
- **Skill Host Folder** is source of truth for skill content.
- **SQLite** is source of truth for management metadata.
- Skills are distributed to projects via symlink or rsync/copy.
- **Skillbox is local-first. The only outbound network is manual-trigger plugin update checks against the user's already-installed plugin source hosts — no background polling, no telemetry, no Skillbox-operated server. The app is fully usable offline (see ADR-0002, supersedes ADR-0001).**

## Start Here

- For broad code discovery, read [`docs/context-map.md`](docs/context-map.md) before searching.
- For feature, cross-layer, review, PR, process, or release-impacting work, read [`docs/playbooks/governance-project.md`](docs/playbooks/governance-project.md).
- For QA runs, QA verdicts, or clean GO decisions, read [`docs/qa/governance.md`](docs/qa/governance.md).
- For concept or source-of-truth changes, read [`docs/playbooks/documentation.md`](docs/playbooks/documentation.md).
- For agent/tmux handoffs only, read [`docs/playbooks/agent-orchestration.md`](docs/playbooks/agent-orchestration.md).

Before editing code or issuing a review/QA verdict, verify target paths and
symbols exist, inspect nearby patterns, confirm the layer boundary being touched,
and map docs/QA impact. Use the full checklist in
[`docs/playbooks/governance-project.md`](docs/playbooks/governance-project.md);
QA verdicts also use [`docs/qa/governance.md`](docs/qa/governance.md).

## Repo Layout

```
apps/desktop/     # Electron + React renderer + main + preload
core-go/          # Go sidecar (domain, services, repositories, providers, RPC, migrations)
shared/           # JSON Schema contracts + generated TS types (committed)
docs/             # Source of truth — see docs/index.md for reading order
docs/.vi/         # Vietnamese authoring/review mirror for docs/ (tracked)
fixtures/         # Test fixtures for provider/filesystem
.scratch/         # Temporary drafts, long handoffs, and task briefs (gitignored)
```

Do not create alternate top-level layouts without updating `docs/10-technical-architecture.md`.

`docs/` is the canonical English documentation used by agents/providers. `docs/.vi/`
is the Vietnamese authoring/review mirror; edit there first when drafting with the
Vietnamese-speaking owner, then translate/sync approved content back to `docs/`.

Scratch files under `.scratch/` must be date-prefixed for sorting and traceability:
`YYYY-MM-DD-<topic>.md`, `YYYY-MM-DD-<topic>-<phase>.md`, or
`YYYY-MM-DD-goal-<slice>-<phase>.md`.

## Commands

```sh
# Frontend / Electron
(cd apps/desktop && pnpm install)
(cd apps/desktop && pnpm dev)                  # Full-stack with real Go sidecar
(cd apps/desktop && pnpm typecheck)
(cd apps/desktop && pnpm test)                 # Vitest
(cd apps/desktop && pnpm build)                # electron-vite build
(cd apps/desktop && pnpm check:contracts-drift)

# Go core
(cd core-go && go test ./...)
(cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...)
```

Three dev modes: **Go-only** (Go tests + JSON-RPC harness), **UI-only** (renderer with mocked client), **Full-stack** (Electron + real Go sidecar).

Release: push tag `v*.*.*` → `.github/workflows/release.yml` builds 4 platforms + creates a GitHub Release. Bump `apps/desktop/package.json` version first; set secrets `APPLE_ID` / `APPLE_APP_SPECIFIC_PASSWORD` / `APPLE_TEAM_ID` for Mac notarize.

## Architecture Boundaries (Hard Rules)

These must not be violated. Full reasoning in `docs/10-technical-architecture.md`.

- **React renderer**: render state, call commands/queries through preload bridge only. No direct `ipcRenderer`, filesystem, DB, or provider adapter access. No raw SQL joins or business rules.
- **Electron main**: window lifecycle, preload bridge, native dialogs, Go process lifecycle only. Validates JSON-RPC allowlist. No business logic.
- **Go core**: owns SQLite, filesystem writes, provider adapters, source integrations, operation runner. All filesystem writes go through `filesystem.Gateway`. Repository layer is the only place with direct SQL. Provider adapters return facts/capabilities only — no DB writes, no filesystem.

Protocol specs (SQLite PRAGMAs, JSON-RPC transport rules, Electron security defaults, CQRS conventions, operation locking) live in `docs/10-technical-architecture.md`, `docs/11-tech-stack-and-scaffold-decisions.md`, `docs/12-implementation-patterns.md`. Read those before changing protocol behavior.

## Conventions

**Language**: preserve the language already used in nearby docs. Do not translate or rewrite prose just to normalize language. Code identifiers, shared API contracts, file names, and commit messages stay in English.

**Commits**: short imperative messages in English, present tense, no trailing period. Examples: `Add implementation patterns document`, `Fix sort instability in plugins table`.

**File naming**: use lowercase kebab-case for Markdown (`release-plan.md`). Major source-of-truth docs keep numbered prefixes (`NN-name.md`). Scratch files under `.scratch/` must be date-prefixed.

**Pull requests**: when creating a PR, include a short summary and the verification performed. Link related specs, issues, or ADRs when useful. For meaningful UI changes, add a screenshot or short recording when it helps reviewers understand the change.

**Testing**: place automated tests near the affected layer (Vitest for renderer, `go test` for `core-go`). Contract changes must regenerate `shared/generated` and pass `pnpm check:contracts-drift`. For user-facing workflows, data-integrity paths, plugin behavior, or release readiness, update/run the QA bank under `docs/qa/` as appropriate.

**Documentation discipline**: when you add/change a concept (schema, RPC method, screen, domain object, provider, etc.), update the corresponding doc in the same slice. Read [`docs/playbooks/documentation.md`](docs/playbooks/documentation.md) — it has the source-of-truth map and update matrix. For architecture / tech stack / domain-level decisions, write an ADR under [`docs/decisions/`](docs/decisions/) (see the README there for criteria).

**Governance & orchestration**: read [`docs/playbooks/governance-project.md`](docs/playbooks/governance-project.md) for phase gates, ownership, review/QA, workflow skills, `.scratch/`, and docs/ADR rules. Use [`docs/playbooks/agent-orchestration.md`](docs/playbooks/agent-orchestration.md) only as the operational playbook for coordinating agents/tmux/hand-offs; it must comply with governance.

## Key Docs

Read `docs/index.md` first for the intended order. Frequently used:

- `docs/context-map.md` — compact map for code/docs/QA discovery
- `docs/10-technical-architecture.md` — architecture boundaries, JSON-RPC, Electron security
- `docs/11-tech-stack-and-scaffold-decisions.md` — stack decisions with status (decided/recommended/open)
- `docs/12-implementation-patterns.md` — 16 implementation patterns
- `docs/06-data-model.md` + `docs/07-schema-dictionary.md` — SQLite schema, PRAGMAs
- `docs/08-provider-model.md` — provider adapter contract
- `docs/playbooks/governance-project.md` — project governance, phase gates, review/QA rules
- `docs/playbooks/agent-orchestration.md` — operational agent/tmux orchestration playbook
- `docs/playbooks/documentation.md` — keeping docs in sync with code
- `docs/decisions/` — ADR for project technical decisions
