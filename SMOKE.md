# Astraler Skillbox — Manual Smoke Checklist

Slice 1 (Skills Library) end-to-end verification. Run this after every significant change to the scaffold or after building a release candidate.

All commands run from the **repo root** unless a different directory is specified.

---

## Pre-Conditions

- [ ] Fresh clone **or** clean working tree (`git status` is clean)
- [ ] Clean database — delete the data directory so migrations run from scratch:

  ```sh
  rm -rf ~/Library/Application\ Support/Astraler\ Skillbox/
  ```

- [ ] macOS 13+ (primary test platform for slice 1; Linux is a secondary target — substitute the path above with `~/.local/share/Astraler Skillbox/`)
- [ ] Node 20+, pnpm 9+, Go 1.22+ installed (verify: `node -v`, `pnpm -v`, `go version`)

---

## 1. Setup Smoke

- [ ] Install JS dependencies:

  ```sh
  (cd apps/desktop && pnpm install)
  ```

  Completes in under 2 minutes with no errors.

- [ ] Download Go modules:

  ```sh
  (cd core-go && go mod download)
  ```

  Completes in under 1 minute.

- [ ] Start the app in full-stack dev mode:

  ```sh
  (cd apps/desktop && pnpm dev)
  ```

  An Electron window opens in under 10 seconds. The terminal shows `[manager] Go core ready`.

- [ ] Confirm no red errors in the Electron DevTools console (open via `Cmd+Option+I`).

---

## 2. Handshake Smoke

- [ ] App renders the **Setup** screen (`/setup`) on first launch — no active host configured.
- [ ] Terminal shows a line containing `server.ready` with `version`, `pid`, and `capabilities` fields (from Go core slog output).
- [ ] The Electron window title is "Astraler Skillbox".
- [ ] Go core stdout contains only NDJSON lines (no stray text). Verify by running Go standalone in a separate terminal:

  ```sh
  (cd core-go && SKILLBOX_DB_PATH=/tmp/smoke-handshake.db go run ./cmd/skillbox-core)
  ```

  First line printed to stdout must be a valid JSON-RPC notification for `server.ready`.

---

## 3. Choose Host Smoke

- [ ] Create an empty test host directory:

  ```sh
  mkdir -p /tmp/skillbox-test-host
  ```

- [ ] In the app, click **"Choose Skill Host Folder…"**.
- [ ] The native macOS folder picker opens.
- [ ] Select `/tmp/skillbox-test-host` and confirm.
- [ ] App navigates to `/skills` immediately (one click — no second confirmation step).
- [ ] The skills directory was created automatically:

  ```sh
  ls /tmp/skillbox-test-host/.agents/skills/
  # Expected: empty directory (no error)
  ```

- [ ] Verify the database recorded the host:

  ```sh
  DB=~/Library/Application\ Support/Astraler\ Skillbox/skillbox.db
  sqlite3 "$DB" "SELECT id, path, status FROM skill_host_folders;"
  # Expected: 1 row, status = 'active'
  sqlite3 "$DB" "SELECT active_skill_host_folder_id FROM app_settings;"
  # Expected: non-null integer matching the host id above
  ```

---

## 4. Scan Smoke

- [ ] Create three skill directories:

  ```sh
  mkdir -p /tmp/skillbox-test-host/.agents/skills/{foo,bar,baz}
  ```

- [ ] In the app on `/skills`, click **"Scan"**.
- [ ] A toast appears showing scan progress phases (e.g., "Scanning skills…").
- [ ] After the scan completes, the toast shows "Skills scanned" (success).
- [ ] The skills table shows 3 rows: `foo`, `bar`, `baz`, all with status **Available**.
- [ ] Verify in the database:

  ```sh
  sqlite3 "$DB" "SELECT name, status FROM skills ORDER BY name;"
  # Expected: bar|available, baz|available, foo|available
  sqlite3 "$DB" \
    "SELECT operation_type, status, metadata_json
     FROM operations ORDER BY id DESC LIMIT 1;"
  # Expected: scan | success | JSON with skillsFound >= 3
  ```

---

## 5. Reconcile Smoke

- [ ] Remove one skill from the filesystem:

  ```sh
  rm -rf /tmp/skillbox-test-host/.agents/skills/foo
  ```

