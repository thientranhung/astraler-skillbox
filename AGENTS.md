# Repository Guidelines

Canonical contributor & agent guide for Astraler Skillbox. Loaded by all AI providers (Claude Code, Codex, OpenCode, etc.). Keep concise: this file orients work; deep specs live under `docs/`.

## Project Overview

Astraler Skillbox is a local-first desktop app for managing agent skills across projects and providers (Claude, Codex, …). It is the local control center, GUI-first.

Core invariants:
- **Skill Host Folder** is source of truth for skill content.
- **SQLite** is source of truth for management metadata.
- Skills are distributed to projects via symlink or rsync/copy.
- **Skillbox is local-first. Outbound network is OFF by default; the only opt-in network feature is plugin update checks against the user's already-installed plugin source hosts (see ADR-0001).**

## Repo Layout

```
apps/desktop/     # Electron + React renderer + main + preload
core-go/          # Go sidecar (domain, services, repositories, providers, RPC, migrations)
shared/           # JSON Schema contracts + generated TS types (committed)
docs/             # Source of truth — see docs/index.md for reading order
fixtures/         # Test fixtures for provider/filesystem
.scratch/         # Throwaway task briefs (gitignored)
```

Do not create alternate top-level layouts without updating `docs/10-technical-architecture.md`.

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

**Language policy**: Vietnamese is the primary working language for this project; docs and prose may be Vietnamese or English. Do not rewrite existing Vietnamese text unless explicitly requested. Code identifiers, commit messages, and shared API contracts stay in English.

**Commits**: short imperative messages, present tense, no trailing period. Examples: `Add implementation patterns document`, `Fix sort instability in plugins table`.

**File naming**: lowercase kebab-case for Markdown (`13-release-plan.md`). Major source-of-truth docs use numbered prefixes (`NN-name.md`).

**Pull requests**: include summary, affected modules, verification performed, linked issue/decision doc. UI changes → screenshots or recordings. Architecture changes → update the relevant numbered doc and `docs/index.md`.

**Testing**: place tests near the affected layer (Vitest for renderer, `go test` for `core-go`). Contract changes must regenerate `shared/generated` and pass `pnpm check:contracts-drift`.

**Documentation discipline**: when you add/change a concept (schema, RPC method, screen, domain object, provider, etc.), update the corresponding doc in the same slice. Read [`docs/playbooks/documentation.md`](docs/playbooks/documentation.md) — it has the source-of-truth map and update matrix. For architecture / tech stack / domain-level decisions, write an ADR under [`docs/decisions/`](docs/decisions/) (see the README there for criteria).

## Key Docs

Read `docs/index.md` first for the intended order. Frequently used:

- `docs/10-technical-architecture.md` — architecture boundaries, JSON-RPC, Electron security
- `docs/11-tech-stack-and-scaffold-decisions.md` — stack decisions with status (decided/recommended/open)
- `docs/12-implementation-patterns.md` — 16 implementation patterns
- `docs/06-data-model.md` + `docs/07-schema-dictionary.md` — SQLite schema, PRAGMAs
- `docs/08-provider-model.md` — provider adapter contract
- `docs/playbooks/documentation.md` — keeping docs in sync with code
- `docs/decisions/` — ADR for project technical decisions
