# Các Quyết Định Về Tech Stack Và Scaffold

Tài liệu này gom các quyết định về tech stack (công nghệ) và scaffold (cấu trúc nền tảng) trước khi bắt đầu tạo codebase thật. Mục tiêu là tiết kiệm thời gian bằng cách chọn framework/library phù hợp, nhưng vẫn giữ các ranh giới kiến trúc (architecture boundary) đã chốt trong `10-technical-architecture.md`.

Trạng thái quyết định:

```text
decided       = đã chốt, dùng làm ràng buộc khi scaffold
recommended   = đề xuất mặc định, cần xem xét/chốt trước khi code
open          = còn cần thảo luận hoặc làm một thử nghiệm nhỏ (spike)
defer         = chưa cần ở Giai đoạn 1 (Phase 1)
```

## Tóm Tắt Tech Stack

```text
Desktop shell:
  trạng thái = đã chốt (decided)
  lựa chọn = Electron

UI runtime:
  trạng thái = đã chốt
  lựa chọn = React

Core runtime:
  trạng thái = đã chốt
  lựa chọn = Golang

Database:
  trạng thái = đã chốt
  lựa chọn = SQLite

Giao tiếp Electron <-> Go (transport):
  trạng thái = đã chốt cho Giai đoạn 1
  lựa chọn = stdio JSON-RPC 2.0

Quản lý vòng đời (lifecycle) của Go core:
  trạng thái = đã chốt cho Giai đoạn 1
  lựa chọn = sidecar process do Electron main quản lý
```

## Nguyên Tắc Scaffold

- Không clone nguyên một boilerplate (mẫu dự án có sẵn) nếu boilerplate đó làm mờ ranh giới của Skillbox.
- Dùng boilerplate để học cách đóng gói (packaging), bảo mật Electron (Electron security), cấu hình Vite, thiết lập test, và các quy tắc đặt tên thư mục (folder conventions).
- Scaffold phải phản ánh đúng ranh giới kiến trúc:
  - React renderer không chạm vào hệ thống file (filesystem) hay database.
  - Electron main chỉ quản lý vòng đời (lifecycle), cầu nối (bridge), và các hộp thoại gốc (native dialogs).
  - Golang core giữ logic nghiệp vụ, SQLite, các adapter cho provider, cổng giao tiếp filesystem (filesystem gateway), tích hợp source, và trình chạy tác vụ (operation runner).
- Ưu tiên cấu trúc dễ review bởi AI và người: thư mục rõ ràng, file nhỏ, contract rõ ràng.
- Không đưa thư viện lớn vào nếu chưa có màn hình hoặc use case nào cần đến nó.

## Cấu Trúc Dự Án Đề Xuất

Đề xuất (Recommended):

```text
astraler-skillbox/
  README.md
  docs/

  apps/
    desktop/
      package.json
      pnpm-lock.yaml
      electron/
        main/
        preload/
        core-process/
      renderer/
        src/
          app/
          screens/
          components/
          features/
          lib/
          styles/

  core-go/
    go.mod
    cmd/
      skillbox-core/
    internal/
      app/
      domain/
      services/
      repositories/
      providers/
      filesystem/
      sources/
      operations/
      migrations/
      rpc/
    migrations/

  shared/
    api-contracts/
    generated/

  scripts/
  fixtures/
```

Lý do (Rationale):

- `apps/desktop` gom chung Electron + React app.
- `core-go` là module Go riêng, có thể build/test độc lập.
- `shared/api-contracts` là nơi giữ JSON Schema hoặc contract của giao thức.
- `fixtures` phục vụ cho việc test provider/filesystem scan.
- `scripts` chứa các helper để build/dev/release.

Đang mở (Open):

- Có cần thư mục `apps/desktop/renderer` hay giữ `ui/` ở thư mục gốc (root) cho ngắn hơn.

Đã chốt (Decided):

- Không dùng `go.work` ở Giai đoạn 1 vì chỉ có một Go module.
- Không dùng pnpm workspace ở Giai đoạn 1 nếu chỉ có một JS package.
- Generated TypeScript types (kiểu TS được tạo tự động) được commit vào repo và CI sẽ kiểm tra nếu có sự sai lệch (drift).

