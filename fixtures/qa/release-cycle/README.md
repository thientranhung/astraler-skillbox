# Release Cycle Fixture

Template for T0 lifecycle cases that need a QA host, a QA project, and managed
install state. Copy this folder into a run folder before execution and mutate
only the copy.

Contents:

- `hosts/host-a` and `hosts/host-b`: distinct Skill Host Folder templates.
- `projects/project-with-provider`: project with an empty Shared Agent Skills
  provider target.
- `projects/project-with-install`: project template for seeded managed install
  cases. The runner should create symlinks inside the run-local copy because Git
  should not track machine-specific absolute symlink targets.
- `scripts/create-large-copy-skill.sh`: creates a run-local large fixture skill
  for restart-during-install QA. The generated payload is never committed.

For `TC-OPS-007`, copy this fixture into the run folder first, then run:

```sh
fixtures/qa/release-cycle/scripts/create-large-copy-skill.sh \
  docs/qa/runs/<run-id>/fixtures/release-cycle 512
```

The script writes `hosts/host-a/.agents/skills/zz-large-copy-skill/` in the
run-local copy. Use that skill with Copy (rsync) mode and poll operation state
before terminating the app.
