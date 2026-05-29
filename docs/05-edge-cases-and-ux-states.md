# Edge Cases And UX States

Tài liệu này mô tả các tình huống không lý tưởng mà Skillbox cần xử lý để UI/UX
chặt chẽ hơn. Đây chưa phải technical implementation detail; mục tiêu là định
nghĩa trạng thái, rủi ro, và phản hồi UI phù hợp.

## 1. Skill Host Folder States

### Chưa cấu hình Skill Host Folder

Tình huống:

- User mở app lần đầu.
- Database chưa có `skill_host_folder`.

UI nên:

- Hiển thị onboarding để user chọn Skill Host Folder.
- Không cho install skill trước khi cấu hình xong.

### Skill Host Folder không tồn tại

Tình huống:

- Folder đã bị move, rename, unmount, hoặc xóa.

UI nên:

- Hiển thị warning rõ trên Dashboard và Settings.
- Cho user chọn lại folder mới.
- Cho user mở danh sách project/install có thể bị ảnh hưởng.

### Skill Host Folder thiếu `.agents/skills`

Tình huống:

- User chọn một folder mới hoặc folder chưa được chuẩn hóa.

UI nên:

- Giải thích Skillbox cần cấu trúc `.agents/skills`.
- Cho user tạo cấu trúc này bằng một action rõ ràng.

### Skill Host Folder rỗng

Tình huống:

- `.agents/skills` tồn tại nhưng chưa có skill nào.

UI nên:

- Hiển thị empty state trong Skills Library.
- Đưa action `Add / Import Skill`.

### Skill Host Folder không có quyền đọc/ghi

Tình huống:

- App không thể scan hoặc import/update skill.

UI nên:

- Phân biệt lỗi read và write.
- Cho user chọn folder khác hoặc mở folder trong file manager để xử lý quyền.

### Skill Host Folder đổi sang path mới

Tình huống:

- User đổi source of truth trong Settings.

UI nên:

- Scan folder mới trước khi áp dụng.
- Hiển thị project/install đang trỏ về folder cũ.
- Đề xuất relink các symlink nếu user muốn.

### Skill Host Folder nằm trên drive ngoài hoặc synced folder

Tình huống:

- Folder nằm trên external drive, iCloud, Dropbox, Google Drive, NAS, v.v.

UI nên:

- Không chặn.
- Báo warning nếu path tạm thời không available.
- Cho user rescan khi folder available trở lại.

## 2. Project States

### Project path không tồn tại

Tình huống:

- Project đã bị move, rename, unmount, hoặc xóa.

UI nên:

- Hiển thị project warning trong Projects và Dashboard.
- Cho user update path hoặc remove project khỏi database.

### Project chưa có provider folder nào

Tình huống:

- User add một folder project mới nhưng chưa có convention nào được nhận diện.

UI nên:

- Hiển thị trạng thái `No provider detected`.
- Cho user chọn provider/convention muốn setup nếu app hỗ trợ tạo cấu trúc.

### Project có nhiều provider folder

Tình huống:

- Project có Claude và `.agents`, hoặc nhiều convention cùng tồn tại.

UI nên:

- Hiển thị provider badges rõ ràng.
- Khi install skill, yêu cầu chọn provider target.

### Project có skill thủ công

Tình huống:

- Skill tồn tại trong provider folder nhưng không có install metadata của
  Skillbox.

UI nên:

- Phân loại là `direct`.
- Không tự nhận là managed install.
- Cho user adopt/import vào Skillbox nếu feature này được hỗ trợ sau.

### Project có skill trùng tên ở nhiều provider

Tình huống:

- `skill-a` tồn tại trong Claude folder và `.agents/skills`.

UI nên:

- Hiển thị theo provider scope, không gộp mù.
- Trong Project Detail, skill row nên thể hiện provider rõ ràng.

## 3. Global Skill States

### Chưa cấu hình global provider location

Tình huống:

- Provider có global level nhưng Skillbox chưa biết path global của provider đó.

UI nên:

- Hiển thị provider global state là `not configured`.
- Cho user configure path nếu provider adapter support global location.

