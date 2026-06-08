# QA Strategy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (khuyến nghị) hoặc superpowers:executing-plans để thực thi plan này theo từng task. Các step dùng checkbox (`- [ ]`) để track.

**Goal:** Dựng bộ QA tầng-2 (validation/acceptance) cho Skillbox trước release — AI agent đóng vai người dùng + kẻ-phá, chạy thật trên sandbox cô lập, verdict neo vào post-condition kiểm-được-bằng-máy, không tin narration.

**Architecture:** Ba loại slice xếp tuần tự theo dependency: (A) **ARTIFACT** — viết tài liệu sống `docs/playbooks/qa.md` + `docs/qa/test-catalog.md`; (B) **SETUP/INFRA** — dựng sandbox cô lập tuyệt đối + chứng minh `~/.claude` thật không bị chạm; (C) **EXECUTE QA** — chạy risk-first T0→T1→T2→T3, rồi Track-2 smoke trên `.app`, rồi completeness pass, mỗi case tách 2 vai Executor/Verifier. Nguồn kỳ vọng: `docs/04-user-flows.md`, `docs/05-edge-cases-and-ux-states.md`, `docs/03-information-architecture.md`. Spec gốc: `docs/superpowers/specs/2026-05-30-qa-strategy-design.md`.

**Tech Stack:** Electron + React renderer (`apps/desktop`), Go core (`core-go`, JSON-RPC qua stdio), SQLite (`qa.db`), agent-browser (CDP) để lái app, `sqlite3` CLI + `readlink`/`stat`/`ls` để verify out-of-band.

---

## Capability / Ownership Map (đọc trước)

Plan này dùng từ vựng capability của `docs/playbooks/agent-orchestration.md`. Mapping đề xuất (user có thể chỉnh):

| Capability | Ai (mặc định) | Dùng ở slice |
|---|---|---|
| **dev** (viết doc, viết script, fix) | **Tom** (`agent-tech-skillbox`) | Slice 0, 1, 2; fix bug spawn ra |
| **reviewer / spec-design review** | **Larry** (`agent-lead-skillbox`) — không edit file | Gate review của Slice 0, 1, 2 |
| **Executor** (đóng vai user + kẻ-phá, lái agent-browser, **không tự tuyên bố PASS**) | agent QA-executor riêng (frontier model) — có thể là Tom ở "chế độ executor" hoặc subagent chuyên | Slice 3–6 |
| **Verifier** (độc lập, **không thấy context Executor**, tự query DB/fs → PASS/FAIL/BLOCKED) | **Larry** / reviewer-capability (frontier model) | Slice 3–6 |

Ràng buộc cứng từ spec §4: Executor và Verifier là **hai agent tách biệt, không chia sẻ context**; **cả hai dùng model frontier** (không dùng model rẻ cho QA). Screenshot = evidence kèm theo, **không** phải verdict.

> **Điểm user cần xác nhận khi duyệt:** mapping Executor↔Tom-executor và Verifier↔Larry ở trên là *đề xuất* khớp role hiện có (Larry vốn là "Reviewer & QA, smoke, không edit"). Nếu user muốn Executor là một agent thứ ba độc lập hẳn (không phải Tom), nói rõ khi duyệt — không phải blocker để viết plan.

## Sequencing & Dependency (tổng quan)

```text
Slice 0 (ARTIFACT: qa.md playbook)
  └─> Slice 1 (ARTIFACT: test-catalog.md)   [cần post-condition vocabulary từ Slice 0]
        └─> Slice 2 (SETUP: sandbox + isolation proof)   [độc lập nội dung, nhưng phải xong trước mọi EXECUTE]
              └─> Slice 3 (EXECUTE: T0 — data-integrity, adversarial nặng)   [GATE: T0 pass 100%]
                    └─> Slice 4 (EXECUTE: T1 — luồng cốt lõi)
                          └─> Slice 5 (EXECUTE: T2 + T3)
                                └─> Slice 6 (EXECUTE: Track-2 smoke trên .app)
                                      └─> Slice 7 (EXECUTE: completeness pass + sign-off recommendation)
```

- Slice 0–1 có thể làm trước khi sandbox sẵn sàng (đều là doc).
- Slice 2 **bắt buộc** xong (kèm isolation proof Larry duyệt) trước khi bất kỳ EXECUTE slice nào chạy.
- Mỗi EXECUTE slice ghi kết quả vào **một** file run report duy nhất: `docs/qa/runs/<version>-qa-report.md` (tạo ở Slice 3, bồi đắp dần tới Slice 7).
- T0 là regression-core gate: **không qua T0 100% thì không tiến T1+** (spec §9).

