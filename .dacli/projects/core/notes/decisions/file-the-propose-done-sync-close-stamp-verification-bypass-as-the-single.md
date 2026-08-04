---
id: d-file-the-propose-done-sync-close-stamp-verification-bypass-as-the-single
kind: note
note_kind: decision
created: 2026-08-04T16:28:50Z
created_by: a-go-auditor-s12cpg
about: "[[279]]"
---
# File the propose:done sync-close stamp/verification bypass as the single highest-value finding
## Chose
File the propose:done sync-close stamp/verification bypass as the single highest-value finding
## Rejected
Filing the CreateNote same-title TOCTOU (store.go:1337 stat-then-write can lose one of two concurrent same-title notes), or the concurrent-Sync MoveTask ENOENT spurious error, or MarkApplied's non-fsync read-modify-write
## Because
The rejected items are low-impact: CreateNote's collision path is only reachable when two writers race on the SAME title AND both stat before either renames, and in the real flow notes are materialized only by the single owner's serialized Sync, so concurrency is near-nil; concurrent Sync producing a MoveTask ENOENT is a self-healing transient (event stays pending, re-applies). mdstore WriteFile is already atomic (temp+rename) and every read fault in store/eventlog is deliberately surfaced-not-swallowed with justifying comments. The propose:done path is the one place the plumbing produces a RECORD THAT DISAGREES WITH REALITY: a task in done/ with unmet acceptance and no completed-by stamp, silently corrupting calibration's input and defeating the supervisor's own unmet==done gate (execution.go:1026) because MoveTask already ran. It reintroduces the exact E1 drift task 037's CloseTask was built to make impossible, and is evidence-grounded (file:line + a non-owner 'task done' repro + real propose:done events on disk) and fully unit-testable.