### Global provider location không tồn tại

Tình huống:

- Global path từng tồn tại nhưng bị move, unmount, hoặc xóa.

UI nên:

- Hiển thị warning trong Dashboard và Global Skills.
- Cho user update path, rescan, hoặc disable location.

### Global provider location tồn tại nhưng rỗng

Tình huống:

- Provider global folder tồn tại nhưng không có skill/global entries.

UI nên:

- Hiển thị empty state theo provider.
- Không coi đây là lỗi.

### Global skill unmanaged/direct

Tình huống:

- Global entry tồn tại nhưng không do Skillbox quản lý.

UI nên:

- Phân loại là `direct`.
- Hiển thị rõ đây là global-level entry.
- Không tự remove hoặc relink.

### Global skill trùng với project-level skill

Tình huống:

- Cùng skill name tồn tại ở global level và project-level.

UI nên:

- Hiển thị warning/informational state để user biết có thể có chồng chéo.
- Không tự quyết định precedence vì provider behavior có thể khác nhau.

### Global symlink bị broken hoặc external

Tình huống:

- Global entry là symlink hỏng hoặc trỏ ngoài Skill Host Folder.

UI nên:

- Phân loại tương tự project install: `broken_symlink` hoặc `external_symlink`.
- Cho user relink, remove, hoặc leave as-is.

## 4. Install States

### Symlink hợp lệ

Tình huống:

- Project skill là symlink trỏ tới Skill Host Folder.

UI nên:

- Hiển thị mode `symlink`.
- Hiển thị source path.
- Cho open source folder và open project folder.

### Symlink bị broken

Tình huống:

- Symlink target không còn tồn tại.

UI nên:

- Hiển thị warning trong Project Detail.
- Cho user relink, remove, hoặc switch sang rsync/copy nếu có source tương ứng.

### Symlink trỏ tới Skill Host Folder cũ

Tình huống:

- User đã đổi Skill Host Folder, nhưng project còn symlink tới host cũ.

UI nên:

- Phân loại là symlink nhưng đánh dấu `old host`.
- Cho user relink sang Skill Host Folder hiện tại.

### Symlink trỏ ngoài Skill Host Folder

Tình huống:

- Skill trong project là symlink nhưng target không nằm trong Skill Host Folder
  hiện tại.

UI nên:

- Phân loại là `external symlink`.
- Không tự sửa.
- Cho user remove, relink, hoặc leave as-is.

### Rsync/copy current

Tình huống:

- Project copy khớp với snapshot/source metadata mới nhất.

UI nên:

- Hiển thị mode `rsync/copy`.
- Hiển thị trạng thái `current`.

### Rsync/copy outdated

Tình huống:

- Skill Host Folder đã update nhưng project copy chưa sync.

UI nên:

- Hiển thị trạng thái `outdated`.
- Cho user sync một skill hoặc nhiều skill.

### Direct install

Tình huống:

- Skill là folder thường, không có metadata Skillbox.

UI nên:

- Hiển thị mode `direct`.
- Không hiển thị update/sync action như managed install.

### Target folder đã tồn tại khi install

Tình huống:

- User install skill nhưng provider folder đã có entry cùng tên.

UI nên:

- Chặn ghi đè mặc định.
- Cho user chọn replace, skip, hoặc cancel.
- Nếu replace, cần confirm rõ vì đây là destructive action.

### Conflict khi switch mode

Tình huống:

- Đổi từ symlink sang copy hoặc ngược lại nhưng target path không thể thay thế.

UI nên:

- Không partial-update metadata nếu filesystem operation fail.
- Hiển thị lỗi và giữ trạng thái cũ.

## 5. Fetch And Update States

### Skill không có source metadata

Tình huống:

- Skill local/manual không biết upstream ở đâu.

UI nên:

- Hiển thị source là `local/manual`.
- Disable Fetch cho skill đó hoặc cho user cấu hình source.

### GitHub repo không truy cập được

Tình huống:

- Repo bị xóa, private, sai URL, hoặc thiếu auth.

UI nên:

