# Documentation Playbook

Giữ docs đồng bộ với code. Đọc khi:

- Vừa thêm/xoá/rename concept (table, RPC, screen, domain, provider, plugin…)
- Muốn check drift trước khi push
- Cần quyết định: việc này có cần ADR không

## TL;DR

- Mỗi concept có **một** source-of-truth trong code, mỗi SoT ánh xạ tới **một** doc canonical. Xem bảng dưới.
- Đổi concept → update doc trong **cùng slice**, không nợ.
- Pre-push hook chặn push vào `main` nếu không có trailer `DOC-VERIFIED: <reason>` ở commit nào trong push range. Hook không tự check drift — agent phải tự chạy Gap-Find.
- ADR chỉ cho 4 loại quyết định lớn (xem cuối). Refactor cục bộ / typo / test → KHÔNG ADR, vẫn cần trailer.

## Source-of-Truth + Update Map

| Concept | Code SoT | Doc canonical | Trigger update khi… |
|---|---|---|---|
| **Schema** (tables, columns, indexes) | `core-go/migrations/*.up.sql` | `docs/06-data-model.md` + `docs/07-schema-dictionary.md` | thêm migration mới |
| **RPC methods** | `shared/api-contracts/methods/*.json` → `shared/generated/methods/*.ts` | `docs/10-technical-architecture.md` (transport + danh sách methods) | thêm/đổi/xoá method file |
| **JSON-RPC notifications** | `shared/api-contracts/notifications/` | `docs/10-technical-architecture.md` | thêm/đổi notification |
| **Domain objects** | `core-go/internal/domain/*.go` | `docs/02-product-notes.md` (intro) + `docs/06-data-model.md` (map) | thêm/rename domain object |
| **Provider adapters** | `core-go/internal/providers/` | `docs/08-provider-model.md` | thêm provider mới |
| **UI screens** | `apps/desktop/renderer/src/screens/` | `docs/03-information-architecture.md` + `docs/09-ui-wireframes.md` | thêm screen mới |
| **UI features** (cross-screen logic) | `apps/desktop/renderer/src/features/` | `docs/04-user-flows.md` | flow user-facing thay đổi |
| **Edge cases / UX states** | feature code + tests | `docs/05-edge-cases-and-ux-states.md` | thêm trạng thái lỗi/empty/conflict mới |
| **Implementation patterns** | code structure | `docs/12-implementation-patterns.md` | pattern mới được áp ≥2 nơi |
| **Architecture boundary** | code + ADR | **ADR** + `docs/10-technical-architecture.md` | đổi boundary, contract, IPC |
| **Tech stack / dep cốt lõi** | `package.json`, `go.mod`, scaffold | **ADR** + `docs/11-tech-stack-and-scaffold-decisions.md` | đổi runtime/framework lớn |
| **Process / workflow / hook** | scripts, hooks, playbook | **ADR** + playbook tương ứng | đổi review rule, branch model, hook |

## Gap-Find Procedure

Khi audit drift giữa code và docs. Output là **danh sách concept thiếu**, không phải opinion.

### 1. Inventory code

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

### 2. Inventory docs

Mỗi concept ở bước 1 → grep trong `docs/` (loại trừ `archive/`):

```sh
grep -rln "<concept>" docs/ --exclude-dir=archive --include="*.md"
```

Output trống → concept thiếu trong docs.

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

### 4. Handoff

In báo cáo. Có gap → tự update docs trước khi push, hoặc giải thích skip rõ ràng.

## Pre-Push Gate (DOC-VERIFIED Trace)

Hook **không tự chạy logic check**. Vai trò:

1. **Reminder**: in checklist nhắc verify docs trước push.
2. **Trace check**: tìm trailer `DOC-VERIFIED: <reason>` trong push range.

Cơ chế:

- Chỉ enforce khi push vào `main`. Push feature branch bỏ qua — gate kích hoạt khi work landing.
- Mỗi `git push` (vào `main`), hook quét commit message của push range.
- **Bất kỳ** commit nào có trailer ở cuối message → cho phép push.
- Không commit nào có trailer → block với checklist (đọc playbook này → chạy Gap-Find → update docs hoặc add trailer giải thích).

Trailer format (chuẩn git trailer, cuối message, có dòng trống phía trên):

```
Add foo bar baz feature

Description body...

DOC-VERIFIED: docs/06 + docs/08 updated for new plugin layer concept
Co-Authored-By: ...
```

Bypass khẩn cấp: `git push --no-verify` (luôn được; để lại vết trong reflog).

**Vì sao thiết kế thế này**: mechanical grep không phân biệt rename vs concept mới, không hiểu junction table không cần doc riêng, không suggest section phù hợp. AI agent với playbook trong tay làm tốt hơn — hook chỉ ép agent phải làm bước đó.

## ADR

### Khi nào tạo ADR

Tạo khi quyết định thuộc 1 trong 4 loại:

1. **Architecture change** — đổi boundary, pattern cross-layer, IPC contract.
2. **Domain change** — thêm/xoá/rename concept cốt lõi.
3. **Tech stack change** — đổi runtime, framework, dependency lớn.
4. **Process change** — đổi workflow, hook, branch model, review rule.

**KHÔNG ADR** cho: refactor cục bộ, bug fix, format change, trivial config.

### Quy trình

1. Copy `docs/decisions/template.md` → `docs/decisions/NNNN-title.md`.
2. Status `proposed`.
3. User duyệt → đổi `accepted`.
4. Sau khi implement xong → **digest** nội dung vào `docs/02-product-notes.md` hoặc spec liên quan. ADR giữ "why", doc thường giữ "what".
5. Update `docs/decisions/index.md`.

## Khi nào skip

Chỉ sửa typo doc / thêm test (không đổi behavior) / refactor nội bộ (interface giữ nguyên) / format & lint → không update doc, không ADR. **Vẫn cần** trailer `DOC-VERIFIED: <lý do>` ở commit message (vd `DOC-VERIFIED: refactor only, no concept changes`).

## References

- [docs/decisions/README.md](../decisions/README.md) — ADR overview
- [docs/decisions/template.md](../decisions/template.md) — ADR template
- [docs/playbooks/agent-orchestration.md](agent-orchestration.md) — vai trò Tom/Larry/Orchestrator, phase gates