## Hướng Nghiên Cứu Boilerplate

Không chọn boilerplate cuối cùng trong tài liệu này, nhưng khi khảo sát nên dùng các tiêu chí sau:

```text
Bảo mật Electron (Electron security):
  contextIsolation = true
  nodeIntegration = false
  preload bridge hẹp (narrow)
  renderer không có quyền truy cập filesystem

Build/dev:
  Vite cho renderer
  HMR (Hot Module Replacement) nhanh
  Hỗ trợ build Electron main/preload
  Hỗ trợ đóng gói binary bên ngoài (external binary packaging)

Đóng gói (Packaging):
  Hỗ trợ electron-builder
  Hỗ trợ extraResources cho Go binary
  Đường dẫn ký xác nhận (signing/notarization) trên macOS rõ ràng

Testing:
  Vitest hoặc tương đương cho UI/core TS
  Có thể dùng Playwright cho desktop/e2e
  Go test chạy độc lập

Khả năng bảo trì (Maintainability):
  Cấu trúc thư mục đơn giản
  Không dùng template SaaS/dashboard quá phức tạp
```

Các nguồn tham khảo để đánh giá:

- `electron-vite-react`
- `vite-electron-builder`
- `electron-react-boilerplate`
- Template chính thức Electron Forge Vite

Khuyến nghị:

- Dùng Vite/Electron boilerplate như tài liệu tham khảo, không phải là chân lý tuyệt đối.
- Tự scaffold cấu trúc riêng nếu template xung đột với Go sidecar, JSON-RPC, hoặc ranh giới bảo mật.

## Công Cụ Build Frontend

Trạng thái: đề xuất (recommended).

Lựa chọn: Vite.

Tại sao:

- Dev server và HMR của React nhanh.
- Phổ biến trong các scaffold Electron + React hiện đại.
- Đóng gói cho production tốt đối với renderer.
- Hoạt động tốt với Tailwind, shadcn/ui, Vitest.

Rủi ro:

- Build Electron main/preload cần cấu hình rõ ràng để các API Node/Electron không bị đóng gói sai.
- Cần cấu hình hoặc build target riêng cho renderer, main, và preload.

Quyết định:

- Dùng `electron-vite` để quản lý các target Vite cho renderer, main, và preload.

## Trình Quản Lý Package (Package Manager)

Trạng thái: đề xuất (recommended).

Lựa chọn: pnpm, gói JS duy nhất tại `apps/desktop`.

Tại sao:

- Cài đặt nhanh.
- Lockfile xác định rõ ràng.
- Hoạt động tốt cho việc phát triển Electron mà không cần chế độ workspace.

Rủi ro:

- Một số tài liệu công cụ Electron mặc định dùng npm/yarn, nên các lệnh cần được ghi chú rõ ràng.

Quyết định:

- Không scaffold `pnpm-workspace.yaml` ngay từ ngày đầu.
- Chỉ thêm pnpm workspace khi có một JS package thứ hai.

## Đóng Gói Electron (Electron Packaging)

Trạng thái: đề xuất (recommended).

Lựa chọn: electron-builder.

Tại sao:

- Trưởng thành trong việc đóng gói app Electron.
- Hỗ trợ `extraResources` cho Go binary đi kèm.
- Hỗ trợ tốt việc signing/notarization trên macOS.
- Kết hợp tốt với `electron-updater` nếu thêm tính năng auto-update.

Rủi ro:

- Việc signing và notarization trên macOS có rủi ro cao và nên được test sớm.
- Go binary phải được đính kèm, signed, và khởi chạy từ đường dẫn resource của production.

Quyết định:

- Dùng `electron-builder` thay vì Electron Forge.
- Lên kế hoạch cho signing/notarization như một cột mốc kỹ thuật, không phải là tác vụ đẩy về cuối lúc release.
- Hoãn dùng `electron-updater` cho đến khi cần luồng release/update.

## Stack Component UI

Trạng thái: đề xuất (recommended).

Lựa chọn:

```text
shadcn/ui
Radix UI primitives
Tailwind CSS
lucide-react
```

Tại sao:

