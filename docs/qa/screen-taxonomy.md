# QA Screen Taxonomy

This file is the canonical registry for user-facing screen names used by the QA
bank. Use it when adding cases, selecting tags, renaming UI labels, or reviewing
screen-related docs drift.

## Rules

- `label` is the user-facing screen name used in `primary_screen`,
  `related_screens`, steps, expected UI, and reports.
- `tag` is the stable QA selection tag. Use lowercase kebab-case.
- `route` is the renderer route when the screen is route-backed.
- `component` is the current renderer component name. It may lag behind the
  user-facing label during safe renames.
- QA cases may use stable sub-surface labels in `related_screens` when a check
  targets a named section inside a route-backed screen.
- Historical `docs/qa/runs/**` evidence is append-only. Do not rewrite old run
  labels just because taxonomy changes.

## Screens

| Label | QA tag | Route | Component / source |
|---|---|---|---|
| Dashboard | `dashboard` | `/` | `apps/desktop/renderer/src/screens/dashboard-screen.tsx` |
| Host Skills | `host-skills` | `/skills` | `apps/desktop/renderer/src/screens/skills-library-screen.tsx` |
| Global Skills | `global-skills` | `/global` | `apps/desktop/renderer/src/screens/global-skills-screen.tsx` |
| Global Plugins | `global-plugins` | `/plugins` | `apps/desktop/renderer/src/screens/plugins-screen.tsx` |
| Projects | `projects` | `/projects` | `apps/desktop/renderer/src/screens/projects-screen.tsx` |
| Project Detail | `project-detail` | `/projects/:projectId` | `apps/desktop/renderer/src/screens/project-detail-screen.tsx` |
| Skill Detail | `skill-detail` | `/skills/:skillId` | `apps/desktop/renderer/src/screens/skill-detail-screen.tsx` |
| Settings | `settings` | `/settings` | `apps/desktop/renderer/src/screens/settings-screen.tsx` |
| About | `about` | `/about` | `apps/desktop/renderer/src/screens/about-screen.tsx` |
| Setup | `setup` | startup/setup flow | `apps/desktop/renderer/src/screens/setup-screen.tsx` |
| Startup Error | `startup-error` | startup error flow | `apps/desktop/renderer/src/screens/startup-error-screen.tsx` |
| Add Skill Wizard | `add-skill` | modal in Project Detail | `apps/desktop/renderer/src/features/projects/add-skill-wizard.tsx` |
| Provider Registry | `provider-registry` | Settings sub-surface | Settings provider registry section |
| Global Skill Detail | `global-skill-detail` | Global Skills sub-surface | Global Skills entry/detail state |
| App Launch | `app-launch` | startup lifecycle | launch/setup/startup-error flow |
| App Shell | `app-shell` | app layout | `apps/desktop/renderer/src/components/app-shell.tsx` |
| Packaged App | `packaged-app` | release artifact | built Electron app |
| Release CLI | `release-cli` | release workflow | GitHub release / local release commands |

## Rename Checklist

When a screen label changes:

1. Update this taxonomy first.
2. Update `docs/03-information-architecture.md` and
   `docs/09-ui-wireframes.md`.
3. Update QA `primary_screen`, `related_screens`, case prose, and tags.
4. Search active docs and QA bank for the old label and old tag.
5. Do not rewrite `docs/qa/runs/**` historical evidence.
6. Ask Quinn to review QA bank selection impact.
