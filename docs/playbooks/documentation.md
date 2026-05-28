# Documentation Playbook

Hướng dẫn AI agent và contributor giữ docs đồng bộ với code. Đọc khi:

- Bạn vừa thêm/xoá concept (table, RPC, screen, domain object) và muốn biết doc nào cần update
- Bạn muốn check gap (drift) giữa code và docs trước khi push
- Bạn muốn quyết định: việc này có cần ADR không

## Source-of-Truth Map

Mỗi concept có **một** nguồn canonical. Khi nguồn đổi, doc tương ứng phải đổi theo.

| Concept | Code source-of-truth | Docs phản ánh |
|---------|---------------------|---------------|
| **Schema** (tables, columns, indexes) | `core-go/migrations/*.up.sql` | `docs/06-data-model.md` + `docs/07-schema-dictionary.md` |
| **RPC methods** (commands + queries) | `shared/api-contracts/methods/*.json` → `shared/generated/methods/*.ts` | `docs/10-technical-architecture.md` (transport + danh sách methods) |
| **JSON-RPC notifications** | `shared/api-contracts/notifications/` | `docs/10-technical-architecture.md` |
| **Domain objects** (Skill, Project, Provider, Plugin, Marketplace, …) | `core-go/internal/domain/*.go` | `docs/02-product-notes.md` + `docs/06-data-model.md` |
| **Provider adapters** | `core-go/internal/providers/` | `docs/08-provider-model.md` |
| **UI screens** | `apps/desktop/renderer/src/screens/` | `docs/03-information-architecture.md` + `docs/09-ui-wireframes.md` |
| **UI features** (cross-screen logic) | `apps/desktop/renderer/src/features/` | `docs/04-user-flows.md` |
| **Edge cases / UX states** | feature code + tests | `docs/05-edge-cases-and-ux-states.md` |
| **Implementation patterns** | code structure | `docs/12-implementation-patterns.md` |

## Update Matrix — Khi Sửa X Phải Update Y

| Thay đổi trong code | Bắt buộc update |
|---------------------|----------------|
| Thêm migration mới (`core-go/migrations/`) | `docs/06-data-model.md`, `docs/07-schema-dictionary.md` |
| Thêm RPC method (file mới ở `shared/api-contracts/methods/`) | `docs/10-technical-architecture.md` (danh sách methods) |
| Thêm domain object mới (`core-go/internal/domain/`) | `docs/02-product-notes.md` (giới thiệu concept) + map ở `docs/06` |
| Thêm provider adapter mới | `docs/08-provider-model.md` |
| Thêm UI screen mới | `docs/03-information-architecture.md` + (nếu có flow) `docs/04-user-flows.md` |
| Đổi architecture boundary | **Tạo ADR** + update `docs/10-technical-architecture.md` |
| Thay tech stack / dependency cốt lõi | **Tạo ADR** + update `docs/11-tech-stack-and-scaffold-decisions.md` |
| Đổi convention / process / workflow | **Tạo ADR** + update playbook tương ứng |

## Gap-Find Procedure (cho AI agent)

Khi muốn audit drift giữa code và docs, làm theo các bước sau. Output là **danh sách concept thiếu**, không phải opinion.

### 1. Inventory code

```sh
# Tables
grep -h "CREATE TABLE" core-go/migrations/*.up.sql | grep -oE "CREATE TABLE[^(]+\(" | sort -u

# RPC methods
ls shared/api-contracts/methods/ | sed 's/\.json$//' | sort -u

# UI screens
ls apps/desktop/renderer/src/screens/*.tsx 2>/dev/null | xargs -n1 basename | sed 's/\.tsx$//' | sort -u

# UI features
ls -d apps/desktop/renderer/src/features/*/ 2>/dev/null | xargs -n1 basename | sort -u

# Domain objects
ls core-go/internal/domain/*.go 2>/dev/null | xargs -n1 basename | sed 's/\.go$//' | sort -u

# Provider adapters
ls core-go/internal/providers/*.go 2>/dev/null | grep -v _test | xargs -n1 basename | sed 's/\.go$//' | sort -u
```

### 2. Inventory docs

Với mỗi concept lấy được ở bước 1, grep trong `docs/` (loại trừ `archive/`):

```sh
grep -rln "<concept>" docs/ --exclude-dir=archive --include="*.md"
```

Nếu output trống → concept thiếu trong docs.

### 3. Output format

```
## Gap Report

### Code → Docs gaps
- Concept: <name>
  - Code source: <path>
  - Docs expected (per map): <doc/section>
  - Status: MISSING | STALE | RENAMED

### Docs → Code gaps (concept ở docs nhưng không tìm thấy code)
- ...
```

### 4. Output handoff

- Nếu chạy thủ công: in báo cáo, gửi cho user/Tom xử lý
- Nếu chạy trong `pre-push` hook: exit 1, in danh sách concept missing, gợi ý doc nào cần update

## Khi Nào Tạo ADR Mới

Tạo ADR khi quyết định thuộc 1 trong các loại sau:

1. **Architecture change** — đổi boundary, đổi pattern cross-layer, đổi IPC contract
2. **Domain change** — thêm/xoá/rename concept cốt lõi
3. **Tech stack change** — đổi runtime, framework, dependency lớn
4. **Process change** — đổi workflow, hook, branch model, review rule

KHÔNG tạo ADR cho refactor cục bộ, bug fix, format change, trivial config.

Quy trình:
1. Copy `docs/decisions/template.md` → `docs/decisions/NNNN-title.md`
2. Status `proposed`
3. User duyệt → đổi thành `accepted`
4. Sau khi implement xong → **digest** nội dung vào `docs/02-product-notes.md` hoặc spec doc liên quan (ADR giữ "why", doc thường giữ "what")
5. Cập nhật `docs/decisions/index.md`

## Khi Nào Bỏ Qua

Nếu bạn chỉ:
- Sửa typo trong doc
- Thêm test (không đổi behavior)
- Refactor nội bộ (interface giữ nguyên)
- Format / lint fix

→ Không cần update doc, không cần ADR. Pre-push hook sẽ tự nhận diện và cho qua (sẽ document khi script được implement).

## References

- [docs/decisions/README.md](../decisions/README.md) — ADR overview
- [docs/decisions/template.md](../decisions/template.md) — ADR template
