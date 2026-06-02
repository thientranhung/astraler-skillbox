# Context Map

Dùng map này trước khi search rộng trong repo. File này chỉ đường tới source of
truth cho các loại task thường gặp; nó không thay thế các docs sâu hơn.

## Application Surfaces

| Nhu cầu | Bắt đầu ở đây |
|---|---|
| Product intent, scope, tradeoffs | `docs/01-product-brief.md`, `docs/02-product-notes.md` |
| Screens, navigation, user-facing concepts | `docs/03-information-architecture.md`, `docs/09-ui-wireframes.md` |
| User flows và edge states | `docs/04-user-flows.md`, `docs/05-edge-cases-and-ux-states.md` |
| Architecture boundaries và protocols | `docs/10-technical-architecture.md` |
| Stack và implementation patterns | `docs/11-tech-stack-and-scaffold-decisions.md`, `docs/12-implementation-patterns.md` |
| Process, review, PR, QA routing | `docs/playbooks/governance-project.md` |
| Documentation source-of-truth map | `docs/playbooks/documentation.md` |
| Agent/tmux handoff operations | `docs/playbooks/agent-orchestration.md` |
| QA policy và run mechanics | `docs/qa/governance.md`, `docs/qa/README.md` |
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

## Contract And Boundary Checks

| Change | Required checks |
|---|---|
| API contract | Update `shared/api-contracts/`, regenerate `shared/generated/`, run `(cd apps/desktop && pnpm check:contracts-drift)`. |
| SQLite schema | Update `core-go/migrations/`, `docs/06-data-model.md`, và `docs/07-schema-dictionary.md`. |
| Renderer behavior | Chỉ dùng preload/core client; không access filesystem, DB, `ipcRenderer`, hoặc provider adapters trực tiếp. |
| Electron main | Chỉ giữ window lifecycle, preload bridge, native dialogs, và Go process lifecycle. |
| Go filesystem writes | Đi qua `filesystem.Gateway`. |
| Provider behavior | Provider adapters chỉ trả facts/capabilities; không write DB/filesystem state. |

## QA Map

| Nhu cầu | Bắt đầu ở đây |
|---|---|
| Chọn case theo feature hoặc risk | `docs/qa/cases/`, `docs/qa/invariants.yaml` |
| Release profile | `docs/qa/profiles/release-full.yaml` |
| Tạo một run | `docs/qa/run-plan-template.yaml`, `docs/qa/report-template.md` |
| Lưu run results | `docs/qa/runs/<run-id>/run-plan.yaml`, `results.jsonl`, `report.md` |
| UI/CDP smoke | `docs/playbooks/agent-browser-smoke.md` |

## Common Discovery Commands

```sh
rg -n "<concept>" docs shared core-go apps/desktop
rg --files apps/desktop/renderer/src/screens apps/desktop/renderer/src/features
rg --files shared/api-contracts shared/generated
rg --files core-go/internal core-go/migrations
rg -n "tier: T0|tier: T1|<feature-tag>" docs/qa/cases docs/qa/invariants.yaml
```
