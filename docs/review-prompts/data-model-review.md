# Data Model Review Prompt

You are reviewing the Astraler Skillbox data model.

Astraler Skillbox is a GUI-first local control center for managing AI agent
skills across many projects and agent providers. It uses a user-configured Skill
Host Folder as the local source of truth for skill content, and SQLite for app
metadata.

## Read These Files In Order

Read the project docs in this exact order before reviewing:

1. `README.md`
2. `docs/index.md`
3. `docs/01-product-brief.md`
4. `docs/02-product-notes.md`
5. `docs/03-information-architecture.md`
6. `docs/04-user-flows.md`
7. `docs/05-edge-cases-and-ux-states.md`
8. `docs/06-data-model.md`

## Review Goal

Review whether `docs/06-data-model.md` is a strong foundation for the product
described in the previous docs.

Focus on whether the data model supports:

- GUI-first product behavior.
- A user-configured Skill Host Folder.
- Provider global-level skills/config separate from project-level installs.
- Skill sources from GitHub, Vercel skills, local, and manual sources.
- Multiple projects.
- Multiple provider conventions per project.
- Installs by `symlink`, `rsync/copy`, `direct`, and external/broken symlink
  states.
- Fetch, update, and sync workflows.
- Edge cases and UX states documented in `docs/05-edge-cases-and-ux-states.md`.
- Future Phase 2 skill format conversion.

## Questions To Answer

Please answer these questions:

1. What is missing from the data model?
2. What is over-modeled or unnecessarily complex?
3. Which relationships look wrong, weak, or ambiguous?
4. Which statuses/enums overlap or need clearer separation?
5. Can every user flow in `docs/04-user-flows.md` be represented cleanly?
6. Can the edge cases in `docs/05-edge-cases-and-ux-states.md` be represented
   cleanly?
7. Is SQLite appropriate for this model? If yes, why? If not, what would you
   change?
8. Does the model leave enough room for provider adapters?
9. Does the model leave enough room for Phase 2 skill format conversion?
10. What should be changed before implementation begins?

## Required Output File

Write your review result to this file:

```text
docs/review-results/data-model-review.md
```

If the file already exists, overwrite it with the latest review.

## Required Output Format

Use this exact structure:

```markdown
# Data Model Review Result

## Reviewer

- Agent/model:
- Review date:
- Context used:

## Executive Summary

Short summary of the review result.

## Critical Issues

List issues that should be fixed before implementation.

## Suggested Improvements

List useful improvements that are not blockers.

## Missing Concepts Or Tables

List any missing entities, tables, fields, or relationships.

## Over-Modeled Or Risky Areas

List anything too complex, premature, or likely to create maintenance burden.

## Enum And Status Review

Review the proposed statuses/enums and identify overlaps or naming issues.

## User Flow Coverage

For each user flow, say whether the data model supports it:

1. First-Time Setup
2. Add Project
3. Scan Project
4. Scan Global Skills
5. Install Skill To Project
6. Fetch Skill Updates
7. Update Skill Host Folder
8. Sync Rsync / Copy Project
9. Switch Install Mode
10. Remove Skill From Project
11. Add Skill To Skill Host Folder
12. Change Skill Host Folder
13. App Startup

## Edge Case Coverage

Summarize whether the model covers:

- Skill Host Folder states
- Global Skill states
- Project states
- Install states
- Fetch and update states
- Provider states
- Database and app state
- UI/UX states

## SQLite Assessment

Assess whether SQLite is appropriate and mention any schema/migration concerns.

## Provider Adapter Readiness

Assess whether the model is ready for provider-specific path and convention
handling.

## Phase 2 Conversion Readiness

Assess whether the model can evolve to support skill format conversion.

## Recommended Data Model Changes

Provide concrete recommended changes. Use bullets. Include table/field names
where relevant.

## Open Questions For The Product Owner

List questions that require product decisions.

## What Looks Solid

List parts of the data model that are well designed.
```

## Review Style

- Be direct and specific.
- Reference exact file names and section names when possible.
- Do not rewrite the entire data model.
- Prioritize issues that affect implementation, data integrity, or UX behavior.
- If something is acceptable for Phase 1 but risky later, say so clearly.
