# Spec: Gỡ network-gate cho Plugin Update-Check (release-blocker)

- **Status:** DRAFT — chờ Larry spec-review + user chốt scope
- **Author:** Tom (planner/senior dev)
- **Date:** 2026-05-30
- **Phase:** Brainstorm + Spec (compressed). **KHÔNG implement, KHÔNG tạo branch ở phase này.**
- **Liên quan:** ADR-0001 (`docs/decisions/0001-outbound-network-update-check.md`) — spec này đề xuất **supersede** một phần.
- **KHÔNG đụng:** QA strategy đang dở (`docs/superpowers/specs/2026-05-30-qa-strategy-design.md`, `.scratch/handoff-qa-strategy.md`).

---

## 0. Đính chính evidence (self-verify đã đọc đúng file:line)

> ⚠️ **Goal brief nhầm UI surface.** Brief nói nút **"Check for Updates" ở About screen** luôn trả `disabled`. Đọc code cho thấy **KHÔNG đúng** — đó là feature **app-update** *khác*, không bị gate:
>
> - **About screen** `apps/desktop/renderer/src/screens/about-screen.tsx:84-89` → `useCheckAppUpdate` (`apps/desktop/renderer/src/features/app-about/use-check-app-update.ts:37`) → RPC `app.checkUpdate` → `UpdateCheckService.CheckAppUpdate` (`core-go/internal/services/update_check_service.go:278-280`). Hàm này comment rõ **`"Always runs — no opt-in gate"`** và auto-check ngay khi mount. **KHÔNG phải bug.**
> - **Bug release-blocker thật** là nút **"Check Updates" ở Plugins screen** (`apps/desktop/renderer/src/screens/plugins-screen.tsx:265-271`, `ArrowUpCircle`) → `useRunUpdateCheck` (`.../features/update-check/use-run-update-check.ts`) → RPC `updateCheck.run` → `UpdateCheckService.RunUpdateCheck` (`update_check_service.go:58-67`). Đây là feature **plugin update-check** bị gate.

Phần còn lại của spec nói tới **plugin update-check** (nút ở Plugins screen), trừ khi ghi rõ "app-update".

---

## 1. Root cause + Fix

### 1.1 Bug gãy **3 tầng** (không phải 2 như brief)

| # | Tầng | File:line | Hậu quả |
|---|------|-----------|---------|
| **B1** | Backend gate | `core-go/internal/services/update_check_service.go:64-67` — `if !settings.UpdateCheckEnabled { return RunResult{Status:"disabled"}, nil }` | Trả `disabled` ngay nếu setting OFF. |
| **B2** | Seed mặc định OFF | `core-go/migrations/000022_plugin_update_check_cache.up.sql:29-30` — `INSERT ... VALUES (1, 0, 6)` → `update_check_enabled = 0` trên **mọi DB mới**. | Mọi máy cài mới → gate OFF → B1 luôn trả `disabled`. |
| **B3** | **Client hardwire NoopClient** | `core-go/cmd/skillbox-core/main.go:96-100` — `updateCheckClient := network.NoopClient{}` (luôn luôn, vô điều kiện). `GitLsRemoteClient` (`network/update_check_client.go:36-44`, `NewGitLsRemoteClient`) **không được wire ở bất kỳ đâu** (grep toàn repo: chỉ xuất hiện trong file định nghĩa + test). | **Dù bật được setting**, client vẫn là stub `NoopClient.LsRemote` → trả `Error:"update_check_disabled"` cho mọi plugin. Feature **chưa từng chạy end-to-end.** |

**Kèm theo (làm bug không thể tự khắc phục):**
- **No UI toggle.** UI bật/tắt (Settings → Network) đã gỡ ở `ca4b604` / `33689b9`. Repo method `NetworkSettingsRepo.SetUpdateCheckEnabled` (`network_settings_repo.go:36-45`) còn đó nhưng **không có RPC handler / không caller nào** (grep rỗng) → dead code, không có đường bật.
- **Reset tắt ngầm:** `core-go/internal/repositories/reset_repo.go:46-50` set `update_check_enabled = 0` mỗi lần Reset All Data → kể cả máy đã (bằng cách nào đó) bật, Reset sẽ tắt lại.

