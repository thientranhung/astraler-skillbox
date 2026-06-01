# Documentation Playbook

Keep docs synchronized with code. Read this when:

- You just added, removed, or renamed a concept (table, RPC, screen, domain, provider, plugin, etc.).
- You want to check docs drift before pushing.
- You need to decide whether this work requires an ADR.

## TL;DR

- Each concept has **one** source of truth in code, and each source of truth maps to **one** canonical doc. See the table below.
- Concept change -> update docs in the **same slice**. Do not defer.
- The pre-push hook blocks pushes to `main` if no commit in the push range has a `DOC-VERIFIED: <reason>` trailer. The hook does not check drift itself; the agent must run Gap-Find.
- ADRs are only for four major decision types (see below). Local refactor / typo / test -> no ADR, but still needs the trailer.

## Source-of-Truth + Update Map

| Concept | Code SoT | Canonical doc | Update trigger |
|---|---|---|---|
| **Schema** (tables, columns, indexes) | `core-go/migrations/*.up.sql` | `docs/06-data-model.md` + `docs/07-schema-dictionary.md` | new migration |
| **RPC methods** | `shared/api-contracts/methods/*.json` -> `shared/generated/methods/*.ts` | `docs/10-technical-architecture.md` (transport + method list) | method file added/changed/deleted |
| **JSON-RPC notifications** | `shared/api-contracts/notifications/` | `docs/10-technical-architecture.md` | notification added/changed |
| **Domain objects** | `core-go/internal/domain/*.go` | `docs/02-product-notes.md` (intro) + `docs/06-data-model.md` (map) | domain object added/renamed |
| **Provider adapters** | `core-go/internal/providers/` | `docs/08-provider-model.md` | new provider added |
| **UI screens** | `apps/desktop/renderer/src/screens/` | `docs/03-information-architecture.md` + `docs/09-ui-wireframes.md` | new screen added |
| **UI features** (cross-screen logic) | `apps/desktop/renderer/src/features/` | `docs/04-user-flows.md` | user-facing flow changes |
| **Edge cases / UX states** | feature code + tests | `docs/05-edge-cases-and-ux-states.md` | new error/empty/conflict state added |
| **Implementation patterns** | code structure | `docs/12-implementation-patterns.md` | pattern is applied in >=2 places |
| **Architecture boundary** | code + ADR | **ADR** + `docs/10-technical-architecture.md` | boundary, contract, or IPC changes |
| **Tech stack / core dependency** | `package.json`, `go.mod`, scaffold | **ADR** + `docs/11-tech-stack-and-scaffold-decisions.md` | major runtime/framework changes |
| **Process / workflow / hook** | scripts, hooks, playbook | **ADR** + relevant playbook | review rule, branch model, or hook changes |

## Gap-Find Procedure

Use this when auditing drift between code and docs. The output is a **list of missing concepts**, not an opinion.

### 1. Inventory Code

```sh
# Tables
grep -h "CREATE TABLE" core-go/migrations/*.up.sql \
  | grep -oE "CREATE TABLE[^(]+\(" | sort -u

# RPC methods
ls shared/api-contracts/methods/ | sed 's/\.json$//' | sort -u

# UI screens
ls apps/desktop/renderer/src/screens/*.tsx 2>/dev/null \
  | xargs -n1 basename | sed 's/\.tsx$//' | sort -u

# UI features
ls -d apps/desktop/renderer/src/features/*/ 2>/dev/null \
  | xargs -n1 basename | sort -u

# Domain objects
ls core-go/internal/domain/*.go 2>/dev/null \
  | xargs -n1 basename | sed 's/\.go$//' | sort -u

# Provider adapters
ls core-go/internal/providers/*.go 2>/dev/null \
  | grep -v _test | xargs -n1 basename | sed 's/\.go$//' | sort -u
```

### 2. Inventory Docs

For each concept from step 1, grep under `docs/` (excluding `archive/`):

```sh
grep -rln "<concept>" docs/ --exclude-dir=archive --include="*.md"
```

Empty output means the concept is missing from docs.

### 3. Output Format

```text
## Gap Report

### Code -> Docs gaps
- Concept: <name>
  - Code source: <path>
  - Docs expected (per map): <doc/section>
  - Status: MISSING | STALE | RENAMED

### Docs -> Code gaps (concept exists in docs but not code)
- ...
```

### 4. Handoff

Print the report. If there are gaps, update docs before pushing or explain the skip clearly.

## Pre-Push Gate (DOC-VERIFIED Trace)

The hook **does not run logic checks itself**. Its roles:

1. **Reminder**: print the checklist to verify docs before push.
2. **Trace check**: find a `DOC-VERIFIED: <reason>` trailer in the push range.

Mechanics:

- Enforced only when pushing to `main`. Feature branch pushes are skipped; the gate activates when work lands.
- On each `git push` to `main`, the hook scans commit messages in the push range.
- **Any** commit with the trailer at the end of the message allows the push.
- If no commit has the trailer, the hook blocks with a checklist (read this playbook -> run Gap-Find -> update docs or add a trailer explaining why not).

Trailer format (standard git trailer, at the end of the message, with a blank line above):

```text
Add foo bar baz feature

Description body...

DOC-VERIFIED: docs/06 + docs/08 updated for new plugin layer concept
Co-Authored-By: ...
```

Emergency bypass: `git push --no-verify` (always allowed; leaves a reflog trace).

**Why this design:** mechanical grep cannot distinguish rename vs new concept, cannot understand when a junction table does not need its own docs, and cannot suggest the right section. An AI agent with this playbook can do that better. The hook only forces the agent to perform the step.

## ADR

### When To Create An ADR

Create an ADR when the decision falls into one of these four types:

1. **Architecture change**: boundary change, cross-layer pattern, IPC contract.
2. **Domain change**: add/remove/rename a core concept.
3. **Tech stack change**: major runtime, framework, or dependency change.
4. **Process change**: workflow, hook, branch model, or review rule change.

**Do not create ADRs** for local refactors, bug fixes, formatting changes, or trivial config.

### Procedure

1. Copy `docs/decisions/template.md` -> `docs/decisions/NNNN-title.md`.
2. Set status to `proposed`.
3. User approval -> change status to `accepted`.
4. After implementation -> **digest** the decision into `docs/02-product-notes.md` or the relevant spec. ADRs keep "why"; normal docs keep "what".
5. Update `docs/decisions/index.md`.

## When To Skip

Doc typo only / add test without behavior change / internal refactor with unchanged interface / format & lint -> no doc update and no ADR. **Still include** a `DOC-VERIFIED: <reason>` trailer in the commit message (for example `DOC-VERIFIED: refactor only, no concept changes`).

## References

- [docs/decisions/README.md](../decisions/README.md) - ADR overview
- [docs/decisions/template.md](../decisions/template.md) - ADR template
- [docs/playbooks/agent-orchestration.md](agent-orchestration.md) - Tom/Larry/Orchestrator roles and phase gates