---

## Slice 0 — [ARTIFACT] QA Playbook (`docs/playbooks/qa.md`)

**Mục tiêu:** Tài liệu vận hành để agent onboard chạy QA: cách dựng sandbox, 2-vai Executor/Verifier, mẫu charter SBTM, exit criteria, ngưỡng model. Đây là "cách làm", không phải danh sách case.

**Ai làm:** dev (Tom). **Gate:** Larry spec/design review (đủ để một agent lạ chạy được QA mà không hỏi thêm?).

**Phụ thuộc:** không (làm đầu tiên). Đọc spec §4, §5, §7, §9, §10 trước.

**Files:**
- Create: `docs/playbooks/qa.md`

- [ ] **Step 1: Tạo file + header + mục lục**

Viết section skeleton chính xác theo thứ tự (mỗi `##` là một section bắt buộc):

```markdown
# QA Playbook (Astraler Skillbox)

> Cách vận hành QA tầng-2 (validation/acceptance). Catalog case nằm ở docs/qa/test-catalog.md. Spec gốc: docs/superpowers/specs/2026-05-30-qa-strategy-design.md.

## 1. Khi nào chạy (Full vs Delta)
## 2. Hai vai: Executor & Verifier
## 3. Dựng sandbox cô lập
## 4. Post-condition kiểm-được-bằng-máy (out-of-band)
## 5. Adversarial charters (SBTM)
## 6. Trình tự thực thi (risk-first)
## 7. Exit criteria & sign-off
## 8. Ngưỡng model
```

- [ ] **Step 2: Viết §1 Full vs Delta**

Nội dung (từ spec §3): Release đầu → Full QA (toàn bộ catalog). Version sau → Delta QA = requirement đổi + feature mới + vùng ảnh hưởng (change-impact) + regression core (T0). PM bất an → gọi Full. Lọc delta bằng tag `version-added` trong catalog. Regression core = tất cả case T0, không bao giờ được vỡ.

- [ ] **Step 3: Viết §2 Executor & Verifier**

Chốt rõ, copy nguyên ràng buộc spec §4:
- **Executor**: đóng vai user + kẻ-phá, thực hiện Bước trong case. **Cấm tự tuyên bố PASS.** Chỉ ghi: đã làm gì + evidence thô (screenshot, log thao tác).
- **Verifier**: agent riêng, **không nhận context của Executor**. Tự chạy query DB / lệnh fs theo cột "Post-condition" của case → phán PASS/FAIL/BLOCKED.
- **Evidence-not-narration**: report trích kết quả query/fs-state/screenshot, KHÔNG trích lời agent kể "tôi đã cài thành công".
- Quy ước handoff giữa hai vai: Executor xong một case → ghi run-state (case-id + screenshot path + thời điểm) → Verifier đọc case-id từ catalog (không đọc narration) → query độc lập.

- [ ] **Step 4: Viết §3 Dựng sandbox (trỏ sang Slice 2)**

Tóm tắt + link: env `SKILLBOX_DB_PATH`, fake `$HOME` với fixtures `.claude/.codex/.antigravity-cli`, `GOCACHE/GOMODCACHE/GOPATH` tường minh, host/project test folder trống. Chi tiết script + isolation proof: mô tả là "xem quy trình setup ở §3 này phải chạy script `scripts/qa-sandbox/` (tạo ở Slice 2)". Ghi rõ gotcha: build cũ ~100 commit → **bắt buộc rebuild trước QA**.

- [ ] **Step 5: Viết §4 Post-condition out-of-band**

Liệt kê 3 kênh verify + ví dụ lệnh thật (dùng schema `installs` thật từ `docs/07`):
- **SQLite**: `sqlite3 "$SKILLBOX_DB_PATH" "SELECT install_mode, install_status, symlink_target_path FROM installs WHERE skill_name='X';"`
- **Filesystem**: `readlink <project>/.claude/skills/X` (symlink target đúng host), `stat -f '%Sp %HT' <path>` (kiểu file/quyền), `ls -la` (folder copy tồn tại).
- **DOM**: `agent-browser --cdp 49222 snapshot -i` đọc trạng thái UI (badge, toast, empty state).
- Nhấn mạnh: **screenshot = evidence kèm theo, KHÔNG phải verdict**.

- [ ] **Step 6: Viết §5 Charter SBTM**

Mẫu charter: **"Explore [X] with [Y] to discover [Z]"**, time-box mỗi session. Copy 9 charter từ spec §7 (input bẩn; mutation ngoài app giữa chừng; FS hiểm; đồng thời; ngắt giữa chừng; biên/giới hạn; stale DB↔FS; plugin-specific; privacy/trust network gate). Mỗi charter ghi: áp lên tier nào (T0/T1) + ví dụ kỹ thuật bẩn.

