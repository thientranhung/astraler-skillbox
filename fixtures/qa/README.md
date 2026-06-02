# QA Fixtures

Reusable safe fixtures for release-grade QA runs. These folders are intentionally
small and non-sensitive. A QA run should copy them into its run folder before
execution, then mutate only the copied files.

Suggested run layout:

```text
docs/qa/runs/<run-id>/fixtures/
  skill-host-a/
  skill-host-b/
  projects/
  homes/
  edge-cases/
  release-cycle/
  provider-overrides/
  db/
  packaged-artifact/
```

Never point destructive QA cases at the repository fixture source directly.
Use the source fixtures as copy templates only.

Release-full P0 templates:

- `release-cycle/` covers host re-point, copy install, switch mode, operation
  deduplication, restart recovery, and full DB lifecycle cases.
- `provider-overrides/project-with-override/` covers provider override reset
  safety.
- `db/partial-migration/` creates a run-local dirty migration DB via script.
- `packaged-artifact/` is a marker for approved packaged artifact metadata; app
  bundles and DMGs are never committed as fixtures.
