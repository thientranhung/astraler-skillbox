# Agent Browser — Tự Động Hoá Smoke Electron

Cách điều khiển **dev app đang chạy** để smoke test / tự động hoá UI qua Chrome
DevTools Protocol (CDP). Tài liệu này tự đủ ngữ cảnh — chỉ cần đọc khi thật sự
chạy browser automation.

## Nguyên tắc: connect, không launch

`pnpm dev` đã chạy app. Agent **attach** vào instance đó — không bao giờ launch
app thứ hai. `pnpm dev` expose CDP trên một localhost port cố định (mặc định
`49222`, override bằng `SKILLBOX_CDP_PORT`; được gate bởi `ELECTRON_RENDERER_URL`
trong `electron/main/index.ts` nên packaged build không bao giờ mở port này).
Dải port dành riêng cho agent-browser: **49222-49250**.

## Quy Trình

```sh
curl -s http://127.0.0.1:49222/json/version   # xác nhận CDP đang live (field Browser)
agent-browser connect 49222                   # attach vào dev app đang chạy
agent-browser --cdp 49222 snapshot -i         # sau đó dùng workflow agent-browser bình thường
```

## Lưu Ý Dễ Sai

- **Không dùng `agent-browser open <url>` cho app smoke test.** Lệnh này spawn
  headless Chrome riêng trên ephemeral port; khi daemon chết có thể để lại process
  mồ côi. Luôn dùng `connect`.
- **`get url` fail trên Electron** (`Target.createTarget: Not supported`). Muốn đọc
  URL hiện tại thì query CDP endpoint: `curl -s http://127.0.0.1:49222/json`, rồi
  đọc `url` của target `page`.
- **Teardown sạch.** Thoát dev app bình thường để đóng port. Nếu kill từ shell,
  kill **electron-vite watcher trước**; nếu không watcher sẽ respawn app:
  `pkill -f "electron-vite.js dev"`, sau đó kill electron + go core. Dừng
  automation daemon còn sót bằng `pkill -f agent-browser-darwin`.
- **Không `pkill -f skillbox-core` khi có nhiều dev instance.** Pattern này match
  Go core của mọi instance và làm crash tất cả (`onFatal -> app.quit`). Hãy target
  đúng process tree cụ thể (launch bằng `setsid` rồi kill process group).
- **Audit dải port bất cứ lúc nào:** `lsof -nP -iTCP:49222-49250 -sTCP:LISTEN`.
