# QA Strategy (Astraler Skillbox)

- **Date:** 2026-05-30
- **Status:** approved (brainstorm) → chờ implementation plan
- **Người yêu cầu:** thienth@astraler.com
- **Bối cảnh:** Sắp release. Unit/contract test (Go `go test`: 101 file; Vitest: 45 file) đã dày, **không có e2e**. Test kỹ thuật luôn pass nhưng người dùng thật vẫn gặp bug → thiếu tầng "đóng vai người dùng". Cần tư duy QA: sinh nhiều kịch bản, có vai "kẻ-phá", chạy thật trên môi trường cô lập, do AI agent thực thi.
- **Cơ sở:** chốt qua brainstorm + một vòng deep-research fact-checked (xem §11).

## 1. Mục tiêu & nguyên tắc

Săn bug mà *người dùng cuối* gặp nhưng test kỹ thuật bỏ sót, trước khi release — bằng AI agent đóng vai người dùng + kẻ-phá.

Nguyên tắc:
- **Phủ đầy đủ theo flow/màn hình, chạy theo rủi ro (nguy hiểm nhất trước).**
- **Logic đúng từ gốc** — mỗi case có *kỳ vọng* rõ dựa trên `docs/04-user-flows.md`, `docs/05-edge-cases-and-ux-states.md`.
- **Tài liệu mỏng (tailored):** mượn *tên gọi & cấu trúc* từ chuẩn (ISO/IEC/IEEE 29119), bỏ giấy tờ thừa. Không làm traceability matrix nặng nề.
- **★ KHÔNG tin lời agent tự kể** (xem §4) — verdict phải neo vào sự thật kiểm chứng được, không phải narration.
- **Không thêm e2e tự động** (chủ ý loại bỏ: chậm, giòn, luôn pass, không phản ánh hành vi thật).

## 2. Hai tầng QA

| Tầng | Tên (thuật ngữ) | Câu hỏi | Gồm | Khi nào |
|---|---|---|---|---|
| **1 — Dev-time** | Verification / quality gate / Definition of Done | "Build đúng theo spec chưa?" | unit + contract-drift + typecheck (+ `go test -race`) + **smoke checklist** | Liên tục, lúc dev/CI (shift-left) |
| **2 — Release** | Validation / Acceptance | "Đúng thứ người dùng cần, hành xử đúng không?" | risk-based + **exploratory (SBTM) + adversarial/destructive** + acceptance | Trước mỗi release |

- Tầng 1: giữ mô hình **Testing Trophy** (mạnh unit/contract, integration vừa phải, **smoke thay e2e**). Spec này **không** mở rộng unit test — chỉ ghi nhận hiện trạng + chốt "không thêm e2e, dùng smoke checklist". (Mở rộng coverage = slice riêng nếu cần.)
- Tầng 2 là trọng tâm của spec này.

## 3. Mô hình regression theo version

- **Release đầu → Full QA** (toàn bộ master catalog).
- **Version sau → Delta QA:** test **requirement đổi + feature mới + vùng bị ảnh hưởng** (change-impact analysis) **+ "regression core"** (bộ critical-path T0 luôn chạy). **PM thấy bất an → gọi Full QA.**
- Mỗi test case **tag** `area / feature / risk / version-added` để lọc delta.
- **Regression core = các case T0** (toàn vẹn dữ liệu) — không bao giờ được vỡ.

## 4. ★ Nguyên tắc AI-QA (cốt lõi — định hình cả thiết kế)

Phát hiện then chốt từ research: **LLM tự làm oracle là không đáng tin** — báo pass giả, ảo giác thao tác thành công, verdict không nhất quán. **Screenshot/live-view một mình KHÔNG phải bằng chứng pass.** Vì vậy:

1. **Post-condition kiểm được bằng máy, out-of-band.** Mỗi case mang điều kiện hậu *cụ thể*, không phải "verify it works". Skillbox có lợi thế (kiểm độc lập ngoài UI):
   - **SQLite:** `SELECT` thẳng vào `qa.db` (install record, mode, settings).
   - **Filesystem:** `readlink`/`ls`/`stat` xác minh symlink thật, target đúng, folder copy tồn tại.
   - **DOM:** agent-browser đọc trạng thái UI.
   - **Screenshot = evidence kèm theo, KHÔNG phải verdict.**