- Radix cung cấp các primitive dễ truy cập cho dialog, menu, tab, popover, tooltip, select, switch, checkbox, toast, scroll area, và nhiều cái khác.
- shadcn/ui cung cấp mã nguồn component có style sẵn, có thể đặt trong repo và tùy chỉnh được.
- Tailwind giữ cho việc styling cục bộ (local) và nhanh chóng cho UI của app.
- lucide-react cung cấp các icon đồng nhất và phù hợp với hướng thiết kế.

Rủi ro:

- Các component của shadcn được sao chép vào repo, nên ta phải tự quản lý (ownership).
- Tailwind có thể trở nên lộn xộn nếu không có tài liệu về các quy tắc layout.
- Cần kiềm chế: không thêm các bộ sưu tập block/template lớn một cách mù quáng.

Quyết định cần xác nhận:

- Dùng shadcn/ui như nguồn component, không phải là một template dashboard tạo sẵn hoàn chỉnh.
- Tạo app shell, sidebar, toolbar, table, warning, và component status đặc thù cho Skillbox thay vì dùng một template SaaS chung chung.

## Phong Cách App UI

Trạng thái: đề xuất (recommended).

Skillbox nên có cảm giác như một công cụ desktop nghiệp vụ (operational desktop tool):

- Dày đặc (dense) nhưng dễ đọc.
- Điều hướng sidebar.
- Bảng/danh sách cho skills, projects, global locations, updates.
- Khung chi tiết cho các thực thể được chọn (selected entities).
- Các huy hiệu (badge) trạng thái và cảnh báo rõ ràng.
- Hạn chế tối đa phong cách marketing/hero.
- Các dialog và wizard mang tính chức năng (functional).

Tránh:

- Layout kiểu landing-page.
- Các phần hero quá khổ.
- Card trang trí nằm lồng trong card khác.
- Gradient/hình minh họa quá nặng.
- Template SaaS dashboard chung chung che khuất đi các chi tiết về filesystem/provider.

## Router

Trạng thái: đã chốt cho Giai đoạn 1.

Lựa chọn: TanStack Router.

Tại sao:

- Định nghĩa route an toàn về kiểu (type-safe).
- Phù hợp cho các màn hình app có route chi tiết lồng nhau (nested detail routes).
- Mạnh mẽ hơn React Router khi params/search state của route trở nên quan trọng.

Rủi ro:

- Cần học/setup nhiều hơn một chút so với React Router.
- Cần giữ mô hình route đơn giản vì đây là app desktop, không phải app web công cộng.

Lựa chọn thay thế:

- React Router nếu team muốn một router đơn giản hơn, nhiều người biết hơn.

Quyết định:

- Dùng TanStack Router với `createMemoryHistory` cho ngữ cảnh Electron/file URL.

## Server State Và View Models

Trạng thái: đã chốt cho Giai đoạn 1.

Lựa chọn: TanStack Query cho các truy vấn JSON-RPC cục bộ.

Tại sao:

- Mặc dù dữ liệu là cục bộ (local), các màn hình vẫn cần trạng thái loading/error/refetch/cache.
- Hoàn thành một operation có thể vô hiệu hóa (invalidate) các query liên quan.
- Tránh việc React UI phải tự quản lý mọi vòng đời request một cách thủ công.

Quy tắc:

- Query gọi đến Electron preload bridge client, không gọi trực tiếp Go.
- Mutation gọi commands và trả về `operation_id` khi cần thiết.
- UI tải lại (re-fetch) view models sau khi hoàn thành operation.

Rủi ro:

- Caching quá mức có thể hiển thị trạng thái filesystem cũ sau khi scan/update.
- Các khóa query (query keys) phải được thiết kế kỷ luật.

Quyết định:

- Dùng TanStack Query ngay từ ngày đầu.
- Giữ thời gian stale (stale time) ngắn và invalidate mạnh tay sau khi command/operation hoàn thành.

## Trạng Thái UI Ở Client

Trạng thái: đề xuất (recommended).

Lựa chọn: Dùng React state trước; Zustand bị hoãn lại (deferred) cho đến khi thực sự cần state chia sẻ giữa các màn hình (cross-screen ephemeral state).

Dùng React state cho:

