---
id: t-01KZV16CQG6QCK4689FH58TZ7M
kind: task
created: 2026-08-12T13:04:41Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 5, pessimistic: 8}"
depends_on: [372, 375, 378]
github:
  issue: 457
  repo: mlnomadpy/dacli
---
# Fix cross-process lost updates in direct task mutations
## So that
Concurrent agents cannot silently erase one another's task state or acceptance evidence
## Acceptance
- [x] A deterministic cross-process test runs two simultaneous task-check mutations against different criteria and the persisted task ends at [2/2]
- [x] Every direct task read-modify-write path rereads and saves under store.WithTask or an equivalent per-task lock
- [x] The new regression test is proven red against the pre-fix mutation path and the failure line is recorded
- [x] go test -race ./... passes
## Log
- 2026-08-12T16:12:27Z claimed by a-codex-maintainer-xm4nzv
- 2026-08-12T16:29:15Z completed by a-root
