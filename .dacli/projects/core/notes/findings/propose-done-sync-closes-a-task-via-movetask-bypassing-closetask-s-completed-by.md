---
id: f-propose-done-sync-closes-a-task-via-movetask-bypassing-closetask-s-completed-by
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-s12cpg
about: "[[t-01KZ6SAYED1SGWXJE4ZZQNKMCK]]"
source_event: 01KZ6SNV79GSAS8PEYB48W14JR
---
# propose:done sync closes a task via MoveTask, bypassing CloseTask's completed-by stamp and acceptance verification
A non-owner running 'dacli task done <ref>' cannot mutate, so planning.go:389 files an EventProposeStatus with body 'propose: done' and returns BEFORE the owner-path acceptance check (planning.go:396-406). The owner's next eventlog.Sync auto-applies it (execution.go:1017 runs Sync between supervisor turns; res.Applied>0). apply() for EventProposeStatus (sync.go:146-164) resolves target=done from model.AllStatuses and calls store.MoveTask(w,t,done) DIRECTLY -- not store.CloseTask. Consequences: (1) NO 'completed by' stamp is written (only 'status done proposed by X, applied'), so calibration.logSpan (calibration.go:368) returns ok=false and the done task is SILENTLY EXCLUDED from calibration samples (calibration.go:155), and doctor's LogHasStamp('completed by') flags it as a broken claim->completion span. (2) NO acceptance verification runs, so a task with unchecked acceptance boxes is moved to done/ -- a 'done' no check supports. This is exactly the E1 drift store.go:1188-1201 says CloseTask (task 037) made impossible: 'no path can close a task without the stamp ... cannot recur' -- reintroduced on the event-sync path. It is also inconsistent with the sibling accept-propose EventComment path, which Sync deliberately leaves pending so verified 'dacli accept' can stamp+verify (sync.go:166-176). Evidence it is exercised: many .dacli/events/**/*-propose-status.md carry 'propose: done'.
