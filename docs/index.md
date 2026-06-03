# Astraler Skillbox Docs Index

Read these documents in order to understand the project from product direction
to app structure.

## 1. Product Brief

[01-product-brief.md](01-product-brief.md)

Read first to understand the problem, product positioning, target users, pain
points, and confirmed design decisions.

## 2. Product Notes

[02-product-notes.md](02-product-notes.md)

Read after Product Brief to understand the product thesis, current scope,
tradeoffs, update model, and key decisions.

## 3. Information Architecture

[03-information-architecture.md](03-information-architecture.md)

Read to understand the core concepts, main app screens, add skill flow, update,
and settings.

## 4. User Flows

[04-user-flows.md](04-user-flows.md)

Read to understand the main user flows: first-time setup, add project, scan,
install skill, fetch update, remove skill, and change Skill Host Folder.

## 5. Edge Cases And UX States

[05-edge-cases-and-ux-states.md](05-edge-cases-and-ux-states.md)

Read to understand error states, warnings, empty states, conflicts,
fetch/update failures, provider mismatches, and how the UI should respond.

## 6. Data Model

[06-data-model.md](06-data-model.md)

Read to understand the high-level SQLite entities, relationships, status enums,
and mapping from user flows/edge cases to metadata the app needs to store.

## 7. Schema Dictionary

[07-schema-dictionary.md](07-schema-dictionary.md)

Read to understand each table/field in detail: expected type, nullable, enum,
and the business meaning of each column.

## 8. Provider Model

[08-provider-model.md](08-provider-model.md)

Read to understand the provider adapter, path candidates, provider detection,
install target resolution, and provider UI state.

## 9. UI Wireframes

[09-ui-wireframes.md](09-ui-wireframes.md)

Read to understand text wireframes for Dashboard, Skills Library, Projects,
Project Detail, Add Skill flow, Updates, Settings, empty states, warnings,
confirmations, and impact previews.

## 10. Technical Architecture

[10-technical-architecture.md](10-technical-architecture.md)

Read to understand architecture boundaries between UI, application services,
domain logic, SQLite repositories, the filesystem gateway, provider adapters,
source integrations, the operation runner, and testing strategy.

## 11. Tech Stack And Scaffold Decisions

[11-tech-stack-and-scaffold-decisions.md](11-tech-stack-and-scaffold-decisions.md)

Read to understand stack/scaffold decisions before creating the real codebase:
Electron, React, Go, Vite, UI kit, router, query, forms, tables, JSON-RPC,
SQLite, keychain, testing, packaging, and open gaps still to confirm.

## 12. Implementation Patterns

[12-implementation-patterns.md](12-implementation-patterns.md)

Read to understand the patterns used when implementing code: Process
Coordinator, preload bridge, JSON-RPC boundary, CQRS, services, repositories,
filesystem gateway, provider/source adapters, operation runner, manual DI, view
models, UI composition, validation, errors, and testing.

## Other Docs

[context-map.md](context-map.md)

Compact routing map for code, docs, contracts, and QA discovery. Read this
before a broad repository search or when starting in a fresh agent context.

[qa/README.md](qa/README.md)

Repo-native QA bank: YAML test cases, cross-screen invariants, run templates,
and evidence/report conventions for agent-driven Electron smoke and release QA.

[superpowers/specs/2026-05-26-provider-registry-settings-design.md](superpowers/specs/2026-05-26-provider-registry-settings-design.md)

Spec for Provider Registry Settings: Settings becomes the source of truth for
declaring built-in provider enablement and path candidates for both global and
project skill scopes.

## Archive

Review/brainstorm history from the pre-implementation phase (May 2026). Kept to
trace the reasoning behind architectural decisions; not current workflow.

- [archive/review-prompts/](archive/review-prompts/) — prompts used to run cross-reviews of data model, provider model, global skills layer, tech stack & scaffold.
- [archive/review-results/](archive/review-results/) — review + brainstorm results (technical architecture, transport decision, tech stack scaffold) incorporated into the numbered docs.

## Suggested Reading Flow

```text
README.md
  -> docs/index.md
  -> docs/context-map.md
  -> docs/01-product-brief.md
  -> docs/02-product-notes.md
  -> docs/03-information-architecture.md
  -> docs/04-user-flows.md
  -> docs/05-edge-cases-and-ux-states.md
  -> docs/06-data-model.md
  -> docs/07-schema-dictionary.md
  -> docs/08-provider-model.md
  -> docs/09-ui-wireframes.md
  -> docs/10-technical-architecture.md
  -> docs/11-tech-stack-and-scaffold-decisions.md
  -> docs/12-implementation-patterns.md
```

## Current Source Of Truth

- Product direction: [01-product-brief.md](01-product-brief.md)
- Code/docs/QA discovery: [context-map.md](context-map.md)
- Decisions and tradeoffs: [02-product-notes.md](02-product-notes.md)
- App structure and core concepts: [03-information-architecture.md](03-information-architecture.md)
- Detailed user flows: [04-user-flows.md](04-user-flows.md)
- Edge cases and UX states: [05-edge-cases-and-ux-states.md](05-edge-cases-and-ux-states.md)
- SQLite metadata model: [06-data-model.md](06-data-model.md)
- Schema dictionary: [07-schema-dictionary.md](07-schema-dictionary.md)
- Provider model: [08-provider-model.md](08-provider-model.md)
- UI wireframes: [09-ui-wireframes.md](09-ui-wireframes.md)
- Technical architecture: [10-technical-architecture.md](10-technical-architecture.md)
- Tech stack and scaffold decisions: [11-tech-stack-and-scaffold-decisions.md](11-tech-stack-and-scaffold-decisions.md)
- Implementation patterns: [12-implementation-patterns.md](12-implementation-patterns.md)
- Provider Registry Settings design: [superpowers/specs/2026-05-26-provider-registry-settings-design.md](superpowers/specs/2026-05-26-provider-registry-settings-design.md)
