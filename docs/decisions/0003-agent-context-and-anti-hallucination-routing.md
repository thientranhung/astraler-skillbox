# ADR-0003: Agent Context And Anti-Hallucination Routing

- **Status:** accepted
- **Date:** 2026-06-02
- **Deciders:** owner + Codex
- **Tags:** process, documentation, governance, qa

## Context

Agents start many tasks from a fresh context. Loading the full project governance
playbook manually at the beginning of every session is costly and easy to skip,
but moving every process rule into `AGENTS.md` would make the always-loaded
context too large. The repository already has detailed governance, QA, docs, and
agent-orchestration playbooks; the missing piece is compact routing plus a small
anti-hallucination guardrail that works in a fresh context.

## Decision

Keep `AGENTS.md` concise and add only task routing plus a short mandatory
pre-edit/pre-verdict guardrail.

Add `docs/context-map.md` as the compact discovery map for code, docs, contracts,
and QA paths.

Keep full process rules in existing playbooks:

- `docs/playbooks/governance-project.md` for phase gates, review, PR, ownership,
  and the full implementer/reviewer anti-hallucination checklist.
- `docs/qa/governance.md` for QA evidence, result semantics, clean GO, and the
  QA anti-hallucination checklist.
- `docs/playbooks/documentation.md` for source-of-truth and docs drift mapping.
- `docs/playbooks/agent-orchestration.md` for tmux/agent handoff operations.

Do not add a new Plan/Act mode. Superpowers, `/goal`, and the existing phase
gates already cover planning and approval.

## Alternatives Considered

- **Move the full governance playbook into `AGENTS.md`** — rejected because it
  increases always-loaded context and duplicates the source of truth.
- **Create a separate `.memory/`, `.rules/`, and `.generators/` tree** —
  rejected for now because this repo already has canonical docs, playbooks,
  `.scratch/`, `/goal`, and QA runs that cover those roles without another
  source-of-truth hierarchy.
- **Keep relying on manual playbook loading** — rejected because fresh agent
  contexts can miss required routing and anti-hallucination checks.
- **Add a new Plan/Act policy** — rejected because it duplicates Superpowers and
  the existing phase gates.

## Consequences

**Positive:**
- Fresh agent contexts get enough routing to start safely.
- Broad repository discovery should consume fewer tokens by starting from a map.
- Full governance and QA rules stay in their existing source-of-truth files.
- Anti-hallucination checks apply before both code edits and review/QA verdicts.

**Negative / chi phí:**
- `docs/context-map.md` must stay current when major folders or source-of-truth
  docs move.
- `AGENTS.md` gains a small amount of always-loaded text.

**Neutral / cần theo dõi:**
- If repeated exceptions or task workflows accumulate, add them to the smallest
  existing playbook surface before considering a new rules/generators tree.

## Implementation Notes

- Add `docs/context-map.md`.
- Update `AGENTS.md` with routing and the short guardrail.
- Update `docs/playbooks/governance-project.md` with the full implementer and
  reviewer checklist.
- Update `docs/qa/governance.md` with the QA verdict checklist.
- Update `docs/index.md` and `docs/decisions/index.md`.

## Verification

- Confirm all linked docs exist.
- Confirm no new process rule duplicates the full governance or QA playbooks.
- For future code or QA tasks, agents should start with `AGENTS.md`, then load
  only the relevant deeper playbook and `docs/context-map.md` when discovery is
  needed.

## References

- `AGENTS.md`
- `docs/context-map.md`
- `docs/playbooks/governance-project.md`
- `docs/qa/governance.md`