2. **Tách 2 vai agent:**
   - **Executor** — đóng vai người dùng + kẻ-phá, thực hiện bước. *Không được tự tuyên bố PASS.*
   - **Verifier** — agent riêng, **không thấy context của Executor**, tự query DB/fs để phán PASS/FAIL/BLOCKED (chống reward-hacking/ảo giác).
3. **Evidence-not-narration:** report trích bằng chứng thật (kết quả query, fs state, screenshot), không trích lời kể của agent.
4. **Model frontier** cho cả hai vai; ghi ngưỡng nhất quán vào playbook (không dùng model rẻ cho QA).

## 5. Môi trường sandbox (cô lập tuyệt đối — "không xóa dữ liệu thật")

- **DB throwaway:** env `SKILLBOX_DB_PATH=<fake-home>/qa.db` (`core-go/cmd/skillbox-core/main.go:138`).
- **Fake `$HOME`:** `os.UserHomeDir()` tôn trọng `$HOME` (`filesystem/gateway.go:76`); core spawn kế thừa env cha (`electron/main/core-process/manager.ts:41`). Fake-home chứa `.claude/ .codex/ .antigravity-cli/` giả (fixtures) → cô lập plugin + global + update-check. `~/.claude` thật không bị chạm.
- **Host Folder test:** `/Users/tranthien/Documents/0.GLOBAL/host-skills-test` (trống → test luôn "thiếu `.agents/skills`").
- **Project test:** `/Users/tranthien/Documents/1.DATA/project-test-astraler-skillbox` (trống → test "no provider detected").

**2-track:**
| Track | Mục đích | Build | CDP |
|---|---|---|---|
| **1 — săn bug logic + adversarial** | agent-browser lái app | `pnpm dev` | ✅ mở (`index.ts:17`, gated `ELECTRON_RENDERER_URL`) |
| **2 — smoke artifact** | boot/path/notarize/signing/update-check trên bản release | rebuild `.app` | ❌ packaged không mở; thử ép `--remote-debugging-port` qua argv, không được thì smoke tay |

**Gotcha:** (a) packaged cố tình không mở CDP → Track 1 dùng dev (renderer + Go core + RPC giống release). (b) `pnpm dev` dùng `go run` cần `$HOME` cho GOCACHE/GOPATH → khi fake HOME phải set tường minh `GOCACHE/GOMODCACHE/GOPATH` về vị trí thật. (c) **build hiện tại cũ ~100 commit → bắt buộc rebuild trước QA.**

## 6. Cấu trúc catalog & phân tầng rủi ro

Tổ chức theo flow/màn hình (`docs/03/04/05`), mỗi mục 3 lớp **Happy → Edge → Adversarial**.

**Mỗi test case:** `ID | Khu vực | Tags(area/feature/risk/version) | Tiền điều kiện | Bước (Executor) | Post-condition máy-kiểm (Verifier) | Loại`.

**Phân tầng rủi ro (quyết định thứ tự chạy + regression core):**
- **T0 — Toàn vẹn dữ liệu / release-blocker (= regression core):** symlink create, rsync/copy, remove, replace existing, switch mode, Reset All Data, reconcile DB↔FS, install conflict (1005).
- **T1 — Luồng cốt lõi:** onboarding host, add project, scan/auto-scan, Add Skill wizard (provider tabs, per-provider "Installed", reset selection khi đổi tab), install to project, plugin toggle global/project.
- **T2 — Phụ:** fetch/update, sync, change host folder, global skills, About + update-check, Dashboard.
- **T3 — UX states:** empty/loading/toast policy, broken-symlink/missing-path warning, cursor affordance.

## 7. Adversarial charters (SBTM)

