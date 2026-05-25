# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Astraler Skillbox là desktop app quản trị agent skills theo hướng GUI-first. App hoạt động như local control center để quản lý skills trên nhiều project và nhiều agent provider (Claude, Codex, v.v.).

Codebase hiện đã có scaffold chạy được: Electron + React desktop app, Go sidecar core, SQLite migrations/repositories/services, JSON-RPC contracts, Skills Library, Settings, Projects list/detail, project scan, soft-remove project, và Open Folder.

Core concept: **Skill Host Folder** là source of truth cho skill content, **SQLite** là source of truth cho metadata quản trị. Skills được phân phối tới projects bằng symlink hoặc rsync/copy.

## Tech Stack (Đã Chốt)

- **Desktop shell**: Electron + electron-vite. Packaging/signing is deferred.
- **UI**: React, TanStack Router (`createMemoryHistory`), TanStack Query, Tailwind CSS, lucide-react, sonner, react-hook-form + Zod
- **Core runtime**: Golang (sidecar process managed by Electron main)
- **Transport**: stdio JSON-RPC 2.0 — `creachadair/jrpc2`, NDJSON framing
- **Database**: SQLite via `modernc.org/sqlite` (no CGO), migrations via `golang-migrate` with embedded SQL
- **Package manager**: pnpm (single package at `apps/desktop`, no workspace yet)
- **Testing**: Vitest + React Testing Library (frontend), `go test` with temp SQLite and fixture folders (backend)

## Current / Target Project Structure

```
astraler-skillbox/
  apps/desktop/
    electron/main/          # Window lifecycle, app menu, native dialogs
      core-process/         # Spawn/monitor/stop Go sidecar, JSON-RPC bridge
    electron/preload/       # Narrow typed bridge to renderer
    renderer/src/
      screens/              # Screen-level components
      components/           # Shared UI components
      features/             # Feature-specific logic
      lib/core-client/      # IPC client — wraps preload bridge calls
  core-go/
    cmd/skillbox-core/      # Entry point
    internal/
      app/                  # Composition root / wiring
      domain/               # Business rules, enums, typed errors
      services/             # Use case orchestration
      repositories/         # SQLite queries — only place with direct SQL
      providers/            # Provider adapter implementations
      filesystem/           # Filesystem gateway
      sources/              # GitHub/Vercel/local source adapters
      operations/           # Operation runner, progress, cancellation
      rpc/                  # JSON-RPC method registration
      migrations/           # Schema migration logic
    migrations/             # Embedded SQL migration files
  shared/
    api-contracts/          # JSON Schema for commands/queries
    generated/              # Generated TypeScript types (committed)
  fixtures/                 # Fixture folders for provider/filesystem tests
```

## Architecture Boundaries (Hard Rules)

**React renderer**:
- Chỉ render state, gọi commands/queries qua preload bridge
- Không import `ipcRenderer`, filesystem API, database client, hoặc provider adapter trực tiếp
- Không tự join raw tables hoặc suy luận business rules

**Electron main**:
- Chỉ làm window lifecycle, preload bridge, native dialogs, Go process lifecycle
- Validate JSON-RPC method allowlist trước khi forward sang Go
- Không chứa business logic

**Go core** (owns everything else):
- SQLite, filesystem writes, provider adapters, source integrations, operation runner
- Stdout chỉ dành cho JSON-RPC protocol messages. Logs đi stderr/log file
- Mọi filesystem write đi qua `filesystem.Gateway` — không service nào gọi `os.WriteFile`/`os.Remove` trực tiếp
- Provider adapters chỉ trả facts/capabilities, không ghi DB, không write filesystem
- Repository layer là nơi duy nhất viết SQL trực tiếp

## SQLite PRAGMAs (Bắt Buộc Trên Mọi Connection)

```sql
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
```

DB path: `~/Library/Application Support/Astraler Skillbox/skillbox.db` (macOS). Override dev/test: `SKILLBOX_DB_PATH`.

## JSON-RPC Transport Rules

