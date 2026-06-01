# CLAUDE.md

Claude-Code-specific delta. All shared project knowledge lives in `AGENTS.md`.

> **MUST READ first:** [`AGENTS.md`](./AGENTS.md)

## Claude Code Notes

- **Project skill**: `.claude/skills/astraler-qa` is a symlink to `.agents/skills/astraler-qa`. Use it for QA bank planning/execution (`docs/qa/`), Electron smoke via CDP/agent-browser, evidence collection, and QA run reports.
- **No project-level hooks or slash commands defined** beyond defaults. If you add any under `.claude/`, document them here.
- **Scratchpad**: `.scratch/` (gitignored) for long task briefs, file-backed `/goal` handoffs, and temporary drafts. Use date-prefixed lowercase kebab-case filenames per `AGENTS.md`.
