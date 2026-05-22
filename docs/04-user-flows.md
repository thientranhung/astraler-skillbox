# User Flows

Tài liệu này mô tả các luồng thao tác chính trong Skillbox. UI là interface
chính; CLI có thể được bổ sung sau.

## 1. First-Time Setup

Mục tiêu: cấu hình Skillbox lần đầu để biết folder nào là source of truth cho
skill.

Flow:

```text
User mở Skillbox lần đầu
  -> Skillbox yêu cầu chọn Skill Host Folder
  -> User chọn một folder trên máy
  -> Skillbox kiểm tra hoặc tạo cấu trúc .agents/skills
  -> Skillbox scan skill hiện có trong folder đó
  -> Skillbox lưu cấu hình vào database
  -> User vào Dashboard
```

Kết quả:

- App có một Skill Host Folder được cấu hình.
- Folder đó trở thành source of truth cho skill.
- Dashboard có dữ liệu ban đầu từ lần scan đầu tiên.

## 2. Add Project

Mục tiêu: đưa một project vào Skillbox để quản lý skill.

Flow:

```text
User chọn Add Project
  -> User chọn project folder
  -> Skillbox scan provider conventions trong project
  -> Skillbox scan các skill đang có trong provider folders
  -> Skillbox lưu project vào database
  -> User được đưa tới Project Detail
```

Kết quả:

- Project được theo dõi bởi Skillbox.
- App biết project đang có provider nào.
- App biết project đang có skill nào và install mode nào nếu phát hiện được.

## 3. Scan Project

Mục tiêu: cập nhật trạng thái thật của project từ filesystem.

Flow:

```text
User mở Project Detail
  -> User chọn Scan hoặc Rescan
  -> Skillbox đọc project folder
  -> Skillbox phát hiện provider folders
  -> Skillbox đọc skill entries trong từng provider
  -> Skillbox phân loại install mode
  -> Skillbox cập nhật database
  -> UI hiển thị trạng thái mới
```

Install mode:

- `symlink`: skill trong project là symlink tới Skill Host Folder.
- `rsync/copy`: skill là folder copy và có metadata do Skillbox quản lý.
- `direct`: skill tồn tại trong project nhưng không do Skillbox quản lý.

Kết quả:

- Project Detail phản ánh trạng thái hiện tại trên filesystem.
- Các thay đổi ngoài app được đưa vào database.

## 4. Scan Global Skills

Mục tiêu: biết provider global level trên máy đang có skill/config nào.

Flow:

```text
User mở Global Skills
  -> User chọn Scan Global
  -> Skillbox đọc global provider locations đã biết/cấu hình
  -> Skillbox phát hiện global skills/entries
  -> Skillbox phân loại mode/status
  -> Skillbox cập nhật database
  -> UI hiển thị global skill state theo provider
```

Kết quả:

- User biết provider global level đang có skill nào.
- App phân biệt global skill với project-level skill.
- Warning được tạo nếu global entry missing, broken, external, hoặc unmanaged.

Phase 1:

- Global Skills là scan/visibility/remediation surface.
- Chưa có flow `Install Skill To Global Location`.
- Add Skill flow chỉ target project providers.

## 5. Install Skill To Project

Mục tiêu: cài một skill từ Skill Host Folder vào project.

Flow:

```text
User mở Project Detail
  -> User chọn Add Skill
  -> Skillbox hiển thị danh sách skill từ Skill Host Folder
  -> User chọn một hoặc nhiều skill
  -> Nếu project có nhiều provider, user chọn provider target
  -> User chọn install mode: symlink hoặc rsync/copy
  -> Skillbox cài skill vào provider folder tương ứng
  -> Skillbox ghi install metadata vào database
  -> UI cập nhật danh sách installed skills
```

Kết quả:

- Skill xuất hiện trong project provider folder.
- Database ghi nhận project, provider, skill, install mode, và source path.

## 6. Fetch Skill Updates

Mục tiêu: kiểm tra upstream để biết skill nào có bản mới.

Flow:

```text
User mở Updates hoặc Skills Library
  -> User chọn Fetch hoặc Fetch All
  -> Skillbox đọc source metadata của skill
  -> Skillbox kiểm tra GitHub hoặc Vercel skills
  -> Skillbox lưu fetch result vào database
  -> UI hiển thị skill có update
```

