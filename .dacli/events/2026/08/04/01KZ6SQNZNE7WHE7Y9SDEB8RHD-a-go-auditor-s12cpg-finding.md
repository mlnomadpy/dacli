---
id: 01KZ6SQNZNE7WHE7Y9SDEB8RHD
kind: event
event_kind: finding
created: 2026-08-04T16:29:30Z
created_by: a-go-auditor-s12cpg
about: "[[t-01KZ6SAYED1SGWXJE4ZZQNKMCK]]"
origin: agent
applied: true
---
279 audit coverage: discarded errors in core plumbing are justified-in-comment; read-modify-write sequences enumerated

Swept store, eventlog, mdstore, workspace, gitx. DISCARDED ERRORS: every ignored error I found is either surfaced or carries a justifying comment -- eventlog.List/Sync log+continue on unreadable events (eventlog.go:109-123,144-153); mdstore.WriteFile cleans up the temp file on every failure branch (mdstore.go:626-642); store.ListProjects/ListTasks/ListNotes skip a broken file to avoid hiding the rest, and ListNotes distinguishes absent-dir (nil) from I/O error (mdstore.go / store.go:176-196,807-819,1426-1445); gitx.Merge deliberately '_ =' the merge --abort (gitx.go:213) after collecting conflicts. No swallowed error becomes a wrong record EXCEPT the propose:done path filed as task 284. READ-MODIFY-WRITE sequences (all either atomic-rename or lock-guarded): mdstore.WriteFile = temp+rename (atomic, no fsync -- documented); store.MoveTask = os.Rename + stale-folder sweep (atomic move, store.go:1157-1180); store.CreateTask seq alloc = acquireSeqLock O_EXCL cross-process lock + gitTaskSeqCeiling (store.go:375-409); eventlog.MarkApplied + store.SaveTask/GradeFinding/SetEstimate/CheckAllAcceptance = ReadFile-mutate-WriteFile, all funneling through the atomic WriteFile (last-write-wins, single-owner in practice). The one RMW that is neither locked nor idempotent-safe under a true concurrent same-key race is CreateNote's stat-then-write same-title collision (store.go:1337) -- noted as low-impact (single-owner Sync serializes it) in the task-284 decision, not separately filed.