- [ ] **Step 7: Viết §6 Trình tự (risk-first) + §7 Exit criteria + §8 Ngưỡng model**

- §6: Setup → T0 (adversarial nặng) → T1 → T2 → T3 → Track-2 smoke → completeness pass (spec §8).
- §7: **T0 pass 100%, không còn bug mất/hỏng dữ liệu mở**; T1 pass ở ngưỡng user-chấp-nhận (ngưỡng cụ thể do PM chốt lúc run, ghi vào run report); T2/T3 ghi nhận, không nhất thiết chặn. Đạt → QA sign-off → Go/No-Go do PM/user quyết.
- §8: cả Executor + Verifier dùng model frontier; cấm model rẻ cho QA (lý do: spec §4 — LLM rẻ false-pass/ảo giác nặng hơn).

- [ ] **Step 8: Self-review + commit**

Đọc lại spec §4/§5/§7/§9/§10 — mỗi mục có phản ánh trong playbook? Sửa gap inline.

```bash
git add docs/playbooks/qa.md
git commit -m "docs(qa): add QA playbook — sandbox, 2-role exec/verify, charters, exit criteria"
```

**Definition of Done:** `docs/playbooks/qa.md` tồn tại với đủ 8 section; một agent lạ đọc xong có thể tự dựng sandbox + chạy 1 case + chấm verdict mà không cần hỏi. Larry review: APPROVE.

**Gate:** Larry spec/design review trên file (target: `docs/playbooks/qa.md`). Verdict approve/block + file:line. Block → Tom fix → re-review.

---

## Slice 1 — [ARTIFACT] Master Test Catalog (`docs/qa/test-catalog.md`)

**Mục tiêu:** Superset mọi test case, tổ chức theo flow/màn hình × (Happy/Edge/Adversarial), mỗi case có **post-condition kiểm-được-bằng-máy** + tag `area/feature/risk/version-added`. Là "plan để sau full test" + nguồn lọc delta.

**Ai làm:** dev (Tom). **Gate:** Larry review **coverage completeness** đối chiếu `docs/05` (mọi edge-case state có ít nhất 1 case?).

**Phụ thuộc:** Slice 0 (cần post-condition vocabulary + định nghĩa tier). Nguồn case: `docs/03` (màn hình), `docs/04` (15 flow), `docs/05` (8 nhóm edge-case state).

**Files:**
- Create: `docs/qa/test-catalog.md`

- [ ] **Step 1: Tạo file + định nghĩa schema case + bảng tier**

Header + giải thích cột chính xác:

```markdown
# Master Test Catalog (Astraler Skillbox)

> Superset mọi QA case. Lọc delta bằng tag `version-added`. Cách chạy: docs/playbooks/qa.md.

## Quy ước case

Mỗi case là một row:

| Cột | Ý nghĩa |
|---|---|
| ID | `<AREA>-<TIER>-<seq>`, vd `INSTALL-T0-003` |
| Khu vực | Màn hình/flow (docs/03, docs/04) |
| Tags | `area:… feature:… risk:T0\|T1\|T2\|T3 version-added:v0.1` |
| Tiền điều kiện | Trạng thái sandbox cần có trước khi chạy |
| Bước (Executor) | Hành động cụ thể agent làm trên UI |
| Post-condition (Verifier) | Lệnh máy-kiểm + kết quả kỳ vọng (SQL/readlink/stat/DOM) |
| Loại | Happy / Edge / Adversarial |

## Phân tầng rủi ro
- **T0** Toàn vẹn dữ liệu / release-blocker (= regression core)
- **T1** Luồng cốt lõi
- **T2** Phụ
- **T3** UX states
```

- [ ] **Step 2: Viết section T0 (data-integrity / release-blocker)**

Mỗi mục dưới đây phải có ≥1 case với post-condition máy-kiểm cụ thể. Bao phủ spec §6 T0: symlink create, rsync/copy, remove, replace existing, switch mode, Reset All Data, reconcile DB↔FS, install conflict (1005). Ví dụ một case đầy đủ (mẫu để nhân bản, **không** để "tương tự"):