**Kết luận root-cause:** việc gỡ gate ở `33689b9` làm **dở** — mới gỡ UI toggle, **chưa gỡ backend gate (B1), seed (B2), và chưa wire real client (B3)**. Đặc biệt **B3 nghĩa là Option B "fix tối thiểu = seed=1" một mình KHÔNG đủ** — vẫn phải sửa wiring ở `main.go`.

### 1.2 Architecture smell phát hiện thêm (ảnh hưởng lựa chọn scope)

Kiến trúc hiện tại **đọc setting tại call-time** (`RunUpdateCheck` gọi `networkSettingsRepo.Get` mỗi lần) **nhưng chọn client một lần tại boot-time** (`main.go`). Hệ quả: nếu giữ gate + cho bật runtime, **toggle lúc runtime sẽ không swap được client** (client cố định = Noop từ lúc boot). Đây là mâu thuẫn thiết kế. **Option A (gỡ gate, luôn wire real client) dọn sạch mâu thuẫn này; Option B kế thừa nó.**

### 1.3 Fix (định hướng, chi tiết theo scope ở §2)

Gỡ điều kiện gate (B1), wire `GitLsRemoteClient` thật thay `NoopClient` (B3), bỏ dòng reset→0, đồng bộ seed/migration (B2), gỡ copy "Settings → Network" trong UI, đồng bộ docs + supersede phần liên quan của ADR-0001.

---

## 2. Đề xuất scope — Khuyến nghị **Option A**

### Option A — RIP gate hoàn toàn (always-on) ⭐ **KHUYẾN NGHỊ**

Update-check trở thành **luôn bật**, đúng tinh thần dọn over-engineering (gate này chưa từng hoạt động end-to-end và đã mất UI).

- **B1:** Gỡ block `if !settings.UpdateCheckEnabled {...}` trong `RunUpdateCheck`.
- **B3:** `main.go` wire `network.NewGitLsRemoteClient()` thay `network.NoopClient{}`.
- **B2:** Migration `000023` **drop cột `update_check_enabled`** khỏi `network_settings` (giữ bảng vì `cache_ttl_hours` còn dùng). Seed cột không còn ý nghĩa.
- **Reset:** Bỏ `update_check_enabled = 0` khỏi câu UPDATE trong `reset_repo.go` (giữ reset `cache_ttl_hours = 6`).
- **Dead code:** Gỡ `NoopClient` (network pkg), `SetUpdateCheckEnabled` (repo), trường `UpdateCheckEnabled` trong domain — *hoặc* để lại nếu tốn công; quyết ở implementation. Khuyến nghị gỡ để sạch.
- **UI:** Sửa toast `use-run-update-check.ts:33` — bỏ nhánh `disabled` hoặc đổi copy (không còn "Settings → Network"). Status type `"disabled"` có thể giữ cho backward-compat contract hoặc gỡ.
- **ADR:** **Supersede ADR-0001 bằng ADR-0002** ("Plugin update-check always-on; gỡ network opt-in gate"). ADR-0001 → status `superseded`.

**Trade-off:**
- ✅ Sạch nhất; loại mâu thuẫn boot-time/call-time (§1.2); một nguồn sự thật; không còn dead code.
- ✅ Khớp [[feedback_no_over_engineering]] — không tôn vinh "offline-first opt-in" vốn table-stakes & đã hỏng.
- ⚠️ **Đổi invariant** "outbound network OFF by default" trong AGENTS.md/ADR-0001 → cần user chốt (đây là *product/privacy decision*, không chỉ bug-fix). App vẫn 100% dùng được offline; chỉ nút "Check Updates" là chủ động gọi mạng khi user bấm (manual trigger giữ nguyên — KHÔNG auto-poll).
- ⚠️ Migration drop-column (xem §6 caveats SQLite).

### Option B — Fix tối thiểu (giữ plumbing)

