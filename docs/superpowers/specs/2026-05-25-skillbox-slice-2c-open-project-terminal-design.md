# Slice 2C: Open Project Terminal — Design

Date: 2026-05-25
Status: approved
Scope: add a macOS-first project action that opens Terminal focused on the project folder.

## Purpose

Users often inspect a project from the terminal after viewing it in Skillbox. Add a one-click project utility that opens Terminal at the project folder without modifying project files.

## In Scope

- Add Electron-handled `dialog.openTerminal { path } -> { opened: true }`.
- Implement macOS first with `open -a Terminal <path>`.
- Add Open Terminal actions to Projects list and Project detail.
- Surface failures as `unknown_error`.
- Keep the action read-only: no command execution inside the terminal.

## Out Of Scope

- Configurable terminal app selection.
- Windows/Linux terminal support.
- Running shell commands after opening.
- Go core or database changes.

## Technical Approach

Electron main handles the operation because it is native desktop behavior. Use `child_process.execFile` or equivalent argument-array invocation, not shell string interpolation, so project paths are passed as data rather than executable script.

Renderer uses a narrow `openTerminal(path)` client wrapper and a project hook. UI adds a `Terminal` icon button beside Open Folder in list and detail.

## Acceptance Criteria

- Clicking Open Terminal from list opens Terminal at the project folder.
- Clicking Open Terminal from detail opens Terminal at the project folder.
- Failure maps to `unknown_error`.
- Desktop typecheck, tests, build, and contract drift checks pass.