- Mở/đóng Dialog.
- Các giá trị form hiện tại.
- Các lựa chọn cục bộ (local selections).

Chỉ dùng Zustand nếu cần cho:

- UI state của App shell.
- Ngữ cảnh project/skill đang chọn được chia sẻ qua nhiều panel.
- State của panel operation tồn tại lâu (long-lived) không thuộc về riêng một màn hình nào.

Tránh:

- Đưa server/database state vào Zustand.
- Lặp lại cache của TanStack Query trong global store.

## Forms Và Validation

Trạng thái: đã chốt cho Giai đoạn 1.

Lựa chọn:

```text
react-hook-form
zod
```

Tại sao:

- Các luồng settings và wizard cần validation rõ ràng.
- Zod schema có thể phản ánh (mirror) lại validation của API contract.
- React Hook Form giúp tránh render lại quá nhiều các input có kiểm soát (controlled-input).

Quy tắc:

- Zod schema là schema validation cho UI/form.
- JSON Schema trong `shared/api-contracts` là validation cho wire contract (dữ liệu truyền tải).
- Go kiểm tra command/query params độc lập ở phía core.

Rủi ro:

- Một số validation cố ý bị lặp lại vì ràng buộc form giao diện cho người dùng và ràng buộc wire contract không phải lúc nào cũng giống hệt nhau.

## Bảng (Tables)

Trạng thái: hoãn lại (defer).

Lựa chọn: bắt đầu với các component bảng đơn giản; thêm TanStack Table khi có màn hình bảng đầu tiên thực sự cần sắp xếp (sort)/lọc (filter).

Tại sao:

- Skillbox có nhiều màn hình chứa bảng:
  - Skills Library
  - Global Skills
  - Projects
  - Project Detail installs
  - Updates affected projects/global installs
- Sắp xếp/lọc/chọn sẽ rất phổ biến.

Rủi ro:

- TanStack Table là headless nên code có thể dài dòng.
- Cần các component bảng dùng chung để tránh lặp lại setup.

Quyết định:

- Không đưa TanStack Table vào scaffold ban đầu.

## Giao Thức JSON-RPC (JSON-RPC Protocol)

Trạng thái: đã chốt một phần.

Đã chốt:

- Giao tiếp (transport) của Giai đoạn 1 là stdio JSON-RPC 2.0.
- Thư viện JSON-RPC cho Go là `creachadair/jrpc2`.
- Định dạng gói tin (framing) là NDJSON.
- Go core gửi `server.ready` trước khi Electron chuyển tiếp request từ renderer.
- Electron main chờ tối đa 10 giây để nhận `server.ready`.
- Tiến độ của Operation sử dụng JSON-RPC notifications.
- Bản Production không mở local HTTP server.

Đang mở:

- Có nên bật debug HTTP server trong chế độ dev hay không.

Luồng xử lý khi startup thất bại:

- Nếu Go thoát (exit) trước khi báo `server.ready`, hiển thị lỗi startup chặn màn hình (blocking) và đưa ra đường dẫn stderr/log.
- Nếu quá thời gian chờ (timeout) `server.ready`, kill process con và hiển thị lỗi chặn màn hình.
- Nều crash giữa chừng có thể khởi động lại tối đa 3 lần, sau đó hiển thị lỗi chặn màn hình.

Khuyến nghị:

- Dev-only debug HTTP server có thể được thêm vào sau cổng `SKILLBOX_DEBUG_PORT` sau khi method JSON-RPC đầu tiên hoạt động.

## API Contracts

Trạng thái: đề xuất (recommended).

Lựa chọn: JSON Schema trong `shared/api-contracts`.

Tại sao:

- Contract dạng người dễ đọc cho commands/queries.
- Có thể tạo (generate) ra TypeScript types.
- Phù hợp với payload của JSON-RPC.
- Nhẹ hơn protobuf/gRPC khi dùng cho IPC cục bộ.

Đang mở:

- Generate Go structs từ JSON Schema hoặc tự viết Go structs khớp bằng tay.
- Quy tắc đặt tên/đánh phiên bản cho các schema command/query.

Quyết định:

