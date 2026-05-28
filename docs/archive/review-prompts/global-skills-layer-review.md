# Global Skills Layer Review Prompt

You are reviewing the newly added Global Skills layer in Astraler Skillbox.

Astraler Skillbox now models three distinct layers:

```text
Skill Host Folder
  Source of truth/library managed by Skillbox.

Global Skills
  Provider global-level skills/config on the user's machine.

Project Installs
  Skills installed into specific project/provider scopes.
```

## Read These Files In Order

Read these files in order before reviewing:

1. `README.md`
2. `docs/index.md`
3. `docs/01-product-brief.md`
4. `docs/02-product-notes.md`
5. `docs/03-information-architecture.md`
6. `docs/04-user-flows.md`
7. `docs/05-edge-cases-and-ux-states.md`
8. `docs/06-data-model.md`
9. `docs/07-schema-dictionary.md`
10. `docs/08-provider-model.md`
11. `docs/09-ui-wireframes.md`
12. Existing review results under `docs/archive/review-results/` if needed for context.

## Review Goal

Review only the Global Skills layer addition. Do not re-review unrelated product
areas unless they conflict with this layer.

Focus on whether the docs now clearly distinguish:

- Skill Host Folder as source of truth.
- Global Skills as provider global-level state.
- Project Installs as project/provider-level state.

## Questions To Answer

Please answer these questions:

1. Is the Global Skills concept clearly defined?
2. Is it clearly distinct from Skill Host Folder?
3. Is it clearly distinct from Project Installs?
4. Do `global_provider_locations` and `global_installs` model the concept well?
5. Is the duplication between `installs` and `global_installs` acceptable, or
   should the model use a polymorphic install table?
6. Are global skill edge cases sufficiently covered?
7. Are global skill user flows sufficiently covered?
8. Does the Provider Model explain global scan boundaries clearly enough?
9. Does the UI Wireframe give Global Skills enough surface area?
10. Are review prompts updated enough so future reviews do not miss Global
    Skills?
11. Are there contradictions across docs?
12. What should be changed before implementation begins?

## Required Output File

Write your review result to:

```text
docs/archive/review-results/global-skills-layer-review.md
```

If the file already exists, overwrite it.

## Required Output Format

Use this exact structure:

```markdown
# Global Skills Layer Review Result

## Reviewer

- Agent/model:
- Review date:
- Context used:
- Browsing used: yes/no

## Executive Summary

Short summary of the review result.

## Decision

Approved / Not approved.

## Critical Issues

List blockers that should be fixed before implementation.

## Suggested Improvements

List useful improvements that are not blockers.

## Concept Separation Review

Assess whether these are clearly separated:

- Skill Host Folder
- Global Skills
- Project Installs

## Data Model Review

Assess:

- `global_provider_locations`
- `global_installs`
- Relationship with `skills`
- Relationship with `provider_definitions`
- Warning scopes
- Whether separate `global_installs` vs polymorphic installs is the right call

## Schema Dictionary Review

Assess whether field-level descriptions are clear and AI/developer friendly.

## User Flow Coverage

Assess the `Scan Global Skills` flow and any missing flows such as removing,
relinking, syncing, or configuring global locations.

## Edge Case Coverage

Assess global location states, global direct installs, broken/external symlinks,
and global/project overlap.

## Provider Model Coverage

Assess whether provider adapters have enough guidance for global provider
locations and global scan.

## UI/UX Coverage

Assess whether Global Skills is represented clearly in navigation, dashboard,
skill detail, settings, empty states, warnings, and impact previews.

## Contradictions Across Docs

List any contradictions or terminology mismatches.

## Recommended Changes

Provide concrete recommended changes with file/section references.

## Open Questions For The Product Owner

List questions that require product decisions.

## What Looks Solid

List parts of the Global Skills layer that are well designed.
```

## Review Style

- Be direct and specific.
- Reference exact file names and section names when possible.
- Prioritize issues that affect implementation safety, data integrity, UX
  clarity, or product model correctness.
- If something is acceptable for Phase 1 but risky later, say so clearly.
