# Context Map

Use this map before broad repository discovery. It points to the source of truth
for common agent tasks; it does not replace the deeper docs.

## Application Surfaces

| Need | Start here |
|---|---|
| Product intent, scope, tradeoffs | `docs/01-product-brief.md`, `docs/02-product-notes.md` |
| Screens, navigation, user-facing concepts | `docs/03-information-architecture.md`, `docs/09-ui-wireframes.md` |
| User flows and edge states | `docs/04-user-flows.md`, `docs/05-edge-cases-and-ux-states.md` |
| Architecture boundaries and protocols | `docs/10-technical-architecture.md` |
| Stack and implementation patterns | `docs/11-tech-stack-and-scaffold-decisions.md`, `docs/12-implementation-patterns.md` |
| Process, review, PR, QA routing | `docs/playbooks/governance-project.md` |
| Documentation source-of-truth map | `docs/playbooks/documentation.md` |
| Agent/tmux handoff operations | `docs/playbooks/agent-orchestration.md` |
| tmux helper scripts | `scripts/harness/agent-send.sh`, `scripts/harness/agent-status.sh` |
| QA policy, taxonomy, and run mechanics | `docs/qa/governance.md`, `docs/qa/screen-taxonomy.md`, `docs/qa/README.md` |
| Architecture decisions | `docs/decisions/index.md` |

## Code Map

| Area | Paths |
|---|---|
| Electron app | `apps/desktop/` |
| Renderer screens | `apps/desktop/renderer/src/screens/` |
| Renderer features | `apps/desktop/renderer/src/features/` |
| Renderer core bridge client | `apps/desktop/renderer/src/lib/core-client/` |
| Electron main / preload | `apps/desktop/electron/main/`, `apps/desktop/electron/preload/` |
| Shared API contracts | `shared/api-contracts/` |
| Generated TypeScript contracts | `shared/generated/` |
| Go sidecar | `core-go/` |
| Go domain objects | `core-go/internal/domain/` |
| Go services / operations | `core-go/internal/services/`, `core-go/internal/operations/` |
| Go repositories / direct SQL | `core-go/internal/repositories/` |
| Go filesystem gateway | `core-go/internal/filesystem/` |
| Go provider adapters | `core-go/internal/providers/` |
| Go RPC transport | `core-go/internal/rpc/` |
| SQLite migrations | `core-go/migrations/` |
| Provider/filesystem fixtures | `fixtures/` |
| Agent harness scripts | `scripts/harness/` |

## Contract And Boundary Checks

| Change | Required checks |
|---|---|
| API contract | Update `shared/api-contracts/`, regenerate `shared/generated/`, run `(cd apps/desktop && pnpm check:contracts-drift)`. |
| SQLite schema | Update `core-go/migrations/`, `docs/06-data-model.md`, and `docs/07-schema-dictionary.md`. |
| Renderer behavior | Use preload/core client only; do not access filesystem, DB, `ipcRenderer`, or provider adapters directly. |
| Electron main | Keep to window lifecycle, preload bridge, native dialogs, and Go process lifecycle. |
| Go filesystem writes | Route through `filesystem.Gateway`. |
| Provider behavior | Provider adapters return facts/capabilities only; they do not write DB/filesystem state. |

## QA Map

| Need | Start here |
|---|---|
| Select cases by feature or risk | `docs/qa/cases/`, `docs/qa/invariants.yaml` |
| Check screen labels, routes, and QA tags | `docs/qa/screen-taxonomy.md` |
| Release profile | `docs/qa/profiles/release-full.yaml` |
| Create a run | `docs/qa/run-plan-template.yaml`, `docs/qa/report-template.md` |
| Preserve run results | `docs/qa/runs/<run-id>/run-plan.yaml`, `results.jsonl`, `report.md` |
| UI/CDP smoke | `docs/playbooks/agent-browser-smoke.md` |

## Common Discovery Commands

```sh
rg -n "<concept>" docs shared core-go apps/desktop
rg --files apps/desktop/renderer/src/screens apps/desktop/renderer/src/features
rg --files shared/api-contracts shared/generated
rg --files core-go/internal core-go/migrations
rg -n "tier: T0|tier: T1|<feature-tag>" docs/qa/cases docs/qa/invariants.yaml
```
