# Product Brief: Astraler Skillbox

## Vấn Đề

Agent skills đang trở thành một phần quan trọng trong workflow với AI agents.
Người dùng ngày càng thử nghiệm nhiều skill, nhiều project, và nhiều agent
provider khác nhau như Claude, Codex, opencode, Antigravity CLI, v.v.

Việc quản lý skill hiện còn rời rạc:

- Skill nằm ở nhiều nơi khác nhau.
- Global skill và project-level skill dễ bị lẫn.
- Mỗi project cần một bộ skill riêng.
- Mỗi provider có convention riêng về folder, path, naming, hoặc format.
- Không chỉ developer dùng skill, nên CLI-only không đủ.
- Update skill bất tiện khi nhiều project dùng chung một skill.

Astraler Skillbox giải quyết vấn đề này bằng một app GUI-first để quản trị
skill local trên nhiều project và provider.

## Định Vị Product

```text
Skillbox là local-first control center cho agent skills.
```

Skillbox quản lý:

- Skill Host Folder: folder do user chọn trong GUI để làm source of truth cho
  skill trên máy.
- Skills: các skill có trong host.
- Sources: GitHub, Vercel skills, local/manual.
- Projects: các project được add vào app.
- Global Skills: skill/config ở provider global level trên máy.
- Providers: Claude, Codex, opencode, Antigravity CLI, và provider khác.
- Installs: skill nào được cài vào project/provider nào, bằng mode nào.
- Updates: Fetch upstream để biết skill nào có bản mới.

## Người Dùng Mục Tiêu

Người dùng không chỉ là developer.

Nhóm người dùng có thể gồm:

- Developer
- Content creator
- Researcher
- Marketer
- Operator
- PM
- Founder
- Analyst

Điểm chung là họ dùng nhiều AI agent workflow và cần quản lý skill có kiểm soát.

## Pain Points Đã Chốt

- Người dùng thử nghiệm nhiều skill, lâu dần không biết skill nào đang ở đâu.
- Mỗi project cần một bộ skill riêng.
- Global skill và project-level skill dễ bị lẫn, gây nhiễu context và chồng
  chéo behavior.
- Người dùng khó nhìn thấy provider global level đang có skill/config nào.
- Nhiều provider agent có convention khác nhau về folder, path, naming.
- Không chỉ developer dùng skill, nên CLI-only là chưa đủ.
- Update skill bất tiện khi nhiều project dùng chung một skill.
- Người dùng khó biết project nào đang dùng skill nào, dùng bằng symlink hay
  copy.
- Skill discovery và skill management hiện đang rời rạc.

## Quyết Định Thiết Kế Đã Chốt

- Skillbox là GUI-first.
- Skill Host Folder là folder do user chọn và cấu hình trong GUI.
- Skill content source of truth nằm trong Skill Host Folder.
- Global Skills là khu vực riêng để quan sát provider global level, không trộn
  với Skill Host Folder hoặc project-level installs.
- App dùng SQLite ngay từ đầu để lưu metadata quản trị.
- Skill source ưu tiên GitHub và Vercel skills.
- Có nút Fetch để kiểm tra upstream update.
- Convert skill format giữa provider là Phase 2.
- Health check chi tiết chưa phải trọng tâm.
- Người dùng cần hiểu các khái niệm kỹ thuật như symlink, rsync/copy, provider,
  Skill Host Folder.

## Skill Host Folder

Skill Host Folder là folder do user chọn trong GUI để làm source of truth cho
skill trên máy.

```text
<skill-host-folder>/
  .agents/
    skills/
      documentation-and-adrs/
      documentation-writer/
      browser-automation/
```

Skillbox đọc danh sách skill từ folder này và cài sang project khác bằng
symlink hoặc rsync/copy.

## Project Install

Project install là việc một skill từ Skill Host Folder được cài vào một
project/provider cụ thể.

Luồng chính:

```text
<skill-host-folder>/.agents/skills/<skill>
        |
        | symlink hoặc rsync/copy
        v
target-project/.agents/skills/<skill>
```

Install mode:

- `symlink`: project trỏ trực tiếp về skill trong Skill Host Folder.
- `rsync/copy`: project nhận một bản snapshot từ Skill Host Folder.
- `direct`: skill đã nằm trong project nhưng không do Skillbox quản lý.

## Provider Model

Skillbox cần provider adapter để hiểu mỗi provider dùng folder/path/convention
nào.

Giả định hiện tại:

- Claude có thế giới riêng.
- Nhiều provider còn lại có thể dùng chung convention kiểu `.agents`.
- Dù vậy, adapter layer vẫn cần tồn tại từ đầu để tránh bị khóa vào một
  convention.

## Updates

Skillbox có nút Fetch để kiểm tra upstream update.

Nguồn skill ưu tiên:

- GitHub repo trực tiếp.
- GitHub repo + subfolder.
- GitHub repo + branch/tag/commit.
- Vercel skills ecosystem.
- Local/manual skill.

Với symlink install, update Skill Host Folder sẽ ảnh hưởng project ngay.

Với rsync/copy install, update Skill Host Folder không đổi project cho tới khi
project được sync lại.

## Phase 2

Phase 2 có thể bao gồm:

- Convert skill format giữa các provider.
- CLI cho automation và power users.
- Advanced doctor/health checks.
- Import/export diagnostics.
- Multi-host management.
