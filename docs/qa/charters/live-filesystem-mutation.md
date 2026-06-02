# Charter: live filesystem mutation

A session-based exploratory charter for the class of bugs where provider
structure on disk changes **while the app is running**. This is the cheapest
detection point for the 2026-06-02 escape
([`../escapes/2026-06-02-provider-partial-structure.md`](../escapes/2026-06-02-provider-partial-structure.md)).

A charter is a mission with guardrails, not a step script. Every finding is
converted into a durable YAML case before the charter is considered done — see
[`../methodology.md`](../methodology.md) §6.

## Charter template

```
Charter:    <one-sentence mission>
Areas:      <screens / providers / operations in scope>
Invariants: <INV-... the charter is stressing>
Time-box:   30–45 min
Fixtures:   run-local QA copies only (never real user data)
Safety:     qa_fixture_only; stop and mark NEEDS_HUMAN before touching real data
Oracle:     after each mutation, the relevant invariant must still hold
Output:     findings list -> new adversarial YAML cases linked to the invariants
```

## Example charter (this escape class)

```
Charter:    Mutate provider structure on disk while the app is running and find
            any state where the UI, DB, or filesystem disagree, or an operation
            fails to reach a terminal state.
Areas:      Projects, Project Detail, Add Skill Wizard; provider generic_agents;
            scan + install operations.
Invariants: INV-PROJECT-001, INV-OPERATION-001, INV-INSTALL-001, INV-DB-001.
Time-box:   45 min.
Fixtures:   run-local copy of a generic_agents project fixture.
Safety:     qa_fixture_only.
Oracle:     after each mutation + rescan, provider facts on disk == DB ==
            every screen; no operation stays queued/running.
```

### Moves to try

- Detect a provider, then delete `.agents/skills` (or all of `.agents`) on disk,
  then rescan — facts must reset, not go stale.
- Delete the skills folder *between* opening Add Skill and clicking Install.
- Replace `.agents/skills` with a file, or a broken symlink, then scan.
- `chmod` the provider folder unreadable, then scan.
- Rename the provider folder mid-scan, if the scan is slow enough to race.
- Restart the app while an install/scan shows "in progress".

### What to record per finding

- Exact mutation and timing relative to the operation.
- Which source (UI / DB / FS) disagreed, and how.
- Whether the operation reached a terminal state.
- The invariant violated → drafted as a new `type: adversarial` case.

Findings from this example charter are already captured as `TC-PROVIDER-006/007/008`
in [`../cases/provider-paths.yaml`](../cases/provider-paths.yaml). Re-run the
charter each release to catch new variants.