- Hiển thị fetch error theo từng skill.
- Không làm hỏng state fetch của skill khác.
- Cho user sửa source metadata.

### Vercel skills fetch fail

Tình huống:

- Vercel skills source tạm thời không truy cập được hoặc response không hợp lệ.

UI nên:

- Hiển thị lỗi recoverable.
- Cho retry.

### Network offline

Tình huống:

- Fetch không thể kết nối network.

UI nên:

- Hiển thị global fetch warning.
- Giữ nguyên last known update state.
- Cho retry khi network có lại.

### Upstream có update

Tình huống:

- Fetch phát hiện version/commit mới.

UI nên:

- Hiển thị skill trong Updates.
- Hiển thị affected projects và install modes.

### Upstream không có update

Tình huống:

- Skill đang ở bản mới nhất.

UI nên:

- Hiển thị trạng thái `up to date`.
- Không đưa vào danh sách cần action.

### Local skill đã sửa khác upstream

Tình huống:

- Skill trong Skill Host Folder có local modifications.

UI nên:

- Không tự overwrite.
- Hiển thị trạng thái cần review.
- Cho user chọn giữ local, overwrite, hoặc tạo snapshot tùy design sau này.

### Update ảnh hưởng nhiều symlink projects

Tình huống:

- Một skill được nhiều project symlink dùng chung.

UI nên:

- Trước update, hiển thị affected projects.
- Sau update, các project symlink được coi là nhận thay đổi ngay.

### Rsync/copy projects cần sync sau update

Tình huống:

- Skill Host Folder đã update, project copy chưa update.

UI nên:

- Hiển thị các project cần sync.
- Cho sync từng project hoặc sync batch.

## 6. Provider States

### Provider được nhận diện rõ

Tình huống:

- Project có folder/path đúng convention của provider adapter.

UI nên:

- Hiển thị provider badge/icon.
- Cho install skill vào provider đó.

### Provider convention chưa được support

Tình huống:

- Project có dấu hiệu dùng provider nhưng Skillbox chưa có adapter.

UI nên:

- Hiển thị là `unsupported provider`.
- Không tự ghi file vào path chưa hiểu rõ.

### Provider folder tồn tại nhưng format lạ

Tình huống:

- Folder đúng tên convention nhưng cấu trúc bên trong không như expected.

UI nên:

- Hiển thị warning.
- Cho user xem path và rescan.

### Claude và `.agents` cùng tồn tại

Tình huống:

- Project dùng cả Claude-specific convention và shared `.agents` convention.

UI nên:

- Tách provider scope rõ.
- Add Skill flow phải chọn provider target.

## 7. Add Skill Wizard States

### 0 installable providers (empty state)

Tình huống:

- Project không có provider nào hợp lệ để install (không có provider nào có
  `detection_status = detected/configured` và `provider_definitions.status =
  supported/experimental` và `skills_path` resolve được).

UI nên:

- Hiển thị empty state card trong wizard: "No provider is ready for install."
- Đưa CTA "Scan project" như primary action.
- Khi user nhấn "Scan project", gọi `useScanProject` và đóng wizard.
- Không hiển thị tab strip, danh sách skill, hoặc nút Install.

### Skill đã installed tại provider của tab đang active

Tình huống:

- Skill trong danh sách đã có install record tại provider của tab đang active.

UI nên:

- Hiển thị checkbox của skill đó ở trạng thái disabled + opacity-50.
- Gắn badge "Installed" cạnh tên skill.
- Không cho phép user tick lại skill đó ở tab hiện tại.
- Skill vẫn có thể chọn được ở tab của provider khác nếu chưa installed tại
  provider đó (installed-state là per-provider, không globally disabled).

### Chuyển tab reset selection

Tình huống:

- User đã tick một số skill ở tab A, sau đó chuyển sang tab B.

UI nên:

- Xóa toàn bộ `selectedSkillIds` khi tab thay đổi.
- Xóa install error (nếu có) khi tab thay đổi.
- Tab B bắt đầu với selection trống, không kế thừa lựa chọn của tab A.

