# QA Bank

Repo-native QA system for Astraler Skillbox. It is intentionally small: test
cases live as YAML, run results are append-only JSONL, and evidence stays under
the run folder.

For status semantics, evidence standards, waivers, T0 handling, and clean GO
rules, read [`governance.md`](governance.md).

## Why This Exists

Skillbox is local-first and filesystem-heavy. Unit tests and contract tests are
necessary, but they do not prove that the Electron UI behaves like the product
description. This QA bank gives agents a durable checklist for UI smoke,
cross-screen data consistency, and unsafe environment edges.

## Quick Start

Use this section when you are new and just need to run QA.

### Run QA For One Feature

Example: a feature changed plugin toggle behavior.

1. Pick a run id:
   `RUN_ID=2026-06-01-plugin-toggle`
2. Create the run folder:
   ```sh
   mkdir -p "docs/qa/runs/$RUN_ID/evidence"
   cp docs/qa/run-plan-template.yaml "docs/qa/runs/$RUN_ID/run-plan.yaml"
   cp docs/qa/report-template.md "docs/qa/runs/$RUN_ID/report.md"
   touch "docs/qa/runs/$RUN_ID/results.jsonl"
   ```
3. Find relevant cases:
   ```sh
   rg -n "plugin|toggle|tier: T0|tier: T1" docs/qa/cases
   ```
4. Edit `docs/qa/runs/$RUN_ID/run-plan.yaml`:
   - set `scope: delta`
   - set QA paths under `environment`
   - add selected case ids under `selection.case_ids`
5. Start the dev Electron app with QA env.
6. Attach agent-browser to CDP.
7. Run the selected case steps.
8. Save screenshots/query outputs/logs under `evidence/`.
9. Append one result JSON object per case to `results.jsonl`.
10. Update `report.md` with the final verdict.

### Run Full QA For The First Baseline

Use this before the first release or when validating the QA bank itself.

1. Create a run folder with `RUN_ID=YYYY-MM-DD-full-baseline`.
2. Select all T0/T1 cases:
   ```sh
   rg -n "tier: T0|tier: T1" docs/qa/cases
   ```
3. Run T0 first, then T1.
4. Mark unclear or unsafe cases as `BLOCKED` or `NEEDS_HUMAN`; do not force a
   pass.
5. Update the QA bank when a case is unclear, missing setup, or missing an
   invariant.

For a first release, prefer the stronger release profile:

1. Create `RUN_ID=YYYY-MM-DD-release-full`.
2. Copy `fixtures/qa/` into the run folder.
3. Select all cases tagged `release-full`, ordered T0, T1, T2, T3.
4. Run automated gates first.
5. Run dev Electron QA with QA env and CDP.
6. Run packaged smoke after a release artifact exists.
7. Mark unsafe, unclear, or artifact-dependent cases `BLOCKED` or
   `NEEDS_HUMAN`; do not force a pass.

### Ask An Agent To Run QA

Use the `astraler-qa` skill:

```text
Use astraler-qa to create a delta QA run for plugin toggle changes. Select the
relevant T0/T1 cases, run them against dev Electron with QA fixtures, collect
evidence, and write results.jsonl plus report.md.
```

The agent should read `.agents/skills/astraler-qa/SKILL.md`, this README, the
case files, and `docs/playbooks/agent-browser-smoke.md`.

## Layout

```text
docs/qa/
  README.md
  governance.md
  schema.md
  invariants.yaml
  profiles/
    release-full.yaml
  cases/
    setup-and-settings.yaml
    skills-and-projects.yaml
    plugins.yaml
  runs/
    <YYYY-MM-DD-release-or-scope>/
      run-plan.yaml
      results.jsonl
      report.md
      evidence/
```

## Risk Tiers

| Tier | Meaning | Release behavior |
|---|---|---|
| T0 | Data integrity or destructive behavior. Failures can lose data, write to the wrong path, or make DB/filesystem disagree. | Blocks release. |
| T1 | Core user journey. Failures break normal product use or cross-screen truth. | Usually blocks release unless user accepts a workaround. |
| T2 | Secondary workflow. | Record and triage. |
| T3 | UX polish or minor state. | Record unless misleading or blocking. |

