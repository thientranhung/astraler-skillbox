# QA Governance

Tài liệu này định nghĩa các rule bền vững cho QA của Astraler Skillbox. Đây là
policy document, không phải run log. Ngày cụ thể, PR, case outcome, waiver, và
evidence thuộc về `docs/qa/runs/<run-id>/`.

## Principles

- QA verify product safety, không chỉ verify test execution.
- T0 cases là release-critical vì chúng bao phủ data integrity, filesystem
  safety, destructive behavior, và DB/filesystem consistency.
- Product scope quyết định case có phải blocker không. QA ghi nhận quyết định
  đó; QA không âm thầm định nghĩa lại product.
- Project folders, provider folders, plugin folders, và host folders thật của
  user không bao giờ là destructive QA targets trừ khi case cho phép opt-in rõ
  ràng và owner đã approve đúng target đó.
- Source fixtures là immutable templates. Một run chỉ được mutate run-local
  fixture copies hoặc explicit temporary paths.
- Full QA tạo release confidence; delta QA bảo vệ confidence đó sau mỗi change.

## Result Semantics

Chỉ dùng các status được định nghĩa trong `schema.md`.

| Status | Meaning |
|---|---|
| `PASS` | Expected behavior đã được verify với evidence đủ cho tier của case. |
| `FAIL` | Product vi phạm expected behavior, invariant, hoặc safety rule. |
| `BLOCKED` | Case không thể hoàn tất vì harness, fixture, setup, hoặc product state ngăn execution hợp lệ. |
| `NEEDS_HUMAN` | Bước tiếp theo cần human judgment, credentials, artifact thật, hoặc explicit approval của opt-in target. |
| `SKIPPED` | Case nằm ngoài approved run scope hoặc product phase hiện tại, và reason đã được ghi lại. |

Waiver không phải status riêng. Ghi result status kèm waiver metadata trong
`results.jsonl` và giải thích risk trong `report.md`. Waiver phải có:

- owner approval;
- scope của waiver;
- vì sao residual risk chấp nhận được;
- mitigation, documentation, hoặc follow-up tracking;
- waiver chỉ áp dụng cho current run hay cho documented phase.

## Scope And Phase

Mỗi release run phải tách rõ current release scope và future scope.

- Current-scope T0 failures block release trừ khi được owner waive rõ ràng.
- Future-phase cases nên là `SKIPPED` với phase/defer reason, không tính là
  blocker của current release.
- Manual-only cases có thể là `NEEDS_HUMAN` cho tới khi artifact, credential,
  platform, hoặc human approval cần thiết tồn tại.
- Nếu product scope chưa rõ, dừng lại và clarify source-of-truth docs trước khi
  ép result.

## T0 Handling

T0 cases không được close sớm chỉ vì automation khó. Trước khi một T0 case được
đánh `BLOCKED`, executor phải thử các safe paths có sẵn:

- automated gate hoặc unit/contract test;
- RPC hoặc sidecar harness;
- dev Electron qua CDP;
- out-of-band DB/filesystem inspection;
- source hoặc contract inspection khi UI automation không chạm được state;
- human/owner input khi case yêu cầu rõ.

Nếu case vẫn blocked, report phải nói điều gì ngăn execution và QA bank,
fixture, harness, hoặc product change nào sẽ unblock.

## Evidence Standard

Evidence depth co giãn theo tier và risk.

| Tier | Minimum evidence |
|---|---|
| T0 | Independent evidence như DB query, filesystem state, RPC output, source/contract check, và screenshot khi có UI. |
| T1 | Screenshot hoặc UI snapshot cộng ít nhất một independent persisted-state hoặc RPC check khi state thay đổi. |
| T2/T3 | Screenshot, log, hoặc note ngắn đủ để reproduce observation. |

Evidence files được gom dưới folder `evidence/` của active run. Mặc định chúng
là local run artifacts và bị ignore bởi git để tránh đưa screenshot, log, cache,
và transient output lớn vào source control. `report.md` và `results.jsonl` được
commit phải summarize decisive evidence đủ rõ để reviewer hiểu result. Chỉ
commit raw evidence khi nó nhỏ, được curate có chủ đích, và cần làm durable
review material.

Generated fixture copies, temporary homes, caches, module downloads, và
outside-target sandboxes là disposable run artifacts, không phải canonical
evidence.

## Anti-Hallucination Checklist

Trước khi ra QA result hoặc clean GO verdict, verify:

- Case IDs, tags, tier, và run scope được chọn khớp với change under test.
- Run folder tồn tại và có `run-plan.yaml`, `results.jsonl`, `report.md`, cùng
  evidence paths cần thiết.
- Mỗi `PASS` có đủ evidence cho tier; T0/T1 state changes có independent check
  khi có thể.
- `FAIL`, `BLOCKED`, `NEEDS_HUMAN`, và `SKIPPED` dùng đúng nghĩa status trong
  tài liệu này và `schema.md`.
- Destructive hoặc filesystem-writing cases dùng run-local fixture copies trừ
  khi case cho phép opt-in real targets rõ ràng và owner approve đúng target đó.
- Report tách riêng gate results, case results, waivers, skipped/future scope,
  residual risk, và final GO/NO-GO.
- Claims trong QA verdict được grounding bằng case output, screenshots, logs,
  DB/RPC checks, filesystem checks, source inspection, hoặc contract inspection.

## Full, Delta, And Clean GO

- First release và release-candidate validation dùng profile `release-full`.
- Changes sau một full run dùng delta QA được chọn theo touched screens, tags,
  invariants, và risk.
- Bug fixes phải chọn hoặc thêm regression case trước khi close.
- Release, T0, filesystem, schema/RPC, hoặc cross-layer fix chưa phải clean GO
  cho tới khi delta hoặc release QA bắt buộc pass trên merge commit.
- Final report phải tách gate results, case results, owner waivers, skipped
  future scope, residual risk, và final GO/NO-GO verdict.

## QA Bank Maintenance

Khi execution phát hiện ambiguity, unsafe setup, missing evidence, hoặc missing
coverage, update QA bank thay vì chỉ encode lesson đó trong run report.

Update bề mặt canonical nhỏ nhất:

- `cases/` cho executable behavior và expected results;
- `invariants.yaml` cho safety và consistency rules lặp lại;
- `schema.md` cho result/run-plan metadata;
- `profiles/` cho selection policy;
- file này cho durable QA operating rules.
- `runs/<run-id>/run-plan.yaml`, `results.jsonl`, và `report.md` cho durable run
  summaries khi một run cần được preserve trong git.

Không thêm one-off historical notes vào đây. Run-specific details giữ trong run
folder.