- Commit các TypeScript types được tạo tự động để AI/người dễ review hơn.
- Giữ các struct Go viết bằng tay trong Giai đoạn 1 trừ khi sự sai lệch (drift) trở nên khó xử lý.
- Thêm contract tests để serialize các response mẫu của Go và validate dựa trên schema.
- Thêm check CI để đảm bảo TypeScript types tạo tự động khớp với types đã commit.

## Go SQLite Stack

Trạng thái: đề xuất (recommended).

Lựa chọn:

```text
driver = modernc.org/sqlite
migrations = embedded SQL migrations (migrations viết bằng SQL nhúng)
```

Tại sao:

- `modernc.org/sqlite` tránh dùng CGO và làm đơn giản việc build cross-platform.
- SQL migrations nhúng có thể kiểm toán (auditable) và có phiên bản.
- SQL giữ tính dễ đọc cho người và AI.

Quyết định:

- Dùng thư mục app data tiêu chuẩn của HĐH cho SQLite.
- macOS: `~/Library/Application Support/Astraler Skillbox/skillbox.db`.
- Windows: `%APPDATA%\Astraler Skillbox\skillbox.db`.
- Linux: `~/.config/astraler-skillbox/skillbox.db`.
- Ghi đè (override) khi Dev/test: biến `SKILLBOX_DB_PATH`.
- Bật WAL.
- Bật foreign keys trên mọi kết nối.
- Đặt `busy_timeout=5000`.
- Dùng `synchronous=NORMAL`.
- Dùng `golang-migrate` với các SQL migrations nhúng.

Các PRAGMA khi khởi động:

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

## Keychain Và Thông Tin Đăng Nhập (Credentials)

Trạng thái: đề xuất (recommended).

Lựa chọn: Go core sở hữu credentials thông qua OS keychain.

Lựa chọn thư viện: `zalando/go-keyring`.

Tại sao:

- Source adapters nằm ở phía Go.
- Secret nên nằm trong process sử dụng nó.
- SQLite chỉ lưu trữ metadata/tham chiếu của credential, không lưu plaintext.

Quyết định:

- Dùng `zalando/go-keyring` trong Go.
- Cho phép dùng environment variable (biến môi trường) thay thế (fallback) khi dev/CI.
- Các biến env: `SKILLBOX_GITHUB_TOKEN`, `SKILLBOX_VERCEL_TOKEN`.
- Viết tài liệu về yêu cầu `libsecret` của Linux nếu dùng một thư viện keychain cần Secret Service API.
- Không lưu token plaintext trong SQLite.

## Go Module Và Chính Sách Phụ Thuộc (Dependency Policy)

Trạng thái: đề xuất (recommended).

Quy tắc:

- Giữ cho Go core ít dependency.
- Dùng standard library khi hợp lý.
- Chỉ dùng các thư viện cho SQLite, migrations, keychain, và JSON-RPC sau khi đã review.
- Giữ các provider adapters phần lớn là code nội bộ (internal code).

Các package Go khởi đầu đề xuất:

```text
modernc.org/sqlite
golang-migrate/migrate
zalando/go-keyring
creachadair/jrpc2
```

## Testing Stack

Trạng thái: đề xuất (recommended).

Frontend/Electron:

```text
Vitest
React Testing Library
Playwright
```

Go:

```text
go test
SQLite database tạm thời
filesystem fixtures
contract tests đối chiếu với JSON Schema
```

Tại sao:

- Vitest hợp với Vite.
- Playwright có thể dùng test các luồng Electron thật sau này.
- Go tests có thể validate hành vi của provider scan/install/fs mà không cần UI.

Đang mở:

- Có nên đưa Playwright vào ngay lập tức hay sau khi có UI shell đầu tiên.
- Cách chạy full-stack tests với Electron + Go sidecar trên CI.

Yêu cầu bắt buộc:

- `go test -race` cho operation runner, provider scan, JSON-RPC, và code filesystem gateway.
- Contract tests từ JSON-RPC method đầu tiên: serialize Go responses và validate với JSON Schema.
- Các mock-core fixtures phải được validate với JSON Schema.

## Luồng Làm Việc Dev (Dev Workflow)

Trạng thái: đề xuất (recommended).

Các lệnh mong muốn:

```text
pnpm install
pnpm dev
pnpm test
pnpm lint
pnpm build
pnpm package
go test ./...
```

