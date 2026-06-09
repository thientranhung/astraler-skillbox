# Escape: project scan used stale global plugin facts (2026-06-09)

Recorded per the escape-analysis loop in [`../methodology.md`](../methodology.md) §7.

## What happened

A real user installed or enabled a provider plugin at the global/user layer, then
opened Project Detail and clicked the project scan action. The newly added
global plugin did not appear in Project Detail until the user separately opened
Global Plugins and ran Scan Global.

The defect was not provider-specific. The reported example used Claude Code, but
the same class applies to any provider whose Project Detail effective plugin
view combines global/user facts with project/local override facts.

## Defect class

| Symptom | Defect class | Invariant |
|---|---|---|
| Project Detail stayed stale after the user clicked Project Scan | Cross-screen freshness / scan composition gap | `INV-PLUGIN-001` |
| Effective plugin resolution depended on persisted global/user facts that Project Scan did not refresh | Source-truth currentness gap | `INV-PLUGIN-001`, `INV-DB-001` |

## Escape analysis

1. **Why did prod have it but QA did not?** Existing plugin cases checked
   cross-screen agreement after both scan paths were available, but did not force
   a stale or absent global scan followed by Project Scan only.
2. **Which case class was missing?** A scan-order/currentness case: mutate the
   global/user source, avoid Global Plugins scan, run Project Detail Scan, and
   require Project Detail to refresh every layer needed for its own truth.
3. **Which method would have generated it?** State freshness matrix and
   workflow-order permutation from [`../methodology.md`](../methodology.md) §§3
   and 5.
4. **Earliest cheap detection point?** A service test for `ScanProjectLayers`
   plus a QA verifier that checks user/global and project layer scan timestamps
   after Project Detail Scan.
5. **What changes so the class cannot recur?** Project Scan now refreshes the
   user/global plugin layer before resolving project/local layers, and
   `TC-PLUGIN-012` covers this scan-order regression.

## Coverage changes

- `ScanProjectLayers` now commits user/global plugin layer facts before
  project/local plugin layer facts.
- `TC-PLUGIN-012` requires Project Detail Scan to show newly added global plugins
  before any separate Global Plugins scan.
- Unit coverage verifies that Project Detail's effective plugin view includes a
  global-only plugin after `ScanProjectLayers`.