Mỗi charter là một session time-box theo mẫu **"Explore [X] with [Y] to discover [Z]"**, áp lên T0/T1 để kiểm app validate tới đâu:
1. **Input bẩn:** path space/unicode/emoji/cực dài, tên trùng, ký tự lạ.
2. **Mutation ngoài app giữa chừng:** xóa folder / đổi target symlink khi đang scan/install.
3. **Trạng thái FS hiểm:** broken symlink, symlink trỏ ngoài host, permission denied (chmod 000), read-only, file-nơi-cần-dir.
4. **Đồng thời:** 2 thao tác cùng target, double-click Install, scan khi đang install.
5. **Ngắt giữa chừng:** kill/đóng app khi đang ghi → kiểm tra không partial state (FS lẫn DB).
6. **Biên/giới hạn:** settings.json too-large/malformed/rỗng, 0 skill, hàng trăm skill/project.
7. **Stale:** DB nói "installed" nhưng FS đã xóa tay; host bị move/unmount.
8. **Plugin-specific:** malformed settings, path escape, settings là symlink, managed/enterprise config, toggle khi file read-only.
9. **Privacy/trust:** network gate vừa gỡ (`33689b9`, update-check now always-on) → kiểm app có gọi mạng ngoài ý muốn không.

## 8. Kế hoạch thực thi (risk-first)

1. **Setup:** rebuild; dựng fake-home + fixtures; verify cô lập (chứng minh `~/.claude` thật không bị chạm).
2. **Vòng T0** (adversarial nặng): filesystem/reset/reconcile → lòi blocker sớm.
3. **Vòng T1** → **T2** → **T3.**
4. **Track 2 smoke** trên `.app` thật.
5. **Completeness pass:** rà "còn khu vực/edge nào trong `docs/05` chưa đụng tới?".
- Suốt quá trình: phát hiện quy trình sai → cập nhật `docs/playbooks/qa.md` ngay (living).
- Mỗi bước: Executor làm → Verifier chấm độc lập (query DB/fs) → ghi evidence.

## 9. Báo cáo & cổng release (exit criteria → sign-off → go/no-go)

- **Kết quả mỗi case:** PASS/FAIL/BLOCKED + bằng chứng (query result + fs state + screenshot), do **Verifier** quyết.
- **Bug:** ghi vào QA report + mỗi bug actionable → một task chip (spawn) riêng.
- **Exit criteria (đã chốt):** **T0 pass 100%**, không còn bug mất/hỏng dữ liệu mở; **T1** pass ở ngưỡng user chấp nhận; **T2/T3** ghi nhận, không nhất thiết chặn.
- Đạt exit criteria → **QA sign-off** → **Go/No-Go** do PM/user quyết.

## 10. Bộ artifact (mỏng, living)

| Artifact | Vai trò | Vòng đời |
|---|---|---|
| `docs/playbooks/qa.md` | **Cách vận hành** QA cho agent onboard (setup sandbox, 2-vai, charter mẫu, exit criteria, ngưỡng model) | Living |
| `docs/qa/test-catalog.md` | **Master regression catalog** — superset mọi case, có post-condition máy-kiểm + tag. *Là "plan để sau full test".* | Living, lớn dần mỗi version |
| `docs/qa/runs/<version>-qa-report.md` | **Một lần chạy** 1 version: scope (full/delta), kết quả, evidence, sign-off | Per-version |

## 11. Cơ sở research & độ tin cậy

- **Đã fact-check (cao):** artifact set 29119 tailorable + nên giữ mỏng; AI agent thực thi QA từ test step khả thi (Nova Act, WebTestPilot); **LLM tự chấm là không đáng tin** (false pass/ảo giác/không nhất quán); 3 cách chống (post-condition DSL out-of-band, verifier agent độc lập, signed receipts); cần model frontier. **Bị bác:** live-view một mình ≠ oracle; định nghĩa cứng "strategy vs plan", test case 10 trường, RTM phủ-100%.
- **Chưa fact-check vòng này (kiến thức ngành):** §2 (pyramid/trophy), §3 (regression core/delta), §7 (SBTM charter), §9 (DoD/exit criteria/go-no-go). Ổn định, dùng tạm; có thể research sâu sau nếu cần.

## 12. Ngoài phạm vi

- E2e tự động (loại bỏ có chủ ý).
- Mở rộng unit-test coverage (slice riêng nếu cần).
- Dọn framing "offline = feature" + network gate trong docs/code → task riêng (đã spawn).
- Test cross-platform (Windows/Linux) — execution chạy macOS; ghi chú rủi ro, không thực thi slice này.
- Receipt ký HMAC (NabaOS) — lý tưởng nhưng cần hạ tầng; Phase sau. Phase 1 dựa vào post-condition out-of-band + verifier độc lập.