The current bank focuses on T0 and T1.

## Profiles

Use profiles to avoid overloading the phrase "full QA":

| Profile | Scope |
|---|---|
| `baseline-smoke` | Fast core safety run: all T0 and the core T1 flows touched by a change. |
| `release-full` | First-release and release-candidate run: all cases tagged `release-full`, ordered T0 → T1 → T2 → T3, plus packaged smoke when an artifact exists. |
| `delta` | Feature/change-specific run selected by tags, screens, and impacted invariants. |
| `packaged-release` | Packaged app launch, sidecar, app-data, CDP, signing/notarization, manifest, and orphan-process checks. |

The first release should use `docs/qa/profiles/release-full.yaml`. A run may
still mark packaged cases `BLOCKED` if no artifact exists, but the report must
make that release risk explicit.

## Fixtures

Reusable safe fixture templates live under `fixtures/qa/`. Always copy them into
the run folder before execution:

```sh
RUN_ID=2026-06-01-release-full
mkdir -p "docs/qa/runs/$RUN_ID/fixtures"
rsync -a fixtures/qa/ "docs/qa/runs/$RUN_ID/fixtures/"
```

Do not mutate source fixtures in place. Destructive cases must use the copied
run-local fixture paths and must keep `real_environment_allowed: false` unless a
case explicitly allows `opt_in` and the user approved that exact target.

## When To Run

| Moment | Scope |
|---|---|
| After feature implementation | Delta QA: cases tagged with the feature plus impacted T0 cases. |
| Before a large merge | Smoke QA: T0 core plus the main T1 flows. |
| Before release | Release QA: all T0/T1 cases plus packaged launch smoke. |
| After a bug fix | Regression QA: the bug reproduction case plus related cases. |

## Start A Run

Create one folder per QA run. The folder is the run's source of truth: plan,
append-only results, final report, and evidence.

```sh
RUN_ID=2026-06-01-full-baseline
mkdir -p "docs/qa/runs/$RUN_ID/evidence"
cp docs/qa/run-plan-template.yaml "docs/qa/runs/$RUN_ID/run-plan.yaml"
cp docs/qa/report-template.md "docs/qa/runs/$RUN_ID/report.md"
touch "docs/qa/runs/$RUN_ID/results.jsonl"
```

Run folder rules:

- `run-plan.yaml` records scope, environment, selected cases, and gates.
- `results.jsonl` gets one JSON object per case, appended as cases finish.
- `report.md` is the human summary for GO / CAUTION / NO-GO.
- `evidence/` stores screenshots, DB query output, filesystem checks, logs, and
  notes.
- Do not overwrite previous run folders. Create a new `RUN_ID` for every QA
  attempt.
- Run-local fixture copies, temporary homes, external target sandboxes, caches,
  module downloads, generated build artifacts, and raw `evidence/` files are
  disposable run artifacts and ignored by git by default. Commit `run-plan.yaml`,
  `results.jsonl`, and `report.md` for durable run summaries; add raw evidence
  only when it is intentionally curated and small.

## Electron Automation

Primary QA runs use the dev Electron app with the real Go sidecar:

1. Start `pnpm dev` from `apps/desktop`.
2. Confirm CDP is live on `127.0.0.1:49222`.
3. Attach with agent-browser; do not launch a second browser.
4. Execute selected cases and write evidence under the active run folder.

Read `docs/playbooks/agent-browser-smoke.md` before browser automation.

Packaged app smoke is separate. It verifies boot, sidecar path, DB path, and a
small manual/agent-assisted smoke. Packaged builds intentionally do not expose
CDP by default.

## Safety Rules

- Never run destructive cases against real user data unless the case explicitly
  says `real_environment: opt_in` and the user approved that exact operation.
- Plugin cases are limited to scan version, toggle enabled state, and
  cross-screen display consistency. Do not install or delete real plugins.
- T0 verifier evidence must include out-of-band checks when available: DB query,
  filesystem state, and screenshot.
- T1 verifier evidence should include at least a screenshot and one independent
  check when the case affects persisted state.
