---
id: t-01KZYA0JMT34QFS2YASXJVHBJB
kind: task
created: 2026-08-13T19:36:31Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 605
  repo: mlnomadpy/dacli
---
# Add retry and dead-letter audit state to queues and stage gates
## So that
retries are classified and repeat transitions cannot silently duplicate work
## Acceptance
- [x] queue and stage-gate transitions carry stable idempotency keys and replaying one transition is a no-op
- [x] retryable and terminal failures are classified explicitly, with terminal failures persisted in inspectable dead-letter state
- [x] every retry, dead-letter, and successful transition appends an attributed audit event, with tests for each path
## Log
- 2026-08-13T19:56:08Z claimed by a-codex-maintainer-vjkw4t
- 2026-08-13T20:12:00Z owner review rejected commit e922104 for durability: success mutates state before its receipt, terminal writes its receipt before mutating state, and the stat-then-write check permits concurrent duplicates. Add injected-write-failure and concurrent-same-key tests, then implement an atomic pending/applied recovery protocol before pushing this branch.
- 2026-08-13T20:41:53Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T20:41:53Z accepted by a-root
- 2026-08-13T20:41:53Z verified by `go test -race ./internal/features/queues ./internal/features/stagegate ./internal/store` (exit 0) in branch main at 10bc086 — proves that tree builds, not that the work is in trunk
- 2026-08-13T20:41:53Z deliverable: dacli/431-add-retry-and-dead-letter-audit-state-to-queues-and-stage-gates is merged into main
- 2026-08-13T20:41:53Z completed by a-root
- 2026-08-13T20:44:30Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/617 (event 01KZYDBDP0PXT21NWHQNW4VKTS)
