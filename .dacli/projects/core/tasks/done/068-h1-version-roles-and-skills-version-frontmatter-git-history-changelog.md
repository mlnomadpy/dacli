---
id: t-01KY5JGGEMXBG4TDWNH93NHACD
kind: task
created: 2026-07-22T18:48:19Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# H1: version roles and skills — version frontmatter + git-history changelog
## Acceptance
- [x] roles and skills carry a version: field; dacli role show / skill show print the current version and a changelog derived from git history (who changed it, when, what)
- [x] a role/skill change bumps or prompts to bump the version; the version is stable and human-legible (e.g. semver or an incrementing v1/v2)
- [x] committed by an agent; build + test green
## Log

- 2026-07-22T20:08:55Z claimed by a-64p4b0yykq
- 2026-07-22T20:16:39Z accepted by a-root
- 2026-07-22T20:16:39Z completed by a-root
## Scope (STRICT): internal/store,internal/features/teamops,internal/features/skillforge only. add a version: frontmatter field to roles and skills; dacli role show / skill show print the version + a git-history changelog (git log on the file). Keep it small and typed.
Do NOT git add -A. Non-worktree-safe. dacli commit; box-check owner-only.
