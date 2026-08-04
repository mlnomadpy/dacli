---
id: t-01KZ6SX3725TJ69J51DK1P6PVM
kind: task
created: 2026-08-04T16:32:27Z
created_by: a-go-auditor-a2hqh6
owner: a-go-auditor-a2hqh6
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 7}"
---
# Identity-check runStillLive's GroupAlive fallback so a recycled pgid can't resurrect a finished run as live
## So that
'dacli wait <ref>' and 'dacli runs prune' stop treating a long-finished agent as still-running when its group-leader PID/PGID has been recycled onto an unrelated live process group during a long unattended run
## Acceptance
- [ ] runStillLive (internal/features/execution/execution.go:2011-2016) no longer trusts GroupAlive(rec.PGID) when the recorded leader PID is ALIVE but its identity does not match rec.PIDStart (recycled leader): in that case it must return false, mirroring procmon.AliveIdentity's leader guard from task 049. The legitimate dacli-177 case — leader PID genuinely absent (Alive(rec.PID)==false) with real surviving children — must still return true (TestRunStillLiveDetectsLiveGroupWithDeadLeader stays green)
- [ ] A new unit test proves the recycle case: a Record whose PID names a LIVE process whose ProcStart differs from the record's PIDStart, and whose PGID equals that live process's group, makes runStillLive return false (dead run, not resurrected). Add a sibling test asserting cmdRunsPrune does NOT print the 'still live … pruning it would orphan a running agent' skip for such a record
- [ ] The fix reuses the existing procmon identity primitives (Alive / ProcStart / AliveIdentity) rather than adding a second identity mechanism; no change to KillTree/liveAgents/cmdKill (already identity-checked via AliveRecord). Note in a comment the residual not covered here: a TRULY-dead leader whose bare pgid integer is reused by an unrelated group is left for a follow-up (children legitimately have unrelated start times), so this task closes only the alive-recycled-leader hole
- [ ] go build ./... clean and go test ./internal/... green (incl. internal/procmon and internal/features/execution)
## Log