- Go gửi `server.ready` notification trước khi Electron forward bất kỳ renderer request nào
- Electron main chờ `server.ready` tối đa 10 giây. Timeout hoặc Go exit trước đó → show blocking startup error
- Mid-session crash: restart tối đa 3 lần, sau đó blocking error
- On quit: SIGTERM → chờ 3s → SIGKILL
- Long-running commands trả `operation_id`; progress dùng `operation.progress` server-push notifications
- App error codes không dùng JSON-RPC reserved range `-32768` đến `-32000`

## Electron Security Defaults (Bắt Buộc)

```
contextIsolation = true
nodeIntegration = false
sandbox = true (nếu compatible)
CSP = default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

## CQRS Pattern

**Queries** (không có side effect): `getDashboard()`, `listSkills()`, `getProjectDetail(projectId)`, v.v.

**Commands** (có thể ghi DB/filesystem, trả `operation_id` nếu long-running): `scanProject(projectId)`, `installSkillToProject(input)`, `syncInstall(installId)`, v.v.

## Manual Constructor DI (Phase 1)

Không dùng DI framework (`google/wire`, `uber-go/dig`). Wiring nằm trong `core-go/cmd/skillbox-core/main.go` hoặc `core-go/internal/app/`. Nếu composition root quá lớn, tách ra `internal/app/`, không thêm DI container.

## Operation Locking

Phase 1: fail-fast per target, không queue tự động. Một Skill Host Folder / project / global provider location chỉ có một active operation. Trả `conflict_error` nếu target đang bận.

## Core Go Dependencies

```
modernc.org/sqlite
golang-migrate/migrate
creachadair/jrpc2
```

Giữ dependencies tối thiểu. Không thêm thư viện nếu standard library đủ dùng. Keychain/source integrations are deferred until the corresponding slice needs them.

## Dev Commands

```bash
# Frontend / Electron
cd apps/desktop
pnpm install
pnpm dev          # Full-stack dev với Go sidecar thật
pnpm typecheck    # TypeScript project references, no emit
pnpm test         # Vitest
pnpm build        # electron-vite build
pnpm generate:contracts
pnpm check:contracts-drift

# Go core
cd core-go
go test ./...
go test -race ./internal/operations/... ./internal/filesystem/... ./internal/providers/...
```

Ba chế độ dev:
- **Go-only**: Go tests và JSON-RPC harness không có Electron
- **UI-only**: React/Electron tests dùng mocked core-client responses
- **Full-stack**: Electron main khởi chạy Go sidecar thật

## UI Style

Skillbox là operational desktop tool, không phải SaaS dashboard. Tránh: hero layout, card lồng card, gradient nặng, template SaaS chung chung. Ưu tiên: sidebar navigation, tables/lists, status badges, detail panes, functional dialogs.

## Key Docs

Đọc theo thứ tự khi cần context sâu hơn:
- `AGENTS.md` — contributor guide for this repository
- `docs/10-technical-architecture.md` — architecture boundaries và module responsibilities
- `docs/11-tech-stack-and-scaffold-decisions.md` — tech stack decisions với trạng thái (decided/recommended/open)
- `docs/12-implementation-patterns.md` — 16 patterns cụ thể khi implement
- `docs/06-data-model.md` + `docs/07-schema-dictionary.md` — SQLite schema
- `docs/08-provider-model.md` — provider adapter contract
- `docs/agent-orchestration-playbook.md` — multi-agent orchestration, `/goal` usage, tmux hygiene, review loop

## Current Implementation Notes

- Skill Host Folder is configured through Settings/Setup and scanned by `host.scan`.
- Skills Library reads active host skills through `skill.list`.
- Projects use `project.add`, `project.list`, `project.get`, `project.scan`, and `project.remove`.
- Project remove is a soft-remove (`projects.status = removed`); it must never delete project files.
- Project Open Folder is Electron-native (`dialog.openPath`) and should map shell failures to `unknown_error`.
- Provider detection implemented so far is `generic_agents` (`.agents/skills`) only.
