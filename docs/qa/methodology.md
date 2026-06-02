# QA Test-Case Generation Methodology

How to *generate* QA cases so severe state / filesystem / DB / UI / async-operation
bugs are caught before release. This is a test-design reference, not a process or
governance doc. For status semantics, waivers, and GO rules see
[`governance.md`](governance.md). For field names see [`schema.md`](schema.md).
For the rules each case protects see [`invariants.yaml`](invariants.yaml).

The bank already had cases for "bad input into a well-formed form". It missed
bugs where the *fixture itself* or the *state transition* was malformed. These
methods exist to close that gap. Each method ends with where its output lands in
the YAML schema.

## 1. State models (highest-value source of cases)

Most severe bugs live on a *transition edge*, not on a state node. Model each
long-lived entity as a small state machine and turn every edge — especially
abnormal edges — into a case.

**Operation** (scan / install / remove / plugin-write / update-check):

```
idle -> queued -> running -> { success | failure | cancelled }
                  running --app-restart--> (must resolve, never stay "running")
                  running --target-vanishes--> failure   <-- often missing
```

**Project provider facts:**

```
unknown -> detected -> { installed | error }
installed --fs-deleted-out-of-band--> stale -> rescan -> reset   <-- often missing
detected(partial structure) -> must show accurate "present, incomplete, 0 skills"
```

Rule: every abnormal edge needs a case. A `running` state with no asserted
terminal state, or an `installed` state with no "what if the target disappears"
edge, is a known gap.

→ YAML: multi-step `steps` walking the edge; `type: adversarial`;
`expected_ui` must assert the *terminal* state explicitly ("does not stay stuck").

## 2. Negative filesystem topology (not just bad values)

Negative testing must cover bad *structure*, not only bad *input strings*. For
every provider path the app reads, enumerate structural edges:

| Edge | Example |
|---|---|
| parent present, child absent | `.agents/` exists, `.agents/skills/` missing |
| empty | provider folder exists but contains nothing |
| wrong node type | `.agents/skills` is a file, not a directory |
| broken symlink | `.agents/skills` points at a deleted target |
| permission denied | provider folder not readable |
| nested / deep | provider marker far below the project root |

Each edge becomes a fixture variant (see [`fixtures-taxonomy.md`](fixtures-taxonomy.md))
and at least one case. The detection and scan layers must present an accurate
state for each — never claim skills that do not exist on disk.

→ YAML: `data_setup.fixture_source` + `mutate_copy`; `preconditions` state the
malformed structure plainly; `verifier.filesystem` proves the structure.

## 3. FS × DB × UI failure matrix

Severe bugs are usually *disagreement between sources of truth*. Lay out a matrix:
rows = source (UI, DB, filesystem); columns = abnormal state (missing / partial /
corrupt / out-of-sync). Every cell where two sources can disagree needs a case
whose `verifier` asserts **all three** sources, so drift is caught instead of
trusted.

The 2026-06-02 escape (see [`escapes/`](escapes/)) sat exactly in the cell
"DB says provider present / filesystem says gone".

→ YAML: pair `verifier.app_db` with `verifier.filesystem` in the same case;
add `cross_screen_checks` when the same truth must appear on multiple screens.

## 4. Async terminal-state rule

Any operation that can show an in-progress state MUST have at least one
adversarial case proving it reaches a terminal state (success / failure / cancel)
and that the UI does not stay stuck — including when the underlying target is
incomplete or vanishes mid-flight. Ties to `INV-OPERATION-001`.

→ YAML: `expected_ui` asserts "in progress only while running" + "does not stay
stuck"; `verifier.app_db` asserts no operation row remains `queued`/`running`.

## 5. Pairwise selection (control the explosion)

Factors that combine: `provider × scope(project|global) × fs-topology ×
install-mode × override-active?`. Do not run the full Cartesian product. Use
2-wise (pairwise) coverage so every *pair* of factor values appears in some case
— roughly 10–15 cases instead of hundreds. Give data-loss / stuck-state pairs T0
priority. Record explicitly (in the run report) any combination intentionally
dropped — silent truncation reads as "covered everything".

→ YAML: split into focused cases; encode the combination in `tags` so `delta`
runs can select them.

## 6. Exploratory charters (find unknown-unknowns)

Negative topology is a known-unknown; the worst bugs are unknown-unknowns.
Each release runs one or two time-boxed (30–45 min) charters. A charter is a
mission, not a script; every finding is converted into a durable YAML case (the
charter is a case *generator*, not a substitute). See
[`charters/`](charters/) for the template and the live-filesystem-mutation
example that reproduces this escape class.

→ YAML: charter output = new `type: adversarial` cases linked to the invariants
the charter stressed.

## 7. Bug-escape analysis (the improvement loop)

For every production escape, answer five fixed questions and require that the
conclusion changes the *method*, not just adds one case:

1. Why did prod have it but QA did not?
2. Which case *class* was missing (state edge / topology / matrix cell / async)?
3. Which method here would have generated it?
4. Where is the earliest cheap detection point (often a charter)?
5. What changes so the whole class cannot recur?

Record each escape under [`escapes/`](escapes/) using the existing escape as the
template.

## Method → schema cheat-sheet

| Method | Primary YAML fields |
|---|---|
| State transition (§1) | multi-step `steps`, `type: adversarial`, terminal `expected_ui` |
| Negative topology (§2) | `data_setup.mutate_copy`, `preconditions`, `verifier.filesystem` |
| Failure matrix (§3) | `verifier.app_db` + `verifier.filesystem`, `cross_screen_checks` |
| Async terminal (§4) | `expected_ui` not-stuck, `verifier.app_db` no live op rows |
| Pairwise (§5) | many small cases, `tags` encoding the combination |
| Charter (§6) | new cases under the stressed `invariants` |
| Escape analysis (§7) | `escapes/<date>-<slug>.md` + regression cases |

Writing rules (keep the bank healthy): product-facing language in
`steps`/`expected_ui`; implementation assertions only inside `verifier`; every
T0 carries DB + filesystem + screenshot evidence; prefer several narrow cases
over one umbrella case; every invariant should have at least one happy case and
one violation-attempt case.
