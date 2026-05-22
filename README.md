# Astraler Skillbox

Astraler Skillbox là một ứng dụng quản trị agent skills theo hướng GUI-first.
Dự án đóng vai trò như một local control center để quản lý skill trên nhiều
project và nhiều agent provider khác nhau.

Skillbox không chỉ dành cho developer. Người dùng có thể là content creator,
researcher, marketer, operator, PM, founder, analyst, hoặc bất kỳ ai dùng agent
skills trong nhiều workflow.

## Định Vị

```text
Skillbox = local-first control center cho agent skills
```

Skillbox giúp người dùng:

- Quản lý một Skill Host Folder làm source of truth cho skill trên máy.
- Xem provider global skills/config để phân biệt global level và project level.
- Add project vào app và scan skill/provider trong project đó.
- Cài skill từ Skill Host Folder vào project bằng symlink hoặc rsync/copy.
- Xem project nào đang dùng skill nào và theo cơ chế nào.
- Fetch upstream để biết skill nào có bản mới.
- Quản lý nhiều provider như Claude, Codex, opencode, Antigravity CLI, và các
  agent provider khác.

## Mô Hình Chính

```text
Skillbox App
  GUI quản trị chính

Skill Host Folder
  Folder do user chọn trong GUI để làm source of truth cho skill trên máy

Projects
  Các project được add vào Skillbox

Global Skills
  Provider global-level skills/config đang tồn tại trên máy

Provider Adapters
  Mapping provider -> folder/path/convention

Database
  SQLite lưu metadata quản trị
```

Skill Host Folder là folder do user chọn và cấu hình trong GUI. Skillbox dùng
folder này làm source of truth để phân phối skill sang các project khác.

```text
<skill-host-folder>/
  .agents/
    skills/
      documentation-and-adrs/
      documentation-writer/
      browser-automation/
```

Project bất kỳ sẽ nhận skill từ Skill Host Folder:

```text
<skill-host-folder>/.agents/skills/<skill>
        |
        | symlink hoặc rsync/copy
        v
target-project/.agents/skills/<skill>
```

## Cơ Chế Cài Đặt

### Symlink

Symlink là cơ chế chính để nhiều project dùng chung một source of truth.

- Sửa skill trong Skill Host Folder một lần thì các project symlink nhận thay
  đổi ngay.
- Phù hợp khi muốn dùng chung skill và update nhanh trên nhiều project.

### Rsync / Copy

Rsync/copy dùng khi một project cần snapshot ổn định.

- Project nhận một bản copy từ Skill Host Folder.
- Update Skill Host Folder không tự động đổi project đó.
- Project cần sync lại nếu muốn nhận thay đổi mới.

## Product Scope Hiện Tại

GUI là trải nghiệm chính. CLI có thể phát triển sau để phục vụ power users và
automation.

Các phần chính của app:

- Dashboard
- Skills Library
- Global Skills
- Projects
- Project Detail
- Skill Detail
- Updates
- Settings

Xem thêm:

- [Docs Index](docs/index.md)
- [01 Product Brief](docs/01-product-brief.md)
- [02 Product Notes](docs/02-product-notes.md)
- [03 Information Architecture](docs/03-information-architecture.md)
- [04 User Flows](docs/04-user-flows.md)
- [05 Edge Cases And UX States](docs/05-edge-cases-and-ux-states.md)
- [06 Data Model](docs/06-data-model.md)
- [07 Schema Dictionary](docs/07-schema-dictionary.md)
- [08 Provider Model](docs/08-provider-model.md)
- [09 UI Wireframes](docs/09-ui-wireframes.md)
- [10 Technical Architecture](docs/10-technical-architecture.md)
- [Data Model Review Prompt](docs/review-prompts/data-model-review.md)
- [Provider Model Review Prompt](docs/review-prompts/provider-model-review.md)
- [Global Skills Layer Review Prompt](docs/review-prompts/global-skills-layer-review.md)
- [Global Skills Layer Follow-Up Review Result](docs/review-results/global-skills-layer-followup-review.md)
