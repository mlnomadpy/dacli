---
id: 01KZ53KFGGWW6ENE87HJS2CDV1
kind: event
event_kind: finding
created: 2026-08-04T00:43:29Z
created_by: a-go-auditor-7f03df
about: "[[t-01KZ53CHDGSA2DR1Y0WWGAJGKK]]"
origin: agent
applied: true
---
acquireSeqLock steal is ownerless and un-throttled, so it breaks task-seq mutual exclusion under contention

internal/store/store.go:420-438. The cross-process seq lock is a bare O_EXCL marker file with NO owner token. Two defects on the steal path: (1) Ownerless release/steal — both the release closure (store.go:427 'func(){ os.Remove(path) }') and the deadline steal (store.go:433 os.Remove) delete whatever file is present, not one the caller owns. After waiter B steals A's lock (removes at :433, re-creates at :424), A's deferred unlock() later removes B's file while B is still mid-CreateTask, letting a third waiter enter the critical section — mutual exclusion is gone. (2) No backoff after the deadline — the 5ms sleep at store.go:436 sits only on the !After(deadline) branch; once time.Now().After(deadline) is true every iteration is OpenFile->Remove->continue with no sleep, a CPU hot-spin, and two waiters both past the deadline livelock stealing each other's file and can both O_EXCL-succeed in the same window. Result: both run 'seq = max+1' over identical on-disk state (store.go:349-355) and write two task files sharing one NNN — the exact dacli-209 collision the lock exists to prevent (FindTask reports the ref ambiguous). Trigger: a slow/paused holder exceeding seqLockTimeout=5s (store.go:413) while >=2 siblings file concurrently, e.g. the swarm's batch task adds. TestCreateTaskConcurrentGetsDistinctSeqs (task_seq_test.go) only exercises fast holders, so the steal path is entirely untested.
