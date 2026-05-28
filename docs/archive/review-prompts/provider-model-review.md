# Provider Model Review Prompt

You are reviewing the Astraler Skillbox Provider Model.

Astraler Skillbox is a GUI-first local control center for managing AI agent
skills across many projects and agent providers. Provider adapters are the layer
that lets Skillbox detect, scan, and install skills into different provider
folder conventions without hardcoding provider logic across the app.

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
9. `docs/07-schema-dictionary.md`
10. `docs/08-provider-model.md`

## Review Goal

Review whether `docs/08-provider-model.md` is a strong foundation for provider
adapters in Skillbox.

Focus on whether the Provider Model supports:

- Multiple agent providers in one project.
- Provider global locations and global skills/config.
- Provider detection from project filesystem conventions.
- Multiple provider path candidates with different purposes.
- Safe install target resolution.
- Clear separation between provider adapter logic and core Skillbox policy.
- Unsupported/experimental provider states.
- UI provider badges/icons/statuses.
- Future Phase 2 skill format conversion.

## Important Context

Some provider conventions may still require research. If you know current
provider conventions for Claude, Codex, opencode, Antigravity CLI, or generic
`.agents`, call out any likely mismatch or missing path. If you are not certain,
label it as an assumption rather than a fact.

Do not browse the web unless explicitly allowed by the user. If you rely on
memory or local docs only, state that limitation in the review.

## Questions To Answer

Please answer these questions:

1. What is missing from the Provider Model?
2. What is over-modeled or unnecessarily complex?
3. Is the adapter boundary clear enough?
4. Are `provider_definitions` and `provider_path_candidates` flexible enough?
5. Does the detection flow handle no provider, multiple providers, unsupported
   provider, invalid structure, and format unknown states?
6. Is install target resolution safe enough?
7. Does the model accidentally mix provider-specific policy with core Skillbox
   policy?
8. Is the unsupported provider policy strict enough to avoid unsafe writes?
9. Are the initial provider assumptions risky or acceptable?
10. Does the model leave enough room for Phase 2 skill format conversion?
11. What should be changed before implementation begins?

## Required Output File

Write your review result to this file:

```text
docs/archive/review-results/provider-model-review.md
```

If the file already exists, overwrite it with the latest review.

## Required Output Format

Use this exact structure:

```markdown
# Provider Model Review Result

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

List issues that should be fixed before implementation.

## Suggested Improvements

List useful improvements that are not blockers.

## Missing Concepts Or Sections

List any missing concepts, sections, provider states, or adapter responsibilities.

## Over-Modeled Or Risky Areas

List anything too complex, premature, or likely to create maintenance burden.

## Adapter Boundary Review

Assess whether provider adapters and core Skillbox logic are separated cleanly.

## Provider Path Candidate Review

Assess whether `provider_path_candidates` and `purpose` values are sufficient.

## Detection Flow Coverage

Assess whether the detection flow handles:

- No provider detected
- One provider detected
- Multiple providers detected
- Unsupported provider
- Missing provider path
- Invalid structure
- Format unknown
- Provider path changed or moved

## Install Target Resolution Review

Assess whether install target resolution is safe enough.

## Initial Provider Assumptions Review

Review the assumptions for:

- Generic `.agents`
- Claude
- Codex
- opencode
- Antigravity CLI

Clearly mark uncertain claims as assumptions.

## UI/UX Review

Assess whether provider badges, icons, support states, warnings, and grouping are
covered well enough for UI design.

## Phase 2 Conversion Readiness

Assess whether the provider model can evolve to skill format conversion.

## Recommended Changes

Provide concrete recommended changes. Reference file sections where relevant.

## Open Questions For The Product Owner

List questions that require product decisions.

## What Looks Solid

List parts of the Provider Model that are well designed.
```

## Review Style

- Be direct and specific.
- Reference exact file names and section names when possible.
- Do not rewrite the entire Provider Model.
- Prioritize issues that affect implementation safety, data integrity, provider
  correctness, or UX clarity.
- If something is acceptable for Phase 1 but risky later, say so clearly.