```markdown
### INSTALL-T0-001 — Install skill bằng symlink tạo đúng symlink + đúng DB row
- Tags: `area:project-detail feature:add-skill risk:T0 version-added:v0.1`
- Tiền điều kiện: project test có ≥1 installable provider; host có skill `demo-skill`; chưa installed.
- Bước (Executor): mở Project Detail → Add Skill → chọn tab provider → tick `demo-skill` → Install.
- Post-condition (Verifier):
  1. `readlink "<project>/.claude/skills/demo-skill"` → trỏ vào `<host>/.agents/skills/demo-skill` (target đúng, không broken).
  2. `sqlite3 "$SKILLBOX_DB_PATH" "SELECT install_mode, install_status, symlink_target_path FROM installs WHERE skill_name='demo-skill';"` → `symlink|current|<host path>`.
  3. DOM: skill xuất hiện trong installed list của Project Detail.
- Loại: Happy
```

Bắt buộc có case **replace existing (destructive)**: install vào target đã tồn tại → app phải chặn ghi đè mặc định (docs/05 §4 "Target folder đã tồn tại"), post-condition: file cũ còn nguyên (checksum/`stat` không đổi) + DB không partial-update. Và case **conflict_error 1005**: post-condition wizard giữ mở + error row + DB không có row mới.

- [ ] **Step 3: Viết section T1 (luồng cốt lõi)**

Bao phủ spec §6 T1 + docs/04 flow 1,2,3,5: onboarding host, add project, scan/auto-scan (silent vs toast — docs/05 §8 "Auto-scan on mount"), Add Skill wizard (provider tabs, per-provider "Installed" badge, reset selection khi đổi tab — docs/04 §5 + docs/05 §7), install to project, plugin toggle global/project. Mỗi case mẫu như Step 2 (post-condition máy-kiểm cụ thể).

- [ ] **Step 4: Viết section T2 (phụ) + T3 (UX states)**

- T2 (spec §6 + docs/04 flow 6,7,8,12,14; docs/03 Dashboard): fetch/update, sync rsync/copy, change host folder, global skills, About + update-check (always-on, ADR-0002), Dashboard counters.
- T3 (docs/05 §8 UI/UX States): empty state, loading/scanning, toast policy (auto-scan silent / manual có toast), broken-symlink & missing-path warning (recoverable), confirm destructive, impact preview, cursor affordance.

- [ ] **Step 5: Viết section Adversarial (Edge/Adversarial layer)**

Cross-cut 9 charter spec §7 thành case adversarial gắn lên T0/T1. Mỗi charter ≥1 case có post-condition "no partial state" (FS + DB nhất quán). Đặc biệt: charter #5 (kill app giữa lúc ghi → kiểm không partial), charter #7 (stale: DB nói installed nhưng FS xóa tay → scan reconcile, FS thắng — docs/05 §8 "Database lệch filesystem"), charter #9 (network gate gỡ → app có gọi mạng ngoài ý muốn không).

- [ ] **Step 6: Coverage self-check vs docs/05 + commit**

Mở `docs/05`, đi qua từng nhóm (§1 Host Folder states … §8 UI/UX states). Mỗi state → point tới ≥1 case ID. Liệt kê gap (nếu có) ở cuối file dưới `## Coverage gaps (known)`. Đây là input cho Slice 7 completeness pass.

```bash
git add docs/qa/test-catalog.md
git commit -m "docs(qa): add master test catalog — T0-T3 cases with machine-checkable post-conditions"
```

**Definition of Done:** mọi mục T0/T1 trong spec §6 có ≥1 case; mọi nhóm state docs/05 map tới ≥1 case ID; mỗi case có post-condition là **lệnh máy chạy được** (không phải "verify it works"). Larry review coverage: APPROVE.

**Gate:** Larry review completeness (target: `docs/qa/test-catalog.md` vs `docs/05`). Block nếu có state docs/05 không có case.

---

## Slice 2 — [SETUP/INFRA] Sandbox cô lập + isolation proof

**Mục tiêu:** Dựng môi trường chạy QA **cô lập tuyệt đối** + **chứng minh `~/.claude` thật không bị chạm**. Output: script tái lập được + bằng chứng cô lập. (Slice này EXECUTE setup nhưng tạo script + proof, nên có verification step.)

**Ai làm:** dev (Tom) viết script. **Gate:** Larry **verify isolation proof độc lập** (tự chạy lại checksum trước/sau).

**Phụ thuộc:** không bắt buộc Slice 0/1 nhưng nên xong trước (script hiện thực hóa §3 playbook). **Phải xong trước mọi EXECUTE slice.**

**Files:**
- Create: `scripts/qa-sandbox/setup.sh` (dựng fake-home + fixtures + env)
- Create: `scripts/qa-sandbox/prove-isolation.sh` (snapshot checksum `~/.claude` trước/sau)
- Create: `scripts/qa-sandbox/run-track1.sh` (export env + `pnpm dev`)
- Create: `.scratch/qa-sandbox/` (fake-home runtime — gitignored)

