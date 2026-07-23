---
id: t-01KY79ESYY289THPNDGF402CCX
kind: task
created: 2026-07-23T10:48:35Z
created_by: a-534c4gav5p
owner: a-root
priority: should
---
# loop --pr must not force-close tasks whose implementer spawn was refused or failed
## So that
a spawn-refused task stays in the backlog for retry instead of being silently marked done with no implementation
## Acceptance
- [x] runCycle records per-task spawn outcome (spawn command exit status, and/or existence of the task's dacli/<seq>-slug branch after wait) instead of treating the whole batch as built
- [x] the --pr LAND step calls 'accept <seq> --force' ONLY for tasks whose implementer spawn succeeded; a task whose spawn was refused/failed is left open (not closed, not box-checked) so the next cycle re-picks it
- [x] a regression test in internal/features/orchestration drives a cycle where one batch task's spawn is refused and asserts that task remains open (not moved to done) while a successfully-spawned sibling is closed
- [x] go build ./... clean and go test ./internal/... green
## Log
- 2026-07-23T13:23:03Z claimed by a-c7sr25jttk
- 2026-07-23T13:29:22Z adopted by a-root (owner a-534c4gav5p orphaned)
- 2026-07-23T13:29:22Z accepted by a-root
- 2026-07-23T13:29:22Z completed by a-root