Giữ cột + interface, chỉ làm feature chạy được mặc định.

- **B2:** Migration `000023` set `update_check_enabled = 1` cho row hiện có **và** đổi seed mặc định (lưu ý: seed nằm ở migration `000022` đã chạy; phải dùng migration mới `UPDATE ... SET = 1`, không sửa `000022` cũ).
- **Reset:** Đổi dòng reset thành `= 1` (hoặc bỏ, để giữ giá trị user) — nếu vẫn coi 1 là "default" thì set 1.
- **B3 (bắt buộc, không né được):** vẫn phải wire client thật. Hoặc (B-boot) `main.go` luôn wire `GitLsRemoteClient`, hoặc (B-runtime) đọc setting lúc boot để chọn client — nhưng B-runtime vẫn dính mâu thuẫn §1.2 (toggle runtime không swap). → Thực tế B cũng phải luôn wire real client ⇒ gate trở nên **hình thức**.
- **B1:** Giữ block gate (nhưng giờ luôn pass vì setting=1).

**Trade-off:**
- ✅ Đổi ít dòng schema/ADR hơn trên giấy; giữ "khả năng tương lai tắt lại".
- ❌ **Gate vô nghĩa**: setting luôn =1, không UI tắt, client luôn real → giữ phức tạp mà không có giá trị (anti-pattern theo [[feedback_no_over_engineering]]).
- ❌ Vẫn phải sửa B3 ⇒ "tối thiểu" chỉ là ảo; effort gần bằng A nhưng để lại nợ.
- ❌ Mâu thuẫn boot/runtime (§1.2) còn nguyên.

### Khuyến nghị

**Chọn A.** B không thật sự rẻ hơn (vẫn buộc sửa B3) và để lại gate chết + mâu thuẫn kiến trúc. A dọn sạch, đúng triết lý dự án. **Nhưng A đổi invariant privacy → cần user chốt** (Orchestrator/user là người quyết, không phải Tom).

---

## 3. Risk Classification

| Tiêu chí | Option A (khuyến nghị) | Option B |
|---|---|---|
| **Layers chạm** | DB (migration), Go (service, wiring `main.go`, network pkg, reset repo, domain), Renderer (toast copy), Docs (6 file), ADR (supersede + new) | DB (migration), Go (wiring `main.go`, reset repo), Renderer (toast copy nhẹ), Docs (đồng bộ seed), ADR (amend) |
| **Breaking change** | Có — đổi invariant network (product/privacy); drop cột DB; có thể đổi contract status `disabled` | Thấp — hành vi mặc định đổi (OFF→ON) nhưng API ổn định |
| **Schema migration** | **Có** — `000023` DROP COLUMN (cần SQLite ≥ 3.35) | **Có** — `000023` UPDATE value (an toàn, không đổi schema) |
| **Est. LOC** | ~120–180 (gồm gỡ dead code + ADR mới + 6 doc) | ~70–110 (gồm B3 + doc-sync) |
| **Workflow đề xuất** | **Branch + PR** (multi-layer + schema + ADR + product decision) — theo [[feedback_orchestration_branch_workflow]] đây là Risk **cao**, KHÔNG commit thẳng main | Branch + PR (vẫn schema + multi-layer) |
| **Reversibility** | Trung bình (drop column khó undo data; nhưng cột vốn vô dụng) | Cao |

---

## 4. Danh sách file đụng tới (đầy đủ)

### Code (Go)
- `core-go/internal/services/update_check_service.go` — [A] gỡ block gate `:64-67`; xét bỏ đọc `networkSettingsRepo.Get` nếu chỉ dùng cho gate (kiểm tra `cache_ttl_hours` còn cần không). [B] giữ.
- `core-go/cmd/skillbox-core/main.go:96-100` — [A&B] wire `network.NewGitLsRemoteClient()` thay `NoopClient{}`.
- `core-go/internal/network/update_check_client.go` — [A] gỡ `NoopClient` (`:28-34`). [B] giữ.
- `core-go/internal/repositories/reset_repo.go:46-50` — [A] bỏ `update_check_enabled = 0`. [B] đổi `= 1` hoặc bỏ.
- `core-go/internal/repositories/network_settings_repo.go` — [A] gỡ `SetUpdateCheckEnabled` (`:36-45`) nếu drop cột; chỉnh `Get` SELECT (`:22`). [B] giữ.
- `core-go/internal/domain/` (file định nghĩa `NetworkSettings`, có trường `UpdateCheckEnabled`) — [A] gỡ trường. *Cần grep xác định file chính xác lúc implement.*