Kết quả:

- User biết skill nào có bản mới.
- UI có thể hiển thị affected projects cho từng skill.

## 7. Update Skill Host Folder

Mục tiêu: cập nhật bản skill trong Skill Host Folder từ upstream.

Flow:

```text
User mở Updates hoặc Skill Detail
  -> User chọn Update skill
  -> Skillbox hiển thị affected projects
  -> User xác nhận update
  -> Skillbox cập nhật skill trong Skill Host Folder
  -> Skillbox cập nhật version/source metadata
  -> UI refresh update status
```

Kết quả:

- Skill trong Skill Host Folder được cập nhật.
- Project dùng `symlink` nhận thay đổi ngay.
- Project dùng `rsync/copy` được đánh dấu cần sync nếu có khác biệt.

## 8. Sync Rsync / Copy Project

Mục tiêu: cập nhật project đang dùng snapshot copy từ Skill Host Folder.

Flow:

```text
User mở Project Detail hoặc Updates
  -> User chọn Sync cho một skill hoặc nhiều skill
  -> Skillbox copy lại skill từ Skill Host Folder sang project
  -> Skillbox cập nhật install metadata
  -> UI đánh dấu project đã sync
```

Kết quả:

- Project dùng bản copy mới nhất từ Skill Host Folder.
- Install mode vẫn là `rsync/copy`.

## 9. Switch Install Mode

Mục tiêu: đổi cơ chế cài skill trong project.

Flow:

```text
User mở Project Detail
  -> User chọn một installed skill
  -> User chọn Switch Mode
  -> User chọn symlink hoặc rsync/copy
  -> Skillbox thay thế entry hiện tại trong provider folder
  -> Skillbox cập nhật install metadata
  -> UI hiển thị mode mới
```

Kết quả:

- Skill trong project chuyển sang install mode mới.
- Database phản ánh mode mới.

## 10. Remove Skill From Project

Mục tiêu: gỡ skill khỏi một project/provider.

Flow:

```text
User mở Project Detail
  -> User chọn installed skill
  -> User chọn Remove
  -> Skillbox xác nhận thao tác
  -> Skillbox xóa symlink hoặc folder copy khỏi provider folder
  -> Skillbox cập nhật database
  -> UI remove skill khỏi danh sách installed skills
```

Kết quả:

- Skill không còn được cài trong project/provider đó.
- Skill gốc trong Skill Host Folder không bị ảnh hưởng.

## 11. Add Skill To Skill Host Folder

Mục tiêu: đưa một skill mới vào source of truth.

Flow:

```text
User mở Skills Library
  -> User chọn Add / Import Skill
  -> User chọn source: GitHub, Vercel skills, local/manual
  -> Skillbox import skill vào Skill Host Folder
  -> Skillbox lưu source metadata vào database
  -> Skillbox scan lại Skills Library
```

Kết quả:

- Skill mới xuất hiện trong Skills Library.
- Skill có thể được cài sang project bằng symlink hoặc rsync/copy.

## 12. Change Skill Host Folder

Mục tiêu: đổi source of truth sang folder khác.

Flow:

```text
User mở Settings
  -> User chọn Change Skill Host Folder
  -> User chọn folder mới
  -> Skillbox scan folder mới
  -> Skillbox cảnh báo nếu install hiện tại đang trỏ về folder cũ
  -> User xác nhận
  -> Skillbox cập nhật config/database
```

Kết quả:

- Skillbox dùng Skill Host Folder mới làm source of truth.
- Project đang symlink tới folder cũ có thể cần được relink nếu user muốn.

## 13. App Startup

Mục tiêu: khi mở app, Skillbox phản ánh trạng thái hệ thống đủ tin cậy.

Flow:

```text
User mở Skillbox
  -> Skillbox load database
  -> Skillbox kiểm tra Skill Host Folder còn tồn tại không
  -> Skillbox kiểm tra global provider locations nếu đã cấu hình
  -> Skillbox kiểm tra các project path còn tồn tại không
  -> Skillbox hiển thị Dashboard
  -> Nếu có missing path, UI hiển thị warning
```

Kết quả:

- User thấy trạng thái tổng quan ngay khi mở app.
- Missing Skill Host Folder, global provider location, hoặc project path được
  báo rõ trong UI.
