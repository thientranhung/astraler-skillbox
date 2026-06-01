# QA Case Schema

Mỗi file dưới `docs/qa/cases/` chứa:

```yaml
area: projects
cases:
  - id: TC-PROJ-001
    title: Add a project and scan detected providers
    tier: T1
    primary_screen: Projects
    related_screens: [Project Detail, Dashboard]
    type: smoke
    tags: [project, scan, cross-screen]
    invariants: [INV-PROJECT-001]
    preconditions:
      - App runs in QA dev Electron mode.
    steps:
      - Open Projects.
    expected_ui:
      - Project appears in the list.
    cross_screen_checks:
      - Dashboard project count matches Projects.
    verifier:
      app_db:
        - Project row exists with the selected path.
      filesystem:
        - No files are written outside the selected project folder.
      evidence:
        - Screenshot after scan.
    safety:
      destructive: false
      allowed_environment: qa_fixture_or_read_only
      real_environment: allowed_read_only
```

## Required Fields

| Field | Purpose |
|---|---|
| `id` | Stable test case id. Dùng `TC-<AREA>-NNN`. |
| `title` | Human-readable behavior đang test. |
| `tier` | `T0`, `T1`, `T2`, hoặc `T3`. |
| `primary_screen` | Screen nơi case bắt đầu. |
| `type` | `critical`, `smoke`, `edge`, `adversarial`, hoặc `packaged-smoke`. |
| `tags` | Filtering keys cho delta QA. |
| `preconditions` | Setup cần thiết trước khi chạy. |
| `steps` | User-visible actions cho executor agent. |
| `expected_ui` | UI phải hiển thị hoặc không hiển thị gì. |
| `verifier` | Independent checks và evidence để collect. |
| `safety` | Destructive/read-only policy. |

## Optional Fields

| Field | Purpose |
|---|---|
| `related_screens` | Screens khác phải reflect cùng truth. |
| `invariants` | References vào `docs/qa/invariants.yaml`. |
| `cross_screen_checks` | UI consistency checks giữa screens. |
| `data_setup` | Fixture data để tạo trước steps. |
| `notes` | Short notes cho known limitations hoặc manual judgment. |

## Result JSONL

Mỗi line trong `runs/<run>/results.jsonl` là kết quả một case:

```json
{"id":"TC-SKILL-003","status":"PASS","tier":"T0","started_at":"2026-06-01T10:00:00+07:00","evidence":["evidence/TC-SKILL-003-after.png","evidence/TC-SKILL-003-fs.txt"],"summary":"Symlink removed and host skill preserved."}
```

Allowed statuses:

- `PASS`
- `FAIL`
- `BLOCKED`
- `NEEDS_HUMAN`
- `SKIPPED`

Dùng `NEEDS_HUMAN` khi next step sẽ touch real plugin/project/provider state mà không có explicit approval.
