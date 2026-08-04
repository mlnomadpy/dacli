---
id: t-01KZ53KN27Y0WCPV4G9HNSC79W
kind: task
created: 2026-08-04T00:43:35Z
created_by: a-go-auditor-7f03df
owner: a-root
priority: should
---
# acquireSeqLock steal is ownerless and can break task-seq mutual exclusion
## So that
concurrent task adds under a slow holder cannot land two tasks on the same NNN
## Acceptance
- [x] acquireSeqLock only removes a lock it owns (owner token or PID+staleness), never a live holder's or a thief's file
- [x] the post-deadline path backs off instead of hot-spinning, and steal only fires against a demonstrably stale lock
- [x] a test exercises the steal path (a pre-existing/stale .seq.lock and a holder exceeding seqLockTimeout) and asserts distinct seqs, extending TestCreateTaskConcurrentGetsDistinctSeqs
## Log
- 2026-08-04T11:33:10Z adopted by a-root (owner a-go-auditor-7f03df orphaned)
- 2026-08-04T11:33:44Z accepted by a-root
- 2026-08-04T11:33:44Z verified by `go test -race -count=2 ./internal/store/` (exit 0)
- 2026-08-04T11:33:44Z completed by a-root
- 2026-08-04T18:18:12Z claimed by a-root (event 01KZ639EPEKEVM62QZ9P5BP0T2)
- 2026-08-04T18:18:12Z status done proposed by a-root, applied (event 01KZ65TBPR45G0HQCYT3NFRQET)
- 2026-08-04T18:18:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/290 (event 01KZ65TJKHE423W7A7R5AQTWQG)
