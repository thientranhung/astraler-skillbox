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
và settings.

## 4. User Flows

[04-user-flows.md](04-user-flows.md)

Đọc để hiểu các luồng thao tác chính của user: setup lần đầu, add project, scan,
install skill, fetch update, remove skill, và đổi Skill Host Folder.

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

## Other Docs

[context-map.md](context-map.md)

Map ngắn để định tuyến khi tìm code, docs, contract, và QA. Đọc file này trước
khi search rộng trong repo hoặc khi bắt đầu từ một agent context mới.

[qa/README.md](qa/README.md)

QA bank nằm trong repo: YAML test cases, invariants giữa các màn hình, run
templates, và quy ước evidence/report cho agent-driven Electron smoke và release
QA.

[superpowers/specs/2026-05-26-provider-registry-settings-design.md](../../superpowers/specs/2026-05-26-provider-registry-settings-design.md)

Spec cho Provider Registry Settings: Settings trở thành nguồn sự thật để khai
báo enablement và path candidates của provider built-in cho global lẫn project
skill scopes.

## Archive

Lịch sử review/brainstorm giai đoạn pre-implementation (May 2026). Giữ lại để
trace lý do của các architectural decision; không phải workflow hiện hành.

- [archive/review-prompts/](archive/review-prompts/) — prompt dùng để chạy review chéo data model, provider model, global skills layer, tech stack & scaffold.
- [archive/review-results/](archive/review-results/) — kết quả review + brainstorm (technical architecture, transport decision, tech stack scaffold) đã chốt vào các doc đánh số.

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
- Provider Registry Settings design: [superpowers/specs/2026-05-26-provider-registry-settings-design.md](../../superpowers/specs/2026-05-26-provider-registry-settings-design.md)
