# QA Case Schema

Each file under `docs/qa/cases/` contains:

```yaml
area: projects
cases:
  - id: TC-PROJ-001
    title: Add a project and scan detected providers
    tier: T1
    primary_screen: Projects
    related_screens: [Project Detail, Dashboard]
    type: smoke
    tags: [project, scan, cross-screen]
    invariants: [INV-PROJECT-001]
    preconditions:
      - App runs in QA dev Electron mode.
    steps:
      - Open Projects.
    expected_ui:
      - Project appears in the list.
    cross_screen_checks:
      - Dashboard project count matches Projects.
    verifier:
      app_db:
        - Project row exists with the selected path.
      filesystem:
        - No files are written outside the selected project folder.
      evidence:
        - Screenshot after scan.
    safety:
      destructive: false
      allowed_environment: qa_fixture_or_read_only
      real_environment: allowed_read_only
```

## Required Fields

| Field | Purpose |
|---|---|
| `id` | Stable test case id. Use `TC-<AREA>-NNN`. |
| `title` | Human-readable behavior under test. |
| `tier` | `T0`, `T1`, `T2`, or `T3`. |
| `primary_screen` | Screen where the case starts. |
| `type` | `critical`, `smoke`, `edge`, `adversarial`, or `packaged-smoke`. |
| `tags` | Filtering keys for delta QA. |
| `preconditions` | Setup required before running. |
| `steps` | User-visible actions for the executor agent. |
| `expected_ui` | What the UI must show or not show. |
| `verifier` | Independent checks and evidence to collect. |
| `safety` | Destructive/read-only policy. |

## Optional Fields

| Field | Purpose |
|---|---|
| `related_screens` | Other screens that must reflect the same truth. |
| `invariants` | References into `docs/qa/invariants.yaml`. |
| `cross_screen_checks` | UI consistency checks across screens. |
| `data_setup` | Fixture data to create before steps. |
| `release_scope` | Optional scope marker: `current`, `future`, `manual`, or `not_applicable`. |
| `phase` | Optional phase marker when a case belongs to a planned product phase. |
| `notes` | Short notes for known limitations or manual judgment. |

## Fixture Policy

Reusable fixture templates live under `fixtures/qa/`. A QA run must copy those
fixtures into its run folder before executing cases. Cases may mutate only the
run-local copy, never the source fixture templates.

Use `data_setup` when a case needs specific copied fixture state, for example:

```yaml
data_setup:
  fixture_source: fixtures/qa/projects/claude-project
  copy_to: runs/<run-id>/fixtures/projects/claude-project
  mutate_copy:
    - create a broken symlink under .claude/skills
```

## Profiles

Run profiles live under `docs/qa/profiles/`.

| Profile | Purpose |
|---|---|
| `baseline-smoke` | Fast safety smoke: T0 plus critical T1. |
| `release-full` | First-release/release-candidate QA: all `release-full` cases across tiers, plus packaged smoke when an artifact exists. |
| `delta` | Feature-specific QA selected by tags, primary screen, and impacted invariants. |
| `packaged-release` | Packaged app launch, sidecar, app-data, signing/notarization, and artifact integrity checks. |

## Result JSONL

Each line in `runs/<run>/results.jsonl` is one case result:

```json
{"id":"TC-SKILL-003","status":"PASS","tier":"T0","started_at":"2026-06-01T10:00:00+07:00","evidence":["evidence/TC-SKILL-003-after.png","evidence/TC-SKILL-003-fs.txt"],"summary":"Symlink removed and host skill preserved."}
```

Allowed statuses:

- `PASS`
- `FAIL`
- `BLOCKED`
- `NEEDS_HUMAN`
- `SKIPPED`

Use `NEEDS_HUMAN` when the next step would touch real plugin/project/provider
state without explicit approval.

Optional result metadata:

| Field | Purpose |
|---|---|
| `decision_basis` | Short reason for the status when not obvious from the summary. |
| `waiver` | Owner-approved waiver details when a result is accepted with known residual risk. |
| `deferred_reason` | Reason a case is `SKIPPED` for current scope or phase. |
| `residual_risk` | Risk that remains after the result is accepted. |

Waivers are metadata, not a separate status. See [`governance.md`](governance.md).
