---
id: f-285-complete-on-branch-dacli-285-runstilllive-rejects-alive-recycled-leader
kind: note
note_kind: finding
created: 2026-08-04T20:12:19Z
created_by: a-maintainer-1e8nm5
about: "[[285]]"
severity: moderate
---
# 285 complete on branch dacli/285-...: runStillLive rejects alive-recycled leader
Commit 83ededb by a-maintainer-1e8nm5 (maintainer). Touched only internal/features/execution/execution.go and internal/features/execution/runstilllive_unix_test.go. FIX: runStillLive (execution.go:2011) now, after AliveRecord(rec)==false, checks procmon.Alive(rec.PID): if the leader PID is ALIVE (recycled onto an unrelated process whose ProcStart != rec.PIDStart) it returns false BEFORE consulting GroupAlive, so a recycled pgid can no longer resurrect a finished run as live. The dacli-177 path (leader PID genuinely absent, real surviving children) still falls through to GroupAlive and returns true. Reuses existing primitives (procmon.Alive/AliveRecord/AliveIdentity) — no second identity mechanism, no change to KillTree/liveAgents/cmdKill. A comment notes the residual left for follow-up: a truly-dead leader whose bare pgid integer is reused by an unrelated group. TESTS (runstilllive_unix_test.go): TestRunStillLiveRejectsAliveRecycledLeader (live PID + mismatched PIDStart + live PGID => runStillLive false) and TestRunsPruneDoesNotSkipRecycledLeader (cmdRunsPrune does NOT print the 'still live ... pruning it would orphan a running agent' skip and prunes the ghost). VERIFIED by reproduction: temporarily reverting the Alive(rec.PID) guard makes BOTH new tests FAIL while TestRunStillLiveDetectsLiveGroupWithDeadLeader stays green; restoring it makes all pass. go build ./... clean, go test ./internal/... all green (incl. procmon + execution), go vet clean, gofmt -l internal/ empty. PR-first off: owner accepts via dacli accept 285 then integrate/merge.
