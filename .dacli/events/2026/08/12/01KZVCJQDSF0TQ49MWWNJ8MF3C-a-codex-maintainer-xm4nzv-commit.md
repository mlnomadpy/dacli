---
id: 01KZVCJQDSF0TQ49MWWNJ8MF3C
kind: event
event_kind: commit
created: 2026-08-12T16:23:39Z
created_by: a-codex-maintainer-xm4nzv
about: "[[t-01KZV16CQG6QCK4689FH58TZ7M]]"
origin: agent
applied: true
---
dc70055 364: serialize direct task mutations

Add a deterministic two-process task-check regression and route direct
read-modify-write paths through store.WithTask so every writer rereads under
the per-task file lock.

Red proof (pre-fix mutation):
--- FAIL: TestTaskCheckConcurrentProcessesPreserveDifferentCriteria
taskcheck_concurrency_test.go:56: persisted acceptance = [1/2], want [2/2]
role: codex-maintainer
