# Escape: plugin layer provenance mismatch (2026-06-09)

Recorded per the escape-analysis loop in [`../methodology.md`](../methodology.md) §7.

## What happened

A real user compared the Codex plugin state across two screens:

- Global Plugins showed several plugin rows as `enabled`.
- Project Detail > Provider Plugins showed the same provider/plugin rows with
  `Project=enabled`, `Global=disabled`, and `Effective=enabled`.

`superpowers` was disabled in both views and was not part of the defect.

The specific plugin names were incidental fixture/user data. The defect class is
generic: Project Detail's Global column and Global Plugins must agree for the
same provider/plugin because both represent the global/user layer.

## Defect class

| Symptom | Defect class | Invariant |
|---|---|---|
| Project Detail displayed a misleading Global column when a project override existed | Cross-screen layer provenance mismatch | `INV-PLUGIN-001` |
| Effective state was correct, but source-layer display was incomplete or misleading | Source-truth / display provenance gap | `INV-PLUGIN-001`, `INV-DB-001` |

## Escape analysis

1. **Why did prod have it but QA did not?** Existing plugin cases checked
   cross-screen agreement, but did not require a source-anchored verifier for
   each displayed layer. Screenshot comparison alone let a layer-provenance bug
   pass.
2. **Which case class was missing?** A plugin layer matrix case: global/user
   enabled or disabled crossed with project override enabled, disabled, or no
   override.
3. **Which method would have generated it?** FS/DB/UI failure matrix and
   pairwise selection from [`../methodology.md`](../methodology.md) §§3 and 5.
4. **Earliest cheap detection point?** A DB/source verifier attached to
   `TC-PLUGIN-003`, querying global/user layer and project layer separately
   while comparing Global Plugins and Project Detail.
5. **What changes so the class cannot recur?** Tighten `INV-PLUGIN-001`, add a
   layer matrix regression case, and require DB/settings evidence for plugin
   provenance checks.

## Coverage changes

- `INV-PLUGIN-001` now states that Global Plugins must reflect only the
  global/user layer and must match Project Detail's Global column, not Effective.
- `TC-PLUGIN-003` now requires DB/source evidence for global/user and project
  layer facts.
- `TC-PLUGIN-011` covers the generic layer matrix across Global Plugins and
  Project Detail.