> **Lưu ý môi trường (đã verify):** `core-go/cmd/skillbox-core/main.go:137` đọc `SKILLBOX_DB_PATH`; `core-go/internal/filesystem/gateway.go:77` trả `os.UserHomeDir()` (tôn trọng `$HOME`); `apps/desktop/electron/main/core-process/manager.ts:41` spawn Go core **không set `env`** → kế thừa `process.env` của Electron → set env ở shell chạy `pnpm dev` là đủ. CDP: `apps/desktop/electron/main/index.ts:18` mở `remote-debugging-port` (default `49222`, override `SKILLBOX_CDP_PORT`), **gated trên `ELECTRON_RENDERER_URL`** → chỉ `pnpm dev` mới mở CDP.

- [ ] **Step 1: Rebuild (gotcha build cũ ~100 commit)**

```bash
cd apps/desktop
pnpm install
pnpm build:core      # rebuild Go core binary
pnpm typecheck
pnpm check:contracts-drift
pnpm test            # baseline unit/contract phải xanh trước QA
```
Expected: tất cả pass. Nếu đỏ → dừng, báo Orchestrator (không QA trên build hỏng).

- [ ] **Step 2: Viết `scripts/qa-sandbox/setup.sh`**

Nội dung script (đầy đủ, không placeholder):

```bash
#!/usr/bin/env bash
set -euo pipefail
SANDBOX="${SANDBOX:-$PWD/.scratch/qa-sandbox}"
FAKE_HOME="$SANDBOX/fake-home"
mkdir -p "$FAKE_HOME"/.claude "$FAKE_HOME"/.codex "$FAKE_HOME"/.antigravity-cli
# fixtures tối thiểu để cô lập plugin + global + update-check
echo '{}' > "$FAKE_HOME/.claude/settings.json"
# DB throwaway
export SKILLBOX_DB_PATH="$SANDBOX/qa.db"
# fake HOME nhưng giữ Go caches ở vị trí THẬT (go run cần)
export HOME="$FAKE_HOME"
export GOCACHE="${REAL_GOCACHE:-$(go env GOCACHE)}"
export GOMODCACHE="${REAL_GOMODCACHE:-$(go env GOMODCACHE)}"
export GOPATH="${REAL_GOPATH:-$(go env GOPATH)}"
echo "SANDBOX ready: HOME=$HOME DB=$SKILLBOX_DB_PATH"
```

> Quan trọng: chạy `go env GOCACHE` … **trước** khi đặt lại `HOME` (gọi trong subshell với HOME thật), nếu không sẽ ra path trong fake-home. Script trên capture qua `REAL_*` hoặc gọi `go env` trước khi export HOME — Tom đảm bảo thứ tự đúng khi hiện thực.

- [ ] **Step 3: Tạo host folder test + project test trống**

Theo spec §5 (path tuyệt đối):
```bash
mkdir -p "<global-documents>/host-skills-test"           # trống → test "thiếu .agents/skills"
mkdir -p "<project-documents>/project-test-astraler-skillbox" # trống → test "no provider detected"
```

- [ ] **Step 4: Viết `scripts/qa-sandbox/prove-isolation.sh` (CHỨNG MINH)**

```bash
#!/usr/bin/env bash
set -euo pipefail
REAL_CLAUDE="$HOME/.claude"   # chạy script này với HOME THẬT
SNAP="${1:-/tmp/claude-snapshot}.txt"
# checksum đệ quy toàn bộ ~/.claude thật
find "$REAL_CLAUDE" -type f -exec shasum {} \; | sort > "$SNAP"
echo "Snapshot -> $SNAP ($(wc -l < "$SNAP") files)"
```
Quy trình proof: chạy `prove-isolation.sh before` (HOME thật) → chạy toàn bộ QA trong sandbox → chạy `prove-isolation.sh after` → `diff before.txt after.txt` phải **rỗng**. Cũng kiểm: `qa.db` nằm trong sandbox, không có file mới trong `~/.claude` thật.

- [ ] **Step 5: Viết `scripts/qa-sandbox/run-track1.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/setup.sh"
cd apps/desktop
exec pnpm dev    # ELECTRON_RENDERER_URL được electron-vite set → CDP mở ở 49222
```

- [ ] **Step 6: Verify cô lập (chạy thử end-to-end nhẹ)**

