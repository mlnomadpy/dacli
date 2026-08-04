---
id: t-01KZ53KN27Y0WCPV4G9HNSC79W
kind: task
created: 2026-08-04T00:43:35Z
created_by: a-go-auditor-7f03df
owner: a-go-auditor-7f03df
priority: should
---
# acquireSeqLock steal is ownerless and can break task-seq mutual exclusion
## So that
concurrent task adds under a slow holder cannot land two tasks on the same NNN
## Acceptance
- [ ] acquireSeqLock only removes a lock it owns (owner token or PID+staleness), never a live holder's or a thief's file
- [ ] the post-deadline path backs off instead of hot-spinning, and steal only fires against a demonstrably stale lock
- [ ] a test exercises the steal path (a pre-existing/stale .seq.lock and a holder exceeding seqLockTimeout) and asserts distinct seqs, extending TestCreateTaskConcurrentGetsDistinctSeqs
## Log
