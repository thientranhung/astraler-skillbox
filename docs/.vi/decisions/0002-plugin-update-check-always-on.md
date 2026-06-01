# ADR-0002: Plugin Update-Check Always-On (remove network opt-in gate)

- **Status:** accepted
- **Date:** 2026-05-31
- **Deciders:** user (thienth@astraler.com), Tom (planner)
- **Tags:** architecture | product | network | privacy
- **Supersedes:** [ADR-0001](./0001-outbound-network-update-check.md) (partial — the default-OFF opt-in gate only)

## Context

ADR-0001 introduced plugin update-check behind a default-OFF opt-in setting
(`network.update_check.enabled`). Bringing the feature to release revealed that the gate
**never worked end-to-end** and could not be turned on through the product:

- **Backend gate (B1).** `UpdateCheckService.RunUpdateCheck` returned `status:"disabled"`
  whenever the setting was off.
- **Seed OFF (B2).** Migration `000022` seeded `update_check_enabled = 0` on every new DB,
  so the gate was off on every install.
- **Client hardwired to no-op (B3).** `cmd/skillbox-core/main.go` always wired
  `network.NoopClient{}` (which returns `update_check_disabled` for every plugin). The real
  `GitLsRemoteClient` was never wired anywhere — so even with the setting forced on, the
  feature returned errors for every plugin. **The feature had never run end-to-end.**
- **No UI toggle.** The Settings → Network toggle was removed in earlier UI cleanup, leaving
  `NetworkSettingsRepo.SetUpdateCheckEnabled` as dead code with no caller — there was no way
  to enable the feature.
- **Reset re-disabled.** `ResetAllData` set `update_check_enabled = 0` on every Reset.
- **Architecture smell.** The setting was read at call-time but the client was chosen once at
  boot-time, so even a hypothetical runtime toggle could never swap the no-op client for the
  real one.

The gate was therefore pure over-engineering: complexity guarding a privacy promise it never
actually delivered, with no path to enable and a permanently-stubbed client.

## Decision

1. **Plugin update-check is always-on.** Remove the `update_check_enabled` gate entirely.
   `RunUpdateCheck` no longer reads any enable setting; it runs whenever the user triggers it.
2. **Manual-trigger-only is retained.** There is still **no background polling and no
   auto-check at launch** for plugin update-check. Network contact only happens when the user
   clicks **"Check Updates"** on the Plugins screen. This preserves the meaningful privacy
   property: the app contacts plugin source hosts only on an explicit user action.
3. **Wire the real client.** `main.go` wires `network.NewGitLsRemoteClient()` so the feature
   actually works.
4. **Drop the gate column.** Migration `000023` drops `network_settings.update_check_enabled`.
   The `network_settings` table is retained for `cache_ttl_hours`.
5. **Remove dead code.** Delete `network.NoopClient`, `NetworkSettingsRepo.SetUpdateCheckEnabled`,
   and the `UpdateCheckEnabled` domain field. Drop the `"disabled"` status from the
   `updateCheck.run` contract enum and the renderer status union (core and renderer ship
   together, so there is no version-skew concern).
6. **App-update check is unchanged.** `app.checkUpdate` (the About-screen "Check for Updates")
   was already always-on and never gated; the dead `network_disabled` error code is removed
   from its contract description for clarity.

**All ADR-0001 safeguards remain in force:** HTTPS-only enforcement, subprocess env stripping,
git-not-found graceful degradation, per-request/batch timeouts, host backoff, allowlist derived
from disk on every call, 6h cache TTL, **no telemetry, and no Skillbox-operated server.**

## Consequences

**Positive:**

- The feature works end-to-end for the first time.
- One source of truth; no dead gate, no boot-time/call-time contradiction, no unreachable code.
- Matches the project's "no over-engineering" philosophy — stops dressing a non-functional
  opt-in as a privacy feature.

**Trade-off / privacy:**

- The invariant "outbound network is OFF by default" changes to "the only outbound network is
  manual-trigger plugin update checks." The app remains 100% usable offline; network contact
  still happens **only when the user clicks "Check Updates"**, against hosts already present in
  the user's local `marketplace.json`. This is a **product/privacy decision approved by the
  user**, not a unilateral implementation choice.

**Migration:**

- `000023` uses `ALTER TABLE ... DROP COLUMN` (requires SQLite ≥ 3.35; the bundled
  `modernc.org/sqlite` v1.50.1 → SQLite 3.53.1 supports it; already used in `000013`/`000021`).
- The down migration re-adds the column (`DEFAULT 0`, row set to `1`) so a rollback to
  pre-ADR-0002 code does not silently re-disable the feature. Data dropped on the up migration
  is unrecoverable but the column was an unused gate.

## Alternatives Considered

- **Option B — minimal fix (keep the plumbing, seed `= 1`).** Rejected: B3 (the hardwired
  no-op client) forces the same wiring change as Option A, so "minimal" is illusory; it leaves
  a meaningless gate (always on, no UI, real client) plus the boot/runtime contradiction.
- **Keep the gate, build a real toggle UI.** Rejected: re-introduces an opt-in for a
  table-stakes, manual-only check against hosts the user already installed — complexity with no
  user value. Can be revisited via a new ADR if a concrete privacy requirement emerges.

## References

- [ADR-0001](./0001-outbound-network-update-check.md) — original gated design (superseded in part)
- Spec: `docs/superpowers/specs/2026-05-30-remove-update-check-gate-design.md` — root-cause
  analysis (B1/B2/B3) and Option A vs B
- `AGENTS.md` — core invariants (network)
- `docs/10-technical-architecture.md` — Outbound Network section
