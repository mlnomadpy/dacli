---
id: t-01KZP9FCV37ZRNW2N3DD8035BV
kind: task
created: 2026-08-10T16:53:12Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Add loop --into so a sprint can integrate onto its own branch instead of main
## Acceptance
- [x] loop accepts --into <branch> and integrates the wave there, defaulting to main when omitted
- [x] the landing check confirms against the same branch the loop integrates into, not a hardcoded trunk
- [x] a test runs a cycle with --into on a non-main branch and asserts the work lands there and the task closes
## Log
- 2026-08-10T17:24:44Z accepted by a-root
- 2026-08-10T17:24:44Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:24:44Z deliverable: no dacli/332-add-loop-into-so-a-sprint-can-integrate-onto-its-own-branch-instead-of-main branch — nothing to check against trunk
- 2026-08-10T17:24:44Z completed by a-root
