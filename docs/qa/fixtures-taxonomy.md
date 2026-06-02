# Fixture Taxonomy (proposal)

A proposed layering for QA fixtures so the methods in
[`methodology.md`](methodology.md) have the malformed inputs they need. This is a
design proposal: it defines tiers and `mutate_copy` recipes expressed in the
existing [`schema.md`](schema.md) `data_setup` vocabulary. It does **not** create
fixture files or build automation yet — recipes are authored per case as needed.

All fixtures follow the existing Fixture Policy: reusable templates live under
`fixtures/qa/`, a run copies them into its run folder, and cases mutate only the
run-local copy.

## Tier 1 — base well-formed (exists today)

A provider project with a complete structure and at least one skill entry.
Used by the happy-path cases. No change.

## Tier 2 — malformed topology (new)

One template (or `mutate_copy` recipe) per structural edge from
[`methodology.md`](methodology.md) §2:

| Recipe name | Structure produced |
|---|---|
| `agents-parent-without-skills` | `.agents/` exists, `.agents/skills/` absent |
| `provider-folder-empty` | provider folder exists, contains nothing |
| `skills-is-file` | `.agents/skills` is a regular file, not a directory |
| `skills-broken-symlink` | `.agents/skills` symlinks to a deleted target |
| `skills-permission-denied` | provider folder present but not readable |
| `provider-marker-nested` | provider marker far below the project root |

Example `data_setup` for the partial-structure case:

```yaml
data_setup:
  fixture_source: fixtures/qa/projects/generic-agents-project
  copy_to: runs/<run-id>/fixtures/projects/generic-agents-project
  mutate_copy:
    - remove the .agents/skills directory, leaving .agents present
```

## Tier 3 — live-mutation recipes (new)

Recipes that mutate the run-local copy **during** a case, to drive state-machine
edges and the live-filesystem-mutation charter
([`charters/live-filesystem-mutation.md`](charters/live-filesystem-mutation.md)).
Expressed as ordered `mutate_copy` steps interleaved with `steps`:

| Recipe name | Mutation timing |
|---|---|
| `delete-skills-before-install` | remove `.agents/skills` after scan, before Install |
| `delete-provider-after-detect` | remove the provider folder after detection, before rescan |
| `swap-skills-to-file-mid-flight` | replace `.agents/skills` with a file between scan and install |

Example for the live-mutation regression case:

```yaml
data_setup:
  fixture_source: fixtures/qa/projects/generic-agents-project
  copy_to: runs/<run-id>/fixtures/projects/generic-agents-project
  mutate_copy:
    - scan project so the provider is detected
    - remove .agents/skills from the run-local copy
    - then attempt Add Skill install
```

## Not in scope yet

No fixture-generator scripts, no automated mutation harness, no CI wiring. Those
are deliberately deferred until the malformed-topology cases prove the recipes
are stable. Authoring recipes by hand in `data_setup` is sufficient for now.
