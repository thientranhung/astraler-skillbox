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
```

Never point destructive QA cases at the repository fixture source directly.
Use the source fixtures as copy templates only.
