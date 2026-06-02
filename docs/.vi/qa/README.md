# QA Bank

Hệ QA nằm trong repo cho Astraler Skillbox. Thiết kế cố ý gọn: test case nằm ở
YAML, kết quả chạy là JSONL append-only, và evidence nằm trong folder của từng
lần chạy.

Với status semantics, evidence standards, waivers, T0 handling, và clean GO
rules, đọc [`governance.md`](governance.md).

## Vì Sao Có QA Bank

Skillbox là app local-first và chạm nhiều vào filesystem. Unit test và contract
test là cần thiết, nhưng chúng không chứng minh được Electron UI có khớp mô tả
sản phẩm hay không. QA bank này giúp agent có checklist bền vững để chạy UI
smoke, kiểm tra logic dữ liệu giữa các màn hình, và bắt lỗi trong môi trường
xấu.

## Bắt Đầu Nhanh

Đọc phần này khi bạn là người mới và chỉ muốn biết cách chạy QA.

### Chạy QA Cho Một Feature

Ví dụ: vừa sửa behavior tắt/bật plugin.

1. Chọn run id:
   `RUN_ID=2026-06-01-plugin-toggle`
2. Tạo run folder:
   ```sh
   mkdir -p "docs/qa/runs/$RUN_ID/evidence"
   cp docs/qa/run-plan-template.yaml "docs/qa/runs/$RUN_ID/run-plan.yaml"
   cp docs/qa/report-template.md "docs/qa/runs/$RUN_ID/report.md"
   touch "docs/qa/runs/$RUN_ID/results.jsonl"
   ```
3. Tìm case liên quan:
   ```sh
   rg -n "plugin|toggle|tier: T0|tier: T1" docs/qa/cases
   ```
4. Sửa `docs/qa/runs/$RUN_ID/run-plan.yaml`:
   - đặt `scope: delta`
   - điền QA paths trong `environment`
   - thêm case id cần chạy vào `selection.case_ids`
5. Start dev Electron app với QA env.
6. Attach agent-browser vào CDP.
7. Chạy các bước trong case như người dùng thật.
8. Lưu screenshot/query output/log vào `evidence/`.
9. Append mỗi case một JSON object vào `results.jsonl`.
10. Cập nhật verdict cuối trong `report.md`.

### Chạy Full QA Baseline Lần Đầu

Dùng trước release đầu tiên hoặc khi muốn kiểm tra chính QA bank có dùng được
không.

1. Tạo run folder với `RUN_ID=YYYY-MM-DD-full-baseline`.
2. Chọn toàn bộ T0/T1:
   ```sh
   rg -n "tier: T0|tier: T1" docs/qa/cases
   ```
3. Chạy T0 trước, sau đó mới chạy T1.
4. Case nào chưa rõ hoặc không an toàn thì đánh `BLOCKED` hoặc `NEEDS_HUMAN`;
   không ép pass.
5. Cập nhật QA bank nếu case thiếu setup, thiếu invariant, hoặc viết chưa đủ rõ.

### Nhờ Agent Chạy QA

Dùng skill `astraler-qa`:

```text
Use astraler-qa to create a delta QA run for plugin toggle changes. Select the
relevant T0/T1 cases, run them against dev Electron with QA fixtures, collect
evidence, and write results.jsonl plus report.md.
```

Agent cần đọc `.agents/skills/astraler-qa/SKILL.md`, README này, các case YAML,
và `docs/playbooks/agent-browser-smoke.md`.

## Cấu Trúc

```text
docs/qa/
  README.md
  governance.md
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

| Tier | Ý nghĩa | Hành động release |
|---|---|---|
| T0 | Data integrity hoặc thao tác destructive. Fail có thể làm mất dữ liệu, ghi nhầm path, hoặc làm DB/filesystem lệch nhau. | Chặn release. |
| T1 | Core user journey. Fail làm hỏng luồng chính hoặc làm sai sự thật giữa các màn hình. | Thường chặn release, trừ khi user chấp nhận workaround. |
| T2 | Workflow phụ. | Ghi nhận và triage. |
| T3 | UX polish hoặc state nhỏ. | Ghi nhận nếu không gây hiểu sai hoặc blocking. |

QA bank hiện tại tập trung vào T0 và T1.

## Khi Nào Chạy

| Thời điểm | Scope |
|---|---|
| Sau khi implement feature | Delta QA: case tag theo feature + T0 bị ảnh hưởng. |
| Trước merge lớn | Smoke QA: T0 core + các T1 flow chính bị chạm. |
| Trước release | Release QA: toàn bộ T0/T1 + packaged launch smoke. |
| Sau khi fix bug | Regression QA: case tái hiện bug + case liên quan. |

## Bắt Đầu Một QA Run

Mỗi lần QA tạo một folder riêng. Folder đó là nguồn sự thật của lần chạy: plan,
result append-only, report cuối, và evidence.

```sh
RUN_ID=2026-06-01-full-baseline
mkdir -p "docs/qa/runs/$RUN_ID/evidence"
cp docs/qa/run-plan-template.yaml "docs/qa/runs/$RUN_ID/run-plan.yaml"
cp docs/qa/report-template.md "docs/qa/runs/$RUN_ID/report.md"
touch "docs/qa/runs/$RUN_ID/results.jsonl"
```

Quy tắc run folder:

- `run-plan.yaml` ghi scope, environment, case được chọn, và gate.
- `results.jsonl` append một JSON object cho mỗi case khi case kết thúc.
- `report.md` là bản tóm tắt cho người đọc: GO / CAUTION / NO-GO.
- `evidence/` chứa screenshot, DB query output, filesystem check, log, và note.
- Không overwrite run folder cũ. Mỗi lần QA tạo một `RUN_ID` mới.

## Electron Automation

QA chính chạy trên dev Electron app với Go sidecar thật:

1. Start `pnpm dev` từ `apps/desktop`.
2. Xác nhận CDP live ở `127.0.0.1:49222`.
3. Attach bằng agent-browser; không launch browser thứ hai.
4. Chạy selected cases và ghi evidence vào active run folder.

Đọc `docs/playbooks/agent-browser-smoke.md` trước khi automation bằng browser.

Packaged app smoke là track riêng. Nó kiểm app boot, sidecar path, DB path, và
smoke nhỏ bằng manual/agent-assisted. Packaged build mặc định không mở CDP.

## Safety Rules

- Không chạy destructive case trên dữ liệu thật trừ khi case ghi rõ
  `real_environment: opt_in` và user đã approve đúng operation đó.
- Plugin case chỉ test scan version, toggle enabled state, và consistency giữa
  các màn hình. Không cài hoặc xóa plugin thật.
- T0 verifier evidence phải có out-of-band checks khi có thể: DB query,
  filesystem state, và screenshot.
- T1 verifier evidence nên có ít nhất screenshot và một independent check khi
  case ảnh hưởng persisted state.