```bash
bash scripts/qa-sandbox/prove-isolation.sh /tmp/claude-before
bash scripts/qa-sandbox/run-track1.sh   # mở app, làm 1 thao tác đọc (scan), tắt
curl -s http://127.0.0.1:49222/json/version   # xác nhận CDP live (field Browser)
bash scripts/qa-sandbox/prove-isolation.sh /tmp/claude-after
diff /tmp/claude-before.txt /tmp/claude-after.txt && echo "ISOLATION OK: ~/.claude untouched"
sqlite3 "$SKILLBOX_DB_PATH" ".tables"   # DB nằm trong sandbox, có schema
```
Expected: `diff` rỗng + "ISOLATION OK" + `.tables` liệt kê bảng trong sandbox db.

- [ ] **Step 7: Commit**

```bash
git add scripts/qa-sandbox/ .gitignore
git commit -m "chore(qa): add isolated sandbox harness + isolation proof scripts"
```
(Đảm bảo `.scratch/` đã trong `.gitignore`.)

**Definition of Done:** chạy `prove-isolation.sh before/after` quanh một session QA cho `diff` rỗng; CDP live ở 49222; `qa.db` trong sandbox; host/project test folder tồn tại & trống; rebuild xanh. **Larry tự chạy lại** before→thao tác→after, xác nhận `~/.claude` thật bytes-identical.

**Gate:** Larry verify isolation proof **độc lập** (không tin log của Tom — tự chạy diff). Block nếu diff khác rỗng hoặc CDP không phải từ dev instance.

---

## Slice 3 — [EXECUTE QA] Vòng T0 (data-integrity, adversarial nặng)

**Mục tiêu:** Chạy thật toàn bộ case T0 + charter adversarial nặng → lòi release-blocker sớm. Đây là **regression core gate**.

**Ai làm:** **Executor** (lái agent-browser, đóng vai user+kẻ-phá) + **Verifier** (Larry, query DB/fs độc lập). **Orchestrator** điều phối handoff, **không** tự chấm.

**Phụ thuộc:** Slice 0 (playbook), Slice 1 (catalog T0), Slice 2 (sandbox + proof APPROVED).

**Files:**
- Create: `docs/qa/runs/<version>-qa-report.md` (mở report, version = version đang QA)
- Đụng (read-only, lái qua CDP): app đang chạy `pnpm dev` trong sandbox

- [ ] **Step 1: Khởi tạo run report**

Tạo `docs/qa/runs/<version>-qa-report.md` với header: version, ngày, scope (Full vì release đầu / Delta), commit SHA đang test, ngưỡng T1 do PM chốt, đường dẫn isolation proof (before snapshot). Section trống: `## T0` `## T1` `## T2` `## T3` `## Track-2 smoke` `## Completeness` `## Sign-off`.

- [ ] **Step 2: Executor chạy từng case T0**

Theo `agent-browser-smoke.md`: `agent-browser connect 49222` (KHÔNG launch instance 2). Với mỗi case T0 trong catalog: thực hiện đúng "Bước (Executor)"; với case adversarial áp charter §7 (input bẩn, mutation giữa chừng, FS hiểm, đồng thời, ngắt giữa chừng, biên, stale, plugin, privacy). Executor ghi: case-id, screenshot path, mô tả thao tác thô. **Không** ghi verdict.

- [ ] **Step 3: Verifier chấm từng case T0 độc lập**

Verifier (không đọc narration Executor) đọc cột "Post-condition" của case-id từ catalog → tự chạy `sqlite3 SELECT` / `readlink` / `stat` / `agent-browser snapshot` → quyết PASS/FAIL/BLOCKED. Ghi evidence thô (output query, fs state, screenshot path) vào report `## T0`.

- [ ] **Step 4: Bug → spawn task chip**

Mỗi FAIL actionable → ghi vào report + tạo **task riêng** (spawn) cho Tom fix theo orchestration flow. Không tự fix trong slice QA (giữ vai). Bug data-loss/data-corruption = blocker.

- [ ] **Step 5: Kiểm gate T0 + isolation re-check**

```bash
bash scripts/qa-sandbox/prove-isolation.sh /tmp/claude-after-t0
diff /tmp/claude-before.txt /tmp/claude-after-t0.txt
```
Expected: diff rỗng (QA không chạm `~/.claude` thật).

**Definition of Done:** mọi case T0 có verdict + evidence trong report; **T0 PASS 100%** (sau khi bug fix + re-run); không còn bug mất/hỏng dữ liệu mở; isolation diff rỗng.

**Gate (hard):** T0 không đạt 100% → **không tiến Slice 4**. Blocker fix → Tom commit → Executor re-run case đó → Verifier re-chấm.

---

## Slice 4 — [EXECUTE QA] Vòng T1 (luồng cốt lõi)

