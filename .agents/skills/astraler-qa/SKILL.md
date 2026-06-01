---
name: astraler-qa
description: "Run and maintain Astraler Skillbox's repo-native QA bank. Use when creating QA run plans, selecting YAML test cases, running Electron UI smoke via agent-browser/CDP, collecting DB/FS/screenshot evidence, writing results.jsonl/report.md, or adding regression cases after bugs."
---

# Astraler QA

Use this skill to operate the QA bank in `docs/qa/`. Keep the workflow lean:
YAML test cases are the source of truth, JSONL stores run results, and evidence
files prove what happened.

## Read First

1. `docs/qa/README.md`
2. `docs/qa/schema.md`
3. `docs/qa/invariants.yaml`
4. Relevant `docs/qa/cases/*.yaml`
5. `docs/playbooks/agent-browser-smoke.md` before browser automation

## Case Selection

- Feature/delta QA: select cases by `tags`, `primary_screen`, and impacted
  invariants. Always include impacted T0 cases.
- Smoke QA: all T0 plus the main T1 flows touched by the change.
- Release QA: all T0/T1 cases.
- Bug regression: the bug reproduction case plus nearby cases with the same
  tags/invariants.

Do not load every case file if the run is clearly scoped. Use `rg` for ids,
tags, tiers, and screens.

## Electron Dev Run

Primary UI QA runs target the dev Electron app with the real Go sidecar.

1. Create a run folder:
   `docs/qa/runs/<YYYY-MM-DD-scope>/evidence/`
2. Copy or adapt `docs/qa/run-plan-template.yaml` into the run folder.
3. Start the app from `apps/desktop` with QA env:
   - `SKILLBOX_DB_PATH=<qa_home>/qa.db`
   - `HOME=<qa_home>`
   - preserve real `GOCACHE`, `GOMODCACHE`, and `GOPATH` if using `go run`
4. Confirm CDP:
   `curl -s http://127.0.0.1:49222/json/version`
5. Attach agent-browser to the running Electron instance. Never launch a second
   browser for app smoke.
6. Execute case steps as the user would.
7. Save screenshots, DB query output, filesystem checks, and notes under
   `evidence/`.
8. Append one JSON object per case to `results.jsonl`.
9. Write/update `report.md` from `docs/qa/report-template.md`.

## Result Protocol

Allowed statuses:

- `PASS`
- `FAIL`
- `BLOCKED`
- `NEEDS_HUMAN`
- `SKIPPED`

Use `FAIL` when observed behavior violates expected UI, cross-screen checks, or
verifier checks. Use `BLOCKED` when setup/tooling prevents execution. Use
`NEEDS_HUMAN` when the next step would touch real project/plugin/provider data
without explicit user approval.

Append-only result example:

```json
{"id":"TC-SKILL-003","status":"PASS","tier":"T0","evidence":["evidence/TC-SKILL-003-after.png","evidence/TC-SKILL-003-fs.txt"],"summary":"Project symlink removed; host skill preserved."}
```

## Safety

- `real_environment: forbidden` means stop if the target is not a QA fixture.
- `real_environment: opt_in` requires explicit user approval for the exact
  target and operation.
- Plugin QA is limited to scan version, toggle enabled state, and cross-screen
  display consistency. Do not install or delete real plugins.
- T0 cases require out-of-band evidence when available: DB, filesystem, and
  screenshot/DOM.

## Updating The Bank

When a bug is found:

1. Record the failing run result.
2. Propose a new regression case or update an existing case.
3. Link the case to the relevant invariant and tags.
4. Keep case language product-facing; avoid implementation-only assertions
   unless they are verifier evidence.

Prefer adding one focused case over a broad umbrella case.