### Provider là experimental

Tình huống:

- Tab trong wizard tương ứng với provider có `provider_definitions.status =
  experimental`.

UI nên:

- Hiển thị badge "experimental" trong tab header cạnh display name.
- Vẫn cho phép install bình thường (experimental không block install).
- Không cần modal confirm thêm chỉ vì experimental.

### Install error (ví dụ: conflict_error 1005)

Tình huống:

- Skillbox trả về lỗi khi user nhấn Install (ví dụ target folder đã tồn tại,
  permission denied, conflict_error code 1005, ...).

UI nên:

- Giữ wizard mở, không đóng sau lỗi.
- Hiển thị error row trong footer (text-red-600) ngay phía trên Cancel/Install.
- Cho user sửa selection hoặc nhấn Cancel để thoát.
- Error row bị xóa nếu user chuyển tab hoặc thay đổi selection.
- Không partial-update database nếu install operation thất bại.

## 8. Database And App State

### Database chưa tồn tại

Tình huống:

- User mở app lần đầu hoặc database bị xóa.

UI nên:

- Tạo database mới.
- Chạy First-Time Setup.

### Database corrupt

Tình huống:

- SQLite file không đọc được hoặc schema lỗi.

UI nên:

- Không crash im lặng.
- Hiển thị lỗi blocking.
- Cho user chọn backup/export file lỗi nếu có thể.

### Database lệch filesystem

Tình huống:

- Database ghi có install nhưng filesystem đã bị sửa ngoài app.

UI nên:

- Rescan để reconcile.
- Ưu tiên filesystem là trạng thái thật.

### Filesystem có skill nhưng database không biết

Tình huống:

- User copy skill thủ công vào project hoặc Skill Host Folder.

UI nên:

- Scan phát hiện và hiển thị.
- Với project install, phân loại `direct` nếu không có metadata.
- Với Skill Host Folder, thêm skill vào library sau scan.

### Schema migration

Tình huống:

- App version mới cần thay đổi SQLite schema.

UI nên:

- Chạy migration trước khi mở app chính.
- Nếu migration fail, hiển thị lỗi rõ và không ghi tiếp dữ liệu mới.

## 8. UI/UX States

### Empty state

Áp dụng cho:

- Chưa có Skill Host Folder.
- Skill Host Folder rỗng.
- Chưa có project.
- Chưa có global skills.
- Project chưa có skill.

UI nên:

- Nói rõ trạng thái hiện tại.
- Đưa một primary action tiếp theo.

### Loading/scanning state

Áp dụng cho:

- Scan Skill Host Folder.
- Scan project.
- Scan global locations.
- Fetch update.
- Sync rsync/copy.

UI nên:

- Hiển thị progress hoặc busy state.
- Không cho chạy trùng thao tác nguy hiểm trên cùng target.

### Confirm destructive action

Áp dụng cho:

- Remove skill khỏi project.
- Replace existing folder.
- Change Skill Host Folder khi có affected symlinks.
- Delete project khỏi database.

UI nên:

- Hiển thị object bị ảnh hưởng.
- Yêu cầu confirm rõ.

### Recoverable warning

Áp dụng cho:

- Missing path.
- Broken symlink.
- Fetch fail.
- Unsupported provider.

UI nên:

- Không chặn toàn app.
- Đưa action cụ thể như rescan, retry, relink, choose folder, remove.

### Blocking error

Áp dụng cho:

- Database corrupt.
- Không thể đọc Skill Host Folder.
- Không thể ghi khi user đang install/update.

UI nên:

- Chặn action liên quan.
- Giải thích lỗi và bước xử lý tiếp theo.

### Impact preview

Áp dụng cho:

- Update skill trong Skill Host Folder.
- Change Skill Host Folder.
- Switch install mode.

UI nên:

- Hiển thị project/provider/skill bị ảnh hưởng trước khi user xác nhận.

### Quick actions

Các trạng thái lỗi nên có action nhanh:

- Open folder.
- Rescan.
- Retry.
- Relink.
- Sync.
- Remove from database.
- Configure source.
