---
id: d-added-a-cross-process-o-excl-lock-file-seq-lock-guarding-createtask-s-scan-then
kind: note
note_kind: decision
created: 2026-08-04T00:09:51Z
created_by: a-fixer-q3xzg4
about: "[[209]]"
---
# added a cross-process O_EXCL lock file (.seq.lock) guarding CreateTask's scan-then-write
## Chose
added a cross-process O_EXCL lock file (.seq.lock) guarding CreateTask's scan-then-write
## Rejected
in-memory sync.Mutex, or O_EXCL on the final task-file path itself
## Because
CreateTask races across separate OS processes (concurrent dacli invocations), not goroutines in one process, so an in-process mutex would not help; O_EXCL on the final path fails to catch collisions because two racers pick the same seq but different slugs, producing different filenames that never collide on their own path -- only a lock around the scan+write critical section (keyed per-project, self-healing via a 5s steal timeout for a crashed holder) closes the race