- [ ] Click **"Scan"** again.
- [ ] The table shows `foo` with status **Missing**; `bar` and `baz` remain **Available**.
- [ ] Verify:

  ```sh
  sqlite3 "$DB" "SELECT name, status FROM skills WHERE name = 'foo';"
  # Expected: foo|missing
  ```

---

## 6. Broken Symlink Warning Smoke

- [ ] Create a broken symlink:

  ```sh
  ln -s /nonexistent /tmp/skillbox-test-host/.agents/skills/broken
  ```

- [ ] Click **"Scan"**.
- [ ] A warning banner appears on the `/skills` screen mentioning the broken symlink.
- [ ] Verify warnings in the database:

  ```sh
  sqlite3 "$DB" \
    "SELECT scope_type, code FROM warnings ORDER BY id DESC LIMIT 1;"
  # Expected: skill_host_folder | broken_symlink   (or similar scope)
  ```

---

## 7. Switch Host Smoke

- [ ] Create a second host with its own skill:

  ```sh
  mkdir -p /tmp/skillbox-test-host-2/.agents/skills/qux
  ```

- [ ] In the app, navigate to **Settings** (`/settings`) via the sidebar.
- [ ] Click **"Change"** next to Skill Host Folder.
- [ ] Select `/tmp/skillbox-test-host-2` in the folder picker.
- [ ] App navigates to `/skills` and shows only `qux` (from the new host).
- [ ] Verify host records:

  ```sh
  sqlite3 "$DB" "SELECT path, status FROM skill_host_folders ORDER BY id;"
  # Expected: 2 rows; first host inactive, second host active
  sqlite3 "$DB" "SELECT active_skill_host_folder_id FROM app_settings;"
  # Expected: id matching the second host
  ```

---

## 8. Lifecycle Smoke

### Graceful Quit

- [ ] Press `Cmd+Q` to quit the app.
- [ ] Verify Go core exited:

  ```sh
  ps aux | grep skillbox-core | grep -v grep
  # Expected: no output (process is gone)
  ```

### Reopen with Persistence

- [ ] Relaunch: `(cd apps/desktop && pnpm dev)`
- [ ] App navigates directly to `/skills` (not `/setup`) — active host persists from DB.
- [ ] The skill list from the second host is visible.

### Crash Restart

- [ ] While the app is running, find the Go core PID (from the terminal output or `server.ready` log).
- [ ] Kill it:

  ```sh
  kill -9 <go_pid>
  ```

- [ ] Electron detects the exit, restarts Go, and the app recovers without a fatal error dialog (restart 1 of 3).
- [ ] The terminal shows `[manager] Go core exited (code=…), restart 1/3`.

### Restart Limit

- [ ] Kill the Go process 3 more times in quick succession (before it finishes `server.ready`).
- [ ] After the 4th crash total (3 restarts exhausted), a **blocking startup error** dialog appears.
- [ ] No further automatic restarts occur.
- [ ] Close and reopen the app to recover (a new `pnpm dev` session resets the counter).

---

## 9. Validation Smoke

### Invalid path via DevTools (file instead of directory)

> The native folder picker enforces directory selection, so triggering this validation requires calling the method directly.

- [ ] Open DevTools (`Cmd+Option+I` → Console).
- [ ] Call:

  ```js
  await window.core.invoke("host.choose", { path: "/etc/hosts" })
  ```

- [ ] A structured error is thrown with `code: "validation_error"` and a human-readable `userMessage`.

### Validation error in the UI (choose host from settings)

- [ ] From the Settings screen, attempt to choose a host at a path that already exists but is not a directory (create a test file):

  ```sh
  touch /tmp/not-a-dir
  ```

- [ ] Invoke via DevTools as above with `path: "/tmp/not-a-dir"`.
- [ ] The error is surfaced in the UI (ErrorDisplay component) with `userMessage`.

---

## Notes

Manual smoke **cannot be fully automated** in a headless environment because it requires:
- A display for the Electron window
- macOS native folder picker interaction
- Visual inspection of the UI state

Run this checklist on a developer machine with a display before tagging a release candidate. The automated test suites (`pnpm test`, `go test -race ./...`) cover unit and integration logic; this checklist covers the end-to-end UI and process wiring.
