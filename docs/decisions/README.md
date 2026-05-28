# Architecture Decision Records

ADR ghi lại quyết định **kỹ thuật / sản phẩm** của Skillbox: tại sao chọn cách này, đã cân nhắc gì, hệ quả ra sao. Đây là nơi AI agent và contributor sau này tra cứu "tại sao" khi đọc code.

## Scope: chỉ quyết định DỰ ÁN

ADR là cho quyết định về **bản thân Skillbox** — code, kiến trúc, sản phẩm. Không phải cho quy ước làm việc giữa user và AI (cái đó nằm ở `docs/playbooks/`).

## Khi nào tạo ADR mới

Tạo ADR khi:

- Thay đổi architecture boundary (cross-layer pattern, IPC contract, data flow)
- Thêm/xoá/đổi domain concept lớn (Skill, Project, Plugin, Marketplace, …)
- Đổi tech stack hoặc dependency cốt lõi
- Quyết định "1 chiều" — khó đảo ngược sau này

KHÔNG tạo ADR cho:

- Refactor code thông thường
- Bug fix cụ thể
- Style/format change
- Trivial config tweak
- **Quy ước workflow / playbook / process làm việc** → vào `docs/playbooks/`

## Vòng đời ADR

```
proposed  →  accepted  →  (superseded by ADR-XXXX)
             ↓
         (digest vào docs/02-product-notes.md hoặc spec doc tương ứng)
```

- **proposed**: vừa viết, chưa duyệt
- **accepted**: user đã duyệt, là quyết định hiện hành
- **superseded**: bị ADR mới thay thế — link tới ADR thay thế ở header

Sau khi `accepted`, **digest** quyết định vào tài liệu thường (product-notes, technical-architecture, …) để doc thường giữ "what we ended up with", ADR giữ "why".

## Naming

`NNNN-kebab-title.md` với `NNNN` là số 4 chữ số zero-pad tăng dần (`0001`, `0002`, …).

## Index

Xem [index.md](./index.md) — danh sách ADR + status hiện tại.

## Template

Xem [template.md](./template.md).
