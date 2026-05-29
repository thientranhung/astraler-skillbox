# ADR-0001: Outbound Network for Plugin Update Check

- **Status:** accepted
- **Date:** 2026-05-29
- **Deciders:** user (thienth@astraler.com), Tom (planner)
- **Tags:** architecture | product | network | privacy

## Context

Skillbox is a **local-first** desktop app. Per `AGENTS.md`, the app currently performs **zero outbound network calls** — all data comes from local filesystem and SQLite. This is a deliberate core invariant: it gives the user offline reliability, privacy by default, and a simple security posture.

A new feature has been proposed (G3c — Update Check): for each installed plugin, show whether a newer version is available upstream (e.g. a new git tag on the plugin's source repo). The installed version is already available locally via `installed_plugins.json` (see ADR-implied feature shipped in PR #4). What's missing is the **upstream latest** value, which requires querying a remote host.

This is the **first feature in the project that would introduce outbound network**. Decision required before any code.

### Inputs to the decision

- `~/.claude/plugins/installed_plugins.json` provides for installed plugins: `version`, `installedAt`, `lastUpdated`, sometimes `gitCommitSha`.
- `~/.claude/plugins/marketplaces/<name>/.claude-plugin/marketplace.json` lists each plugin's `source`:
  - `git-subdir` / `git` — has `url`, `ref` (tag or branch), often `sha`
  - inline source — has no remote
- For git sources, the cheapest upstream signal is `git ls-remote <url> <ref>` (returns current SHA of a ref) or — for tag lists — GitHub REST `/repos/{owner}/{repo}/tags` (60 req/h unauthenticated, 5000/h authenticated).
- "Updatable" = installed `gitCommitSha`/`version` ≠ upstream resolved SHA/tag for the configured `ref`.

### Constraints

- User has not opted in to anything related to network. Doing so silently is a breach of trust.
- Private plugin repos exist (auth needed). Phase 1 must handle the public-only case cleanly.
- Skillbox is a productivity tool, not a scheduler — there's no compelling case for background polling.
- Renderer must NOT make network calls (existing architecture boundary). All network goes through Go core.

## Decision

1. **Add outbound network capability** to Skillbox, scoped strictly to "plugin update check" for Phase 1.
2. **Default OFF.** Feature is gated by a new opt-in setting `network.update_check.enabled: bool` (default `false`). When `false`, no remote host is contacted under any circumstance, and the relevant UI controls are hidden or disabled with a one-line explanation linking to Settings.
3. **Manual trigger only in Phase 1.** No background polling, no auto-check at launch. User clicks **"Check Updates"** in Updates screen (and optionally per-plugin in Project Detail / Global Plugins). No timer, no scheduler.
4. **Public sources only in Phase 1.** No credential storage for plugin sources. Private repos surface a clear "auth required — not supported yet" state. (Phase 2 may add credential handling — separate ADR.)
5. **Network adapter lives in Go core** (`core-go/internal/network/`), behind a single interface (`UpdateCheckClient`) so it can be mocked in tests and disabled wholesale at boot.
6. **Mechanism per source type:**
   - `git-subdir` / `git` with HTTPS URL → use Go's `git ls-remote` via shelling out to system `git` (already a required tool for the user; no new dependency). Returns SHA for `ref`.
   - For semver-tag comparisons, also enumerate refs via `git ls-remote --tags` and parse semver client-side. NO GitHub REST in Phase 1 — keeps auth surface and rate-limit surface zero.
   - Plugins from inline / local sources → `latestVersion = null`, no remote call.
7. **Update-available rule:**
   - If installed has `gitCommitSha` AND ref resolves remotely → `updateAvailable = installed.gitCommitSha != remote.sha`.
   - Else if installed has semver-shaped `version` AND ref is a tag pattern → `updateAvailable = semver.gt(remote.latestSemverTag, installed.version)`.
   - Else `updateAvailable = unknown` (rendered as `?` or omitted).
8. **Timeouts & limits:**
   - Per-request timeout: **8s**. Whole-batch deadline: **60s**.
   - Max concurrent `ls-remote` processes: **4**.
   - Per-host backoff after consecutive failures (3 fails → skip host for the remainder of the batch).
9. **Cache TTL: 6 hours** (configurable via `network.update_check.cache_ttl_hours`, default 6). Cache stored in SQLite (new table `plugin_update_check_cache` — see Implementation Notes). User clicking "Check Updates" with a fresh cache returns instantly; with a stale cache, triggers refresh and shows freshness timestamp.
10. **Telemetry: none.** No analytics, no error reporting outbound. Failures are surfaced in-UI only.
11. **No URL composed from user input.** Only URLs that already exist in `marketplace.json` (which the user has installed locally) are contacted. Logged plainly so user can audit which hosts were called and when.
12. **Settings UI surfaces** (under Settings → Network):
    - Toggle: "Enable plugin update checks (queries plugin source hosts over network)" — off by default.
    - Cache TTL slider/input.
    - "View recent network activity" — table of `host | when | result` (last 100 entries).
13. **First-time opt-in dialog.** When the user toggles the setting on the first time, show a confirmation explaining what is sent (git refs to plugin source hosts) and what is NOT (no analytics, no Skillbox-controlled servers, no telemetry).
14. **Contract additions (additive, optional):**
    - `PPGlobalEntry.updateAvailable?: boolean | null`
    - `PPGlobalEntry.latestVersion?: string | null`
    - `PPGlobalEntry.lastCheckedAt?: string | null`
    - Same three additions on `PPProjectEntry`.
    - All NOT in `required`. Backward compatible.
15. **Local-first invariant remains.** The invariant is amended in `AGENTS.md` to: *"Skillbox is local-first. Outbound network is OFF by default; the only opt-in network feature is plugin update checks against the user's already-installed plugin source hosts (see ADR-0001)."* Listing/scanning/installing skills never requires network. The app remains fully usable offline.

## Alternatives Considered

- **Status quo (no update check)** — would leave users to discover staleness manually (e.g. `git pull` inside plugin cache directories). Rejected: real user value lost, and the data is cheaply derivable.
- **Always-on background polling** — convenient but violates default-off principle, surprises the user with network activity, requires scheduler infra. Rejected.
- **GitHub REST API as primary mechanism** — richer metadata but introduces auth surface (rate limit forces token storage even for public repos at scale), couples us to one host, and bypasses non-GitHub git sources. Rejected for Phase 1; may be added in Phase 2 for richer release-notes UX behind a separate opt-in.
- **Renderer-side fetch (browser `fetch`)** — would violate the architecture boundary (renderer is render-only, all I/O via preload bridge to Go) and bypass the central network gate. Rejected.
- **Trigger updates by side-effect during normal scan** — couples a fast local operation to a slow remote one and would make "scan" itself a network operation. Rejected. Keep scan offline; update-check is a separate explicit user action.
- **Cache TTL = 0 (always fresh)** — punishes the user with latency on every render of the screen. Rejected. 6h matches user-perceived "today" without surprise staleness.
- **Cache TTL = days** — risks showing very stale data without a freshness signal. Rejected. 6h + visible "Checked X minutes ago" copy is the better balance.
- **Allow Skillbox to contact a central updates server we operate** — would be the simplest aggregation point but introduces a Skillbox-controlled outbound endpoint, telemetry temptation, and the need for us to scrape every plugin source. Rejected — keeps Skillbox out of the data path between user and plugin authors.

## Consequences

**Positive:**

- Users can see "X updates available" without leaving Skillbox.
- Local-first invariant preserved in spirit: out-of-the-box experience is still 100% offline; network is a conscious choice.
- Architecture boundaries respected (renderer → preload → Go core → network).
- No new third-party SDK; `git` is already required to use the app meaningfully.
- Future Phase 2 (auth for private repos, scheduled checks, GitHub release notes) can be added incrementally on top of the same gate.

**Negative / costs:**

- First time the project allows outbound network — new failure modes (DNS, TLS, host down, captive portals, corporate proxies). Must be handled without blocking the rest of the UI.
- New SQLite table + cache invalidation logic.
- New Settings surface, new "Network activity" log view.
- `git ls-remote` cost: ~1 process per plugin source per check. For 200+ plugins this is non-trivial; mitigated by the 4-concurrent cap and 6h cache.
- Privacy: plugin source hosts now learn the user's IP and the timing of their checks. Documented in the opt-in dialog.

**Neutral / monitor:**

- If users overwhelmingly opt-in, the design assumption that "network is rare" weakens, and we may want to revisit caching, batching, and possibly an authenticated mode.
- `git` CLI behavior varies by version; the adapter must tolerate older versions (>= 2.20 should be safe — verify in Phase 2).
- The 8s/60s timeouts are educated guesses; revisit after dogfooding.

## Implementation Notes

(For the Phase 1 implementation PR — NOT this ADR.)

- **New files:**
  - `core-go/internal/network/update_check_client.go` — interface + `gitLsRemoteClient` implementation.
  - `core-go/internal/services/update_check_service.go` — orchestrator: read installed_plugins.json + marketplace.json → compute remotes → call client → diff → persist cache → return DTO.
  - `core-go/internal/repositories/update_check_cache_repo.go`.
  - `core-go/migrations/000022_plugin_update_check_cache.up.sql` / `.down.sql` —
    ```sql
    CREATE TABLE plugin_update_check_cache (
      id INTEGER PRIMARY KEY,
      provider_key TEXT NOT NULL,
      plugin_name TEXT NOT NULL,
      marketplace_name TEXT NOT NULL,
      source_url TEXT NOT NULL,
      source_ref TEXT,
      installed_sha TEXT,
      installed_version TEXT,
      remote_sha TEXT,
      remote_latest_tag TEXT,
      update_available INTEGER,                   -- 0/1/NULL for unknown
      checked_at TEXT NOT NULL,
      error TEXT,
      UNIQUE(provider_key, plugin_name, marketplace_name)
    );
    ```
  - `core-go/internal/rpc/handlers/update_check_run.go` — new RPC method `updateCheck.run` (manual trigger).
  - `apps/desktop/renderer/src/features/update-check/*` + UI integration in Updates screen.
- **Settings:** new keys under `app_settings` (or a new `network_settings` table — TBD in implementation). Default values must be the privacy-safe ones (`enabled=false`, `cache_ttl_hours=6`).
- **Process management:** use Go `exec.CommandContext` with the per-request 8s deadline; never inherit user env beyond what's needed (PATH, plus `GIT_TERMINAL_PROMPT=0` to suppress credential prompts on private repos — this is how "auth required" is detected: non-zero exit + recognizable stderr).
- **Allowlist:** the only URLs reachable are those already present in user's local marketplace.json files. Service must derive URLs from disk on every check, not from a long-lived cache, so that removing a plugin source also removes the host from the reachable set.
- **Logging:** structured log per request to `app_logs` (existing table if present, else new). Visible in Settings → Network activity. Never log query params or response bodies — just `host`, `result`, `duration_ms`.
- **[Larry-1] HTTPS-only enforcement:** Before spawning any subprocess, validate that the plugin source URL scheme is exactly `https`. Reject `git://`, `ssh://`, `http://`, `file://`, and any other scheme — mark the entry as `error: "non-https scheme rejected"` without contacting the host. This check happens inside `gitLsRemoteClient` so the interface itself enforces the invariant.
- **[Larry-2] Subprocess env stripping:** `exec.CommandContext` for `git ls-remote` sets `cmd.Env` explicitly to only `["PATH=<os-PATH>", "GIT_TERMINAL_PROMPT=0"]`. All other environment variables from the parent process are stripped. This prevents leaking tokens, credentials, or any secret env vars from the user's shell into the subprocess.
- **[Larry-3] git-not-found UX:** At service invocation time, look up `git` via `exec.LookPath("git")`. If not found, return a top-level error result `{status: "git_not_found", message: "git is required for update checks"}` without crashing. The UI surfaces this as a dismissible inline notice (not a modal error). The app continues to function normally in all other respects (airplane-mode-safe).
- **[Larry-4] Button rate-limit:** The "Check Updates" UI button is disabled for the duration of any in-flight `updateCheck.run` call (via `operationId` state), preventing overlapping subprocesses. Additionally, enforce a client-side minimum re-trigger interval of 10 seconds after a completed run to prevent process storms from rapid clicks.

## Verification

- ADR is accepted by user → digest "default OFF, opt-in, public-only Phase 1" into `AGENTS.md` core invariants block and `docs/10-technical-architecture.md` "Network" section.
- Phase 1 implementation:
  - Boot-time check: with setting OFF, `UpdateCheckClient` is replaced by a no-op stub at wiring time (compile-time assertion no other site uses real client when stub is in place).
  - Test: `go test ./... -run NetworkOffSmokesNoRemote` ensures with default settings, ANY call to `updateCheck.run` returns "disabled" without invoking the client.
  - Test: with setting ON + mock client, end-to-end flow returns expected `updateAvailable` per plugin.
  - Manual: airplane-mode dogfood. Skillbox must remain fully usable; only the "Check Updates" button surfaces a network-down state.
- Doc updates that block merge:
  - `AGENTS.md` invariant block amended (one line).
  - `docs/10-technical-architecture.md` adds "Outbound Network" subsection citing this ADR.
  - `docs/11-tech-stack-and-scaffold-decisions.md` records `git` CLI as a runtime dependency for update checks.
  - `docs/decisions/index.md` table updated.

## Scope phasing summary

| Phase | Includes | Excludes |
|---|---|---|
| **Phase 1 (this ADR)** | Default OFF; opt-in toggle; manual "Check Updates" button; `git ls-remote` only; public sources only; 6h cache; activity log; airplane-safe | Background polling; private repo auth; GitHub REST; release notes; non-plugin update checks |
| **Phase 2 (future ADR)** | Optional scheduled checks; private repo credential handling; possibly GitHub REST for richer release info | Telemetry of any kind; Skillbox-operated update server |
| **Out of scope (no ADR planned)** | Auto-update of plugins (would mutate user files); telemetry; account login | — |

## References

- `AGENTS.md` — Core invariants (local-first)
- `docs/10-technical-architecture.md` — Architecture boundaries
- `docs/08-provider-model.md` — Provider adapter contract (adapter returns facts; reads from disk only)
- `.scratch/spec-plugin-version.md` — How `installed_plugins.json` is currently read (no network)
- `.scratch/spec-plugin-version-project.md` — Project-layer version display spec (sibling work, network-free)
- PR #4 (`865c00e`) — Plugin version display feature (user layer, no network) — current baseline
