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
- [ ] roles and skills carry a version: field; dacli role show / skill show print the current version and a changelog derived from git history (who changed it, when, what)
- [ ] a role/skill change bumps or prompts to bump the version; the version is stable and human-legible (e.g. semver or an incrementing v1/v2)
- [ ] committed by an agent; build + test green
## Log