### Migration
- `core-go/migrations/000023_remove_update_check_gate.up.sql` + `.down.sql` — **MỚI**. [A] up: `ALTER TABLE network_settings DROP COLUMN update_check_enabled` + bump `database_version = 23`; down: add lại cột `DEFAULT 0` + set `=1` cho row + `database_version = 22`. [B] up: `UPDATE network_settings SET update_check_enabled = 1 WHERE id=1` + bump version; down: set về 0.
- *(KHÔNG sửa `000022_*.up.sql` cũ — migration đã chạy là immutable.)*

### Renderer (UI)
- `apps/desktop/renderer/src/features/update-check/use-run-update-check.ts:31-33` — sửa/bỏ nhánh `disabled` + toast "Settings → Network".
- `apps/desktop/renderer/src/features/update-check/use-run-update-check.ts:9` — [A] cân nhắc gỡ `"disabled"` khỏi `UpdateCheckStatus` (đồng bộ với contract nếu đổi).
- *(About screen KHÔNG đụng — không phải bug; xem §0.)*

### Docs (đồng bộ cùng slice — bắt buộc theo `docs/playbooks/documentation.md`)
- `docs/decisions/0001-outbound-network-update-check.md` — [A] status → `superseded by ADR-0002`. [B] amend §2 "default OFF" → "default ON".
- `docs/decisions/0002-*.md` — **MỚI** (chỉ [A]): ADR always-on, supersede 0001.
- `docs/decisions/index.md:5` — thêm dòng ADR-0002 + đổi status 0001.
- `AGENTS.md:13` — sửa invariant network (bỏ/đổi "Outbound network is OFF by default").
- `docs/10-technical-architecture.md:785-792` — sửa block `default = OFF ... opt_in = network.update_check.enabled`. **Đồng thời sửa `:292`** (`app.checkUpdate ... (opt-in)`) vì **stale**: code `CheckAppUpdate` không gate (xem §0).
- `docs/03-information-architecture.md:328-330` — sửa "Khi network bị tắt (Settings → Network)..." + dòng `:330` nói `app.checkUpdate` gated → **stale**, sửa.
- `docs/04-user-flows.md:327,338,353` — bỏ "Enable in Settings → Network"; sửa "opt-in mặc định tắt"; sửa reset flow.
- `docs/06-data-model.md:1027-1095` — sửa mô tả `network_settings`/`update_check_enabled` (default, cột bị drop nếu A).
- `docs/07-schema-dictionary.md:459-475` — cập nhật mục `network_settings` (drop cột nếu A; đổi default nếu B).

### Tests (cần cập nhật/đụng)
- `core-go/internal/network/update_check_client_test.go` — nếu gỡ `NoopClient` (A).
- `apps/desktop/renderer/src/screens/__tests__/plugins-screen.test.tsx` — assert hành vi mới (không còn disabled).
- Test cũ `NetworkOffSmokesNoRemote` (ADR-0001 §Verification, nếu tồn tại) — [A] xóa/đảo nghĩa. *Grep lúc implement.*

---

## 5. Smoke scenarios (end-to-end, đề xuất)

