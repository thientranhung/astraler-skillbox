# Astraler Skillbox Docs Index

Đọc tài liệu theo thứ tự này để nắm dự án từ product direction đến cấu trúc app.

## 1. Product Brief

[01-product-brief.md](01-product-brief.md)

Đọc trước để hiểu vấn đề, định vị product, người dùng mục tiêu, pain points, và
các quyết định thiết kế đã chốt.

## 2. Product Notes

[02-product-notes.md](02-product-notes.md)

Đọc sau Product Brief để nắm product thesis, scope hiện tại, tradeoffs, update
model, và các decision quan trọng.

## 3. Information Architecture

[03-information-architecture.md](03-information-architecture.md)

Đọc để hiểu các core concepts, màn hình chính trong app, flow add skill, update,
sync, và settings.

## 4. User Flows

[04-user-flows.md](04-user-flows.md)

Đọc để hiểu các luồng thao tác chính của user: setup lần đầu, add project, scan,
install skill, fetch update, sync, switch mode, remove skill, và đổi Skill Host
Folder.

## 5. Edge Cases And UX States

[05-edge-cases-and-ux-states.md](05-edge-cases-and-ux-states.md)

Đọc để hiểu các trạng thái lỗi, warning, empty state, conflict, fetch/update
failure, provider mismatch, và cách UI nên phản hồi.

## 6. Data Model

[06-data-model.md](06-data-model.md)

Đọc để hiểu các entity SQLite cấp cao, relationship, status enum, và mapping từ
user flows/edge cases sang metadata mà app cần lưu.

## 7. Schema Dictionary

[07-schema-dictionary.md](07-schema-dictionary.md)

Đọc để hiểu chi tiết từng table/field: type dự kiến, nullable, enum, và ý nghĩa
nghiệp vụ của từng cột.

## 8. Provider Model

[08-provider-model.md](08-provider-model.md)

Đọc để hiểu provider adapter, path candidates, provider detection, install target
resolution, provider UI state, và hướng Phase 2 conversion.

## 9. UI Wireframes

[09-ui-wireframes.md](09-ui-wireframes.md)

Đọc để hiểu text wireframes cho Dashboard, Skills Library, Projects, Project
Detail, Add Skill flow, Updates, Settings, empty states, warnings, confirmations,
và impact previews.

## 10. Technical Architecture

[10-technical-architecture.md](10-technical-architecture.md)

Đọc để hiểu architecture boundary giữa UI, application services, domain logic,
SQLite repositories, filesystem gateway, provider adapters, source integrations,
operation runner, và testing strategy.

## 11. Tech Stack And Scaffold Decisions

[11-tech-stack-and-scaffold-decisions.md](11-tech-stack-and-scaffold-decisions.md)

Đọc để hiểu các quyết định stack/scaffold trước khi tạo codebase thật: Electron,
React, Go, Vite, UI kit, router, query, forms, tables, JSON-RPC, SQLite,
keychain, testing, packaging, và các GAP còn cần chốt.

## 12. Implementation Patterns

[12-implementation-patterns.md](12-implementation-patterns.md)

Đọc để hiểu các pattern sẽ dùng khi implement code: Process Coordinator, preload
bridge, JSON-RPC boundary, CQRS, services, repositories, filesystem gateway,
provider/source adapters, operation runner, manual DI, view models, UI
composition, validation, errors, và testing.

## Review Prompts

[review-prompts/data-model-review.md](review-prompts/data-model-review.md)

Prompt dùng để nhờ agent/chuyên gia khác review chéo data model. Kết quả review
nên được ghi vào `docs/review-results/data-model-review.md`.

[review-prompts/provider-model-review.md](review-prompts/provider-model-review.md)

Prompt dùng để nhờ agent/chuyên gia khác review chéo provider model. Kết quả
review nên được ghi vào `docs/review-results/provider-model-review.md`.

[review-prompts/global-skills-layer-review.md](review-prompts/global-skills-layer-review.md)

Prompt dùng để nhờ agent/chuyên gia khác review riêng Global Skills layer. Kết
quả review nên được ghi vào `docs/review-results/global-skills-layer-review.md`.

Follow-up review sau khi xử lý blocker nên được ghi vào
`docs/review-results/global-skills-layer-followup-review.md`.

[review-results/technical-architecture-brainstorm.md](review-results/technical-architecture-brainstorm.md)

Brainstorm kỹ thuật với Agent Tech cho architecture decisions sau khi đã chốt
Electron + React + Golang.

[review-results/transport-decision-brainstorm.md](review-results/transport-decision-brainstorm.md)

Brainstorm riêng cho transport decision giữa Electron main process và Golang
core. Kết luận: Phase 1 dùng stdio JSON-RPC 2.0.

[review-prompts/tech-stack-scaffold-review.md](review-prompts/tech-stack-scaffold-review.md)

Prompt dùng để nhờ agent/chuyên gia khác review riêng Tech Stack And Scaffold
Decisions. Kết quả review nên được ghi vào
`docs/review-results/tech-stack-scaffold-review.md`.

[review-results/tech-stack-scaffold-review.md](review-results/tech-stack-scaffold-review.md)

Kết quả review Tech Stack And Scaffold Decisions. Review này chốt các thay đổi
cần làm trước scaffold như SQLite PRAGMAs, JSON-RPC framing/library, startup
timeout, Electron security defaults, app data path, và dependency deferral.

[agent-orchestration-playbook.md](agent-orchestration-playbook.md)

Playbook hardening cho điều phối `agent-tech-skillbox` và
`agent-lead-skillbox`: tmux hygiene, prompt dài qua file, phase gates, review
loop, và recovery khi TUI input bị kẹt hoặc trôi sang shell.

[superpowers/specs/2026-05-26-provider-registry-settings-design.md](superpowers/specs/2026-05-26-provider-registry-settings-design.md)

Spec cho Provider Registry Settings: Settings trở thành nguồn sự thật để khai
báo built-in/custom provider, icon, enablement, và path candidates cho global
lẫn project skill scopes.

## Suggested Reading Flow

```text
README.md
  -> docs/index.md
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
- Data model review prompt: [review-prompts/data-model-review.md](review-prompts/data-model-review.md)
- Provider model review prompt: [review-prompts/provider-model-review.md](review-prompts/provider-model-review.md)
- Global Skills layer review prompt: [review-prompts/global-skills-layer-review.md](review-prompts/global-skills-layer-review.md)
- Tech stack scaffold review prompt: [review-prompts/tech-stack-scaffold-review.md](review-prompts/tech-stack-scaffold-review.md)
- Global Skills layer follow-up review result: [review-results/global-skills-layer-followup-review.md](review-results/global-skills-layer-followup-review.md)
- Technical architecture brainstorm result: [review-results/technical-architecture-brainstorm.md](review-results/technical-architecture-brainstorm.md)
- Transport decision brainstorm result: [review-results/transport-decision-brainstorm.md](review-results/transport-decision-brainstorm.md)
- Tech stack scaffold review result: [review-results/tech-stack-scaffold-review.md](review-results/tech-stack-scaffold-review.md)
- Agent orchestration playbook: [agent-orchestration-playbook.md](agent-orchestration-playbook.md)
- Provider Registry Settings design: [superpowers/specs/2026-05-26-provider-registry-settings-design.md](superpowers/specs/2026-05-26-provider-registry-settings-design.md)
