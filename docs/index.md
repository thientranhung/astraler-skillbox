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
```

## Current Source Of Truth

- Product direction: [01-product-brief.md](01-product-brief.md)
- Decisions and tradeoffs: [02-product-notes.md](02-product-notes.md)
- App structure and core concepts: [03-information-architecture.md](03-information-architecture.md)
- Detailed user flows: [04-user-flows.md](04-user-flows.md)
- Edge cases and UX states: [05-edge-cases-and-ux-states.md](05-edge-cases-and-ux-states.md)
- SQLite metadata model: [06-data-model.md](06-data-model.md)