1. **DB mới + bấm "Check Updates" (Plugins screen):** cài mới (chạy hết migration tới 000023) → mở Plugins → bấm "Check Updates" → status **`ok`**, KHÔNG còn `disabled`; plugin có git HTTPS source hiện `↑ update` đúng (cần ≥1 plugin có upstream mới).
2. **Sau Reset All Data:** bấm Reset → quay lại Plugins → "Check Updates" vẫn **`ok`** (không tắt ngầm).
3. **Airplane-mode / offline:** tắt mạng → "Check Updates" → degrade gracefully (per-plugin `error: timeout/git_ls_remote_failed`), app KHÔNG crash, các phần khác vẫn dùng được.
4. **git không cài (`exec.LookPath` fail):** status `git_not_found`, inline notice, không crash.
5. **About screen (regression-guard):** "Check for Updates" vẫn auto-check & hoạt động như cũ (đảm bảo slice không vô tình đụng app-update).
6. **(A) Migration down rồi up lại:** rollback 000023 → 000022 → app vẫn boot; up lại → vẫn `ok`.

---

## 6. ⚠️ WARN — Rủi ro migration & backward-compat

- **(A) `ALTER TABLE ... DROP COLUMN`** cần **SQLite ≥ 3.35** (2021). Phải xác nhận version driver Go đang build (lúc implement: kiểm tra `modernc.org/sqlite` hay `mattn/go-sqlite3` + version). Nếu rủi ro, **fallback**: giữ cột nhưng ngừng đọc (no-op migration) — kém sạch nhưng zero-risk. → *Open question cho implementation.*
- **DB cũ đã có `update_check_enabled = 1`** (máy nào đó bật tay): [A] vô hại — code không còn đọc cột. [B] vô hại — vẫn =1.
- **DB cũ `= 0` (đa số):** [A] cột bị drop, không ảnh hưởng; [B] migration `UPDATE ... = 1` bật lên — đúng ý.
- **Down migration:** phải khôi phục schema/giá trị để rollback an toàn (đã ghi ở §4). Down của A re-add cột; lưu ý dữ liệu cột cũ mất khi drop — chấp nhận được vì cột vô dụng.
- **Contract `status:"disabled"`:** nếu gỡ khỏi enum renderer mà core (cũ) còn trả, sẽ lệch. [A] nên giữ tạm `"disabled"` trong type union để an toàn, hoặc gỡ đồng bộ cả 2 phía trong cùng commit.
- **Invariant đổi (A):** "outbound network OFF by default" là cam kết privacy trong ADR-0001 + AGENTS.md. Đổi nó là **product decision** — KHÔNG để Tom tự quyết. Manual-trigger-only vẫn giữ (không auto-poll) → privacy impact giới hạn ở "user chủ động bấm thì mới gọi mạng".

---

## 7. Final deliverable

- `[OK] 1` — Spec tại `docs/superpowers/specs/2026-05-30-remove-update-check-gate-design.md` (file này), mô tả root cause (§1, 3 tầng) + fix (§1.3, §2).
- `[OK] 2` — Option A vs B + khuyến nghị **A** + trade-off (§2).
- `[OK] 3` — Risk Classification table (§3).
- `[OK] 4` — Danh sách đầy đủ file: code + migration + renderer + docs + ADR + tests (§4).
- `[OK] 5` — 6 smoke scenarios end-to-end (§5).
- `[OK] 6` — Evidence: mọi file:line đã đọc & verify (§0, §1, §4).
- `[FILE]` — `docs/superpowers/specs/2026-05-30-remove-update-check-gate-design.md`
- `[LOG]` — **Recommend Option A** (gỡ gate hoàn toàn). Lý do: B3 (NoopClient hardwire) khiến Option B vẫn buộc sửa wiring ⇒ "tối thiểu" là ảo; gate đã mất UI + chưa từng chạy end-to-end + mâu thuẫn boot/runtime ⇒ giữ lại là over-engineering. A cần user chốt vì đổi invariant privacy.
- `[WARN]` — (1) `DROP COLUMN` cần SQLite ≥3.35, có fallback no-op; (2) DB cũ mọi giá trị đều an toàn với A; (3) down-migration phải restore; (4) contract `status:"disabled"` gỡ đồng bộ 2 phía; (5) đổi invariant network = product decision.
- `[STOP]` — **Spec xong — chờ Larry spec-review + user approve scope (A hay B). Orchestrator điều phối, KHÔNG tự tiếp sang code/branch.**
