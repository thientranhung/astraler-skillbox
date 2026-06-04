# ADR-0004: Local Diagnostic Export

- **Status:** accepted
- **Date:** 2026-06-05
- **Deciders:** owner / Claude Code (Tom)
- **Tags:** architecture, product, privacy, observability

## Context

After releasing v0.1.2, the owner needs a way to collect diagnostic information from users to help diagnose bugs. The current product promise is strictly local-first: no telemetry, no background polling, no Skillbox-operated server (ADR-0002). Go core logs to stderr only (no log files). There is no built-in way for users to share diagnostic context when filing bug reports.

## Decision

Implement local-only, user-triggered diagnostic export:

- Diagnostics are collected entirely in the Electron main process (no Go RPC involved).
- Collection includes: app version, Electron/Chrome/Node versions, platform/arch, DB path, and the last 100 lines of Go core stderr output buffered in memory since app start.
- All home directory paths are redacted to `~` before export.
- No skill contents, plugin contents, or credentials are included.
- Two user-triggered actions are exposed in the About screen: "Export Diagnostics…" (native save dialog → writes `.txt` file) and "Copy to Clipboard".
- No automatic upload, no network call, no background job.

## Alternatives Considered

- **Sentry or remote error reporting** — violates local-first promise and requires explicit opt-in infrastructure. Deferred per OBS-001 triage note; would require a separate ADR.
- **Go RPC method for log collection** — unnecessary; Go's logs arrive on stderr which is already piped to the Electron main process. Keeping collection in main avoids adding a Go RPC method for data that main already has.
- **Renderer reading logs directly** — violates the architecture boundary (renderer has no filesystem access). Main process is the correct layer for native file I/O and clipboard.

## Consequences

**Positive:**
- Users can share diagnostics for bug reports without any privacy concern — export is always manual and local.
- No new Go code, no new JSON-RPC contract, no network surface.
- Consistent with ADR-0002 local-first invariant and the architecture boundary (renderer → main → native).

**Negative / cost:**
- Log capture is in-memory only; logs from before the current process launch are not available.
- The ring buffer holds the last 100 lines; earlier output is discarded.

**Neutral / watch:**
- If log volume grows significantly, the ring buffer size may need tuning.

## Implementation Notes

- `apps/desktop/electron/main/core-process/manager.ts`: adds `coreLogBuffer` ring buffer (100 lines), `pushLogChunk()`, and exported `getCoreLogs()`.
- `apps/desktop/electron/main/core-process/diagnostics.ts`: pure `buildDiagnosticsText(opts)` function with no Electron dependency (unit-testable).
- `apps/desktop/electron/main/core-process/ipc-bridge.ts`: native handlers for `dialog.exportDiagnostics` and `dialog.copyDiagnostics` — never forwarded to Go.
- `apps/desktop/electron/main/core-process/method-allowlist.ts`: both methods added.
- `apps/desktop/renderer/src/screens/about-screen.tsx`: Diagnostics section with Export and Copy buttons.

## Verification

- `pnpm test` passes the `diagnostics.test.ts` suite covering redaction, log tail, format.
- `pnpm typecheck` passes with no new type errors.
- QA case TC-DIAG-001 covers the user-triggered export flow.
- The invariant INV-PRIVACY-001 (no automatic outbound network) is not violated — both actions are user-triggered and write only to the local filesystem or clipboard.

## References

- OBS-001 in `.scratch/2026-06-04-v012-user-feedback-triage.md`
- ADR-0002 (local-first, no telemetry)
