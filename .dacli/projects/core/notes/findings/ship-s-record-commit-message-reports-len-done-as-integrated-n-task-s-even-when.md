---
id: f-ship-s-record-commit-message-reports-len-done-as-integrated-n-task-s-even-when
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-8b74h81fsz
about: [[t-01KY59FNENE0C7CRCSXM3WH9DD]]
source_event: 01KY59PV8TAQQVHZ26ZFATPQGB
---
# ship's record commit message reports len(done) as 'integrated N task(s)' even when zero branches actually merged
ship.go:136 calls commitRecord(ctx, w, id, len(done)) and commitRecord (ship.go:182) builds the message 'ship: record workspace after integrating %d task(s)' from that count. The number is the DONE-task count, not the count of branches actually merged by integrate — tasks with no branch are skipped (lifecycle.go:267-270), and (per the half-ship finding) integrate can no-op entirely while still exiting 0. So the committed history line overstates what was integrated (the current HEAD 'ship: record workspace after integrating 40 task(s)' is this count, not 40 real merges). Fix: thread integrate's actual merged-count back to the record message, or word it as 'recording after ship of N done task(s)'.
