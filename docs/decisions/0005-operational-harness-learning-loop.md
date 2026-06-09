# ADR-0005: Operational Harness Learning Loop

- **Status:** accepted
- **Date:** 2026-06-09
- **Deciders:** Tran Thien, Codex
- **Tags:** process, governance, harness, agents, qa

## Context

Astraler Skillbox now has a mature repo-native operating harness:
`AGENTS.md`, governance and orchestration playbooks, documentation rules, QA
bank, and review roles. Recent work showed that the harness catches real drift,
but also that operational mistakes repeat:

- tmux prompts can remain in the input area and require a second Enter;
- reviewer and QA routing is sometimes applied late;
- terminology and QA tag drift can survive build checks;
- lessons from one slice can remain in chat instead of becoming durable rules;
- too much ceremony can make small work heavier than needed.

The project needs a controlled way to learn from execution without letting an
agent write arbitrary new rules into governance.

## Decision

Adopt an operational harness learning loop:

- Keep governance tool-agnostic and phase-gated.
- Use `/goal` only for long, stateful, cross-phase, or high-context handoffs.
  Do not use it as a mandatory wrapper for every task.
- Add a repo-native retrospective path:
  `.scratch/<date>-harness-retro.md` -> candidate lesson -> promote only if
  recurring or high-risk -> update playbook/ADR -> independent review.
- Add small tmux helper scripts as an Agent-Computer Interface for common
  handoff operations:
  - `scripts/harness/agent-send.sh`
  - `scripts/harness/agent-status.sh`
- Add a QA screen taxonomy so UI labels, QA tags, routes, and component names
  have one canonical registry.
- Keep task/feature/release closing checklists outside the project-level
  governance core. They may exist as phase-specific playbooks later, but this
  ADR does not add one.

## Alternatives Considered

- **Let agents edit governance directly after every mistake** - rejected because
  one-off lessons would accumulate into brittle ceremony.
- **Rely only on chat memory** - rejected because future agents and reviewers
  cannot inspect or review transient chat state.
- **Make `/goal` mandatory for all work** - rejected because small, clear tasks
  would become slower and noisier without reducing risk.
- **Create a broad "project closing" checklist now** - rejected because the
  owner clarified that closing belongs to task, feature, or release scope, not
  the always-on project governance layer.

## Consequences

**Positive:**

- Repeated operational failures have a durable path into the harness.
- tmux handoffs become less dependent on operator memory.
- QA tag/screen drift has a canonical registry to check.
- The harness can improve over time without turning every lesson into policy.

**Negative / cost:**

- A small number of extra docs and helper scripts must be maintained.
- Agents must distinguish candidate lessons from accepted rules.
- Helper scripts still require human inspection; they are not a full orchestration
  system.

**Neutral / to monitor:**

- If helper scripts become complex, move them behind tests or reduce scope.
- If `/goal` usage drifts toward ceremony, tighten the usage rules again.
- If the QA taxonomy becomes stale, Quinn should block QA bank changes until it
  is updated.

## Implementation Notes

- Update `docs/playbooks/governance-project.md` with the learning-loop rule and
  review routing.
- Update `docs/playbooks/agent-orchestration.md` with helper script usage.
- Add `docs/qa/screen-taxonomy.md` and reference it from QA maintenance.
- Add shell helpers under `scripts/harness/`.
- Update `mkdocs.yml` to publish the taxonomy.

## Verification

- `python3 -m mkdocs build --strict`
- QA YAML parse for `docs/qa/**/*.yaml` excluding run folders
- `shellcheck` if available for harness scripts; otherwise run script smoke
  checks with safe targets and `bash -n`
- Larry review for process/script changes
- Quinn review for QA taxonomy changes

## References

- [Agent Context And Anti-Hallucination Routing](./0003-agent-context-and-anti-hallucination-routing.md)
- [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [SWE-agent: Agent-Computer Interfaces Enable Automated Software Engineering](https://arxiv.org/abs/2405.15793)
- [LangGraph Persistence](https://langchain-ai.github.io/langgraph/concepts/persistence/)
