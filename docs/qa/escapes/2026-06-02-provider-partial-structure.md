# Escape: provider partial-structure detection + stale rescan (2026-06-02)

First production escape after the first release of Astraler Skillbox. Recorded
per the escape-analysis loop in [`../methodology.md`](../methodology.md) §7.

## What happened

A real user added a project folder that contained `.agents/` but **not**
`.agents/skills/`. Three distinct defects fired in sequence:

| # | Symptom | Defect class | Invariant |
|---|---|---|---|
| A | App detected the provider and presented provider facts even though the skills structure did not exist | Negative filesystem topology — partial structure | `INV-PROJECT-001` |
| B | Add Skill install entered an unclear, stuck "Installing" state with no terminal outcome | Async operation missing a terminal state | `INV-OPERATION-001`, `INV-INSTALL-001` |
| C | After the user deleted the provider folders, a rescan still showed the old provider facts | State staleness / rescan not idempotent | `INV-PROJECT-001`, `INV-DB-001` |

## Fix reference

- PR #27 — *Fix provider folder install regressions* (commit `d0be2f9`).
- Branch: `codex/project-provider-regressions` (base `main`).

## Escape analysis (five questions)

1. **Why did prod have it but QA did not?** Every provider fixture in the bank
   was well-formed — a provider folder *with* a skills subfolder and at least one
   skill entry. No fixture modelled a partial or malformed structure.
2. **Which case class was missing?** Three: negative filesystem topology (A),
   async terminal-state (B), and state-staleness / FS↔DB disagreement (C).
3. **Which method would have generated it?** State models (§1, the
   `installed --fs-deleted--> rescan` and `running --target-vanishes--> failure`
   edges), negative topology (§2, `.agents` without `.agents/skills`), and the
   FS×DB×UI failure matrix (§3, the "DB present / FS gone" cell).
4. **Earliest cheap detection point?** A live-filesystem-mutation exploratory
   charter that changes provider structure while the app is running — it would
   have surfaced A and C directly. See
   [`../charters/live-filesystem-mutation.md`](../charters/live-filesystem-mutation.md).
5. **What changes so the class cannot recur?** A malformed-topology fixture tier
   ([`../fixtures-taxonomy.md`](../fixtures-taxonomy.md)), the async terminal-state
   rule, and a recurring live-mutation charter — all now documented in the
   methodology.

## Coverage (cases protecting this escape)

Added by **PR #27** (in `cases/skills-and-projects.yaml`):

- `TC-PROJ-003` — rescan marks removed provider folders missing. *(covers C)*
- `TC-SKILL-007` — Shared Agent install creates missing `.agents/skills` without
  hanging. *(covers B and the install path of A)*

Added by **this PR** (in `cases/provider-paths.yaml`), closing the gaps PR #27
did not cover:

- `TC-PROVIDER-006` — scan of a project with `.agents` but no `.agents/skills`
  presents an accurate "present, incomplete, zero skills" state and claims no
  phantom skills. *(covers A as a pure detection-truth case, distinct from the
  install path in TC-SKILL-007)*
- `TC-PROVIDER-007` — `.agents/skills` is a file or a broken symlink: scan
  reaches a terminal empty/error state with no DB drift. *(negative-topology
  variants beyond the single missing-folder case)*
- `TC-PROVIDER-008` — `.agents/skills` deleted immediately before install (live
  mutation): the install reaches a terminal state and the UI does not stay stuck.
  Because `generic_agents` may recreate `.agents/skills`, success is allowed only
  if the target is recreated and DB/FS agree; otherwise it ends in a clear
  terminal failure with no active install. *(charter-derived; the race that A+B+C
  together hinted at)*

## Method changes shipped with this record

- New [`../methodology.md`](../methodology.md): state models, negative topology,
  FS×DB×UI matrix, async terminal-state rule, pairwise selection, charters.
- New [`../fixtures-taxonomy.md`](../fixtures-taxonomy.md): malformed-topology
  fixture tier and live-mutation recipes.
- New [`../charters/live-filesystem-mutation.md`](../charters/live-filesystem-mutation.md).