**Mục tiêu:** Chạy thật case T1 (onboarding, add project, scan/auto-scan, Add Skill wizard, install, plugin toggle).

**Ai làm:** Executor + Verifier (như Slice 3). **Phụ thuộc:** Slice 3 PASS 100%.

**Files:** Modify: `docs/qa/runs/<version>-qa-report.md` (section `## T1`).

- [ ] **Step 1:** Executor chạy từng case T1 + charter áp lên T1 (đặc biệt wizard: per-provider "Installed" badge, reset selection khi đổi tab, empty state 0 provider — docs/05 §7).
- [ ] **Step 2:** Verifier chấm độc lập (DOM cho wizard state + SQL cho install rows). Ghi evidence vào `## T1`.
- [ ] **Step 3:** Bug actionable → spawn task chip. Re-run case sau fix.
- [ ] **Step 4:** Isolation re-check (`prove-isolation.sh after-t1`, diff rỗng).

**Definition of Done:** mọi case T1 có verdict + evidence; pass đạt **ngưỡng user-chấp-nhận** (PM chốt, ghi trong report header); isolation diff rỗng.

**Gate:** review report `## T1` — dưới ngưỡng → Orchestrator báo PM quyết tiếp tục hay block.

---

## Slice 5 — [EXECUTE QA] Vòng T2 + T3

**Mục tiêu:** Chạy case T2 (fetch/update, sync, change host, global skills, About/update-check, Dashboard) + T3 (UX states: empty/loading/toast/warning/cursor).

**Ai làm:** Executor + Verifier. **Phụ thuộc:** Slice 4.

**Files:** Modify: `docs/qa/runs/<version>-qa-report.md` (sections `## T2`, `## T3`).

- [ ] **Step 1:** Executor chạy T2. Chú ý charter #9 (privacy): bật theo dõi mạng (vd `lsof -i`/proxy log) khi mở About → xác nhận update-check chỉ gọi `api.github.com`, không gọi đâu khác ngoài ý muốn (network gate vừa gỡ, ADR-0002).
- [ ] **Step 2:** Verifier chấm T2 (SQL `fetch_results`, DOM update list/Dashboard counters).
- [ ] **Step 3:** Executor chạy T3 (toast policy: auto-scan `silent:true` không toast / manual có toast — docs/05 §8; broken-symlink warning recoverable).
- [ ] **Step 4:** Verifier chấm T3. Ghi evidence vào report.
- [ ] **Step 5:** Bug → spawn (ghi nhận; T2/T3 không nhất thiết block). Isolation re-check.

**Definition of Done:** mọi case T2/T3 có verdict + evidence; bug ghi nhận đầy đủ (không bắt buộc 100% pass per spec §9).

**Gate:** review report `## T2`/`## T3` đầy đủ evidence (không narration).

---

## Slice 6 — [EXECUTE QA] Track-2 smoke trên `.app` thật

**Mục tiêu:** Smoke artifact release: boot/path/notarize/signing/update-check trên bản đóng gói (packaged **không** mở CDP → smoke tay hoặc ép argv).

**Ai làm:** **Verifier/Larry** (smoke là vai Larry per orchestration playbook) + Executor hỗ trợ thao tác tay. **Phụ thuộc:** Slice 5 (logic đã sạch trước khi đóng gói smoke).

**Files:** Modify: `docs/qa/runs/<version>-qa-report.md` (`## Track-2 smoke`).

- [ ] **Step 1: Build `.app`**

```bash
cd apps/desktop
pnpm package:mac:unsigned   # hoặc package:mac nếu cần test signing/notarize thật
```
Expected: artifact `.app`/`.dmg` trong `dist/` (hoặc theo electron-builder config).

- [ ] **Step 2: Thử ép CDP trên packaged (best-effort)**

```bash
open dist/mac*/Skillbox.app --args --remote-debugging-port=49222 || true
curl -s http://127.0.0.1:49222/json/version || echo "packaged không mở CDP → smoke tay"
```
Spec §5: packaged cố tình không mở CDP → nếu fail thì smoke tay.

- [ ] **Step 3: Smoke checklist (tay nếu cần)**

Verify: app boot không crash; DB path đúng (packaged dùng HOME thật, KHÔNG sandbox — note rõ đây là instance thật, chạy trên data thật cần thận trọng hoặc dùng fake HOME qua `open`); About screen hiện version + update-check gọi GitHub Releases OK; signing/notarize (nếu `package:mac`) qua `spctl --assess` / `codesign -dv`.

- [ ] **Step 4:** Ghi pass/fail + evidence (log boot, `codesign` output, screenshot About) vào report.

