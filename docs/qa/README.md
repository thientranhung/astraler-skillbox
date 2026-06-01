# QA Bank

Repo-native QA system for Astraler Skillbox. It is intentionally small: test
cases live as YAML, run results are append-only JSONL, and evidence stays under
the run folder.

## Why This Exists

Skillbox is local-first and filesystem-heavy. Unit tests and contract tests are
necessary, but they do not prove that the Electron UI behaves like the product
description. This QA bank gives agents a durable checklist for UI smoke,
cross-screen data consistency, and unsafe environment edges.

## Layout

```text
docs/qa/
  README.md
  schema.md
  invariants.yaml
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

## When To Run

| Moment | Scope |
|---|---|
| After feature implementation | Delta QA: cases tagged with the feature plus impacted T0 cases. |
| Before a large merge | Smoke QA: T0 core plus the main T1 flows. |
| Before release | Release QA: all T0/T1 cases plus packaged launch smoke. |
| After a bug fix | Regression QA: the bug reproduction case plus related cases. |

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