Các chế độ dev:

```text
Go-only:
  chạy core tests và JSON-RPC harness không có Electron

UI-only:
  React app dùng mock core client/view models

Full-stack:
  Electron main khởi chạy Go sidecar và renderer kết nối qua preload
```

Đang mở:

- Dùng `air` hoặc watcher Go nào khác để hot reload.
- Mock core client được viết tay hay sinh tự động từ API contracts.

Quyết định:

- Hỗ trợ ba chế độ dev trong phần README của scaffold:
  - Go-only: Go tests và JSON-RPC harness không có Electron.
  - UI-only: Electron/React dùng các mock core fixture responses.
  - Full-stack: Electron main khởi chạy Go sidecar thật.
- Các mock-core fixtures được tạo ra từ việc capture lại các Go integration test hoặc validate với JSON Schema trên CI.

## Mặc Định Bảo Mật (Security Defaults)

Trạng thái: đã chốt cho scaffold.

Electron:

```text
contextIsolation = true
nodeIntegration = false
sandbox = true nếu tương thích
preload chỉ lộ ra API hẹp (narrow API)
renderer không bao giờ nhận được đường dẫn Go process hoặc chi tiết về transport
Electron main xác nhận (validate) method JSON-RPC dựa trên allowlist trước khi chuyển tiếp
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

Go:

```text
stdout = chỉ dành cho giao thức JSON-RPC
stderr/log file = logs
validate mọi lệnh ghi vào hệ thống file (filesystem writes)
không bao giờ tin tưởng các đường dẫn từ renderer cung cấp mà không validate ở core
```

Đóng gói (Packaging):

```text
Go binary được đính kèm qua electron-builder extraResources
Việc signing/notarization trên macOS được test sớm
```

## Ghi Chú Về Kích Thước Bản Build (Build Size Notes)

Trạng thái: để cung cấp thông tin (informational).

- Electron chiếm phần lớn kích thước app vì Chromium và Node được đóng gói kèm.
- Radix/shadcn/Tailwind không phải là rủi ro chính về kích thước.
- shadcn/ui copy component source; kích thước bundle phụ thuộc vào việc import cái gì.
- lucide-react nên import từng icon riêng biệt.
- Go binary nên dùng release flags như `-ldflags="-s -w"` khi đóng gói.

## Các Quyết Định Trước Khi Scaffold

Cần chốt:

- Có dev debug HTTP server hay không (yes/no).
- Chính sách sinh ra (generation policy) cho mock-core fixture.

Có thể hoãn:

- Hành vi Auto-update.
- Trình nền (persistent daemon).
- Multi-window.

## Tập Quyết Định Scaffold Đề Xuất Cho Giai Đoạn 1

```text
workspace:
  pnpm
  package đơn lẻ ở apps/desktop
  không dùng pnpm workspace cho tới khi có JS package thứ hai

desktop:
  Electron
  electron-vite
  React
  electron-builder

ui:
  shadcn/ui
  Radix UI
  Tailwind CSS
  lucide-react
  TanStack Router
  TanStack Query
  React Hook Form
  Zod
  TanStack Table (hoãn lại)
  Zustand (hoãn lại)

core:
  Golang
  SQLite thông qua modernc.org/sqlite
  golang-migrate với SQL migrations nhúng
  zalando/go-keyring
  không dùng go.work cho tới khi có Go module thứ hai

transport:
  stdio JSON-RPC 2.0
  creachadair/jrpc2
  định dạng NDJSON
  tiến độ operation qua JSON-RPC notifications
  handshake server.ready với timeout 10 giây

sqlite:
  WAL
  foreign_keys=ON
  busy_timeout=5000
  synchronous=NORMAL
  thư mục app data của OS
  ghi đè qua SKILLBOX_DB_PATH

testing:
  Vitest
  React Testing Library
  Playwright sau này hoặc sau khi có shell
  go test
  go test -race cho concurrent code
  filesystem fixtures
  contract tests

runtime CLI dependencies:
  git >= 2.20 — required for plugin update checks (updateCheck.run, ADR-0001)
  absent git → service returns status='git_not_found'; app remains fully usable offline
```