**Definition of Done:** smoke artifact có verdict + evidence cho boot/path/update-check (+signing/notarize nếu áp dụng).

**Gate:** Larry verdict smoke. Boot crash / signing fail = blocker.

---

## Slice 7 — [EXECUTE QA] Completeness pass + sign-off recommendation

**Mục tiêu:** Rà "còn khu vực/edge nào trong `docs/05` chưa đụng tới?" → đóng gap → tổng hợp run report + đề xuất Go/No-Go (PM/user quyết).

**Ai làm:** Verifier/Larry (rà soát) + Executor (chạy case bù nếu phát hiện gap). **Phụ thuộc:** Slice 3–6.

**Files:** Modify: `docs/qa/runs/<version>-qa-report.md` (`## Completeness`, `## Sign-off`); có thể Modify: `docs/qa/test-catalog.md` (thêm case phát hiện thiếu).

- [ ] **Step 1:** Đi lại từng nhóm `docs/05` (§1–§8) + 15 flow `docs/04` → đánh dấu khu vực đã có verdict trong report. Liệt kê gap.
- [ ] **Step 2:** Gap → thêm case vào `test-catalog.md` (living) + Executor chạy + Verifier chấm bù.
- [ ] **Step 3:** Quy trình QA sai phát hiện trong lúc chạy → cập nhật `docs/playbooks/qa.md` ngay (living, spec §8).
- [ ] **Step 4:** Tổng hợp `## Sign-off`: T0 pass %, T1 pass % vs ngưỡng, T2/T3 ghi nhận, danh sách bug + task chip đã spawn, isolation proof cuối (diff rỗng). Kết luận **exit criteria đạt?** → khuyến nghị Go/No-Go.
- [ ] **Step 5: Commit report + catalog updates**

```bash
git add docs/qa/runs/ docs/qa/test-catalog.md docs/playbooks/qa.md
git commit -m "docs(qa): <version> QA run report + completeness pass + sign-off recommendation"
```

**Definition of Done:** mọi nhóm docs/05 có trạng thái (covered/gap-closed/known-risk); run report có Sign-off với khuyến nghị Go/No-Go neo vào exit criteria; living docs cập nhật. **Quyết định Go/No-Go cuối do PM/user**, không phải agent.

**Gate:** user/PM đọc `## Sign-off` → Go/No-Go.

---

## Self-Review (đối chiếu spec)

**1. Spec coverage:**
- §1 mục tiêu/nguyên tắc → toàn plan (risk-first, logic-từ-gốc qua docs/04-05, tài liệu mỏng).
- §2 hai tầng → plan tập trung tầng-2 (validation); tầng-1 chỉ ghi nhận (đúng spec, không mở rộng unit test).
- §3 regression theo version → Slice 0 §1 playbook + tag `version-added` Slice 1.
- §4 AI-QA core (Executor/Verifier, post-condition out-of-band, evidence-not-narration, model frontier) → Capability Map + Slice 0 §2/§4/§8 + mọi EXECUTE slice.
- §5 sandbox → Slice 2 (đầy đủ env, fixtures, host/project test, isolation proof, gotcha rebuild + GOCACHE).
- §6 catalog & tier → Slice 1.
- §7 charters → Slice 0 §5 + áp dụng Slice 3–5.
- §8 trình tự risk-first → Slice 3→7 sequencing.
- §9 báo cáo & exit criteria → run report + gate T0-100% (Slice 3) + Sign-off (Slice 7).
- §10 artifact set → Slice 0 (qa.md), Slice 1 (test-catalog.md), Slice 3+ (runs/<version>).
- §12 ngoài phạm vi → tôn trọng (không e2e tự động, không mở unit coverage, không cross-platform exec, không HMAC receipt phase này).

**2. Placeholder scan:** mỗi case mẫu (INSTALL-T0-001) có lệnh máy-kiểm thật; script có nội dung đầy đủ; không có "tương tự Task N" (đã ghi mẫu để nhân bản kèm yêu cầu cụ thể hóa).

**3. Type/path consistency:** dùng tên bảng/cột thật từ `docs/07` (`installs.install_mode/install_status/symlink_target_path`); path file đã verify (`main.go:137`, `gateway.go:77`, `manager.ts:41`, `index.ts:18`); build script thật (`package:mac:unsigned`, `build:core`); CDP port `49222`/`SKILLBOX_CDP_PORT`.

**Known open item (không phải gap, cần user xác nhận lúc duyệt):** mapping Executor↔Tom-executor (xem Capability Map). Nếu user muốn Executor là agent thứ ba riêng → chỉnh ở Capability Map, không ảnh hưởng cấu trúc slice.
