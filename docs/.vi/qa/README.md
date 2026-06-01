# QA Bank

Hệ thống QA riêng trong repo cho Astraler Skillbox. Thiết kế ý định rất gọn nhẹ: test cases nằm ở YAML, kết quả chạy là JSONL append-only, và evidence nằm dưới folder run.

## Tại Sao Tồn Tại

Skillbox là local-first và filesystem-heavy. Unit tests và contract tests là cần thiết, nhưng chúng không chứng minh được rằng Electron UI hoạt động như mô tả sản phẩm. QA bank này cung cấp một checklist durable cho agent để chạy UI smoke, cross-screen data consistency, và unsafe environment edges.

## Layout

```text
docs/qa/
  README.md
  schema.md
  invariants.yaml
  cases/
    setup-and-settings.yaml
    skills-and-projects.yaml
    plugins.yaml
  runs/
    <YYYY-MM-DD-release-or-scope>/
      run-plan.yaml
      results.jsonl
      report.md
      evidence/
```

## Risk Tiers

| Tier | Ý Nghĩa | Hành Động Release |
|---|---|---|
| T0 | Data integrity hoặc destructive behavior. Failures có thể mất dữ liệu, ghi vào path sai, hoặc làm DB/filesystem không nhất quán. | Chặn release. |
| T1 | Core user journey. Failures phá vỡ product sử dụng bình thường hoặc cross-screen truth. | Thường chặn release trừ khi user chấp nhận workaround. |
| T2 | Secondary workflow. | Ghi lại và triage. |
| T3 | UX polish hoặc minor state. | Ghi lại nếu không misleading hoặc blocking. |

QA bank hiện tại tập trung vào T0 và T1.

## Khi Nào Chạy

| Thời Điểm | Scope |
|---|---|
| Sau feature implementation | Delta QA: cases được tag với feature cộng impacted T0 cases. |
| Trước một large merge | Smoke QA: T0 core cộng main T1 flows. |
| Trước release | Release QA: tất cả T0/T1 cases cộng packaged launch smoke. |
